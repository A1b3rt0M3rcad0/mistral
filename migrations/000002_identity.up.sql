CREATE TABLE character_ownerships (
    character_id TEXT PRIMARY KEY REFERENCES characters(id) ON DELETE CASCADE,
    subject_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX character_ownerships_subject_id_idx ON character_ownerships (subject_id);
