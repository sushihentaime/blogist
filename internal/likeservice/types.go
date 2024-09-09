package likeservice

import (
	"database/sql"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
)

type Like struct {
	BlogID    int       `json:"blog_id"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type LikedUser struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
}

type likeModel struct {
	db *sql.DB
}

type LikeService struct {
	m *likeModel
	c *common.Cache
}
