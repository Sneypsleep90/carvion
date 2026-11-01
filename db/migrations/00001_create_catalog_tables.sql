-- +goose Up
CREATE TABLE manufacturers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE vehicle_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manufacturer_id UUID NOT NULL REFERENCES manufacturers (id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT vehicle_models_manufacturer_name_key UNIQUE (manufacturer_id, name)
);

CREATE TABLE infotainment_platforms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    manufacturer_id UUID REFERENCES manufacturers (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- These leading manufacturer_id indexes also support the foreign key lookups.
CREATE UNIQUE INDEX infotainment_platforms_manufacturer_name_key
    ON infotainment_platforms (manufacturer_id, name)
    WHERE manufacturer_id IS NOT NULL;

CREATE UNIQUE INDEX infotainment_platforms_global_name_key
    ON infotainment_platforms (name)
    WHERE manufacturer_id IS NULL;

-- +goose Down
DROP TABLE infotainment_platforms;
DROP TABLE vehicle_models;
DROP TABLE manufacturers;
