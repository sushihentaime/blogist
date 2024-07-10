package mailservice

import (
	"bytes"
	"testing"

	"github.com/go-mail/mail/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSendEmail(t *testing.T) {
	mockParser := new(MockTemplate)
	mockDialer := new(MockDialer)

	recipient := "test@example.com"
	mailer := Mail{
		dialer: mockDialer,
		parser: mockParser,
		sender: "sender@example.com",
	}

	subject := bytes.NewBufferString("Test Subject")
	plainBody := bytes.NewBufferString("Test Plain Body")
	htmlBody := bytes.NewBufferString("Test HTML Body")
	mockParser.On("ParseTemplate", "template.html", mock.Anything).Return(subject, plainBody, htmlBody, nil)

	msg := mail.NewMessage()
	msg.SetHeader("From", mailer.sender)
	msg.SetHeader("To", recipient)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	mockDialer.On("DialAndSend", mock.AnythingOfType("[]*mail.Message")).Return(nil)

	err := mailer.send("test@example.com", msg, "template.html")
	assert.NoError(t, err)

	mockParser.AssertExpectations(t)
	mockDialer.AssertExpectations(t)
}
