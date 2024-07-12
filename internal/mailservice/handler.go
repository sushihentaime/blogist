package mailservice

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sushihentaime/blogist/internal/common"
)

func NewMailService(mb common.MessageConsumer, host, username, password, sender string, port int, logger *slog.Logger) *MailService {
	return &MailService{
		mb: mb,
		m:  NewMailer(host, port, username, password, sender, NewTemplate()),
		l:  logger,
	}
}

// Send function that consumes messages from the message broker and sends emails to the user. Currently this function runs forever and I want a way to stop it.
// ! Add a logger.
// how to perform a graceful shutdown.
func (s *MailService) SendActivationEmail() {
	msgs, err := s.mb.Consume(common.UserCreatedKey, common.UserExchange, common.UserCreatedQueue)
	if err != nil {
		fmt.Printf("could not consume messages: %v\n", err)
		return
	}

	var forever chan struct{}

	go func() {
		for msg := range msgs {
			fmt.Printf("received message: %s\n", msg.Body)
			var data struct {
				Email string
				Token string
			}

			err := json.Unmarshal(msg.Body, &data)
			if err != nil {
				s.l.Error(fmt.Sprintf("could not unmarshal message: %v", err))
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
				s.l.Error(fmt.Sprintf("could not send activation email: %v", err))
				continue
			}

			s.l.Info(fmt.Sprintf("activation email sent to %s", data.Email))
		}
	}()

	<-forever
}
