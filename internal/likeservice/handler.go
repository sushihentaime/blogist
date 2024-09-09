package likeservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sushihentaime/blogist/internal/common"
)

var (
	ErrInvalidBlogID = errors.New("invalid blog ID")
	ErrInvalidUserID = errors.New("invalid user ID")
)

func NewLikeService(db *sql.DB, c *common.Cache) *LikeService {
	return &LikeService{m: newLikeModel(db), c: c}
}

func (s *LikeService) CreateLike(ctx context.Context, blogID, userID int) error {
	if blogID <= 0 {
		return ErrInvalidBlogID
	}

	if userID <= 0 {
		return ErrInvalidUserID
	}

	return s.m.createLike(ctx, blogID, userID)
}

func (s *LikeService) DeleteLike(ctx context.Context, blogID, userID int) error {
	if blogID <= 0 {
		return ErrInvalidBlogID
	}

	if userID <= 0 {
		return ErrInvalidUserID
	}

	return s.m.deleteLike(ctx, blogID, userID)
}

func (s *LikeService) GetLikeCount(ctx context.Context, blogID int) (int, error) {
	if blogID <= 0 {
		return 0, ErrInvalidBlogID
	}

	// Check the cache first
	if count, ok := s.c.Get(common.CacheKeyLikeCount(blogID)); ok {
		return count.(int), nil
	}

	count, err := s.m.getLikeCount(ctx, blogID)
	if err != nil {
		return 0, fmt.Errorf("failed to count likes for blogID %d: %w", blogID, err)
	}

	s.c.Set(common.CacheKeyLikeCount(blogID), count)

	return count, nil
}

func (s *LikeService) GetUsersWhoLikedPost(ctx context.Context, blogID int) ([]LikedUser, error) {
	if blogID <= 0 {
		return nil, ErrInvalidBlogID
	}

	// Check the cache first
	if users, ok := s.c.Get(common.CacheKeyLikedUsers(blogID)); ok {
		return users.([]LikedUser), nil
	}

	users, err := s.m.getUsersWhoLikedPost(ctx, blogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users who liked post %d: %w", blogID, err)
	}

	s.c.Set(common.CacheKeyLikedUsers(blogID), users)

	return users, nil
}
