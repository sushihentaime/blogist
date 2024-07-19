package main

import "github.com/spf13/viper"

type Config struct {
	Port        string `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`

	DB struct {
		Host     string `mapstructure:"POSTGRES_HOST"`
		Port     string `mapstructure:"POSTGRES_PORT"`
		User     string `mapstructure:"POSTGRES_USER"`
		Password string `mapstructure:"POSTGRES_PASSWORD"`
		Name     string `mapstructure:"POSTGRES_DB"`
	}

	Mail struct {
		Host     string `mapstructure:"MAIL_HOST"`
		Port     int    `mapstructure:"MAIL_PORT"`
		User     string `mapstructure:"MAIL_USER"`
		Password string `mapstructure:"MAIL_PASSWORD"`
		Sender   string `mapstructure:"MAIL_SENDER"`
	}

	RabbitMQ struct {
		Host     string `mapstructure:"RABBITMQ_HOST"`
		Port     string `mapstructure:"RABBITMQ_PORT"`
		User     string `mapstructure:"RABBITMQ_USER"`
		Password string `mapstructure:"RABBITMQ_PASSWORD"`
	}
}

func loadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
