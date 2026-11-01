package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LocalizationRepository struct {
	pool *pgxpool.Pool
}

func nowUTC() time.Time { return time.Now().UTC() }

func NewLocalizationRepository(pool *pgxpool.Pool) *LocalizationRepository {
	return &LocalizationRepository{pool: pool}
}

func (r *LocalizationRepository) CreateLocalizationProject(ctx context.Context, project domain.LocalizationProject) (domain.LocalizationProject, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.LocalizationProject{}, localizationDBError(ctx, "begin project import", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	err = tx.QueryRow(ctx, `
		INSERT INTO localization_projects (name, locale, resource_format, source_filename, source_sha256, source_content)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, created_at, updated_at`,
		project.Name, project.Locale, string(project.Format), project.Filename, project.SourceSHA256, project.Source,
	).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return domain.LocalizationProject{}, localizationDBError(ctx, "insert project", err)
	}
	for i := range project.Resources {
		resource := project.Resources[i]
		metadata := resource.Metadata
		if len(metadata) == 0 {
			metadata = []byte(`{}`)
		}
		var groupKey, pluralCategory any
		if resource.GroupKey != "" {
			groupKey = resource.GroupKey
		}
		if resource.PluralCategory != "" {
			pluralCategory = resource.PluralCategory
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO localization_resources
				(project_id, resource_key, group_key, resource_kind, source_text, source_sha256, order_index, item_index, plural_category, translatable, formatted, present, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			project.ID, resource.Key, groupKey, string(resource.Kind), resource.SourceText, resource.SourceSHA256,
			resource.Order, resource.ItemIndex, pluralCategory, resource.Translatable, resource.Formatted, resource.Present, metadata,
		)
		if err != nil {
			return domain.LocalizationProject{}, localizationDBError(ctx, "insert project resource", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.LocalizationProject{}, localizationDBError(ctx, "commit project import", err)
	}
	return project, nil
}

func (r *LocalizationRepository) ListLocalizationProjects(ctx context.Context) ([]domain.LocalizationProjectSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, locale, resource_format, source_filename, source_sha256, updated_at
		FROM localization_projects
		ORDER BY updated_at DESC, id`)
	if err != nil {
		return nil, localizationDBError(ctx, "list projects", err)
	}
	defer rows.Close()
	projects := make([]domain.LocalizationProjectSummary, 0)
	for rows.Next() {
		var project domain.LocalizationProjectSummary
		if err := rows.Scan(&project.ID, &project.Name, &project.Locale, &project.Format, &project.Filename, &project.SourceSHA256, &project.UpdatedAt); err != nil {
			return nil, localizationDBError(ctx, "scan project", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, localizationDBError(ctx, "read projects", err)
	}
	return projects, nil
}

func (r *LocalizationRepository) GetLocalizationProject(ctx context.Context, projectID string) (domain.LocalizationProject, error) {
	var project domain.LocalizationProject
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, locale, resource_format, source_filename, source_sha256, source_content, created_at, updated_at
		FROM localization_projects WHERE id = $1::uuid`, projectID,
	).Scan(&project.ID, &project.Name, &project.Locale, &project.Format, &project.Filename, &project.SourceSHA256, &project.Source, &project.CreatedAt, &project.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LocalizationProject{}, domain.ErrLocalizationProjectNotFound
	}
	if err != nil {
		return domain.LocalizationProject{}, localizationDBError(ctx, "get project", err)
	}
	project.Resources, err = r.listLocalizationResources(ctx, project.ID)
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	project.Translations, err = r.listLocalizationTranslations(ctx, project.ID)
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return project, nil
}

func (r *LocalizationRepository) ReplaceLocalizationSource(ctx context.Context, projectID string, format domain.LocalizationFormat, filename, sourceHash string, source []byte, resources []domain.LocalizationResource) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return localizationDBError(ctx, "begin source replacement", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var lockedID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM localization_projects WHERE id = $1::uuid FOR UPDATE`, projectID).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrLocalizationProjectNotFound
		}
		return localizationDBError(ctx, "lock project source", err)
	}
	keys := make([]string, 0, len(resources))
	for _, resource := range resources {
		keys = append(keys, resource.Key)
		var existingHash string
		err := tx.QueryRow(ctx, `SELECT source_sha256 FROM localization_resources WHERE project_id = $1::uuid AND resource_key = $2`, projectID, resource.Key).Scan(&existingHash)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return localizationDBError(ctx, "read previous resource source", err)
		}
		if err == nil && existingHash != resource.SourceSHA256 {
			if err := invalidateApprovedTranslation(ctx, tx, projectID, resource.Key); err != nil {
				return localizationDBError(ctx, "invalidate approval after source change", err)
			}
		}
	}
	rows, err := tx.Query(ctx, `SELECT resource_key FROM localization_resources WHERE project_id = $1::uuid AND NOT (resource_key = ANY($2::text[]))`, projectID, keys)
	if err != nil {
		return localizationDBError(ctx, "find removed resources", err)
	}
	removed := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return localizationDBError(ctx, "scan removed resource", err)
		}
		removed = append(removed, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return localizationDBError(ctx, "read removed resources", err)
	}
	rows.Close()
	for _, key := range removed {
		if err := invalidateApprovedTranslation(ctx, tx, projectID, key); err != nil {
			return localizationDBError(ctx, "invalidate approval for removed resource", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE localization_resources SET present = false WHERE project_id = $1::uuid AND NOT (resource_key = ANY($2::text[]))`, projectID, keys); err != nil {
		return localizationDBError(ctx, "mark removed resources", err)
	}
	for _, resource := range resources {
		metadata := resource.Metadata
		if len(metadata) == 0 {
			metadata = []byte(`{}`)
		}
		var groupKey, pluralCategory any
		if resource.GroupKey != "" {
			groupKey = resource.GroupKey
		}
		if resource.PluralCategory != "" {
			pluralCategory = resource.PluralCategory
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO localization_resources (project_id, resource_key, group_key, resource_kind, source_text, source_sha256, order_index, item_index, plural_category, translatable, formatted, present, metadata)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, true, $12)
			ON CONFLICT (project_id, resource_key) DO UPDATE SET group_key = EXCLUDED.group_key,
				resource_kind = EXCLUDED.resource_kind, source_text = EXCLUDED.source_text, source_sha256 = EXCLUDED.source_sha256,
				order_index = EXCLUDED.order_index, item_index = EXCLUDED.item_index, plural_category = EXCLUDED.plural_category,
				translatable = EXCLUDED.translatable, formatted = EXCLUDED.formatted, present = true, metadata = EXCLUDED.metadata`,
			projectID, resource.Key, groupKey, string(resource.Kind), resource.SourceText, resource.SourceSHA256,
			resource.Order, resource.ItemIndex, pluralCategory, resource.Translatable, resource.Formatted, metadata)
		if err != nil {
			return localizationDBError(ctx, "upsert replacement resource", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE localization_projects SET resource_format = $2, source_filename = $3, source_sha256 = $4, source_content = $5, updated_at = now() WHERE id = $1::uuid`, projectID, string(format), filename, sourceHash, source); err != nil {
		return localizationDBError(ctx, "update project source", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return localizationDBError(ctx, "commit source replacement", err)
	}
	return nil
}

func invalidateApprovedTranslation(ctx context.Context, tx pgx.Tx, projectID, resourceKey string) error {
	rows, err := tx.Query(ctx, `
		SELECT locale, target_text, approved_source_sha256
		FROM localization_translations
		WHERE project_id = $1::uuid AND resource_key = $2 AND status = 'approved'
		FOR UPDATE`, projectID, resourceKey)
	if err != nil {
		return err
	}
	type item struct{ locale, target, approvedHash string }
	changed := make([]item, 0)
	for rows.Next() {
		var current item
		if err := rows.Scan(&current.locale, &current.target, &current.approvedHash); err != nil {
			rows.Close()
			return err
		}
		changed = append(changed, current)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if _, err := tx.Exec(ctx, `
		UPDATE localization_translations SET status = 'needs_review', approved_source_sha256 = NULL, updated_at = now()
		WHERE project_id = $1::uuid AND resource_key = $2 AND status = 'approved'`, projectID, resourceKey); err != nil {
		return err
	}
	for _, current := range changed {
		if _, err := tx.Exec(ctx, `
			INSERT INTO translation_history (project_id, resource_key, locale, previous_target_text, next_target_text, previous_status, next_status, approved_source_sha256)
			VALUES ($1::uuid, $2, $3, $4, $4, 'approved', 'needs_review', $5)`, projectID, resourceKey, current.locale, current.target, current.approvedHash); err != nil {
			return err
		}
	}
	return nil
}

func (r *LocalizationRepository) SaveLocalizationTranslation(ctx context.Context, projectID string, translation domain.LocalizationTranslation) (domain.LocalizationTranslation, error) {
	if translation.UpdatedAt.IsZero() {
		translation.UpdatedAt = nowUTC()
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.LocalizationTranslation{}, localizationDBError(ctx, "begin translation update", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := saveTranslation(ctx, tx, projectID, translation); err != nil {
		return domain.LocalizationTranslation{}, localizationDBError(ctx, "save translation", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.LocalizationTranslation{}, localizationDBError(ctx, "commit translation update", err)
	}
	return translation, nil
}

func (r *LocalizationRepository) SaveLocalizationTranslations(ctx context.Context, projectID string, translations []domain.LocalizationTranslation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return localizationDBError(ctx, "begin translation import", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	for _, translation := range translations {
		if translation.UpdatedAt.IsZero() {
			translation.UpdatedAt = nowUTC()
		}
		if err := saveTranslation(ctx, tx, projectID, translation); err != nil {
			return localizationDBError(ctx, "save imported translation", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return localizationDBError(ctx, "commit translation import", err)
	}
	return nil
}

func saveTranslation(ctx context.Context, tx pgx.Tx, projectID string, translation domain.LocalizationTranslation) error {
	var previousTarget, previousStatus pgtype.Text
	err := tx.QueryRow(ctx, `
		SELECT target_text, status FROM localization_translations
		WHERE project_id = $1::uuid AND resource_key = $2 AND locale = $3 FOR UPDATE`,
		projectID, translation.ResourceKey, translation.Locale,
	).Scan(&previousTarget, &previousStatus)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var approvedHash any
	if translation.ApprovedSourceSHA256 != "" {
		approvedHash = translation.ApprovedSourceSHA256
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO localization_translations (project_id, resource_key, locale, target_text, status, approved_source_sha256, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (project_id, resource_key, locale) DO UPDATE SET
			target_text = EXCLUDED.target_text,
			status = EXCLUDED.status,
			approved_source_sha256 = EXCLUDED.approved_source_sha256,
			updated_at = EXCLUDED.updated_at`,
		projectID, translation.ResourceKey, translation.Locale, translation.TargetText,
		string(translation.Status), approvedHash, translation.UpdatedAt,
	)
	if err != nil {
		return err
	}
	var previousTargetValue, previousStatusValue any
	if previousTarget.Valid {
		previousTargetValue = previousTarget.String
	}
	if previousStatus.Valid {
		previousStatusValue = previousStatus.String
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO translation_history
			(project_id, resource_key, locale, previous_target_text, next_target_text, previous_status, next_status, approved_source_sha256)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)`,
		projectID, translation.ResourceKey, translation.Locale, previousTargetValue, translation.TargetText,
		previousStatusValue, string(translation.Status), approvedHash,
	)
	return err
}

func (r *LocalizationRepository) SaveLocalizationValidation(ctx context.Context, projectID string, report domain.LocalizationValidationReport) (string, error) {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("encode localization validation report: %w", err)
	}
	var id string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO localization_validation_runs (project_id, is_valid, report)
		VALUES ($1::uuid, $2, $3) RETURNING id::text`, projectID, report.Valid, reportJSON,
	).Scan(&id)
	if err != nil {
		return "", localizationDBError(ctx, "save validation run", err)
	}
	return id, nil
}

func (r *LocalizationRepository) SaveLocalizationPackage(ctx context.Context, projectID string, item domain.LocalizationPackage, validationRunID string, artifact []byte) (domain.LocalizationPackage, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO language_packages (project_id, validation_run_id, filename, sha256, artifact, size_bytes)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		RETURNING id::text`, projectID, validationRunID, item.Filename, item.SHA256, artifact, len(artifact)).Scan(&item.ID)
	if err != nil {
		return domain.LocalizationPackage{}, localizationDBError(ctx, "save language package", err)
	}
	return item, nil
}

func (r *LocalizationRepository) GetLocalizationPackage(ctx context.Context, projectID, packageSHA256 string) ([]byte, error) {
	var artifact []byte
	err := r.pool.QueryRow(ctx, `
		SELECT artifact FROM language_packages
		WHERE project_id = $1::uuid AND sha256 = $2
		ORDER BY created_at DESC, id DESC LIMIT 1`, projectID, packageSHA256,
	).Scan(&artifact)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrLocalizationPackageNotFound
	}
	if err != nil {
		return nil, localizationDBError(ctx, "get language package", err)
	}
	return artifact, nil
}

func (r *LocalizationRepository) listLocalizationResources(ctx context.Context, projectID string) ([]domain.LocalizationResource, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT resource_key, COALESCE(group_key, ''), resource_kind, source_text, source_sha256,
			order_index, item_index, COALESCE(plural_category, ''), translatable, formatted, present, metadata
		FROM localization_resources WHERE project_id = $1::uuid ORDER BY order_index, resource_key`, projectID)
	if err != nil {
		return nil, localizationDBError(ctx, "list project resources", err)
	}
	defer rows.Close()
	resources := make([]domain.LocalizationResource, 0)
	for rows.Next() {
		var resource domain.LocalizationResource
		if err := rows.Scan(&resource.Key, &resource.GroupKey, &resource.Kind, &resource.SourceText, &resource.SourceSHA256,
			&resource.Order, &resource.ItemIndex, &resource.PluralCategory, &resource.Translatable, &resource.Formatted, &resource.Present, &resource.Metadata); err != nil {
			return nil, localizationDBError(ctx, "scan project resource", err)
		}
		resources = append(resources, resource)
	}
	if err := rows.Err(); err != nil {
		return nil, localizationDBError(ctx, "read project resources", err)
	}
	return resources, nil
}

func (r *LocalizationRepository) listLocalizationTranslations(ctx context.Context, projectID string) ([]domain.LocalizationTranslation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT resource_key, locale, target_text, status, COALESCE(approved_source_sha256, ''), updated_at
		FROM localization_translations WHERE project_id = $1::uuid ORDER BY locale, resource_key`, projectID)
	if err != nil {
		return nil, localizationDBError(ctx, "list project translations", err)
	}
	defer rows.Close()
	translations := make([]domain.LocalizationTranslation, 0)
	for rows.Next() {
		var translation domain.LocalizationTranslation
		if err := rows.Scan(&translation.ResourceKey, &translation.Locale, &translation.TargetText, &translation.Status, &translation.ApprovedSourceSHA256, &translation.UpdatedAt); err != nil {
			return nil, localizationDBError(ctx, "scan project translation", err)
		}
		translations = append(translations, translation)
	}
	if err := rows.Err(); err != nil {
		return nil, localizationDBError(ctx, "read project translations", err)
	}
	return translations, nil
}

func localizationDBError(ctx context.Context, operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: %s: %v", domain.ErrLocalizationUnavailable, operation, err)
}
