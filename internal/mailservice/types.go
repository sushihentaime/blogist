package mailservice

import (
	"bytes"
	"log/slog"

	"github.com/go-mail/mail/v2"

	"github.com/sushihentaime/blogist/internal/common"
)

type MailService struct {
	mb common.MessageConsumer
	m  Mailer
	l  *slog.Logger
}

type Mail struct {
	dialer Dialer
	parser TemplateParser
	sender string
}

type Mailer interface {
	send(recipient string, data any, templateFile string) error
}

type Template struct{}

type Dialer interface {
	DialAndSend(m ...*mail.Message) error
}

type TemplateParser interface {
	ParseTemplate(name string, data any) (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer, error)
}
