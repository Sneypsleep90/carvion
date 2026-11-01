-- +goose Up
CREATE TABLE localization_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    locale TEXT NOT NULL,
    resource_format TEXT NOT NULL,
    source_filename TEXT NOT NULL,
    source_sha256 CHAR(64) NOT NULL,
    source_content BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT localization_projects_name_not_empty CHECK (length(btrim(name)) > 0),
    CONSTRAINT localization_projects_locale_not_empty CHECK (length(btrim(locale)) > 0),
    CONSTRAINT localization_projects_format_check CHECK (resource_format IN ('android-strings-xml', 'carvion-json-v1')),
    CONSTRAINT localization_projects_filename_not_empty CHECK (length(btrim(source_filename)) > 0),
    CONSTRAINT localization_projects_sha256_check CHECK (source_sha256 ~ '^[0-9a-f]{64}$')
);

CREATE TABLE localization_resources (
    project_id UUID NOT NULL REFERENCES localization_projects (id) ON DELETE CASCADE,
    resource_key TEXT NOT NULL,
    group_key TEXT,
    resource_kind TEXT NOT NULL,
    source_text TEXT NOT NULL,
    source_sha256 CHAR(64) NOT NULL,
    order_index INTEGER NOT NULL,
    item_index INTEGER NOT NULL DEFAULT 0,
    plural_category TEXT,
    translatable BOOLEAN NOT NULL DEFAULT true,
    formatted BOOLEAN NOT NULL DEFAULT true,
    present BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT localization_resources_project_key PRIMARY KEY (project_id, resource_key),
    CONSTRAINT localization_resources_project_key_hash_key UNIQUE (project_id, resource_key, source_sha256),
    CONSTRAINT localization_resources_key_not_empty CHECK (length(btrim(resource_key)) > 0),
    CONSTRAINT localization_resources_kind_check CHECK (resource_kind IN ('string', 'string-array-item', 'plural-item')),
    CONSTRAINT localization_resources_order_nonnegative CHECK (order_index >= 0),
    CONSTRAINT localization_resources_item_nonnegative CHECK (item_index >= 0),
    CONSTRAINT localization_resources_plural_category_check CHECK (plural_category IS NULL OR plural_category IN ('zero', 'one', 'two', 'few', 'many', 'other')),
    CONSTRAINT localization_resources_plural_group_check CHECK (resource_kind <> 'plural-item' OR (group_key IS NOT NULL AND plural_category IS NOT NULL))
);

CREATE TABLE localization_translations (
    project_id UUID NOT NULL,
    resource_key TEXT NOT NULL,
    locale TEXT NOT NULL,
    target_text TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    approved_source_sha256 CHAR(64),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT localization_translations_project_resource_fkey
        FOREIGN KEY (project_id, resource_key)
        REFERENCES localization_resources (project_id, resource_key) ON DELETE CASCADE,
    CONSTRAINT localization_translations_approved_source_fkey
        FOREIGN KEY (project_id, resource_key, approved_source_sha256)
        REFERENCES localization_resources (project_id, resource_key, source_sha256),
    CONSTRAINT localization_translations_project_key_locale PRIMARY KEY (project_id, resource_key, locale),
    CONSTRAINT localization_translations_locale_not_empty CHECK (length(btrim(locale)) > 0),
    CONSTRAINT localization_translations_status_check CHECK (status IN ('draft', 'approved', 'needs_review')),
    CONSTRAINT localization_translations_approval_hash_check CHECK (status <> 'approved' OR approved_source_sha256 IS NOT NULL)
);

CREATE TABLE translation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    resource_key TEXT NOT NULL,
    locale TEXT NOT NULL,
    previous_target_text TEXT,
    next_target_text TEXT NOT NULL,
    previous_status TEXT,
    next_status TEXT NOT NULL,
    approved_source_sha256 CHAR(64),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT translation_history_translation_fkey
        FOREIGN KEY (project_id, resource_key, locale)
        REFERENCES localization_translations (project_id, resource_key, locale),
    CONSTRAINT translation_history_next_status_check CHECK (next_status IN ('draft', 'approved', 'needs_review')),
    CONSTRAINT translation_history_previous_status_check CHECK (previous_status IS NULL OR previous_status IN ('draft', 'approved', 'needs_review'))
);

CREATE TABLE localization_validation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES localization_projects (id) ON DELETE CASCADE,
    is_valid BOOLEAN NOT NULL,
    report JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT localization_validation_runs_id_project_key UNIQUE (id, project_id)
);

CREATE TABLE language_packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES localization_projects (id) ON DELETE CASCADE,
    validation_run_id UUID NOT NULL,
    filename TEXT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    artifact BYTEA NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT language_packages_validation_project_fkey
        FOREIGN KEY (validation_run_id, project_id)
        REFERENCES localization_validation_runs (id, project_id),
    CONSTRAINT language_packages_filename_not_empty CHECK (length(btrim(filename)) > 0),
    CONSTRAINT language_packages_sha256_check CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT language_packages_size_check CHECK (size_bytes = octet_length(artifact))
);

CREATE INDEX localization_projects_updated_idx ON localization_projects (updated_at DESC, id);
CREATE INDEX localization_resources_group_order_idx ON localization_resources (project_id, group_key, order_index);
CREATE INDEX localization_resources_translation_queue_idx ON localization_resources (project_id, translatable, order_index);
CREATE INDEX localization_translations_locale_status_idx ON localization_translations (project_id, locale, status);
CREATE INDEX translation_history_resource_changed_idx ON translation_history (project_id, resource_key, locale, changed_at DESC);
CREATE INDEX localization_validation_project_created_idx ON localization_validation_runs (project_id, created_at DESC);
CREATE INDEX language_packages_project_created_idx ON language_packages (project_id, created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION prevent_translation_history_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'translation_history is append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER translation_history_immutable
    BEFORE UPDATE OR DELETE ON translation_history
    FOR EACH ROW EXECUTE FUNCTION prevent_translation_history_mutation();

-- +goose Down
DROP TRIGGER translation_history_immutable ON translation_history;
DROP FUNCTION prevent_translation_history_mutation();
DROP TABLE language_packages;
DROP TABLE localization_validation_runs;
DROP TABLE translation_history;
DROP TABLE localization_translations;
DROP TABLE localization_resources;
DROP TABLE localization_projects;
