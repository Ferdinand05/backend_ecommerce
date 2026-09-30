package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	SSLMode  string
	User     string
	Password string
	AppPort  string
}

type JWTConfig struct {
	SecretKey string
}

type MailConfig struct {
	Host       string
	Port       string
	Username   string
	Password   string
	From       string
	AppBaseURL string
}

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Mail     MailConfig
}

func Load() (Config, error) {

	_ = godotenv.Load()
	// if err != nil {
	// 	return Config{}, fmt.Errorf("Error loading .env file:%w", err)
	// }

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return Config{}, fmt.Errorf("DB_USER is required")
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return Config{}, fmt.Errorf("DB_PASSWORD is required")
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return Config{}, fmt.Errorf("DB_HOST is required")
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		return Config{}, fmt.Errorf("DB_PORT is required")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return Config{}, fmt.Errorf("DB_NAME is required")
	}

	dbSSLMode := os.Getenv("DB_SSLMODE")
	if dbSSLMode == "" {
		return Config{}, fmt.Errorf("DB_SSLMODE is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	appBaseURL := os.Getenv("APP_BASE_URL")
	if appBaseURL == "" {
		appBaseURL = "http://localhost:8080"
	}

	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		return Config{}, errors.New("SMTP_HOST is required")
	}

	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		return Config{}, errors.New("SMTP_PORT is required")
	}

	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		return Config{}, errors.New("SMTP_FROM is required")
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		return Config{}, errors.New("APP_PORT is required")
	}

	dbConfig := DatabaseConfig{
		Host:     dbHost,
		Port:     dbPort,
		Name:     dbName,
		SSLMode:  dbSSLMode,
		User:     dbUser,
		Password: dbPassword,
		AppPort:  appPort,
	}

	return Config{
		Database: dbConfig,
		JWT: JWTConfig{
			SecretKey: jwtSecret,
		},
		Mail: MailConfig{
			Host:       smtpHost,
			Port:       smtpPort,
			From:       smtpFrom,
			Username:   os.Getenv("SMTP_USERNAME"),
			Password:   os.Getenv("SMTP_PASSWORD"),
			AppBaseURL: appBaseURL,
		},
	}, nil

}
