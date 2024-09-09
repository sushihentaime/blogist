CREATE TABLE IF NOT EXISTS likes (
    blog_id INT NOT NULL REFERENCES blogs(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    deleted_at timestamptz,
    PRIMARY KEY (blog_id, user_id)
);

CREATE INDEX idx_likes_blog_id ON likes (blog_id);
CREATE INDEX idx_likes_user_id ON likes (user_id);
CREATE INDEX idx_likes_created_at ON likes (created_at);
CREATE INDEX idx_likes_deleted_at ON likes (deleted_at);
