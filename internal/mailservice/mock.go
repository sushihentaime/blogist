package mailservice

import (
	"bytes"
	"sync"

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
	mu     sync.Mutex
	Called bool
	Email  string
	mock.Mock
}

func (m *MockMailer) send(recipient string, data any, templateFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Called = true
	m.Email = recipient
	return nil
}

func (m *MockMailer) IsCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Called
}

func (m *MockMailer) GetEmail() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Email
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

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Error(msg string, fields ...any) {
	m.Called(msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}
