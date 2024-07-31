package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sushihentaime/blogist/internal/userservice"
	"golang.org/x/crypto/bcrypt"
)

func intptr(i int) *int {
	return &i
}

func TestRegisterUserHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		payload    any
		setup      func(db *sql.DB) error
		wantStatus int
		wantBody   envelope
	}{
		{
			name: "Valid Request",
			payload: map[string]any{
				"username": "testuser",
				"email":    "testuser@example.com",
				"password": "Test_1234!",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "Invalid Payload",
			payload: map[string]any{
				"username": "testuser",
				"email":    "test",
				"password": "Test_1234!",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   envelope{"error": map[string]string{"email": "must be a valid email address"}},
		},
		{
			name: "Duplicate Email",
			payload: map[string]any{
				"username": "user1",
				"email":    "testuser@example.com",
				"password": "Test_1234!",
			},
			setup: func(db *sql.DB) error {
				randomHash := make([]byte, 16)
				_, err := rand.Read(randomHash)
				if err != nil {
					return err
				}

				_, err = db.Exec("INSERT INTO users (username, email, password) VALUES ($1, $2, $3)", "testuser", "testuser@example.com", randomHash)
				return err
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   envelope{"error": map[string]string{"email": "a user with this email address already exists"}},
		},
		{
			name: "Duplicate Username",
			payload: map[string]any{
				"username": "testuser",
				"email":    "testuser1@example.com",
				"password": "Test_1234!",
			},
			setup: func(db *sql.DB) error {
				randomHash := make([]byte, 16)
				_, err := rand.Read(randomHash)
				if err != nil {
					return err
				}

				_, err = db.Exec("INSERT INTO users (username, email, password) VALUES ($1, $2, $3)", "testuser", "testuser@example.co", randomHash)
				return err
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   envelope{"error": map[string]string{"username": "this username is already taken"}},
		},
		{
			name: "Invalid Password",
			payload: map[string]any{
				"username": "testuser",
				"email":    "testuser@example.com",
				"password": "password",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   envelope{"error": map[string]string{"password": "must be between 8 and 72 characters long and contain at least one uppercase letter, one lowercase letter, one number, and one symbol"}},
		},
		{
			name:       "Empty Payload",
			payload:    map[string]any{},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   envelope{"error": map[string]string{"email": "must be provided", "password": "must be provided", "username": "must be provided"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the test database
			if tc.setup != nil {
				err := tc.setup(db)
				assert.NoError(t, err)
			}

			// Create a new test request
			status, _, gotBody := ts.post(t, "/api/v1/users/register", tc.payload, nil)
			assert.Equal(t, tc.wantStatus, status)
			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestLoginUserHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	setup := func(db *sql.DB) error {
		// set the password for the test user
		b, err := bcrypt.GenerateFromPassword([]byte("Test_1234!"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = db.Exec("INSERT INTO users (username, email, password) VALUES ($1, $2, $3)", "testuser", "testuser@example.com", b)
		return err
	}

	testCases := []struct {
		name       string
		payload    any
		setup      func(db *sql.DB) error
		wantStatus int
		wantBody   envelope
	}{
		{
			name: "Valid Request",
			payload: map[string]any{
				"username": "testuser",
				"password": "Test_1234!",
			},
			setup:      setup,
			wantStatus: http.StatusOK,
		},
		{
			name: "Invalid Username",
			payload: map[string]any{
				"username": "testuser1",
				"password": "Test_1234!",
			},
			setup:      setup,
			wantStatus: http.StatusUnauthorized,
			wantBody:   envelope{"error": "invalid authentication credentials"},
		},
		{
			name: "Invalid Password",
			payload: map[string]any{
				"username": "testuser",
				"password": "Test1234!",
			},
			setup:      setup,
			wantStatus: http.StatusUnauthorized,
			wantBody:   envelope{"error": "invalid authentication credentials"},
		},
		{
			name:       "Empty Payload",
			payload:    map[string]any{},
			setup:      setup,
			wantStatus: http.StatusUnprocessableEntity,
			wantBody: envelope{"error": map[string]string{
				"password": "must be provided",
				"username": "must be provided",
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				err := tc.setup(db)
				assert.NoError(t, err)
			}

			status, _, gotBody := ts.post(t, "/api/v1/users/login", tc.payload, nil)
			assert.Equal(t, tc.wantStatus, status)
			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestLogoutUserHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	setup := func(db *sql.DB) (*string, error) {
		// set the password for the test user
		b, err := bcrypt.GenerateFromPassword([]byte("Test_1234!"), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		var userId int

		err = db.QueryRow("INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id", "testuser", "testuser@example.com", b).Scan(&userId)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// add permission to the test user
		_, err = db.Exec("INSERT INTO user_permissions (user_id, permission) VALUES ($1, $2)", userId, userservice.PermissionWriteBlog)
		if err != nil {
			return nil, fmt.Errorf("failed to add user permissions: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		token, err := app.userService.LoginUser(ctx, "testuser", "Test_1234!")
		if err != nil {
			return nil, fmt.Errorf("failed to login user: %w", err)
		}

		return &token.AccessTokenPlain, nil
	}

	testCases := []struct {
		name       string
		setup      func(db *sql.DB) (*string, error)
		wantStatus int
		wantBody   envelope
	}{
		{
			name:       "Valid Request",
			setup:      setup,
			wantStatus: http.StatusOK,
			wantBody:   envelope{"message": "user logged out"},
		},
		{
			name:       "Invalid Token",
			setup:      func(db *sql.DB) (*string, error) { return strptr("invalid-token"), nil },
			wantStatus: http.StatusForbidden,
			wantBody:   envelope{"error": "invalid or missing authentication token"},
		},
		{
			name:       "No Token",
			setup:      func(db *sql.DB) (*string, error) { return strptr(""), nil },
			wantStatus: http.StatusForbidden,
			wantBody:   envelope{"error": "invalid or missing authentication token"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := tc.setup(db)
			assert.NoError(t, err)

			req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/users/logout", nil)
			assert.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+*token)

			res, err := ts.Client().Do(req)
			assert.NoError(t, err)

			status, _, gotBody := readResponse(t, res)

			assert.Equal(t, tc.wantStatus, status)
			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func createTestUser(app *application, db *sql.DB, u *userservice.User) (*string, *int, error) {
	// set the password for the test user
	b, err := bcrypt.GenerateFromPassword([]byte("Test_1234!"), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	var userId int

	err = db.QueryRow("INSERT INTO users (username, email, password, activated) VALUES ($1, $2, $3, $4) RETURNING id", u.Username, u.Email, b, true).Scan(&userId)
	if err != nil {
		return nil, nil, err
	}

	// add permission to the test user
	_, err = db.Exec("INSERT INTO user_permissions (user_id, permission) VALUES ($1, $2)", userId, string(userservice.PermissionWriteBlog))
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := app.userService.LoginUser(ctx, u.Username, "Test_1234!")
	if err != nil {
		return nil, nil, err
	}

	return &token.AccessTokenPlain, &userId, nil
}

func TestCreateBlogHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		payload    any
		setup      func(app *application, db *sql.DB, u *userservice.User) (*string, *int, error)
		wantStatus int
		wantBody   envelope
	}{
		{
			name: "Valid Request",
			payload: map[string]any{
				"title":   "Test Blog",
				"content": "This is a test blog",
			},
			setup:      createTestUser,
			wantStatus: http.StatusCreated,
			wantBody:   envelope{"message": "blog created"},
		},
		{
			name: "Invalid Title",
			payload: map[string]any{
				"title":   "",
				"content": "This is a test blog",
			},
			setup:      createTestUser,
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   envelope{"error": map[string]string{"title": "must be provided"}},
		},
		{
			name: "Invalid Content",
			payload: map[string]any{
				"title":   "Test Blog",
				"content": "",
			},
			setup:      createTestUser,
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   envelope{"error": map[string]string{"content": "must be provided"}},
		},
		{
			name: "No Authentication Token",
			payload: map[string]any{
				"title":   "Test Blog",
				"content": "This is a test blog",
			},
			setup: func(app *application, db *sql.DB, u *userservice.User) (*string, *int, error) {
				return strptr(""), nil, nil
			},
			wantStatus: http.StatusForbidden,
			wantBody:   envelope{"error": "invalid or missing authentication token"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, _, err := tc.setup(app, db, &userservice.User{Username: "testuser", Email: "testuser@example.com"})
			assert.NoError(t, err)

			status, _, gotBody := ts.post(t, "/api/v1/blogs/create", tc.payload, token)
			assert.Equal(t, tc.wantStatus, status)
			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM blogs")
				assert.NoError(t, err)

				_, err = db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func createTestBlog(app *application, db *sql.DB) (*string, *int, *int, error) {
	authToken, userId, err := createTestUser(app, db, &userservice.User{Username: "testuser", Email: "testuser@example.com"})
	if err != nil {
		return nil, nil, nil, err
	}

	var blogId int
	err = db.QueryRow("INSERT INTO blogs (title, content, user_id) VALUES ($1, $2, $3) RETURNING id", "Test Blog", "This is a test blog", *userId).Scan(&blogId)
	if err != nil {
		return nil, nil, nil, err
	}

	return authToken, userId, &blogId, nil
}

func TestGetBlogHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		setup      func(app *application, db *sql.DB) (*string, *int, *int, error)
		wantStatus int
		wantBody   envelope
	}{
		{
			name:       "Valid Request with Authentication Token",
			setup:      createTestBlog,
			wantStatus: http.StatusOK,
		},
		{
			name: "No Authentication Token",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, userId, blogId, err := createTestBlog(app, db)
				return nil, userId, blogId, err
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Invalid Blog ID",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				token, userId, _, err := createTestBlog(app, db)
				return token, userId, intptr(10), err
			},
			wantStatus: http.StatusNotFound,
			wantBody:   envelope{"error": "resource not found"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, _, blogId, err := tc.setup(app, db)
			assert.NoError(t, err)

			if token == nil {
				status, _, gotBody := ts.get(t, fmt.Sprintf("/api/v1/blogs/view/%d", *blogId), nil, nil)
				assert.Equal(t, tc.wantStatus, status)

				if tc.wantBody != nil {
					assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
				}
			} else {
				status, _, gotBody := ts.get(t, fmt.Sprintf("/api/v1/blogs/view/%d", *blogId), token, nil)
				assert.Equal(t, tc.wantStatus, status)

				if tc.wantBody != nil {
					assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
				}
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM blogs")
				assert.NoError(t, err)

				_, err = db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestUpdateBlogHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		payload    any
		setup      func(app *application, db *sql.DB) (*string, *int, *int, error)
		wantStatus int
		wantBody   envelope
	}{
		{
			name: "Valid Request",
			payload: map[string]any{
				"title":   "Updated Blog",
				"content": "This is an updated blog",
			},
			setup:      createTestBlog,
			wantStatus: http.StatusOK,
			wantBody:   envelope{"message": "blog updated"},
		},
		{
			name: "Empty Title",
			payload: map[string]any{
				"title":   "",
				"content": "This is an updated blog",
			},
			setup:      createTestBlog,
			wantStatus: http.StatusOK,
			wantBody:   envelope{"message": "blog updated"},
		},
		{
			name: "Empty Content",
			payload: map[string]any{
				"title":   "Updated Blog",
				"content": "",
			},
			setup:      createTestBlog,
			wantStatus: http.StatusOK,
			wantBody:   envelope{"message": "blog updated"},
		},
		{
			name: "No Authentication Token",
			payload: map[string]any{
				"title":   "Updated Blog",
				"content": "This is an updated blog",
			},
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, userId, blogId, err := createTestBlog(app, db)
				return nil, userId, blogId, err
			},
			wantStatus: http.StatusForbidden,
			wantBody:   envelope{"error": "invalid or missing authentication token"},
		},
		{
			name: "Invalid Blog ID",
			payload: map[string]any{
				"title":   "Updated Blog",
				"content": "This is an updated blog",
			},
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				token, userId, _, err := createTestBlog(app, db)
				return token, userId, intptr(10), err
			},
			wantStatus: http.StatusNotFound,
			wantBody:   envelope{"error": "resource not found"},
		},
		{
			name: "Update Another User's Blog",
			payload: map[string]any{
				"title":   "Updated Blog",
				"content": "This is an updated blog",
			},
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, _, blogId, err := createTestBlog(app, db)
				if err != nil {
					return nil, nil, nil, err
				}

				token, userId2, err := createTestUser(app, db, &userservice.User{Username: "testuser2", Email: "testuser2@example.com"})
				if err != nil {
					return nil, nil, nil, err
				}

				return token, userId2, blogId, nil
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   envelope{"error": "unauthorized access"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, _, blogId, err := tc.setup(app, db)
			assert.NoError(t, err)

			status, _, gotBody := ts.put(t, fmt.Sprintf("/api/v1/blogs/update/%d", *blogId), token, tc.payload)
			assert.Equal(t, tc.wantStatus, status)

			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM blogs")
				assert.NoError(t, err)

				_, err = db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestDeleteBlogHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		setup      func(app *application, db *sql.DB) (*string, *int, *int, error)
		wantStatus int
		wantBody   envelope
	}{
		{
			name:       "Valid Request",
			setup:      createTestBlog,
			wantStatus: http.StatusOK,
			wantBody:   envelope{"message": "blog deleted"},
		},
		{
			name: "No Authentication Token",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, userId, blogId, err := createTestBlog(app, db)
				return nil, userId, blogId, err
			},
			wantStatus: http.StatusForbidden,
			wantBody:   envelope{"error": "invalid or missing authentication token"},
		},
		{
			name: "Invalid Blog ID",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				token, userId, _, err := createTestBlog(app, db)
				return token, userId, intptr(10), err
			},
			wantStatus: http.StatusNotFound,
			wantBody:   envelope{"error": "resource not found"},
		},
		{
			name: "Delete Another User's Blog",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, _, blogId, err := createTestBlog(app, db)
				if err != nil {
					return nil, nil, nil, err
				}

				token, userId2, err := createTestUser(app, db, &userservice.User{Username: "testuser2", Email: "testuser2@example.com"})
				if err != nil {
					return nil, nil, nil, err
				}

				return token, userId2, blogId, nil
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   envelope{"error": "unauthorized access"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, _, blogId, err := tc.setup(app, db)
			assert.NoError(t, err)

			status, _, gotBody := ts.delete(t, fmt.Sprintf("/api/v1/blogs/delete/%d", *blogId), token)
			assert.Equal(t, tc.wantStatus, status)

			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM blogs")
				assert.NoError(t, err)

				_, err = db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetBlogsByUserIdHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		setup      func(app *application, db *sql.DB) (*string, *int, *int, error)
		wantStatus int
		wantBody   envelope
	}{
		{
			name:       "Valid Request",
			setup:      createTestBlog,
			wantStatus: http.StatusOK,
		},
		{
			name: "No Authentication Token",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, userId, blogId, err := createTestBlog(app, db)
				return nil, userId, blogId, err
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Invalid User ID",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				token, _, blogId, err := createTestBlog(app, db)
				return token, intptr(10), blogId, err
			},
			wantStatus: http.StatusNotFound,
			wantBody:   envelope{"error": "resource not found"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, userId, _, err := tc.setup(app, db)
			assert.NoError(t, err)

			status, _, gotBody := ts.get(t, fmt.Sprintf("/api/v1/blogs/user/%d", *userId), token, nil)
			fmt.Printf("gotBody: %v\n", gotBody)
			assert.Equal(t, tc.wantStatus, status)

			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM blogs")
				assert.NoError(t, err)

				_, err = db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetAllBlogsHandler(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		setup      func(app *application, db *sql.DB) (*string, *int, *int, error)
		limit      int
		offset     int
		wantStatus int
		wantBody   envelope
	}{
		{
			name:       "Valid Request",
			setup:      createTestBlog,
			limit:      10,
			offset:     0,
			wantStatus: http.StatusOK,
		},
		{
			name: "No Authentication Token",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, userId, blogId, err := createTestBlog(app, db)
				return nil, userId, blogId, err
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "No Blogs",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				token, userId, err := createTestUser(app, db, &userservice.User{Username: "testuser", Email: "testuser@example.com", Activated: true})
				return token, userId, nil, err
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, _, _, err := tc.setup(app, db)
			assert.NoError(t, err)

			status, _, gotBody := ts.get(t, fmt.Sprintf("/api/v1/blogs?limit=%d&offset=%d", tc.limit, tc.offset), token, nil)
			assert.Equal(t, tc.wantStatus, status)

			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM blogs")
				assert.NoError(t, err)

				_, err = db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestGetBlogsByTitle(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	testCases := []struct {
		name       string
		setup      func(app *application, db *sql.DB) (*string, *int, *int, error)
		title      string
		limit      int
		offset     int
		wantStatus int
		wantBody   envelope
	}{
		{
			name:       "Valid Request",
			setup:      createTestBlog,
			title:      "Test Blog",
			wantStatus: http.StatusOK,
		},
		{
			name: "No Authentication Token",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				_, userId, blogId, err := createTestBlog(app, db)
				return nil, userId, blogId, err
			},
			title:      "Test Blog",
			wantStatus: http.StatusOK,
		},
		{
			name: "No Blogs",
			setup: func(app *application, db *sql.DB) (*string, *int, *int, error) {
				token, userId, err := createTestUser(app, db, &userservice.User{Username: "testuser", Email: "testuser@example.com", Activated: true})
				return token, userId, nil, err
			},
			title:      "Test Blog",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid Title",
			setup:      createTestBlog,
			title:      "Invalid Title",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, _, _, err := tc.setup(app, db)
			assert.NoError(t, err)

			title := url.QueryEscape(tc.title)
			url := fmt.Sprintf("/api/v1/blogs/search?q=%s&limit=%d&offset=%d", title, tc.limit, tc.offset)
			status, _, gotBody := ts.get(t, url, token, nil)
			assert.Equal(t, tc.wantStatus, status)

			if tc.wantBody != nil {
				assert.JSONEq(t, tc.wantBody.JSON(), gotBody.JSON())
			}

			t.Cleanup(func() {
				_, err := db.Exec("DELETE FROM blogs")
				assert.NoError(t, err)

				_, err = db.Exec("DELETE FROM users")
				assert.NoError(t, err)
			})
		})
	}
}

func TestConcurrentGetAndUpdate(t *testing.T) {
	app, db := newTestApplication(t)

	ts := newTestServer(t, app.routes())

	token, _, blogId, err := createTestBlog(app, db)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		t.Log("Starting update blog request")
		status, _, gotBody := ts.put(t, fmt.Sprintf("/api/v1/blogs/update/%d", *blogId), token, map[string]any{"title": "Updated Blog", "content": "This is an updated blog"})
		assert.Equal(t, http.StatusOK, status)
		assert.JSONEq(t, `{"message":"blog updated"}`, gotBody.JSON())
		t.Log("Finished update blog request")
	}()

	go func() {
		defer wg.Done()
		t.Log("Starting get blog request")
		status, _, gotBody := ts.get(t, fmt.Sprintf("/api/v1/blogs/view/%d", *blogId), token, nil)
		assert.Equal(t, http.StatusOK, status)
		t.Logf("gotBody: %+v\n", gotBody)
		t.Log("Finished get blog request")
	}()

	wg.Wait()

	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM blogs")
		assert.NoError(t, err)

		_, err = db.Exec("DELETE FROM users")
		assert.NoError(t, err)
	})
}
