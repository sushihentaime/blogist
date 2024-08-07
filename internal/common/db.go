package common

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

var (
	ErrRecordNotFound = fmt.Errorf("record not found")
)

func NewDB(host, user, password, name string, maxOpenConns, maxIdleConns int, maxIdleTime time.Duration) (*sql.DB, error) {
	URI := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, host, name)
	return connectDB(URI, maxOpenConns, maxIdleConns, maxIdleTime)
}

// connectDB connects to the database and returns the connection
func connectDB(URI string, maxOpenConns int, maxIdleConns int, maxIdleTime time.Duration) (*sql.DB, error) {
	db, err := sql.Open("postgres", URI)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database connection: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxIdleTime(maxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		CloseDB(db)
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}

	return db, nil
}

// CloseDB closes the database connection
func CloseDB(db *sql.DB) error {
	return db.Close()
}
