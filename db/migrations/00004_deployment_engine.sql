-- +goose Up
CREATE TABLE deployment_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES localization_projects (id) ON DELETE RESTRICT,
    target_profile_id TEXT,
    mode TEXT NOT NULL,
    provider_id TEXT,
    package_sha256 CHAR(64) NOT NULL,
    state TEXT NOT NULL DEFAULT 'queued',
    result_status TEXT NOT NULL DEFAULT 'pending',
    preflight JSONB NOT NULL DEFAULT '{"ready":false,"checks":[]}'::jsonb,
    result_verification JSONB NOT NULL DEFAULT '{"verified":false,"simulated":false,"message":""}'::jsonb,
    recovery JSONB NOT NULL DEFAULT '{"available":false,"simulated":false,"message":""}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT deployment_jobs_mode_check CHECK (mode IN ('test', 'production')),
    CONSTRAINT deployment_jobs_state_check CHECK (state IN (
        'queued', 'preflighting', 'blocked', 'preflight-passed', 'simulating',
        'verifying', 'simulated', 'failed', 'cancelled'
    )),
    CONSTRAINT deployment_jobs_result_status_check CHECK (result_status IN (
        'pending', 'simulated', 'blocked', 'failed', 'cancelled'
    )),
    CONSTRAINT deployment_jobs_package_sha256_check CHECK (package_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT deployment_jobs_target_profile_not_empty CHECK (target_profile_id IS NULL OR length(btrim(target_profile_id)) > 0),
    CONSTRAINT deployment_jobs_provider_not_empty CHECK (provider_id IS NULL OR length(btrim(provider_id)) > 0),
    CONSTRAINT deployment_jobs_terminal_result_check CHECK (
        (state IN ('simulated', 'blocked', 'failed', 'cancelled') AND result_status = state)
        OR (state NOT IN ('simulated', 'blocked', 'failed', 'cancelled') AND result_status = 'pending')
    )
);

CREATE TABLE deployment_job_events (
    job_id UUID NOT NULL REFERENCES deployment_jobs (id) ON DELETE RESTRICT,
    sequence BIGINT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    state TEXT NOT NULL,
    stage TEXT NOT NULL,
    message TEXT NOT NULL,
    CONSTRAINT deployment_job_events_job_sequence_pkey PRIMARY KEY (job_id, sequence),
    CONSTRAINT deployment_job_events_sequence_positive CHECK (sequence > 0),
    CONSTRAINT deployment_job_events_state_check CHECK (state IN (
        'queued', 'preflighting', 'blocked', 'preflight-passed', 'simulating',
        'verifying', 'simulated', 'failed', 'cancelled'
    )),
    CONSTRAINT deployment_job_events_stage_not_empty CHECK (length(btrim(stage)) > 0),
    CONSTRAINT deployment_job_events_message_not_empty CHECK (length(btrim(message)) > 0),
    CONSTRAINT deployment_job_events_stage_length CHECK (length(stage) <= 80),
    CONSTRAINT deployment_job_events_message_length CHECK (length(message) <= 2048)
);

CREATE INDEX deployment_jobs_created_idx ON deployment_jobs (created_at DESC, id DESC);
CREATE INDEX deployment_jobs_project_created_idx ON deployment_jobs (project_id, created_at DESC, id DESC);

-- +goose StatementBegin
CREATE FUNCTION prevent_deployment_job_event_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'deployment_job_events is append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER deployment_job_events_immutable
    BEFORE UPDATE OR DELETE ON deployment_job_events
    FOR EACH ROW EXECUTE FUNCTION prevent_deployment_job_event_mutation();

-- +goose Down
DROP TRIGGER deployment_job_events_immutable ON deployment_job_events;
DROP FUNCTION prevent_deployment_job_event_mutation();
DROP TABLE deployment_job_events;
DROP TABLE deployment_jobs;
