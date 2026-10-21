package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

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
	frontendBaseURL := os.Getenv("FRONTEND_BASE_URL")
	if frontendBaseURL == "" {
		frontendBaseURL = "http://localhost:5173"
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

	r2AccountID := os.Getenv("R2_ACCOUNT_ID")
	if r2AccountID == "" {
		return Config{}, errors.New("R2_ACCOUNT_ID is required")

	}

	r2AccessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	if r2AccessKeyID == "" {
		return Config{}, errors.New("R2_ACCESS_KEY_ID is required")
	}

	r2SecretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	if r2SecretAccessKey == "" {
		return Config{}, errors.New("R2_SECRET_ACCESS_KEY is required")
	}
	r2Bucket := os.Getenv("R2_BUCKET")
	if r2Bucket == "" {
		return Config{}, errors.New("R2_BUCKET is required")
	}

	r2PublicURL := os.Getenv("R2_PUBLIC_URL")
	if r2PublicURL == "" {
		return Config{}, errors.New("R2_PUBLIC_URL is required")
	}

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		return Config{}, errors.New("REDIS_HOST is required")
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		return Config{}, errors.New("REDIS_PORT is required")
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisPassword == "" {
		return Config{}, errors.New("REDIS_PASSWORD is required")
	}

	biteshipApiKey := os.Getenv("BITESHIP_API_KEY")
	if biteshipApiKey == "" {
		return Config{}, errors.New("BITESHIP_API_KEY is required")
	}

	biteshipBaseURL := os.Getenv("BITESHIP_BASE_URL")
	if biteshipBaseURL == "" {
		return Config{}, errors.New("BITESHIP_BASE_URL is required")
	}

	biteshipTimeout := os.Getenv("BITESHIP_TIMEOUT")
	if biteshipTimeout == "" {
		return Config{}, errors.New("BITESHIP_TIMEOUT is required")
	}

	biteshipOriginAreaID := os.Getenv("BITESHIP_ORIGIN_AREA_ID")
	if biteshipOriginAreaID == "" {
		return Config{}, errors.New("BITESHIP_ORIGIN_AREA_ID is required")
	}

	biteshipCouriers := os.Getenv("BITESHIP_COURIERS")
	if biteshipCouriers == "" {
		return Config{}, errors.New("BITESHIP_COURIERS is required")
	}

	xenditApiKey := os.Getenv("XENDIT_API_KEY")
	if xenditApiKey == "" {
		return Config{}, errors.New("XENDIT_API_KEY is required")
	}

	xenditWebhookToken := os.Getenv("XENDIT_WEBHOOK_TOKEN")
	if xenditWebhookToken == "" {
		return Config{}, errors.New("XENDIT_WEBHOOK_TOKEN is required")
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
			Host:            smtpHost,
			Port:            smtpPort,
			From:            smtpFrom,
			Username:        os.Getenv("SMTP_USERNAME"),
			Password:        os.Getenv("SMTP_PASSWORD"),
			AppBaseURL:      appBaseURL,
			FrontendBaseURL: frontendBaseURL,
		},
		R2: R2Storage{
			AccountID:       r2AccountID,
			AccessKeyID:     r2AccessKeyID,
			SecretAccessKey: r2SecretAccessKey,
			Bucket:          r2Bucket,
			PublicURL:       r2PublicURL,
		},
		Redis: RedisConfig{
			Host:     redisHost,
			Port:     redisPort,
			Password: redisPassword,
		},
		Biteship: BiteshipConfig{
			ApiKey:       biteshipApiKey,
			BaseURL:      biteshipBaseURL,
			Timeout:      biteshipTimeout,
			OriginAreaID: biteshipOriginAreaID,
			Couriers:     biteshipCouriers,
		},
		Xendit: XenditConfig{
			ApiKey:       xenditApiKey,
			WebhookToken: xenditWebhookToken,
		},
	}, nil

}
