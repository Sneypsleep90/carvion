//go:build integration

package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDeploymentRepositoryPersistsOrderedAppendOnlyAudit(t *testing.T) {
	ctx, pool := deploymentIntegrationPool(t)
	defer pool.Close()
	repository := NewDeploymentRepository(pool)
	projectID := createDeploymentIntegrationProject(t, ctx, pool)

	job, err := repository.CreateDeploymentJob(ctx, domain.DeploymentJob{
		ProjectID: projectID, Mode: domain.DeploymentModeTest, PackageSHA256: strings.Repeat("a", 64),
		State: domain.DeploymentStateQueued,
	})
	if err != nil {
		t.Fatalf("CreateDeploymentJob() error = %v", err)
	}
	if len(job.Events) != 1 || job.Events[0].Sequence != 1 || job.Events[0].State != domain.DeploymentStateQueued {
		t.Fatalf("created job events = %+v, want initial queued event at sequence 1", job.Events)
	}

	invalid := job
	invalid.State = domain.DeploymentStateSimulating
	if err := repository.TransitionDeploymentJob(ctx, invalid, domain.DeploymentEvent{
		Stage: "simulate", Message: "invalid transition must not be persisted",
	}); err == nil {
		t.Fatal("TransitionDeploymentJob() accepted queued → simulating")
	}
	unchanged, err := repository.GetDeploymentJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetDeploymentJob() after rejected transition error = %v", err)
	}
	if unchanged.State != domain.DeploymentStateQueued || len(unchanged.Events) != 1 {
		t.Fatalf("rejected transition changed persisted job: %+v", unchanged)
	}

	preflighting := unchanged
	preflighting.State = domain.DeploymentStatePreflighting
	preflighting.Preflight = domain.DeploymentPreflight{Checks: []domain.DeploymentCheck{}}
	if err := repository.TransitionDeploymentJob(ctx, preflighting, domain.DeploymentEvent{
		Stage: "preflight", Message: "Synthetic preflight started",
	}); err != nil {
		t.Fatalf("TransitionDeploymentJob() queued → preflighting error = %v", err)
	}
	productionJob, err := repository.CreateDeploymentJob(ctx, domain.DeploymentJob{
		ProjectID: projectID, Mode: domain.DeploymentModeProduction, PackageSHA256: strings.Repeat("c", 64),
		State: domain.DeploymentStateQueued,
	})
	if err != nil {
		t.Fatalf("CreateDeploymentJob() production job error = %v", err)
	}
	productionPreflight := productionJob
	productionPreflight.State = domain.DeploymentStatePreflighting
	if err := repository.TransitionDeploymentJob(ctx, productionPreflight, domain.DeploymentEvent{
		Stage: "preflight", Message: "Production readiness review started",
	}); err != nil {
		t.Fatalf("TransitionDeploymentJob() production preflight error = %v", err)
	}
	blocked := productionPreflight
	blocked.State = domain.DeploymentStateBlocked
	blocked.Preflight = domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{{
		ID: "production-disabled", Status: domain.DeploymentCheckBlocked, Required: true, Message: "Production Mode is unavailable",
	}}}
	if err := repository.TransitionDeploymentJob(ctx, blocked, domain.DeploymentEvent{
		Stage: "blocked", Message: "Production Mode is blocked by Core policy",
	}); err != nil {
		t.Fatalf("TransitionDeploymentJob() → blocked error = %v", err)
	}

	newRepository := NewDeploymentRepository(pool)
	persisted, err := newRepository.GetDeploymentJob(ctx, productionJob.ID)
	if err != nil {
		t.Fatalf("GetDeploymentJob() from new repository error = %v", err)
	}
	if persisted.State != domain.DeploymentStateBlocked || persisted.Mode != domain.DeploymentModeProduction {
		t.Fatalf("persisted blocked job = %+v", persisted)
	}
	if persisted.Preflight.Ready || len(persisted.Preflight.Checks) != 1 || persisted.Preflight.Checks[0].ID != "production-disabled" {
		t.Fatalf("persisted preflight = %+v, want production blocker", persisted.Preflight)
	}
	if len(persisted.Events) != 3 {
		t.Fatalf("persisted production events = %d, want 3", len(persisted.Events))
	}
	for i, event := range persisted.Events {
		if event.Sequence != int64(i+1) {
			t.Fatalf("event sequence at index %d = %d, want %d", i, event.Sequence, i+1)
		}
	}

	if _, err := pool.Exec(ctx, `UPDATE deployment_job_events SET message = 'tampered' WHERE job_id = $1::uuid AND sequence = 1`, productionJob.ID); err == nil {
		t.Fatal("deployment_job_events accepted UPDATE, want append-only trigger")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM deployment_job_events WHERE job_id = $1::uuid AND sequence = 1`, productionJob.ID); err == nil {
		t.Fatal("deployment_job_events accepted DELETE, want append-only trigger")
	}

	summaries, err := newRepository.ListDeploymentJobs(ctx, 10)
	if err != nil {
		t.Fatalf("ListDeploymentJobs() error = %v", err)
	}
	found := false
	for _, summary := range summaries {
		if summary.ID == productionJob.ID {
			found = summary.State == domain.DeploymentStateBlocked && summary.ProjectID == projectID
		}
	}
	if !found {
		t.Fatalf("ListDeploymentJobs() did not return blocked job %q: %+v", productionJob.ID, summaries)
	}
}

func deploymentIntegrationPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("CARVION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARVION_TEST_DATABASE_URL to a dedicated migrated test database")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	return ctx, pool
}

func createDeploymentIntegrationProject(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var projectID string
	err := pool.QueryRow(ctx, `
		INSERT INTO localization_projects (name, locale, resource_format, source_filename, source_sha256, source_content)
		VALUES ($1, 'ru-RU', 'carvion-json-v1', 'empty.json', $2, $3)
		RETURNING id::text`, "deployment integration "+time.Now().UTC().Format("20060102150405.000000000"), strings.Repeat("b", 64), []byte(`{}`)).Scan(&projectID)
	if err != nil {
		t.Fatalf("create localization project fixture: %v", err)
	}
	return projectID
}
