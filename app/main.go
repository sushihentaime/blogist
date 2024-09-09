package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/sushihentaime/blogist/docs"
	"github.com/sushihentaime/blogist/internal/blogservice"
	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/likeservice"
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
	likeService *likeservice.LikeService
	broker      *common.MessageBroker
}

// @title Blog API
// @version 1.0
// @description This is a blog service API for managing users and blog posts.

// @license.name MIT License
// @license.url http://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the token
func main() {
	// Initialize the logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load the configuration
	cfg, err := loadConfig(".env")
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.Environment == "development" {
		cfg.RateLimitEnabled = false
	} else {
		cfg.RateLimitEnabled = true
	}

	// Initialize the database
	db, err := common.NewDB(cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, 10, 5, 15*time.Minute)
	if err != nil {
		logger.Error("failed to connect to the database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer common.CloseDB(db)

	// Initialize the message broker
	// Create the URI and connect to the message broker
	URI := fmt.Sprintf("amqp://%s:%s@%s/", cfg.MQUser, cfg.MQPassword, cfg.MQHost)
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

	cache := common.NewCache(5*time.Minute, 10*time.Minute)

	// Initialize the services
	app := &application{
		config:      cfg,
		logger:      logger,
		userService: userservice.NewUserService(db, broker, cache),
		blogService: blogservice.NewBlogService(db, cache),
		broker:      broker,
		mailService: mailservice.NewMailService(broker, cfg.MailHost, cfg.MailUser, cfg.MailPassword, cfg.MailSender, cfg.MailPort, logger),
		likeService: likeservice.NewLikeService(db, cache),
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
