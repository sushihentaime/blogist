package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config.env")
	if err != nil {
		t.Fatalf("Failed to create temporary config file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test configuration to the temporary file
	configData := []byte(`
PORT=8080
ENVIRONMENT=development
VERSION=1.0.0
TRUSTED_ORIGINS="http://localhost:3000,http://localhost:3001"
POSTGRES_HOST=localhost
POSTGRES_USER=testuser
POSTGRES_PASSWORD=testpassword
POSTGRES_DB=testdb
MAIL_HOST=smtp.example.com
MAIL_PORT=587
MAIL_USER=testuser@example.com
MAIL_PASSWORD=testpassword
MAIL_SENDER=sender@example.com
RABBITMQ_HOST=rabbitmq.example.com
RABBITMQ_USER=testuser
RABBITMQ_PASSWORD=testpassword
`)
	if _, err := tempFile.Write(configData); err != nil {
		t.Fatalf("Failed to write test configuration to temporary file: %v", err)
	}

	// Load the config from the temporary file
	config, err := loadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify the loaded config values
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, []string{"http://localhost:3000", "http://localhost:3001"}, config.TrustedOrigins)
	assert.Equal(t, "localhost", config.DBHost)
	assert.Equal(t, "testuser", config.DBUser)
	assert.Equal(t, "testpassword", config.DBPassword)
	assert.Equal(t, "testdb", config.DBName)
	assert.Equal(t, "smtp.example.com", config.MailHost)
	assert.Equal(t, 587, config.MailPort)
	assert.Equal(t, "testuser@example.com", config.MailUser)
	assert.Equal(t, "testpassword", config.MailPassword)
	assert.Equal(t, "sender@example.com", config.MailSender)
	assert.Equal(t, "rabbitmq.example.com", config.MQHost)
	assert.Equal(t, "testuser", config.MQUser)
	assert.Equal(t, "testpassword", config.MQPassword)

}
