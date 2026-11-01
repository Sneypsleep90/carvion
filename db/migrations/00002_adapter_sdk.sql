-- +goose Up
CREATE TABLE adapter_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adapter_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT adapter_definitions_key_not_empty CHECK (length(btrim(adapter_key)) > 0),
    CONSTRAINT adapter_definitions_name_not_empty CHECK (length(btrim(name)) > 0)
);

CREATE TABLE adapter_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adapter_definition_id UUID NOT NULL REFERENCES adapter_definitions (id),
    version TEXT NOT NULL,
    api_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT adapter_versions_definition_version_key UNIQUE (adapter_definition_id, version),
    CONSTRAINT adapter_versions_version_not_empty CHECK (length(btrim(version)) > 0),
    CONSTRAINT adapter_versions_api_version_not_empty CHECK (length(btrim(api_version)) > 0)
);

CREATE TABLE hardware_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id UUID NOT NULL REFERENCES infotainment_platforms (id),
    revision TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT hardware_variants_platform_revision_key UNIQUE (platform_id, revision),
    CONSTRAINT hardware_variants_id_platform_key UNIQUE (id, platform_id),
    CONSTRAINT hardware_variants_revision_not_empty CHECK (length(btrim(revision)) > 0)
);

CREATE TABLE firmware_builds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id UUID NOT NULL,
    hardware_variant_id UUID NOT NULL,
    build_id TEXT NOT NULL,
    version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT firmware_builds_variant_platform_fkey
        FOREIGN KEY (hardware_variant_id, platform_id)
        REFERENCES hardware_variants (id, platform_id),
    CONSTRAINT firmware_builds_variant_build_key UNIQUE (hardware_variant_id, build_id),
    CONSTRAINT firmware_builds_id_variant_platform_key UNIQUE (id, hardware_variant_id, platform_id),
    CONSTRAINT firmware_builds_build_not_empty CHECK (length(btrim(build_id)) > 0),
    CONSTRAINT firmware_builds_version_not_empty CHECK (length(btrim(version)) > 0)
);

CREATE TABLE compatibility_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adapter_version_id UUID NOT NULL REFERENCES adapter_versions (id),
    platform_id UUID NOT NULL,
    hardware_variant_id UUID NOT NULL,
    firmware_build_id UUID NOT NULL,
    status TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT compatibility_rules_hardware_platform_fkey
        FOREIGN KEY (hardware_variant_id, platform_id)
        REFERENCES hardware_variants (id, platform_id),
    CONSTRAINT compatibility_rules_build_variant_platform_fkey
        FOREIGN KEY (firmware_build_id, hardware_variant_id, platform_id)
        REFERENCES firmware_builds (id, hardware_variant_id, platform_id),
    CONSTRAINT compatibility_rules_adapter_platform_hardware_build_key
        UNIQUE (adapter_version_id, platform_id, hardware_variant_id, firmware_build_id),
    CONSTRAINT compatibility_rules_status_check
        CHECK (status IN ('supported', 'unsupported', 'unknown'))
);

CREATE INDEX firmware_builds_hardware_platform_idx
    ON firmware_builds (hardware_variant_id, platform_id);
CREATE INDEX compatibility_rules_hardware_platform_idx
    ON compatibility_rules (hardware_variant_id, platform_id);
CREATE INDEX compatibility_rules_firmware_build_hardware_platform_idx
    ON compatibility_rules (firmware_build_id, hardware_variant_id, platform_id);

-- +goose Down
DROP TABLE compatibility_rules;
DROP TABLE firmware_builds;
DROP TABLE hardware_variants;
DROP TABLE adapter_versions;
DROP TABLE adapter_definitions;
