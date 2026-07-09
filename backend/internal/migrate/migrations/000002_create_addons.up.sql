CREATE TABLE addons (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_owner  TEXT NOT NULL,
    github_repo   TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (github_owner, github_repo)
);

CREATE TABLE addon_versions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    addon_id           UUID NOT NULL REFERENCES addons(id) ON DELETE CASCADE,
    github_release_id  BIGINT NOT NULL,
    tag_name           TEXT NOT NULL,
    zip_asset_name     TEXT,
    zip_asset_url      TEXT,
    published_at       TIMESTAMPTZ NOT NULL,
    properties         JSONB,
    properties_error   TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (addon_id, github_release_id)
);

CREATE INDEX addon_versions_addon_id_idx ON addon_versions (addon_id);
