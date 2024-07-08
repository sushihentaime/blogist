package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
)

var (
	ErrUnauthorized = fmt.Errorf("unauthorized access")
)

func NewService(m *UserModel, mb *common.MessageBroker, t *TokenModel) *UserService {
	return &UserService{
		m:  m,
		mb: mb,
		t:  t,
	}
}

// CreateUser creates a new user account and publish an user.created event.
func (s *UserService) CreateUser(ctx context.Context, u User) error {
	// Perform validation
	v := common.NewValidator()
	validateUsername(v, u.Username)
	validateEmail(v, u.Email)
	validatePassword(v, u.Password.Plain)
	if !v.Valid() {
		return fmt.Errorf("validation error: %v", v.Errors)
	}

	// Set the password hash
	err := u.Password.set(u.Password.Plain)
	if err != nil {
		return err
	}

	// Insert the user into the database
	err = s.m.insert(ctx, &u)
	if err != nil {
		return err
	}

	// create the token
	token, err := s.t.createToken(ctx, u.ID, ActivationTokenTime, TokenScopeActivate)
	if err != nil {
		return err
	}

	plainToken, err := json.Marshal(token.Plain)
	if err != nil {
		return err
	}

	// Publish the user created event
	err = s.mb.Publish(ctx, plainToken, common.UserCreatedKey, common.UserExchange)
	if err != nil {
		return err
	}

	return nil
}

// ActivateUser activates a user account using the token and deletes the token from the database.
func (s *UserService) ActivateUser(ctx context.Context, token string) error {
	// Validate the token
	v := common.NewValidator()
	validateToken(v, token)
	if !v.Valid() {
		return fmt.Errorf("validation error: %v", v.Errors)
	}

	// Hash the token
	hash := hashToken(token)

	user, err := s.t.getUser(ctx, TokenScopeActivate, hash)
	if err != nil {
		return err
	}

	tx, err := s.m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// activate the user account
	err = s.m.activate(tx, ctx, user.ID, user.Version)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// delete the token
	err = s.t.delete(tx, ctx, user.ID, TokenScopeActivate)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// LoginUser logs in a user and returns the access token and refresh token.
func (s *UserService) LoginUser(ctx context.Context, username, password string, ip string, userAgent string) (*AuthToken, error) {
	// Validate the username
	v := common.NewValidator()
	validateUsername(v, username)
	validatePassword(v, password)
	if !v.Valid() {
		return nil, fmt.Errorf("validation error: %v", v.Errors)
	}

	// Get the user from the database
	user, err := s.m.getByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Compare the password hash
	ok, err := user.Password.compare(password)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrUnauthorized
	} else {
		// rehash the password and update the user
		if err := user.Password.set(password); err != nil {
			return nil, err
		}

		if err := s.m.updatePassword(ctx, user.Password, user.ID, user.Version); err != nil {
			return nil, err
		}
	}

	// get the token from the database
	dbToken, err := s.t.getAuthToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if dbToken != nil {
		// Check if the token expiry is still valid
		if dbToken.AccessTokenExpiry.After(time.Now()) && dbToken.RefreshTokenExpiry.After(time.Now()) {
			return dbToken, nil
		} else {
			tx, err := s.m.db.BeginTx(ctx, nil)
			if err != nil {
				return nil, err
			}

			// delete the token
			err = s.t.deleteAuthToken(tx, ctx, user.ID)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}

			authToken, err := s.t.createAuthToken(tx, ctx, user.ID, ip, userAgent)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}

			if err := tx.Commit(); err != nil {
				return nil, err
			}

			return authToken, nil
		}
	}

	tx, err := s.m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	authToken, err := s.t.createAuthToken(tx, ctx, user.ID, ip, userAgent)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return authToken, nil
}
