CREATE TYPE permission AS ENUM ('blog:write');

CREATE TABLE IF NOT EXISTS user_permissions (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission permission NOT NULL,
    PRIMARY KEY (user_id, permission)
);
