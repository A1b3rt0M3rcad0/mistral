ALTER TABLE content_releases
    ADD COLUMN payload_schema_version SMALLINT NOT NULL DEFAULT 1,
    ADD CONSTRAINT content_releases_payload_schema_version_check CHECK (payload_schema_version > 0);
