package mailservice

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
	"golang.org/x/exp/rand"
)

func NewMailService(mb common.MessageConsumer, host, username, password, sender string, port int, logger *slog.Logger) *MailService {
	return &MailService{
		mb:     mb,
		m:      NewMailer(host, port, username, password, sender, NewTemplate()),
		logger: logger,
	}
}

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

			payload := struct {
				ActivationToken string
			}{
				ActivationToken: data.Token,
			}

			// using exponential backoff with jitter
			const maxRetries = 5
			const baseDelay = 500 * time.Millisecond

			var attempt int
			for attempt = 0; attempt < maxRetries; attempt++ {
				err = s.m.send(data.Email, payload, "activation_email.html")
				if err == nil {
					s.logger.Info("activation email sent", slog.String("email", data.Email))
					msg.Ack(false)
					break
				}

				delay := time.Duration(rand.Int63n(int64(baseDelay) << uint(attempt)))
				s.logger.Info("delaying activation email", slog.String("email", data.Email), slog.Int("attempt", attempt), slog.Duration("delay", delay))
				time.Sleep(delay)
			}

			if attempt == maxRetries {
				s.logger.Error("could not send activation email", slog.String("email", data.Email))
				msg.Nack(false, true)
			}
		}
	}()

	<-forever
}
