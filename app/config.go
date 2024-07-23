package main

import "github.com/spf13/viper"

type Config struct {
	Port           string   `mapstructure:"PORT"`
	Environment    string   `mapstructure:"ENVIRONMENT"`
	Version        string   `mapstructure:"VERSION"`
	TrustedOrigins []string `mapstructure:"TRUSTED_ORIGINS"`

	DBHost string `mapstructure:"POSTGRES_HOST"`
	// DBPort     string `mapstructure:"POSTGRES_PORT"`
	DBUser     string `mapstructure:"POSTGRES_USER"`
	DBPassword string `mapstructure:"POSTGRES_PASSWORD"`
	DBName     string `mapstructure:"POSTGRES_DB"`

	MailHost     string `mapstructure:"MAIL_HOST"`
	MailPort     int    `mapstructure:"MAIL_PORT"`
	MailUser     string `mapstructure:"MAIL_USER"`
	MailPassword string `mapstructure:"MAIL_PASSWORD"`
	MailSender   string `mapstructure:"MAIL_SENDER"`

	MQHost string `mapstructure:"RABBITMQ_HOST"`
	// MQPort     string `mapstructure:"RABBITMQ_PORT"`
	MQUser     string `mapstructure:"RABBITMQ_USER"`
	MQPassword string `mapstructure:"RABBITMQ_PASSWORD"`
}

func loadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
