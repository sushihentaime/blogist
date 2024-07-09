CREATE TABLE IF NOT EXISTS auth_tokens (
    access_token BYTEA,
    refresh_token BYTEA,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    access_token_expiry timestamptz NOT NULL,
    refresh_token_expiry timestamptz NOT NULL,
    PRIMARY KEY (access_token, user_id)
);
