package userservice

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sushihentaime/blogist/internal/common"
)

func strptr(s string) *string {
	return &s
}

func testUser() User {
	return User{
		Username: "testuser",
		Email:    "testuser@example.com",
		Password: Password{
			Plain: "TestPassword123!",
		},
	}
}

func setupTestEnvironment(t *testing.T) (*UserService, *sql.DB, func() error, error) {
	db := common.TestDB("file://../../migrations", t)
	connURL := common.TestRabbitMQ(t)
	mb, err := common.NewMessageBroker(connURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not create message broker: %w", err)
	}

	err = common.SetupUserExchange(mb)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not setup user exchange: %w", err)
	}

	cleanup := func() error {
		_, err := db.Exec("DELETE FROM user_permissions")
		if err != nil {
			return err
		}

		_, err = db.Exec("DELETE FROM auth_tokens")
		if err != nil {
			return err
		}

		_, err = db.Exec("DELETE FROM tokens")
		if err != nil {
			return err
		}

		_, err = db.Exec("DELETE FROM users")
		if err != nil {
			return err
		}

		return nil
	}

	return NewUserService(db, mb), db, cleanup, nil
}

func TestSignUpUser(t *testing.T) {
	s, db, cleanup, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	testCases := []struct {
		name        string
		payload     User
		expectedErr error
	}{
		{
			name:        "valid user",
			payload:     testUser(),
			expectedErr: nil,
		},
		{
			name: "empty username",
			payload: User{
				Email:    testUser().Email,
				Password: testUser().Password,
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"username": "must be provided"}},
		},
		{
			name: "empty email",
			payload: User{
				Username: testUser().Username,
				Password: testUser().Password,
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"email": "must be provided"}},
		},
		{
			name: "empty password",
			payload: User{
				Username: testUser().Username,
				Email:    testUser().Email,
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"password": "must be provided"}},
		},
		{
			name:        "empty payload",
			payload:     User{},
			expectedErr: common.ValidationError{Errors: map[string]string{"username": "must be provided", "email": "must be provided", "password": "must be provided"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			fmt.Printf("username: %s\n", tc.payload.Username)

			_, err := s.CreateUser(ctx, tc.payload.Username, tc.payload.Email, tc.payload.Password.Plain)
			assert.Equal(t, tc.expectedErr, err)

			var count int

			if err == nil {
				err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)

				err = db.QueryRow("SELECT COUNT(*) FROM tokens").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)

			} else {
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
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

func TestActivateUser(t *testing.T) {
	s, db, cleanup, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	setup := func(ctx context.Context, s *UserService, u User) (*string, error) {
		err := u.Password.set(u.Password.Plain)
		if err != nil {
			return nil, err
		}

		err = s.m.insertUser(&u)
		if err != nil {
			return nil, err
		}

		token, err := s.m.createToken(u.ID, ActivationTokenTime, TokenScopeActivate)
		if err != nil {
			return nil, err
		}

		return &token.Plain, nil
	}

	testCases := []struct {
		name        string
		token       func(context.Context, *UserService, User) (*string, error)
		expectedErr error
	}{
		{
			name:        "valid token",
			token:       setup,
			expectedErr: nil,
		},
		{
			name: "invalid token",
			token: func(ctx context.Context, s *UserService, u User) (*string, error) {
				return strptr("invalid token"), nil
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"token": "invalid token"}},
		},
		{
			name: "empty token",
			token: func(ctx context.Context, s *UserService, u User) (*string, error) {
				return strptr(""), nil
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"token": "must be provided"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var plainToken string

			if tc.token != nil {
				token, err := tc.token(ctx, s, testUser())
				assert.NoError(t, err)
				assert.NotNil(t, token)
				plainToken = *token
			}

			err := s.ActivateUser(ctx, plainToken)
			assert.Equal(t, tc.expectedErr, err)

			var count int

			if err == nil {
				err = db.QueryRow("SELECT COUNT(*) FROM tokens").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)

				err = db.QueryRow("SELECT COUNT(*) FROM users WHERE activated = true").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)

				err = db.QueryRow("SELECT COUNT(*) FROM user_permissions").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)

			} else {
				err = db.QueryRow("SELECT COUNT(*) FROM tokens").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)

				err = db.QueryRow("SELECT COUNT(*) FROM users WHERE activated = true").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)

				err = db.QueryRow("SELECT COUNT(*) FROM user_permissions").Scan(&count)
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

func TestLoginUser(t *testing.T) {
	s, db, cleanup, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	setup := func(ctx context.Context, s *UserService, u User) error {
		err := u.Password.set(u.Password.Plain)
		if err != nil {
			return err
		}

		err = s.m.insertUser(&u)
		if err != nil {
			return err
		}

		return nil
	}

	testCases := []struct {
		name        string
		setup       func(context.Context, *UserService, User) error
		user        User
		expectedErr error
	}{
		{
			name:        "valid user",
			setup:       setup,
			user:        testUser(),
			expectedErr: nil,
		},
		{
			name:  "invalid user",
			setup: setup,
			user: User{
				Username: "invaliduser",
				Password: Password{
					Plain: "InvalidPassword123!",
				},
			},
			expectedErr: ErrAuthenticationFailure,
		},
		{
			name: "second-time login",
			setup: func(ctx context.Context, s *UserService, u User) error {
				err := setup(ctx, s, u)
				if err != nil {
					return err
				}

				_, err = s.LoginUser(ctx, u.Username, u.Password.Plain)
				if err != nil {
					return err
				}

				return nil
			},
			user:        testUser(),
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			if tc.setup != nil {
				err := tc.setup(ctx, s, testUser())
				assert.NoError(t, err)
			}

			_, err := s.LoginUser(ctx, tc.user.Username, tc.user.Password.Plain)
			assert.Equal(t, tc.expectedErr, err)

			var count int

			if err == nil {
				err = db.QueryRow("SELECT COUNT(*) FROM auth_tokens").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			} else {
				err = db.QueryRow("SELECT COUNT(*) FROM auth_tokens").Scan(&count)
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

func TestGetUserByAccessToken(t *testing.T) {
	s, db, cleanup, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	setup := func(ctx context.Context, s *UserService, u User) (*string, error) {
		err := u.Password.set(u.Password.Plain)
		if err != nil {
			return nil, err
		}

		err = s.m.insertUser(&u)
		if err != nil {
			return nil, err
		}

		tx, err := db.Begin()
		if err != nil {
			return nil, err
		}

		token, err := s.m.createAuthToken(tx, u.ID)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		// add permissions
		err = s.m.addUserPermission(tx, ctx, u.ID, PermissionWriteBlog)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}

		return &token.AccessTokenPlain, nil
	}

	testCases := []struct {
		name        string
		token       func(context.Context, *UserService, User) (*string, error)
		expectedErr error
	}{
		{
			name:        "valid token",
			token:       setup,
			expectedErr: nil,
		},
		{
			name: "invalid token",
			token: func(ctx context.Context, s *UserService, u User) (*string, error) {
				return strptr("invalid token"), nil
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"token": "invalid token"}},
		},
		{
			name: "empty token",
			token: func(ctx context.Context, s *UserService, u User) (*string, error) {
				return strptr(""), nil
			},
			expectedErr: common.ValidationError{Errors: map[string]string{"token": "must be provided"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var plainToken string

			if tc.token != nil {
				token, err := tc.token(ctx, s, testUser())
				assert.NoError(t, err)
				assert.NotNil(t, token)
				plainToken = *token
			}

			_, err := s.GetUserByAccessToken(ctx, plainToken)
			assert.Equal(t, tc.expectedErr, err)

			t.Cleanup(func() {
				err := cleanup()
				assert.NoError(t, err)
			})
		})
	}
}

func TestLogoutUser(t *testing.T) {
	s, db, cleanup, err := setupTestEnvironment(t)
	assert.NoError(t, err)

	setup := func(ctx context.Context, s *UserService, u User) error {
		err := u.Password.set(u.Password.Plain)
		if err != nil {
			return err
		}

		err = s.m.insertUser(&u)
		if err != nil {
			return err
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		_, err = s.m.createAuthToken(tx, u.ID)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}

	testCases := []struct {
		name        string
		setup       func(context.Context, *UserService, User) error
		expectedErr error
	}{
		{
			name:        "valid user",
			setup:       setup,
			expectedErr: nil,
		},
		{
			name:        "invalid user",
			setup:       setup,
			expectedErr: common.ErrRecordNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if tc.setup != nil {
				err := tc.setup(ctx, s, testUser())
				assert.NoError(t, err)
			}

			err := s.LogoutUser(ctx, 1)
			assert.Equal(t, tc.expectedErr, err)

			var count int

			if err == nil {
				err = db.QueryRow("SELECT COUNT(*) FROM auth_tokens").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)
			} else {
				err = db.QueryRow("SELECT COUNT(*) FROM auth_tokens").Scan(&count)
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
