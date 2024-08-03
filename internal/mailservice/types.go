package mailservice

import (
	"bytes"
	"context"
	"sync"

	"github.com/go-mail/mail/v2"

	"github.com/sushihentaime/blogist/internal/common"
)

type MailService struct {
	mb     common.MessageConsumer
	m      Mailer
	logger MailLogger
	ctx    context.Context
	cancel context.CancelFunc
}

type MailLogger interface {
	Error(msg string, args ...any)
	Info(msg string, args ...any)
}

type Mail struct {
	mu     sync.Mutex
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
