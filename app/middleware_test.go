package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/userservice"
	"golang.org/x/crypto/bcrypt"
)

func strptr(s string) *string {
	return &s
}

func newTestApplication(t *testing.T) (*application, *sql.DB) {
	db := common.TestDB("file://../migrations", t)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	return &application{
		logger:      logger,
		userService: userservice.NewUserService(db, nil),
	}, db
}

func TestRecoverPanic(t *testing.T) {
	// Create a new instance of your application
	app, _ := newTestApplication(t)

	// Create a test HTTP handler that will panic
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	// Wrap the handler with the recoverPanic middleware
	middleware := app.recoverPanic(handler)

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Create a test response recorder
	res := httptest.NewRecorder()

	// Call the middleware with the test request and response recorder
	middleware.ServeHTTP(res, req)

	assert.Equal(t, res.Code, http.StatusInternalServerError)
}

func TestAuthenticate(t *testing.T) {
	app, db := newTestApplication(t)

	setup := func(db *sql.DB) (*string, error) {
		// should I mock this?
		randomBytes := make([]byte, 16)
		_, err := rand.Read(randomBytes)
		if err != nil {
			return nil, err
		}

		user := userservice.User{
			Username: "testuser",
			Email:    "testuser@example.com",
			Password: userservice.Password{Plain: "Test_1234!"},
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password.Plain), 12)
		if err != nil {
			return nil, err
		}

		var userId int

		// Create a new user
		err = db.QueryRow("INSERT INTO users (username, email, password) VALUES ($1, $2, $3) RETURNING id", user.Username, user.Email, hash).Scan(&userId)
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// add the user permissions
		_, err = db.ExecContext(ctx, "INSERT INTO user_permissions (user_id, permission) VALUES ($1, $2)", userId, userservice.PermissionWriteBlog)
		if err != nil {
			return nil, err
		}

		// login the user
		token, err := app.userService.LoginUser(ctx, user.Username, user.Password.Plain)
		if err != nil {
			return nil, err
		}

		return &token.AccessTokenPlain, nil
	}

	tests := []struct {
		name           string
		authHeader     func(db *sql.DB) (*string, error)
		expectedStatus int
	}{
		{
			name:           "No Authentication Header",
			authHeader:     func(db *sql.DB) (*string, error) { return nil, nil },
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Authentication Header",
			authHeader:     func(db *sql.DB) (*string, error) { return strptr("invalid-token"), nil },
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Valid Authentication Header",
			authHeader:     setup,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test HTTP handler that will be wrapped by the authenticate middleware
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap the handler with the authenticate middleware
			middleware := app.authenticate(handler)

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != nil {
				token, err := tt.authHeader(db)
				assert.NoError(t, err)

				if token != nil {
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
				}
			}

			// Create a test response recorder
			res := httptest.NewRecorder()

			// Call the middleware with the test request and response recorder
			middleware.ServeHTTP(res, req)

			// Check the response status code
			assert.Equal(t, res.Code, tt.expectedStatus)
		})
	}
}
