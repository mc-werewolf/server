CREATE TABLE game_servers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash       BYTEA NOT NULL,
    display_name     TEXT NOT NULL,
    world_name       TEXT NOT NULL,
    host_name        TEXT,
    host_port        INTEGER,
    connection_mode  TEXT NOT NULL DEFAULT 'pending',
    player_count     INTEGER NOT NULL DEFAULT 0,
    max_players      INTEGER NOT NULL DEFAULT 10,
    status           TEXT NOT NULL DEFAULT 'starting',
    lease_expires_at TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (length(display_name) BETWEEN 1 AND 80),
    CHECK (length(world_name) BETWEEN 1 AND 80),
    CHECK (host_port IS NULL OR host_port BETWEEN 1 AND 65535),
    CHECK (connection_mode IN ('pending', 'direct', 'relay')),
    CHECK (player_count >= 0),
    CHECK (max_players BETWEEN 1 AND 100),
    CHECK (status IN ('starting', 'online', 'stopping'))
);

CREATE INDEX game_servers_active_idx
    ON game_servers (lease_expires_at DESC)
    WHERE status IN ('starting', 'online');
