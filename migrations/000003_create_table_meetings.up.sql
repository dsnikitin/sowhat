CREATE table IF NOT EXISTS sowhat.meetings(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES sowhat.users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    transcript TEXT,
    summary TEXT,
    embedding tsvector,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
