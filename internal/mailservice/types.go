package mailservice

import (
	"github.com/go-mail/mail/v2"
	"github.com/sushihentaime/blogist/internal/common"
)

type MailService struct {
	mb common.MessageConsumer
	m  *Mailer
}

type Mailer struct {
	dialer *mail.Dialer
	sender string
}
