package mailservice

import (
	"time"

	"github.com/go-mail/mail/v2"
)

// NewMailer creates a new mailer with the given host, port, username, password, sender, and template.
func NewMailer(host string, port int, username, password, sender string, tp *Template) *Mail {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return &Mail{
		dialer: dialer,
		sender: sender,
		parser: tp,
	}
}

func (m *Mail) send(recipient string, data any, templateFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	subject, plainBody, htmlBody, err := m.parser.ParseTemplate(templateFile, data)
	if err != nil {
		return err
	}

	msg := mail.NewMessage()
	msg.SetHeader("From", m.sender)
	msg.SetHeader("To", recipient)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	err = m.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}

	return nil
}
