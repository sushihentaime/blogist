package blogservice

import (
	"database/sql"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/userservice"
)

type Blog struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	// Content is stored in Markdown format.
	Content   string           `json:"content"`
	User      userservice.User `json:"user"`
	UserID    int              `json:"user_id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Version   int              `json:"version"`
}

type BlogModel struct {
	db *sql.DB
}

type BlogService struct {
	m *BlogModel
	c *common.Cache
}
