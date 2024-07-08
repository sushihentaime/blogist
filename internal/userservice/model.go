package userservice

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrDuplicateUsername = errors.New("duplicate username")
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrNotFound          = errors.New("user not found")
)

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{db: db}
}

func (m *UserModel) insert(ctx context.Context, u *User) error {
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

// func (m *UserModel) getByID(ctx context.Context, id int) (*User, error) {
// 	query := `
// 		SELECT id, username, email, activated, created_at, updated_at, version
// 		FROM users
// 		WHERE id = $1`

// 	var u User
// 	err := m.db.QueryRowContext(ctx, query, id).Scan(
// 		&u.ID,
// 		&u.Username,
// 		&u.Email,
// 		&u.Activated,
// 		&u.CreatedAt,
// 		&u.UpdatedAt,
// 		&u.Version,
// 	)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, sql.ErrNoRows):
// 			return nil, ErrNotFound
// 		default:
// 			return nil, err
// 		}
// 	}

// 	return &u, nil
// }

func (m *UserModel) getByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, password, version
		FROM users
		WHERE username = $1`

	var u User

	err := m.db.QueryRowContext(ctx, query, username).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Version)
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

// func (m *UserModel) delete(ctx context.Context, id int) error {
// 	query := `
// 		DELETE FROM users
// 		WHERE id = $1`

// 	res, err := m.db.ExecContext(ctx, query, id)
// 	if err != nil {
// 		return err
// 	}

// 	rows, err := res.RowsAffected()
// 	if err != nil {
// 		return err
// 	}
// 	if rows != 1 {
// 		switch {
// 		case rows == 0:
// 			return ErrNotFound
// 		default:
// 			return errors.New("too many rows affected")
// 		}
// 	}

// 	return nil
// }

func (m *UserModel) activate(tx *sql.Tx, ctx context.Context, id int, version int) error {
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

func (m *UserModel) updatePassword(ctx context.Context, pwd Password, id int, version int) error {
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
