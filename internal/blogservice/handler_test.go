package blogservice

import (
	"context"
	"crypto/rand"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sushihentaime/blogist/internal/common"
)

// setupTestUser is a helper function to create a test user in the database.
func setupTestUser(db *sql.DB) (*int, error) {
	// set the password
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO users (username, email, password)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int
	err = db.QueryRow(query, "testuser", "testuser@example.com", randomBytes).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

func setupTestEnvironment(t *testing.T) (*BlogService, *sql.DB, func() error, *int, error) {
	db := common.TestDB("file://../../migrations", t)
	cache := common.NewCache(5*time.Minute, 10*time.Minute)

	// set the password
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	id, err := setupTestUser(db)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	cleanup := func() error {
		_, err := db.Exec("DELETE FROM blogs")
		if err != nil {
			return err
		}

		cache.Flush()

		return nil
	}

	return NewBlogService(db, cache), db, cleanup, id, nil
}

func createRandomBlog(db *sql.DB, userId int) (*int, *int, error) {
	query := `
		INSERT INTO blogs (title, content, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, version`

	var id, version int
	err := db.QueryRow(query, "Test Blog", "This is a test blog.", userId).Scan(&id, &version)
	if err != nil {
		return nil, nil, err
	}

	return &id, &version, nil
}

func TestCreateBlog(t *testing.T) {
	s, db, cleanup, userId, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	testCases := []struct {
		name        string
		blog        *CreateBlogRequest
		expectedErr error
	}{
		{
			name: "valid blog",
			blog: &CreateBlogRequest{
				Title:   "Test Blog",
				Content: "This is a test blog.",
				UserID:  *userId,
			},
			expectedErr: nil,
		},
		{
			name: "empty title",
			blog: &CreateBlogRequest{
				Title:   "",
				Content: "This is a test blog.",
				UserID:  *userId,
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"title": "must be provided"}},
		},
		{
			name: "empty content",
			blog: &CreateBlogRequest{
				Title:   "Test Blog",
				Content: "",
				UserID:  *userId,
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"content": "must be provided"}},
		},
		{
			name: "empty user ID",
			blog: &CreateBlogRequest{
				Title:   "Test Blog",
				Content: "This is a test blog.",
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"user_id": "must be provided"}},
		},
		{
			name: "invalid user ID",
			blog: &CreateBlogRequest{
				Title:   "Test Blog",
				Content: "This is a test blog.",
				UserID:  999,
			},
			expectedErr: ErrUserForeignKey,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			err := s.CreateBlog(ctx, tc.blog)
			assert.Equal(t, tc.expectedErr, err)

			if err == nil {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM blogs").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			}

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetBlogById(t *testing.T) {
	s, db, cleanup, userId, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	blogId, _, err := createRandomBlog(db, *userId)
	assert.NoError(t, err)

	testCases := []struct {
		name        string
		id          int
		expectedErr error
	}{
		{
			name:        "valid ID",
			id:          *blogId,
			expectedErr: nil,
		},
		{
			name:        "invalid ID",
			id:          999,
			expectedErr: common.ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			blog, err := s.GetBlogByID(ctx, tc.id)
			if tc.expectedErr != nil {
				assert.Nil(t, blog)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NotNil(t, blog)
				assert.NoError(t, err)
			}

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}

func TestUpdateBlog(t *testing.T) {
	s, db, cleanup, userId, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	testCases := []struct {
		name           string
		blog           *Blog
		setup          func(db *sql.DB, userId int) (*int, *int, error)
		expectedResult *Blog
		expectedErr    error
	}{
		{
			name: "valid blog",
			blog: &Blog{
				Title:   "Updated Blog",
				Content: "This is an updated blog.",
			},
			setup: createRandomBlog,
			expectedResult: &Blog{
				Title:   "Updated Blog",
				Content: "This is an updated blog.",
			},
			expectedErr: nil,
		},
		{
			name: "empty title",
			blog: &Blog{
				Title:   "",
				Content: "This is an updated blog.",
			},
			setup: createRandomBlog,
			expectedResult: &Blog{
				Title:   "Test Blog",
				Content: "This is an updated blog.",
			},
			expectedErr: nil,
		},
		{
			name: "empty content",
			blog: &Blog{
				Title:   "Updated Blog",
				Content: "",
			},
			setup: createRandomBlog,
			expectedResult: &Blog{
				Title:   "Updated Blog",
				Content: "This is a test blog.",
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if tc.setup != nil {
				blogId, versionId, err := tc.setup(db, *userId)
				assert.NoError(t, err)

				tc.blog.ID = *blogId
				tc.blog.UserID = *userId
				tc.blog.Version = *versionId
			}

			err := s.UpdateBlog(ctx, tc.blog.Title, tc.blog.Content, &tc.blog.ID, &tc.blog.UserID, &tc.blog.Version)
			assert.Equal(t, tc.expectedErr, err)

			var b Blog
			err = db.QueryRow("SELECT title, content FROM blogs WHERE id = $1", tc.blog.ID).Scan(&b.Title, &b.Content)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResult.Title, b.Title)
			assert.Equal(t, tc.expectedResult.Content, b.Content)

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}

func TestDeleteBlog(t *testing.T) {
	s, db, cleanup, userId, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	blogId, _, err := createRandomBlog(db, *userId)
	assert.NoError(t, err)

	testCases := []struct {
		name        string
		blogId      int
		userId      int
		expectedErr error
	}{
		{
			name:        "valid ID",
			blogId:      *blogId,
			userId:      *userId,
			expectedErr: nil,
		},
		{
			name:        "invalid ID",
			blogId:      999,
			userId:      *userId,
			expectedErr: common.ErrRecordNotFound,
		},
		{
			name:        "invalid user ID",
			blogId:      *blogId,
			userId:      999,
			expectedErr: common.ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			err := s.DeleteBlog(ctx, tc.blogId, tc.userId)
			assert.Equal(t, tc.expectedErr, err)

			if err == nil {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM blogs").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
			}

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetBlogsByUserId(t *testing.T) {
	s, db, cleanup, userId, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	// create a loop to create multiple blog posts
	for i := 0; i < 5; i++ {
		_, _, err := createRandomBlog(db, *userId)
		assert.NoError(t, err)
	}

	testCases := []struct {
		name        string
		userId      int
		expectedErr error
	}{
		{
			name:        "valid ID",
			userId:      *userId,
			expectedErr: nil,
		},
		{
			name:        "invalid ID",
			userId:      999,
			expectedErr: common.ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			_, err := s.GetBlogsByUserId(ctx, tc.userId)
			assert.Equal(t, tc.expectedErr, err)

			if err == nil {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM blogs WHERE user_id = $1", tc.userId).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 5, count)
			}

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetBlogs(t *testing.T) {
	s, db, cleanup, userId, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	setup := func() error {
		for i := 0; i < 5; i++ {
			_, _, err := createRandomBlog(db, *userId)
			if err != nil {
				return err
			}
		}
		return nil
	}

	testCases := []struct {
		name          string
		setup         func() error
		limit         int
		offset        int
		expectedCount int
		expectedErr   error
	}{
		{
			name:          "valid limit and offset",
			setup:         setup,
			limit:         2,
			offset:        0,
			expectedCount: 2,
			expectedErr:   nil,
		},
		{
			name:          "invalid limit",
			setup:         setup,
			limit:         0,
			offset:        0,
			expectedCount: 5,
			expectedErr:   nil,
		},
		{
			name:          "invalid offset",
			setup:         setup,
			limit:         5,
			offset:        -1,
			expectedCount: 5,
			expectedErr:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			err := tc.setup()
			assert.NoError(t, err)

			blogs, err := s.GetBlogs(ctx, tc.limit, tc.offset)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedCount, len(*blogs))

			if err == nil {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM blogs").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 5, count)
			}

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetBlogsByTitle(t *testing.T) {
	s, db, cleanup, userId, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	setup := func() error {
		for i := 0; i < 5; i++ {
			_, _, err := createRandomBlog(db, *userId)
			if err != nil {
				return err
			}
		}
		return nil
	}

	testCases := []struct {
		name        string
		setup       func() error
		title       string
		limit       int
		offset      int
		expectedErr error
	}{
		{
			name:        "valid title",
			setup:       setup,
			title:       "Test Blog",
			limit:       5,
			offset:      0,
			expectedErr: nil,
		},
		{
			name:        "invalid title",
			setup:       setup,
			title:       "Invalid Blog",
			limit:       5,
			offset:      0,
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			err := tc.setup()
			assert.NoError(t, err)

			_, err = s.GetBlogsByTitle(ctx, tc.title, tc.limit, tc.offset)
			assert.Equal(t, tc.expectedErr, err)

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}
