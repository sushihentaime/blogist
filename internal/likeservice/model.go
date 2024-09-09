package likeservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
)

var (
	ErrAlreadyLiked = errors.New("like found")
	ErrLikeNotFound = errors.New("like not found")
)

func newLikeModel(db *sql.DB) *likeModel {
	return &likeModel{db: db}
}

func (m *likeModel) createLike(ctx context.Context, blogID, userID int) error {
	query := `
		INSERT INTO likes (blog_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (blog_id, user_id)
		DO UPDATE SET deleted_at = NULL
		RETURNING (xmax = 0) AS inserted`

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var inserted bool
	err := m.db.QueryRowContext(dbCtx, query, blogID, userID).Scan(&inserted)
	if err != nil {
		switch {
		case common.ForeignKeyError(err, "likes_blog_id_fkey"):
			return ErrInvalidBlogID
		case common.ForeignKeyError(err, "likes_user_id_fkey"):
			return ErrInvalidUserID
		default:
			return fmt.Errorf("failed to create like: %w", err)
		}
	}

	if !inserted {
		return ErrAlreadyLiked
	}

	return nil
}

func (m *likeModel) deleteLike(ctx context.Context, blogID int, userID int) error {
	query := `
		UPDATE likes
		SET deleted_at = $3
		WHERE blog_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := m.db.ExecContext(dbCtx, query, blogID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete like: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// case 1 - like not found
	// case 2 - blog_id or user_id is invalid
	if rowsAffected == 0 {
		return ErrLikeNotFound
	}

	return nil
}

func (m *likeModel) getUsersWhoLikedPost(ctx context.Context, blogID int) ([]LikedUser, error) {
	query := `
		SELECT u.id, u.username
		FROM likes l
		INNER JOIN users u ON l.user_id = u.id
		WHERE l.blog_id = $1 AND l.deleted_at IS NULL`

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := m.db.QueryContext(dbCtx, query, blogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users who liked post: %w", err)
	}
	defer rows.Close()

	var users []LikedUser
	for rows.Next() {
		var user LikedUser
		err := rows.Scan(&user.UserID, &user.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows: %w", err)
	}

	return users, nil
}

func (m *likeModel) getLikeCount(ctx context.Context, blogID int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM likes
		WHERE blog_id = $1 AND deleted_at IS NULL`

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := m.db.QueryRowContext(dbCtx, query, blogID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count likes: %w", err)
	}

	return count, nil
}
