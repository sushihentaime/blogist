package mailservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendActivationEmail(t *testing.T) {
	mockMC := new(MockMessageConsumer)
	mockMailer := new(MockMailer)

	s := &MailService{
		mb: mockMC,
		m:  mockMailer,
	}

	go s.SendActivationEmail()

	time.Sleep(100 * time.Millisecond)

	// verify that the mockMailer.send method was called
	assert.True(t, mockMailer.Called, "expected mockMailer.send to be called")
	// verify that the email was sent to the correct recipient
	assert.Equal(t, "test@example.com", mockMailer.Email, "expected email to be sent to the recipient")
}
