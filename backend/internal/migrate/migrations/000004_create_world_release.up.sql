CREATE TABLE world_release (
    singleton_id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (singleton_id = 1),
    version      TEXT NOT NULL,
    file_name    TEXT NOT NULL,
    original_name TEXT NOT NULL,
    file_size    BIGINT NOT NULL CHECK (file_size > 0),
    sha256       TEXT NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
