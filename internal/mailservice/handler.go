package mailservice

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sushihentaime/blogist/internal/common"
)

func NewMailService(mb common.MessageConsumer, host, username, password, sender string, port int, logger *slog.Logger) *MailService {
	return &MailService{
		mb:     mb,
		m:      NewMailer(host, port, username, password, sender, NewTemplate()),
		logger: logger,
	}
}

// Send function that consumes messages from the message broker and sends emails to the user. Currently this function runs forever and I want a way to stop it.
// ! Add a logger.Debug call to log the message received from the message broker.
func (s *MailService) SendActivationEmail() {
	msgs, err := s.mb.Consume(common.UserCreatedKey, common.UserExchange, common.UserCreatedQueue)
	if err != nil {
		s.logger.Error("could not consume message", slog.String("error", err.Error()))
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
				s.logger.Error("could not unmarshal message", slog.String("error", err.Error()))
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
				s.logger.Error("could not send activation email", slog.String("error", err.Error()))
				continue
			}

			s.logger.Info("activation email sent", slog.String("email", data.Email))

			msg.Ack(false)
		}
	}()

	<-forever
}
