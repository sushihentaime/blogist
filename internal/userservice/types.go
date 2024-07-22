package userservice

import (
	"database/sql"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
)

type tokenScope string

type Permission string
type Permissions []Permission

const (
	TokenScopeActivate tokenScope = "token:activate"

	ActivationTokenTime time.Duration = 3 * 24 * time.Hour
	AccessTokenTime     time.Duration = 7 * 24 * time.Hour
	RefreshTokenTime    time.Duration = 30 * 24 * time.Hour

	PermissionWriteBlog Permission = "blog:write"
)

var (
	AnonymousUser = User{}
)

type UserService struct {
	m  *DBModel
	mb common.MessageProducer
	c  *common.Cache
}

type DBModel struct {
	db *sql.DB
}

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  Password  `json:"-"`
	Activated bool      `json:"activated"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`

	Permissions Permissions `json:"permissions"`
}

type Password struct {
	Plain string `json:"-"`
	hash  []byte `json:"-"`
}

type Token struct {
	Plain  string     `json:"token"`
	Hash   []byte     `json:"-"`
	UserID int        `json:"-"`
	Expiry time.Time  `json:"expiry"`
	Scope  tokenScope `json:"-"`
}

// Authentication Token
type AuthToken struct {
	AccessTokenPlain   string    `json:"access_token"`
	AccessTokenHash    []byte    `json:"-"`
	RefreshTokenPlain  string    `json:"refresh_token"`
	RefreshTokenHash   []byte    `json:"-"`
	UserID             int       `json:"user_id"`
	AccessTokenExpiry  time.Time `json:"access_token_expiry"`
	RefreshTokenExpiry time.Time `json:"refresh_token_expiry"`
}
