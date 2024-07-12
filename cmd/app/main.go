package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sushihentaime/blogist/internal/blogservice"
	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/mailservice"
	"github.com/sushihentaime/blogist/internal/userservice"
)

type application struct {
	// config, logger, db, broker, services, etc.
	config      *Config
	logger      *slog.Logger
	userService *userservice.UserService
	blogService *blogservice.BlogService
	mailService *mailservice.MailService
	broker      *common.MessageBroker
}

func main() {
	// Initialize the logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load the configuration
	cfg, err := loadConfig(".env")
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize the database
	db, err := common.NewDB(cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, 10, 5, 15*time.Minute)
	if err != nil {
		logger.Error("failed to connect to the database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer common.CloseDB(db)

	// Initialize the message broker
	// Create the URI and connect to the message broker
	URI := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.RabbitMQ.User, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	broker, err := common.NewMessageBroker(URI)
	if err != nil {
		logger.Error("failed to connect to the message broker", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer broker.Close()

	// Setup the exchange, queue, and binding key
	err = common.SetupUserExchange(broker)
	if err != nil {
		logger.Error("failed to setup the user exchange", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize the services
	app := &application{
		config:      cfg,
		logger:      logger,
		userService: userservice.NewUserService(db, broker),
		blogService: blogservice.NewBlogService(db),
		broker:      broker,
		mailService: mailservice.NewMailService(broker, cfg.Mail.Host, cfg.Mail.User, cfg.Mail.Password, cfg.Mail.Sender, cfg.Mail.Port, logger),
	}

	// Initialize the consumer
	go app.mailService.SendActivationEmail()

	// Start the HTTP server
	err = app.serve(cfg.Port)
	if err != nil {
		logger.Error("failed to start the server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
