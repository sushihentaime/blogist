package blogservice

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sushihentaime/blogist/internal/common"
)

func NewBlogService(db *sql.DB, c *common.Cache) *BlogService {
	return &BlogService{m: newBlogModel(db), c: c}
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

	return s.m.insert(req.Title, req.Content, req.UserID)
}

// GetBlogByID returns a blog post by its ID.
func (s *BlogService) GetBlogByID(ctx context.Context, id int) (*Blog, error) {
	v := common.NewValidator()
	validateInt(v, id, "id")
	if !v.Valid() {
		return nil, v.ValidationError()
	}

	fmt.Println("I am working")
	// Check cache first before querying the database.
	if blog, ok := s.c.Get(common.CacheKeyBlog(id)); ok {
		return blog.(*Blog), nil
	}

	fmt.Println("I am working 2")

	blog, err := s.m.getBlogById(id)
	if err != nil {
		return nil, err
	}

	// Cache the pointer to the blog post.
	s.c.Set(common.CacheKeyBlog(id), blog)

	return blog, nil
}

// UpdateBlog updates a blog post. The user ID must be provided. Only the user who created the blog post can update it.
func (s *BlogService) UpdateBlog(ctx context.Context, title, content string, id, userId *int, version *int) error {
	v := common.NewValidator()
	if title != "" {
		validateTitle(v, title)
	}

	if content != "" {
		validateContent(v, content)
	}

	if id != nil {
		validateInt(v, *id, "id")
	}

	if userId != nil {
		validateInt(v, *userId, "user_id")
	}

	if version != nil {
		validateInt(v, *version, "version")
	}

	if !v.Valid() {
		return v.ValidationError()
	}

	blog := Blog{
		ID:      *id,
		Title:   title,
		Content: content,
		UserID:  *userId,
		Version: *version,
	}

	return s.m.updateBlog(&blog)
}

// DeleteBlog deletes a blog post. Only the user who created the blog post can delete it.
func (s *BlogService) DeleteBlog(ctx context.Context, blogId, userId int) error {
	v := common.NewValidator()
	validateInt(v, blogId, "id")
	validateInt(v, userId, "user_id")
	if !v.Valid() {
		return v.ValidationError()
	}

	return s.m.deleteBlog(blogId, userId)
}

// GetBlogsByUserId returns all blog posts by a user.
func (s *BlogService) GetBlogsByUserId(ctx context.Context, userID int) (*[]Blog, error) {
	v := common.NewValidator()
	validateInt(v, userID, "user_id")
	if !v.Valid() {
		return nil, v.ValidationError()
	}

	// Check cache first before querying the database.
	if blogs, ok := s.c.Get(common.CacheKeyBlogsByUserId(userID)); ok {
		return blogs.(*[]Blog), nil
	}

	blogs, err := s.m.getBlogsByUserId(userID)
	if err != nil {
		return nil, err
	}

	// Cache the pointer to the slice of blog posts.
	s.c.Set(common.CacheKeyBlogsByUserId(userID), blogs)

	return blogs, nil
}

// GetBlogs returns all blog posts. Default limit is 10 and default offset is 0.
func (s *BlogService) GetBlogs(ctx context.Context, limit, offset int) (*[]Blog, error) {
	if limit < 1 {
		limit = 10
	}

	if offset < 0 {
		offset = 0
	}

	// Check cache first before querying the database.
	if blogs, ok := s.c.Get(common.CacheKeyBlogs(limit, offset)); ok {
		return blogs.(*[]Blog), nil
	}

	blogs, err := s.m.getBlogs(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Cache the pointer to the slice of blog posts.
	s.c.Set(common.CacheKeyBlogs(limit, offset), blogs)

	return blogs, nil
}

func (s *BlogService) GetBlogsByTitle(ctx context.Context, title string, limit, offset int) (*[]Blog, error) {
	v := common.NewValidator()
	validateTitle(v, title)
	if !v.Valid() {
		return nil, v.ValidationError()
	}

	if limit < 1 {
		limit = 10
	}

	if offset < 0 {
		offset = 0
	}

	// Check cache first before querying the database.
	if blogs, ok := s.c.Get(title); ok {
		return blogs.(*[]Blog), nil
	}

	blogs, err := s.m.getBlogsByTitle(ctx, title, limit, offset)
	if err != nil {
		return nil, err
	}

	// Cache the pointer to the slice of blog posts.
	s.c.Set(title, blogs)

	return blogs, nil
}
