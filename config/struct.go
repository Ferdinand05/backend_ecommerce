package config

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
	Host            string
	Port            string
	Username        string
	Password        string
	From            string
	AppBaseURL      string
	FrontendBaseURL string
}

type R2Storage struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

type BiteshipConfig struct {
	ApiKey             string
	BaseURL            string
	Timeout            string
	OriginAreaID       string
	Couriers           string
	SignatureKey       string
	SignatureSecret    string
	OriginContactName  string
	OriginContactPhone string
	OriginAddress      string
}

type XenditConfig struct {
	ApiKey       string
	WebhookToken string
}

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Mail     MailConfig
	R2       R2Storage
	Redis    RedisConfig
	Biteship BiteshipConfig
	Xendit   XenditConfig
}
