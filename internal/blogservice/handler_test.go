package blogservice

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
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
	db := common.TestDB(t)

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

		return nil
	}

	return NewBlogService(db), db, cleanup, id, nil
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
	fmt.Printf("userId: %v\n", *userId)
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
			expectedErr: fmt.Errorf("validation failed: map[title:must be provided]"),
		},
		{
			name: "empty content",
			blog: &CreateBlogRequest{
				Title:   "Test Blog",
				Content: "",
				UserID:  *userId,
			},
			expectedErr: fmt.Errorf("validation failed: map[content:must be provided]"),
		},
		{
			name: "empty user ID",
			blog: &CreateBlogRequest{
				Title:   "Test Blog",
				Content: "This is a test blog.",
			},
			expectedErr: fmt.Errorf("validation failed: map[id:must be greater than zero]"),
		},
		{
			name: "invalid user ID",
			blog: &CreateBlogRequest{
				Title:   "Test Blog",
				Content: "This is a test blog.",
				UserID:  999,
			},
			expectedErr: fmt.Errorf("user with ID 999 does not exist"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			fmt.Printf("blog: %v\n", tc.blog)
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
			expectedErr: ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			blog, err := s.GetBlogByID(ctx, tc.id)
			fmt.Printf("blog: %v\n", blog)
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

	blogId, versionId, err := createRandomBlog(db, *userId)
	assert.NoError(t, err)

	testCases := []struct {
		name string
		blog *Blog
		err  error
	}{
		{
			name: "valid blog",
			blog: &Blog{
				ID:      *blogId,
				Title:   "Updated Blog",
				Content: "This is an updated blog.",
				UserID:  *userId,
				Version: *versionId,
			},
			err: nil,
		},
		{
			name: "empty title",
			blog: &Blog{
				ID:      *blogId,
				Title:   "",
				Content: "This is an updated blog.",
				UserID:  *userId,
				Version: *versionId,
			},
			err: fmt.Errorf("validation failed: map[title:must be provided]"),
		},
		{
			name: "empty content",
			blog: &Blog{
				ID:      *blogId,
				Title:   "Updated Blog",
				Content: "",
				UserID:  *userId,
				Version: *versionId,
			},
			err: fmt.Errorf("validation failed: map[content:must be provided]"),
		},
		{
			name: "empty user ID",
			blog: &Blog{
				ID:      *blogId,
				Title:   "Updated Blog",
				Content: "This is an updated blog.",
				Version: *versionId,
			},
			err: fmt.Errorf("validation failed: map[id:must be greater than zero]"),
		},
		{
			name: "invalid user ID",
			blog: &Blog{
				ID:      *blogId,
				Title:   "Updated Blog",
				Content: "This is an updated blog.",
				UserID:  999,
				Version: *versionId,
			},
			err: ErrRecordNotFound,
		},
		{
			name: "invalid version",
			blog: &Blog{
				ID:      *blogId,
				Title:   "Updated Blog",
				Content: "This is an updated blog.",
				UserID:  *userId,
				Version: 999,
			},
			err: ErrRecordNotFound,
		},
		{
			name: "invalid ID",
			blog: &Blog{
				ID:      999,
				Title:   "Updated Blog",
				Content: "This is an updated blog.",
				UserID:  *userId,
				Version: *versionId,
			},
			err: ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := s.UpdateBlog(ctx, tc.blog)
			assert.Equal(t, tc.err, err)

			if err == nil {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM blogs WHERE title = 'Updated Blog'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			} else {
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM blogs WHERE title = 'Updated Blog'").Scan(&count)
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
			expectedErr: ErrRecordNotFound,
		},
		{
			name:        "invalid user ID",
			blogId:      *blogId,
			userId:      999,
			expectedErr: ErrRecordNotFound,
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
			expectedErr: ErrRecordNotFound,
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
		name        string
		setup       func() error
		limit       int
		offset      int
		expectedErr error
	}{
		{
			name:        "valid limit and offset",
			setup:       setup,
			limit:       5,
			offset:      0,
			expectedErr: nil,
		},
		{
			name:        "invalid limit",
			setup:       setup,
			limit:       0,
			offset:      0,
			expectedErr: nil,
		},
		{
			name:        "invalid offset",
			setup:       setup,
			limit:       5,
			offset:      -1,
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			err := tc.setup()
			assert.NoError(t, err)

			_, err = s.GetBlogs(ctx, tc.limit, tc.offset)
			assert.Equal(t, tc.expectedErr, err)

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
