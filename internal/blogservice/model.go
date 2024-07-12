package blogservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrUserForeignKey = errors.New("user_id does not exist")
)

func newBlogModel(db *sql.DB) *BlogModel {
	return &BlogModel{db: db}
}

// ForeignKeyError is a helper function to check if the error is a foreign key constraint error.
func ForeignKeyError(err error, name string) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if pqErr.Code == "23503" && pqErr.Constraint == name {
			return true
		}
	}

	return false
}

func (m *BlogModel) insert(ctx context.Context, title, content string, id int) error {
	query := `
		INSERT INTO blogs (title, content, user_id)
		VALUES ($1, $2, $3)`

	_, err := m.db.ExecContext(ctx, query, title, content, id)
	if err != nil {
		switch {
		case ForeignKeyError(err, "blogs_user_id_fkey"):
			return ErrUserForeignKey
		default:
			return err
		}
	}

	return nil
}

// getBlogById is a method to get a blog by its ID joining the users table to get the user's name.
func (m *BlogModel) getBlogById(ctx context.Context, id int) (*Blog, error) {
	query := `
		SELECT b.id, b.title, b.content, b.user_id, b.created_at, b.updated_at, b.version, u.username
		FROM blogs b
		JOIN users u ON b.user_id = u.id
		WHERE b.id = $1`

	row := m.db.QueryRowContext(ctx, query, id)

	var blog Blog
	err := row.Scan(&blog.ID, &blog.Title, &blog.Content, &blog.User.ID, &blog.CreatedAt, &blog.UpdatedAt, &blog.Version, &blog.User.Username)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &blog, nil

}

func (m *BlogModel) updateBlog(ctx context.Context, blog *Blog) error {
	query := `
		UPDATE blogs
		SET title = $1, content = $2, version = version + 1
		WHERE id = $3 AND version = $4 AND user_id = $5
		RETURNING version, created_at, updated_at`

	err := m.db.QueryRowContext(ctx, query, blog.Title, blog.Content, blog.ID, blog.Version, blog.UserID).Scan(&blog.Version, &blog.CreatedAt, &blog.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	return nil
}

func (m *BlogModel) deleteBlog(ctx context.Context, blogId, userId int) error {
	query := `
		DELETE FROM blogs
		WHERE id = $1 AND user_id = $2`

	res, err := m.db.ExecContext(ctx, query, blogId, userId)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		switch {
		case rows == 0:
			return ErrRecordNotFound
		default:
			return fmt.Errorf("expected 1 row to be affected, got %d", rows)
		}
	}

	return nil
}

func (m *BlogModel) getBlogsByUserId(ctx context.Context, userID int) (*[]Blog, error) {
	query := `
		SELECT id, title, content, user_id, created_at, updated_at, version
		FROM blogs
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// found is a flag to check if any records were found
	found := false

	var blogs []Blog
	for rows.Next() {
		var blog Blog
		err := rows.Scan(&blog.ID, &blog.Title, &blog.Content, &blog.User.ID, &blog.CreatedAt, &blog.UpdatedAt, &blog.Version)
		if err != nil {
			return nil, err
		}
		blogs = append(blogs, blog)
		found = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !found {
		return nil, ErrRecordNotFound
	}

	return &blogs, nil
}

// getBlogs to get all blogs. set limit and offset to get paginated results and sort the results by created_at descending order
func (m *BlogModel) getBlogs(ctx context.Context, limit, offset int) ([]Blog, error) {
	query := `
		SELECT id, title, content, user_id, created_at, updated_at, version
		FROM blogs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := m.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var blog Blog
		err := rows.Scan(&blog.ID, &blog.Title, &blog.Content, &blog.User.ID, &blog.CreatedAt, &blog.UpdatedAt, &blog.Version)
		if err != nil {
			return nil, err
		}
		blogs = append(blogs, blog)
	}

	return blogs, nil
}

// getBlogsByTitle is a method to get blogs by title. This method is used to demonstrate the use of LIKE operator in SQL query.
func (m *BlogModel) getBlogsByTitle(ctx context.Context, title string, limit, offset int) ([]Blog, error) {
	query := `
		SELECT id, title, content, user_id, created_at, updated_at, version
		FROM blogs
		WHERE title LIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := m.db.QueryContext(ctx, query, "%"+title+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var blog Blog
		err := rows.Scan(&blog.ID, &blog.Title, &blog.Content, &blog.User.ID, &blog.CreatedAt, &blog.UpdatedAt, &blog.Version)
		if err != nil {
			return nil, err
		}
		blogs = append(blogs, blog)
	}

	return blogs, nil
}
