CREATE TABLE IF NOT EXISTS token_scopes (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

INSERT INTO token_scopes (name)
VALUES
    ('token:activate');

CREATE TABLE IF NOT EXISTS tokens (
    hash BYTEA,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope_id INT NOT NULL REFERENCES token_scopes(id) ON DELETE CASCADE,
    expiry timestamptz NOT NULL,
    PRIMARY KEY(user_id, scope_id)
);
