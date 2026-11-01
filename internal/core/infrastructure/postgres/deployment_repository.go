package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeploymentRepository struct {
	pool *pgxpool.Pool
}

func NewDeploymentRepository(pool *pgxpool.Pool) *DeploymentRepository {
	return &DeploymentRepository{pool: pool}
}

func (r *DeploymentRepository) CreateDeploymentJob(ctx context.Context, job domain.DeploymentJob) (domain.DeploymentJob, error) {
	if err := validateNewDeploymentJob(job); err != nil {
		return domain.DeploymentJob{}, err
	}
	preflight, result, recovery, err := deploymentJobJSON(job)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.DeploymentJob{}, deploymentDBError(ctx, "begin job creation", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var targetProfileID, providerID any
	if job.TargetProfileID != "" {
		targetProfileID = job.TargetProfileID
	}
	if job.ProviderID != "" {
		providerID = job.ProviderID
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO deployment_jobs
			(project_id, target_profile_id, mode, provider_id, package_sha256, state, result_status,
			 preflight, result_verification, recovery, error_message)
		VALUES ($1::uuid, $2, $3, $4, $5, 'queued', 'pending', $6::jsonb, $7::jsonb, $8::jsonb, $9)
		RETURNING id::text, created_at, updated_at`,
		job.ProjectID, targetProfileID, string(job.Mode), providerID, job.PackageSHA256,
		preflight, result, recovery, job.ErrorMessage,
	).Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return domain.DeploymentJob{}, deploymentDBError(ctx, "insert job", err)
	}
	initialEvent := domain.DeploymentEvent{
		JobID: job.ID, Sequence: 1, At: nowUTC(), State: domain.DeploymentStateQueued,
		Stage: "queued", Message: "Deployment job created",
	}
	if err := insertDeploymentEvent(ctx, tx, initialEvent); err != nil {
		return domain.DeploymentJob{}, deploymentDBError(ctx, "insert initial job event", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.DeploymentJob{}, deploymentDBError(ctx, "commit job creation", err)
	}
	job.State = domain.DeploymentStateQueued
	job.Events = []domain.DeploymentEvent{initialEvent}
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	return job, nil
}

func (r *DeploymentRepository) TransitionDeploymentJob(ctx context.Context, job domain.DeploymentJob, event domain.DeploymentEvent) error {
	if strings.TrimSpace(job.ID) == "" {
		return fmt.Errorf("%w: deployment job ID is required", domain.ErrInvalidArgument)
	}
	if event.Sequence != 0 || (event.JobID != "" && event.JobID != job.ID) || (event.State != "" && event.State != job.State) {
		return fmt.Errorf("%w: deployment event identity or state does not match the transition", domain.ErrInvalidArgument)
	}
	if err := validateDeploymentEvent(event); err != nil {
		return err
	}
	preflight, result, recovery, err := deploymentJobJSON(job)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return deploymentDBError(ctx, "begin job transition", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var currentState, projectID, mode, packageHash string
	var currentTargetID, currentProviderID pgtype.Text
	err = tx.QueryRow(ctx, `
		SELECT state, project_id::text, target_profile_id, mode, provider_id, package_sha256
		FROM deployment_jobs WHERE id = $1::uuid FOR UPDATE`, job.ID,
	).Scan(&currentState, &projectID, &currentTargetID, &mode, &currentProviderID, &packageHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrDeploymentJobNotFound
	}
	if err != nil {
		return deploymentDBError(ctx, "lock job for transition", err)
	}
	if projectID != job.ProjectID || nullableText(currentTargetID) != job.TargetProfileID || mode != string(job.Mode) || nullableText(currentProviderID) != job.ProviderID || packageHash != job.PackageSHA256 {
		return fmt.Errorf("%w: immutable deployment job fields changed", domain.ErrDeploymentStateConflict)
	}
	if err := domain.ValidateDeploymentTransition(domain.DeploymentState(currentState), job.State); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrDeploymentStateConflict, err)
	}
	// Persist timestamps from this process, not caller-provided values.
	event.At = nowUTC()
	var sequence int64
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM deployment_job_events WHERE job_id = $1::uuid`, job.ID).Scan(&sequence)
	if err != nil {
		return deploymentDBError(ctx, "allocate event sequence", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE deployment_jobs SET
			state = $2,
			result_status = $3,
			preflight = $4::jsonb,
			result_verification = $5::jsonb,
			recovery = $6::jsonb,
			error_message = $7,
			updated_at = now()
		WHERE id = $1::uuid`,
		job.ID, string(job.State), deploymentResultStatus(job.State), preflight, result, recovery, job.ErrorMessage,
	)
	if err != nil {
		return deploymentDBError(ctx, "update job state", err)
	}
	event.JobID = job.ID
	event.Sequence = sequence
	event.State = job.State
	if err := insertDeploymentEvent(ctx, tx, event); err != nil {
		return deploymentDBError(ctx, "append job event", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return deploymentDBError(ctx, "commit job transition", err)
	}
	return nil
}

func (r *DeploymentRepository) AppendDeploymentEvent(ctx context.Context, jobID string, expectedState domain.DeploymentState, event domain.DeploymentEvent) (domain.DeploymentEvent, error) {
	if strings.TrimSpace(jobID) == "" || (event.JobID != "" && event.JobID != jobID) || event.Sequence != 0 || (event.State != "" && event.State != expectedState) {
		return domain.DeploymentEvent{}, fmt.Errorf("%w: deployment event identity or state is invalid", domain.ErrInvalidArgument)
	}
	if err := validateDeploymentEvent(event); err != nil {
		return domain.DeploymentEvent{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.DeploymentEvent{}, deploymentDBError(ctx, "begin progress event", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var currentState string
	err = tx.QueryRow(ctx, `SELECT state FROM deployment_jobs WHERE id = $1::uuid FOR UPDATE`, jobID).Scan(&currentState)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DeploymentEvent{}, domain.ErrDeploymentJobNotFound
	}
	if err != nil {
		return domain.DeploymentEvent{}, deploymentDBError(ctx, "lock job for progress event", err)
	}
	if domain.DeploymentState(currentState) != expectedState || expectedState.Terminal() {
		return domain.DeploymentEvent{}, fmt.Errorf("%w: progress event no longer matches the active job state", domain.ErrDeploymentStateConflict)
	}
	var sequence int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM deployment_job_events WHERE job_id = $1::uuid`, jobID).Scan(&sequence); err != nil {
		return domain.DeploymentEvent{}, deploymentDBError(ctx, "allocate progress event sequence", err)
	}
	event.JobID = jobID
	event.Sequence = sequence
	event.State = expectedState
	event.At = nowUTC()
	if err := insertDeploymentEvent(ctx, tx, event); err != nil {
		return domain.DeploymentEvent{}, deploymentDBError(ctx, "append progress event", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.DeploymentEvent{}, deploymentDBError(ctx, "commit progress event", err)
	}
	return event, nil
}

func (r *DeploymentRepository) GetDeploymentJob(ctx context.Context, jobID string) (domain.DeploymentJob, error) {
	job, err := scanDeploymentJob(r.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, target_profile_id, mode, provider_id, package_sha256,
			state, preflight::text, result_verification::text, recovery::text, error_message, created_at, updated_at
		FROM deployment_jobs WHERE id = $1::uuid`, jobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DeploymentJob{}, domain.ErrDeploymentJobNotFound
	}
	if err != nil {
		return domain.DeploymentJob{}, deploymentDBError(ctx, "get job", err)
	}
	job.Events, err = r.listDeploymentEvents(ctx, job.ID)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	return job, nil
}

func (r *DeploymentRepository) ListDeploymentJobs(ctx context.Context, limit int) ([]domain.DeploymentJobSummary, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("%w: deployment job limit must be between 1 and 100", domain.ErrInvalidArgument)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, project_id::text, target_profile_id, mode, package_sha256, state, created_at, updated_at
		FROM deployment_jobs ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, deploymentDBError(ctx, "list jobs", err)
	}
	defer rows.Close()
	jobs := make([]domain.DeploymentJobSummary, 0)
	for rows.Next() {
		var job domain.DeploymentJobSummary
		var targetProfileID pgtype.Text
		if err := rows.Scan(&job.ID, &job.ProjectID, &targetProfileID, &job.Mode, &job.PackageSHA256, &job.State, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, deploymentDBError(ctx, "scan job summary", err)
		}
		job.TargetProfileID = nullableText(targetProfileID)
		job.CreatedAt = job.CreatedAt.UTC()
		job.UpdatedAt = job.UpdatedAt.UTC()
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, deploymentDBError(ctx, "read job summaries", err)
	}
	return jobs, nil
}

type deploymentRowScanner interface {
	Scan(...any) error
}

func scanDeploymentJob(row deploymentRowScanner) (domain.DeploymentJob, error) {
	var job domain.DeploymentJob
	var targetProfileID, providerID pgtype.Text
	var preflightJSON, resultJSON, recoveryJSON string
	err := row.Scan(
		&job.ID, &job.ProjectID, &targetProfileID, &job.Mode, &providerID, &job.PackageSHA256,
		&job.State, &preflightJSON, &resultJSON, &recoveryJSON, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	job.TargetProfileID = nullableText(targetProfileID)
	job.ProviderID = nullableText(providerID)
	if err := json.Unmarshal([]byte(preflightJSON), &job.Preflight); err != nil {
		return domain.DeploymentJob{}, fmt.Errorf("decode deployment preflight: %w", err)
	}
	if err := json.Unmarshal([]byte(resultJSON), &job.ResultVerification); err != nil {
		return domain.DeploymentJob{}, fmt.Errorf("decode deployment result verification: %w", err)
	}
	if err := json.Unmarshal([]byte(recoveryJSON), &job.Recovery); err != nil {
		return domain.DeploymentJob{}, fmt.Errorf("decode deployment recovery: %w", err)
	}
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	return job, nil
}

func (r *DeploymentRepository) listDeploymentEvents(ctx context.Context, jobID string) ([]domain.DeploymentEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT job_id::text, sequence, occurred_at, state, stage, message
		FROM deployment_job_events WHERE job_id = $1::uuid ORDER BY sequence`, jobID)
	if err != nil {
		return nil, deploymentDBError(ctx, "list job events", err)
	}
	defer rows.Close()
	events := make([]domain.DeploymentEvent, 0)
	for rows.Next() {
		var event domain.DeploymentEvent
		if err := rows.Scan(&event.JobID, &event.Sequence, &event.At, &event.State, &event.Stage, &event.Message); err != nil {
			return nil, deploymentDBError(ctx, "scan job event", err)
		}
		event.At = event.At.UTC()
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, deploymentDBError(ctx, "read job events", err)
	}
	return events, nil
}

func validateNewDeploymentJob(job domain.DeploymentJob) error {
	if strings.TrimSpace(job.ProjectID) == "" || job.State != domain.DeploymentStateQueued ||
		(job.Mode != domain.DeploymentModeTest && job.Mode != domain.DeploymentModeProduction) || !validDeploymentSHA256(job.PackageSHA256) {
		return fmt.Errorf("%w: deployment job identity, mode, state or package hash is invalid", domain.ErrInvalidArgument)
	}
	return nil
}

func validateDeploymentEvent(event domain.DeploymentEvent) error {
	if strings.TrimSpace(event.Stage) == "" || strings.TrimSpace(event.Message) == "" || len(event.Stage) > 80 || len(event.Message) > 2048 {
		return fmt.Errorf("%w: deployment event requires a stage and a message within size limits", domain.ErrInvalidArgument)
	}
	return nil
}

func deploymentJobJSON(job domain.DeploymentJob) ([]byte, []byte, []byte, error) {
	preflight, err := json.Marshal(job.Preflight)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encode deployment preflight: %w", err)
	}
	result, err := json.Marshal(job.ResultVerification)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encode deployment result verification: %w", err)
	}
	recovery, err := json.Marshal(job.Recovery)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encode deployment recovery: %w", err)
	}
	return preflight, result, recovery, nil
}

func insertDeploymentEvent(ctx context.Context, tx pgx.Tx, event domain.DeploymentEvent) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO deployment_job_events (job_id, sequence, occurred_at, state, stage, message)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)`,
		event.JobID, event.Sequence, event.At, string(event.State), event.Stage, event.Message,
	)
	return err
}

func deploymentResultStatus(state domain.DeploymentState) string {
	switch state {
	case domain.DeploymentStateSimulated, domain.DeploymentStateBlocked, domain.DeploymentStateFailed, domain.DeploymentStateCancelled:
		return string(state)
	default:
		return "pending"
	}
}

func validDeploymentSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func nullableText(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func deploymentDBError(ctx context.Context, operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: %s: %v", domain.ErrDeploymentUnavailable, operation, err)
}
