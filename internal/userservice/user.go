package userservice

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrDuplicateUsername = errors.New("duplicate username")
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrNotFound          = errors.New("user not found")
)

func newUserModel(db *sql.DB) *DBModel {
	return &DBModel{db: db}
}

func (m *DBModel) insertUser(ctx context.Context, u *User) error {
	query := `
		INSERT INTO users (username, email, password)
		VALUES ($1, $2, $3)
		RETURNING id`

	args := []any{
		u.Username,
		u.Email,
		u.Password.hash,
	}

	err := m.db.QueryRowContext(ctx, query, args...).Scan(&u.ID)
	if err != nil {
		switch {
		case err.Error() == "pq: duplicate key value violates unique constraint \"users_username_key\"":
			return ErrDuplicateUsername
		case err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"":
			return ErrDuplicateEmail
		default:
			return err
		}
	}
	return nil
}

func (m *DBModel) getUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, password, version
		FROM users
		WHERE username = $1`

	var u User

	err := m.db.QueryRowContext(ctx, query, username).Scan(&u.ID, &u.Username, &u.Email, &u.Password.hash, &u.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return &u, nil
}

func (m *DBModel) activateUserAccount(tx *sql.Tx, ctx context.Context, id int, version int) error {
	query := `
		UPDATE users
		SET activated = true
		WHERE id = $1 AND version = $2`

	res, err := tx.ExecContext(ctx, query, id, version)
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

func (m *DBModel) updateUserPassword(ctx context.Context, pwd Password, id int, version int) error {
	query := `
		UPDATE users
		SET password = $1
		WHERE id = $2 AND version = $3`

	_, err := m.db.ExecContext(ctx, query, pwd.hash, id, version)
	if err != nil {
		return err
	}

	return nil
}

func (m *DBModel) getToken(ctx context.Context, token []byte) (*User, error) {
	var u User

	query := `
		SELECT u.id, u.username, u.email, u.activated, u.version, p.permission
		FROM users u
		INNER JOIN auth_tokens t ON u.id = t.user_id
		INNER JOIN user_permissions p on u.id = p.user_id
		WHERE t.access_token = $1 AND t.access_token_expiry > $2`

	rows, err := m.db.QueryContext(ctx, query, token, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Permission
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Activated, &u.Version, &p)
		if err != nil {
			return nil, err
		}

		u.Permissions = append(u.Permissions, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if u.ID == 0 {
		return nil, ErrNotFound
	}

	return &u, nil
}
