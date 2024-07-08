CREATE TABLE IF NOT EXISTS auth_tokens (
    access_token BYTEA,
    refresh_token BYTEA,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    access_token_expiry timestamptz NOT NULL,
    refresh_token_expiry timestamptz NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT NOT NULL,
    PRIMARY KEY (hash, user_id)
);
