package userservice

import (
	"database/sql"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  Password  `json:"-"`
	Activated bool      `json:"activated"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

type Password struct {
	Plain string `json:"-"`
	hash  []byte `json:"-"`
}

type UserModel struct {
	db *sql.DB
}

type UserService struct {
	m  *UserModel
	t  *TokenModel
	mb common.MessageProducer
}

type tokenScope string

const (
	TokenScopeActivate tokenScope = "token:activate"

	ActivationTokenTime time.Duration = 3 * 24 * time.Hour
	AccessTokenTime     time.Duration = 7 * 24 * time.Hour
	RefreshTokenTime    time.Duration = 30 * 24 * time.Hour
)

type Token struct {
	Plain  string     `json:"token"`
	Hash   []byte     `json:"-"`
	UserID int        `json:"-"`
	Expiry time.Time  `json:"expiry"`
	Scope  tokenScope `json:"-"`
}

type TokenModel struct {
	db *sql.DB
}

type AuthToken struct {
	AccessTokenPlain   string    `json:"access_token"`
	AccessTokenHash    []byte    `json:"-"`
	RefreshTokenPlain  string    `json:"refresh_token"`
	RefreshTokenHash   []byte    `json:"-"`
	UserID             int       `json:"user_id"`
	AccessTokenExpiry  time.Time `json:"access_token_expiry"`
	RefreshTokenExpiry time.Time `json:"refresh_token_expiry"`
	IPAddress          string    `json:"ip_address"`
	UserAgent          string    `json:"user_agent"`
}

type Permission struct{}
