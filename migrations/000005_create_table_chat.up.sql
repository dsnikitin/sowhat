CREATE table IF NOT EXISTS sowhat.chat(
	user_id BIGINT REFERENCES sowhat.users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
	id TEXT NOT NULL,
	query TEXT NOT NULL,
	answer TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_user_id ON sowhat.chat(user_id);
