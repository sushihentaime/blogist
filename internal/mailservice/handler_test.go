package mailservice

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendActivationEmail(t *testing.T) {
	mockMC := new(MockMessageConsumer)
	mockMailer := new(MockMailer)
	mockLogger := new(MockLogger)

	expectedArgs := []interface{}{slog.Attr{Key: "email", Value: slog.StringValue("test@example.com")}}
	mockLogger.On("Info", "activation email sent", expectedArgs).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())

	s := &MailService{
		mb:     mockMC,
		m:      mockMailer,
		logger: mockLogger,
		ctx:    ctx,
		cancel: cancel,
	}

	go s.SendActivationEmail()

	time.Sleep(1 * time.Second)

	if mockMailer.IsCalled() {
		recipientEmail := mockMailer.GetEmail()
		assert.Equal(t, "test@example.com", recipientEmail, "expected email to be sent to the recipient")
	}

	// verify that the message consumer consume method was called
	mockMC.AssertExpectations(t)

	// verify that the logger.Info method was called
	mockLogger.AssertExpectations(t)

	t.Cleanup(func() {
		s.Close()
	})
}
