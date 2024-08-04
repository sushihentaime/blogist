package mailservice

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
	"golang.org/x/exp/rand"
)

func NewMailService(mb common.MessageConsumer, host, username, password, sender string, port int, logger *slog.Logger) *MailService {
	ctx, cancel := context.WithCancel(context.Background())
	return &MailService{
		mb:     mb,
		m:      NewMailer(host, port, username, password, sender, NewTemplate()),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *MailService) SendActivationEmail() {
	msgs, err := s.mb.Consume(common.UserCreatedKey, common.UserExchange, common.UserCreatedQueue)
	if err != nil {
		s.logger.Error("could not consume message", slog.String("error", err.Error()))
		return
	}

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}

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
					msg.Ack(false)
				}

			case <-s.ctx.Done():
				s.logger.Info("stopping SendActivationEmail due to context cancellation")
				return
			}
		}
	}()
}

func (s *MailService) Close() {
	s.cancel()
}
