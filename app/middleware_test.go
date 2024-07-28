package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sushihentaime/blogist/internal/userservice"
	"golang.org/x/crypto/bcrypt"
)

func strptr(s string) *string {
	return &s
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

func TestEnableCORS(t *testing.T) {
	app := &application{
		config: &Config{
			TrustedOrigins: []string{"http://example.com"},
		},
	}

	// Create a test HTTP handler that will be wrapped by the enableCORS middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the enableCORS middleware
	middleware := app.enableCORS(handler)

	server := httptest.NewServer(middleware)
	defer server.Close()

	tests := []struct {
		name                       string
		origin                     string
		method                     string
		accessControlRequestMethod *string
		expectedStatus             int
	}{
		{
			name:           "Valid Origin and Method",
			origin:         "http://example.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:                       "Valid Origin and Preflight Request",
			origin:                     "http://example.com",
			method:                     http.MethodOptions,
			accessControlRequestMethod: strptr(http.MethodPut),
			expectedStatus:             http.StatusOK,
		},
		{
			name:           "Invalid Origin",
			origin:         "http://invalid.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			req.Header.Set("Origin", tt.origin)
			if tt.accessControlRequestMethod != nil {
				req.Header.Set("Access-Control-Request-Method", *tt.accessControlRequestMethod)
			}

			res := httptest.NewRecorder()

			middleware.ServeHTTP(res, req)

			assert.Equal(t, tt.expectedStatus, res.Code)

			for i := range app.config.TrustedOrigins {
				if tt.origin == app.config.TrustedOrigins[i] {
					assert.Equal(t, tt.origin, res.Header().Get("Access-Control-Allow-Origin"))
				} else {
					assert.Empty(t, res.Header().Get("Access-Control-Allow-Origin"))
				}
			}

			// Check the Access-Control-Allow-Methods header for preflight requests
			if tt.method == http.MethodOptions && tt.origin != "" {

				assert.Equal(t, "OPTIONS, PUT, PATCH, DELETE", res.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Content-Type, Authorization", res.Header().Get("Access-Control-Allow-Headers"))
			} else {
				assert.Empty(t, res.Header().Get("Access-Control-Allow-Methods"))
				assert.Empty(t, res.Header().Get("Access-Control-Allow-Headers"))
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	app := &application{
		config: &Config{
			RateLimitRPS:     2,
			RateLimitBurst:   4,
			RateLimitEnabled: true,
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := app.rateLimit(handler)

	server := httptest.NewServer(middleware)
	defer server.Close()

	tests := []struct {
		name           string
		requests       int
		expectedStatus int
	}{
		{
			name:           "Within Limit",
			requests:       4,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Over Limit",
			requests:       6,
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lastStatusCode int

			for i := 0; i < tt.requests; i++ {
				res, err := http.Get(server.URL)
				assert.NoError(t, err)

				lastStatusCode = res.StatusCode
			}

			assert.Equal(t, tt.expectedStatus, lastStatusCode)
		})

	}
}
