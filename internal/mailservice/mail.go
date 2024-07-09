package mailservice

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/go-mail/mail/v2"
	"github.com/sushihentaime/blogist/internal/common"
)

//go:embed templates/*
var templateFS embed.FS

func NewMailer(host string, port int, username, password, sender string) *Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return &Mailer{
		dialer: dialer,
		sender: sender,
	}
}

func NewMailService(mb common.MessageConsumer, dialer *Mailer) *MailService {
	return &MailService{
		mb: mb,
		m:  dialer,
	}
}

// Send function that consumes messages from the message broker and sends emails to the user.
func (s *MailService) SendActivationEmail() {
	msgs, err := s.mb.Consume(common.UserCreatedKey, common.UserExchange, common.UserCreatedQueue)
	if err != nil {
		fmt.Printf("could not consume messages: %v\n", err)
		return
	}

	var forever chan struct{}

	go func() {
		for msg := range msgs {
			var data struct {
				Email string
				Token string
			}

			err := json.Unmarshal(msg.Body, &data)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}

			link := fmt.Sprintf("http://localhost:8080/activate?token=%s", data.Token)

			activationLink := struct {
				ActivationLink string
				LinkName       string
			}{
				ActivationLink: link,
				LinkName:       "Activate Account",
			}

			err = s.m.send(data.Email, activationLink, "activation_email.html")
			if err != nil {
				fmt.Printf("could not send activation email: %v\n", err)
			}
		}
	}()

	<-forever
}

func (m *Mailer) send(recipient string, data any, templateFile string) error {
	t, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = t.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = t.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = t.ExecuteTemplate(htmlBody, "htmlBody", data)
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
