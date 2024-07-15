package userservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"time"
)

func hashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func newToken(userID int, ttl time.Duration, scope tokenScope) (*Token, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token := &Token{
		Plain:  base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes),
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	token.Hash = hashToken(token.Plain)

	return token, nil
}

func (m *DBModel) insertToken(ctx context.Context, token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope_id)
		VALUES ($1, $2, $3, (SELECT id FROM token_scopes WHERE name = $4))`

	_, err := m.db.ExecContext(ctx, query, token.Hash, token.UserID, token.Expiry, string(token.Scope))
	return err
}

func (m *DBModel) createToken(ctx context.Context, userID int, ttl time.Duration, scope tokenScope) (*Token, error) {
	token, err := newToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.insertToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (m *DBModel) getUser(ctx context.Context, tokenScope tokenScope, token []byte) (*User, error) {
	var user User

	query := `
		SELECT u.id, u.username, u.email, u.activated, u.version
		FROM users u
		INNER JOIN tokens t ON u.id = t.user_id
		INNER JOIN token_scopes s ON t.scope_id = s.id
		WHERE t.hash = $1 AND s.name = $2 AND t.expiry > $3`

	err := m.db.QueryRowContext(ctx, query, token, tokenScope, time.Now()).Scan(&user.ID, &user.Username, &user.Email, &user.Activated, &user.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m *DBModel) deleteToken(tx *sql.Tx, ctx context.Context, userID int, scope tokenScope) error {
	query := `
		DELETE FROM tokens
		WHERE user_id = $1 AND scope_id = (SELECT id FROM token_scopes WHERE name = $2)`

	res, err := tx.ExecContext(ctx, query, userID, string(scope))
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		switch {
		case rows == 0:
			return ErrNotFound
		default:
			return errors.New("too many rows affected")
		}
	}

	return nil
}

func newAuthToken(userID int, accessTokenTTL time.Duration, refreshTokenTTL time.Duration) (*AuthToken, error) {
	accessToken := make([]byte, 16)
	_, err := rand.Read(accessToken)
	if err != nil {
		return nil, err
	}

	refreshToken := make([]byte, 16)
	_, err = rand.Read(refreshToken)
	if err != nil {
		return nil, err
	}

	token := &AuthToken{
		AccessTokenPlain:   base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(accessToken),
		RefreshTokenPlain:  base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(refreshToken),
		UserID:             userID,
		AccessTokenExpiry:  time.Now().Add(accessTokenTTL),
		RefreshTokenExpiry: time.Now().Add(refreshTokenTTL),
	}

	token.AccessTokenHash = hashToken(token.AccessTokenPlain)
	token.RefreshTokenHash = hashToken(token.RefreshTokenPlain)

	return token, nil
}

func (m *DBModel) createAuthToken(tx *sql.Tx, ctx context.Context, userID int) (*AuthToken, error) {
	authToken, err := newAuthToken(userID, AccessTokenTime, RefreshTokenTime)
	if err != nil {
		return nil, err
	}

	err = m.insertAuthToken(tx, ctx, authToken)
	if err != nil {
		return nil, err
	}

	return authToken, nil
}

func (m *DBModel) insertAuthToken(tx *sql.Tx, ctx context.Context, authToken *AuthToken) error {
	query := `
		INSERT INTO auth_tokens (access_token, refresh_token, user_id, access_token_expiry, refresh_token_expiry)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := tx.ExecContext(ctx, query, authToken.AccessTokenHash, authToken.RefreshTokenHash, authToken.UserID, authToken.AccessTokenExpiry, authToken.RefreshTokenExpiry)
	return err
}

func (m *DBModel) getAuthToken(ctx context.Context, userid int) (*AuthToken, error) {
	var authToken AuthToken

	query := `
		SELECT access_token, refresh_token, user_id, access_token_expiry, refresh_token_expiry
		FROM auth_tokens
		WHERE user_id = $1`

	err := m.db.QueryRowContext(ctx, query, userid).Scan(&authToken.AccessTokenHash, &authToken.RefreshTokenHash, &authToken.UserID, &authToken.AccessTokenExpiry, &authToken.RefreshTokenExpiry)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, nil
		default:
			return nil, err
		}
	}

	return &authToken, nil
}

func (m *DBModel) deleteAuthToken(tx *sql.Tx, ctx context.Context, userID int) error {
	query := `
		DELETE FROM auth_tokens
		WHERE user_id = $1`

	res, err := tx.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		switch {
		case rows == 0:
			return ErrNotFound
		default:
			return errors.New("too many rows affected")
		}
	}

	return nil
}
