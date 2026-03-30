CREATE table IF NOT EXISTS sowhat.users(
    id BIGINT PRIMARY KEY,
    first_name TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()    
)