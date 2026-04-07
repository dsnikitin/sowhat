CREATE table IF NOT EXISTS sowhat.transcriptions(
	meeting_id BIGINT PRIMARY KEY REFERENCES sowhat.meetings(id) ON UPDATE CASCADE ON DELETE CASCADE,
	transcriber_rq_file_id TEXT,
	transcriber_task_id TEXT,
	transcriber_rs_file_id TEXT,
	is_completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
