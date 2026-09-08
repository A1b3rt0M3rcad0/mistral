CREATE TABLE content_releases (
    release_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    payload JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT content_releases_name_check CHECK (length(name) > 0),
    CONSTRAINT content_releases_version_check CHECK (length(version) > 0),
    CONSTRAINT content_releases_hash_format_check CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT content_releases_identity_check CHECK (release_id = version || '@sha256:' || content_hash)
);
