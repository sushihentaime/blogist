package common

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRabbitMQ(t *testing.T) string {
	ctx := context.Background()

	container, err := rabbitmq.Run(ctx, "rabbitmq:3.12.11-management-alpine", rabbitmq.WithAdminUsername("guest"), rabbitmq.WithAdminPassword("guest"))
	if err != nil {
		t.Fatalf("could not start rabbitmq container: %v", err)
	}

	connURL, err := container.AmqpURL(ctx)
	if err != nil {
		t.Fatalf("could not get rabbitmq connection URL: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("could not terminate container: %v", err)
		}
	})

	return connURL
}

// file parameter changes according to the caller filelocation relative to the migrations file and it should be in the format of "file://../../migrations"
func dbMigrate(file, dsn string) (*migrate.Migrate, error) {
	m, err := migrate.New(file, dsn)
	if err != nil {
		return nil, err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, err
	}

	return m, nil
}

func TestDB(filepath string, t *testing.T) *sql.DB {
	ctx := context.Background()

	c, err := postgres.Run(ctx,
		"docker.io/postgres:14.11-bookworm",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)))
	if err != nil {
		t.Fatalf("could not start postgres container: %v", err)
	}

	connURL, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}

	m, err := dbMigrate(filepath, connURL)
	if err != nil {
		t.Fatalf("could not run migrations: %v", err)
	}

	db, err := sql.Open("postgres", connURL)
	if err != nil {
		t.Fatalf("could not open database: %v", err)
	}

	t.Cleanup(func() {
		m.Drop()
		c.Terminate(ctx)
	})

	return db
}
