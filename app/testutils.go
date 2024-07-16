package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sushihentaime/blogist/internal/blogservice"
	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/mailservice"
	"github.com/sushihentaime/blogist/internal/userservice"
)

type testServer struct {
	*httptest.Server
}

func newTestServer(t *testing.T, h http.Handler) *testServer {
	ts := httptest.NewServer(h)

	t.Cleanup(ts.Close)

	return &testServer{ts}
}

func readResponse(t *testing.T, res *http.Response) (int, http.Header, envelope) {
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	var envelope envelope
	err = json.Unmarshal(responseBody, &envelope)
	if err != nil {
		t.Fatal(err)
	}

	return res.StatusCode, res.Header, envelope
}

func newTestApplication(t *testing.T) (*application, *sql.DB) {
	db := common.TestDB("file://../migrations", t)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	rabbitURI := common.TestRabbitMQ(t)
	rabbitmq, err := common.NewMessageBroker(rabbitURI)
	assert.NoError(t, err)

	err = common.SetupUserExchange(rabbitmq)
	assert.NoError(t, err)

	cfg, err := loadConfig("../.test.env")
	assert.NoError(t, err)

	app := &application{
		config:      cfg,
		logger:      logger,
		userService: userservice.NewUserService(db, rabbitmq),
		mailService: mailservice.NewMailService(rabbitmq, cfg.Mail.Host, cfg.Mail.User, cfg.Mail.Password, cfg.Mail.Sender, cfg.Mail.Port, logger),
		broker:      rabbitmq,
		blogService: blogservice.NewBlogService(db),
	}

	return app, db
}

func (ts *testServer) post(t *testing.T, path string, data any, token *string) (int, http.Header, envelope) {
	jsonPayload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader(jsonPayload)
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	return readResponse(t, res)
}

func (ts *testServer) get(t *testing.T, path string, token *string, payload any) (int, http.Header, envelope) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader(jsonPayload)
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	}
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	return readResponse(t, res)
}

func (ts *testServer) put(t *testing.T, path string, token *string, payload any) (int, http.Header, envelope) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader(jsonPayload)
	req, err := http.NewRequest(http.MethodPut, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	}
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	return readResponse(t, res)
}

func (ts *testServer) delete(t *testing.T, path string, token *string) (int, http.Header, envelope) {
	req, err := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	}
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	return readResponse(t, res)
}
