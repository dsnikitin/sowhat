CREATE table IF NOT EXISTS sowhat.meetings(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES sowhat.users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    transcript TEXT,
    summary TEXT,
	chatter_file_id TEXT,
	transcript_tsv TSVECTOR GENERATED ALWAYS AS (to_tsvector('russian', transcript)) STORED,
	is_transcription_failed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    raw_transcript TEXT
);

CREATE INDEX idx_meetings_user_id ON sowhat.meetings(user_id);
CREATE INDEX idx_meetings_transcript_tsv ON sowhat.meetings USING GIN (transcript_tsv);
