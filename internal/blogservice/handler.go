package blogservice

import (
	"context"
	"database/sql"

	"github.com/sushihentaime/blogist/internal/common"
)

func NewBlogService(db *sql.DB) *BlogService {
	return &BlogService{m: newBlogModel(db)}
}

type CreateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	UserID  int    `json:"user_id"`
}

// CreateBlog creates a new blog post. The user ID must be provided.
func (s *BlogService) CreateBlog(ctx context.Context, req *CreateBlogRequest) error {
	v := common.NewValidator()
	validateTitle(v, req.Title)
	validateContent(v, req.Content)
	validateInt(v, req.UserID, "user_id")
	if !v.Valid() {
		return v.ValidationError()
	}

	return s.m.insert(ctx, req.Title, req.Content, req.UserID)
}

// GetBlogByID returns a blog post by its ID.
func (s *BlogService) GetBlogByID(ctx context.Context, id int) (*Blog, error) {
	v := common.NewValidator()
	validateInt(v, id, "id")
	if !v.Valid() {
		return nil, v.ValidationError()
	}

	return s.m.getBlogById(ctx, id)
}

// UpdateBlog updates a blog post. The user ID must be provided. Only the user who created the blog post can update it.
func (s *BlogService) UpdateBlog(ctx context.Context, blog *Blog) error {
	v := common.NewValidator()
	validateTitle(v, blog.Title)
	validateContent(v, blog.Content)
	validateInt(v, blog.ID, "id")
	validateInt(v, blog.UserID, "user_id")
	if !v.Valid() {
		return v.ValidationError()
	}

	return s.m.updateBlog(ctx, blog)
}

// DeleteBlog deletes a blog post. Only the user who created the blog post can delete it.
func (s *BlogService) DeleteBlog(ctx context.Context, blogId, userId int) error {
	v := common.NewValidator()
	validateInt(v, blogId, "id")
	validateInt(v, userId, "user_id")
	if !v.Valid() {
		return v.ValidationError()
	}

	return s.m.deleteBlog(ctx, blogId, userId)
}

// GetBlogsByUserId returns all blog posts by a user.
func (s *BlogService) GetBlogsByUserId(ctx context.Context, userID int) (*[]Blog, error) {
	v := common.NewValidator()
	validateInt(v, userID, "user_id")
	if !v.Valid() {
		return nil, v.ValidationError()
	}

	return s.m.getBlogsByUserId(ctx, userID)
}

// GetBlogs returns all blog posts. Default limit is 10 and default offset is 0.
func (s *BlogService) GetBlogs(ctx context.Context, limit, offset *int) ([]Blog, error) {
	if *limit < 1 {
		*limit = 10
	}

	if *offset < 0 {
		*offset = 0
	}

	return s.m.getBlogs(ctx, *limit, *offset)
}

func (s *BlogService) GetBlogsByTitle(ctx context.Context, title string, limit, offset *int) ([]Blog, error) {
	v := common.NewValidator()
	validateTitle(v, title)
	if !v.Valid() {
		return nil, v.ValidationError()
	}

	if *limit < 1 {
		*limit = 10
	}

	if *offset < 0 {
		*offset = 0
	}

	return s.m.getBlogsByTitle(ctx, title, *limit, *offset)
}
