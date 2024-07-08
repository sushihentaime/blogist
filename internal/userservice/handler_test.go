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
	db := common.TestDB(t)
	m := NewUserModel(db)
	tokenModel := NewTokenModel(db)
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
		_, err := db.Exec("DELETE FROM tokens")
		if err != nil {
			return err
		}

		_, err = db.Exec("DELETE FROM users")
		if err != nil {
			return err
		}

		return nil
	}

	return NewService(m, mb, tokenModel), db, cleanup, nil
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
			expectedErr: fmt.Errorf("validation error: map[username:must be provided]"),
		},
		{
			name: "empty email",
			payload: User{
				Username: testUser().Username,
				Password: testUser().Password,
			},
			expectedErr: fmt.Errorf("validation error: map[email:must be provided]"),
		},
		{
			name: "empty password",
			payload: User{
				Username: testUser().Username,
				Email:    testUser().Email,
			},
			expectedErr: fmt.Errorf("validation error: map[password:must be provided]"),
		},
		{
			name:        "empty payload",
			payload:     User{},
			expectedErr: fmt.Errorf("validation error: map[email:must be provided password:must be provided username:must be provided]"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			fmt.Printf("username: %s\n", tc.payload.Username)

			err := s.CreateUser(ctx, tc.payload)
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

		err = s.m.insert(ctx, &u)
		if err != nil {
			return nil, err
		}

		token, err := s.t.createToken(ctx, u.ID, ActivationTokenTime, TokenScopeActivate)
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
			expectedErr: fmt.Errorf("validation error: map[token:invalid token]"),
		},
		{
			name: "empty token",
			token: func(ctx context.Context, s *UserService, u User) (*string, error) {
				return strptr(""), nil
			},
			expectedErr: fmt.Errorf("validation error: map[token:must be provided]"),
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
			} else {
				err = db.QueryRow("SELECT COUNT(*) FROM tokens").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count)

				err = db.QueryRow("SELECT COUNT(*) FROM users WHERE activated = true").Scan(&count)
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
