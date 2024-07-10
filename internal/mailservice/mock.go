package mailservice

import (
	"bytes"

	"github.com/go-mail/mail/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/mock"
	"github.com/sushihentaime/blogist/internal/common"
)

type MockTemplate struct {
	mock.Mock
}

func (m *MockTemplate) ParseTemplate(name string, data any) (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer, error) {
	args := m.Called(name, data)
	return args.Get(0).(*bytes.Buffer), args.Get(1).(*bytes.Buffer), args.Get(2).(*bytes.Buffer), args.Error(3)
}

type MockDialer struct {
	mock.Mock
}

func (d *MockDialer) DialAndSend(m ...*mail.Message) error {
	args := d.Called(m)
	return args.Error(0)
}

type MockMailer struct {
	Called bool
	Email  string
	mock.Mock
}

func (m *MockMailer) send(recipient string, data any, templateFile string) error {
	m.Called = true
	m.Email = recipient
	return nil
}

type MockMessageConsumer struct {
	mock.Mock
}

func (m *MockMessageConsumer) Consume(key common.BindingKey, exchange common.Exchange, queue common.Queue) (<-chan amqp.Delivery, error) {
	msgsChan := make(chan amqp.Delivery)

	go func() {
		defer close(msgsChan)

		mockMessage := `{"Email": "test@example.com", "Token": "testtoken"}`
		mockDelivery := amqp.Delivery{Body: []byte(mockMessage)}
		msgsChan <- mockDelivery
	}()

	return msgsChan, nil
}
