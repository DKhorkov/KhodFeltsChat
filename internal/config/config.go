package config

import (
	"fmt"
	"github.com/DKhorkov/khodfeltschat/internal/common"
	"time"

	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
)

func New() Config {
	return Config{
		Environment: loadenv.GetEnv("ENVIRONMENT", "local"),
		Version:     loadenv.GetEnv("VERSION", "latest"),
		HTTP: HTTPConfig{
			Host: loadenv.GetEnv("HOST", "0.0.0.0"),
			Port: loadenv.GetEnvAsInt("PORT", 8080),
			ReadTimeout: time.Second * time.Duration(
				loadenv.GetEnvAsInt("HTTP_READ_TIMEOUT", 3),
			),
			ReadHeaderTimeout: time.Second * time.Duration(
				loadenv.GetEnvAsInt("HTTP_READ_HEADER_TIMEOUT", 1),
			),
			TimeoutHandlerTimeout: time.Second * time.Duration(
				loadenv.GetEnvAsInt("HTTP_TIMEOUT_HANDLER_TIMEOUT", 2),
			),
		},
		Database: postgresql.Config{
			Host:         loadenv.GetEnv("POSTGRES_HOST", "0.0.0.0"),
			Port:         loadenv.GetEnvAsInt("POSTGRES_PORT", 5432),
			User:         loadenv.GetEnv("POSTGRES_USER", "postgres"),
			Password:     loadenv.GetEnv("POSTGRES_PASSWORD", "postgres"),
			DatabaseName: loadenv.GetEnv("POSTGRES_DB", "postgres"),
			SSLMode:      loadenv.GetEnv("POSTGRES_SSL_MODE", "disable"),
			Driver:       loadenv.GetEnv("POSTGRES_DRIVER", "postgres"),
			Pool: postgresql.PoolConfig{
				MaxIdleConnections: loadenv.GetEnvAsInt("MAX_IDLE_CONNECTIONS", 1),
				MaxOpenConnections: loadenv.GetEnvAsInt("MAX_OPEN_CONNECTIONS", 1),
				MaxConnectionLifetime: time.Second * time.Duration(
					loadenv.GetEnvAsInt("MAX_CONNECTION_LIFETIME", 20),
				),
				MaxConnectionIdleTime: time.Second * time.Duration(
					loadenv.GetEnvAsInt("MAX_CONNECTION_IDLE_TIME", 10),
				),
			},
		},
		Logging: logging.Config{
			Level:       logging.Levels.DEBUG,
			LogFilePath: fmt.Sprintf(common.LogsPath, time.Now().In(common.Timezone).Format(common.DateFormat)),
		},
		Email: EmailConfig{
			SMTP: SMTPConfig{
				Host:     loadenv.GetEnv("EMAIL_SMTP_HOST", "smtp.freesmtpservers.com"),
				Port:     loadenv.GetEnvAsInt("EMAIL_SMTP_PORT", 25),
				Login:    loadenv.GetEnv("EMAIL_SMTP_LOGIN", "smtp"),
				Password: loadenv.GetEnv("EMAIL_SMTP_PASSWORD", "smtp"),
			},
			VerifyEmailURL: loadenv.GetEnv(
				"VERIFY_EMAIL_URL",
				"http://localhost:3000/verify-email",
			),
			ForgetPasswordURL: loadenv.GetEnv(
				"FORGET_PASSWORD_URL",
				"http://localhost:3000/forget-password",
			),
		},
		Cache: CacheConfig{
			Password: loadenv.GetEnv("REDIS_PASSWORD", ""),
			Host:     loadenv.GetEnv("REDIS_HOST", "0.0.0.0"),
			Port:     loadenv.GetEnvAsInt("REDIS_PORT", 6379),
		},
		Validation: ValidationConfig{
			EmailRegExp: loadenv.GetEnv(
				"EMAIL_REGEXP",
				"^[a-z0-9._%+\\-]+@[a-z0-9.\\-]+\\.[a-z]{2,4}$",
			),
			PasswordRegExps: loadenv.GetEnvAsSlice(
				"PASSWORD_REGEXPS",
				[]string{
					".{8,}",
					"[a-z]",
					"[A-Z]",
					"[0-9]",
					"[^\\d\\w]",
				},
				";",
			),
			UsernameRegExps: loadenv.GetEnvAsSlice(
				"USERNAME_REGEXPS",
				[]string{
					`^.{5,70}$`,   // длина 5-70 символов
					`^[A-Za-z]+$`, // только латинница
				},
				";",
			),
		},
	}
}

type HTTPConfig struct {
	Host                  string
	Port                  int
	ReadHeaderTimeout     time.Duration
	ReadTimeout           time.Duration
	TimeoutHandlerTimeout time.Duration
}

type EmailConfig struct {
	SMTP              SMTPConfig
	VerifyEmailURL    string
	ForgetPasswordURL string
	TicketUpdatedURL  string
	TicketDeletedURL  string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Login    string
	Password string
}

type ValidationConfig struct {
	EmailRegExp     string
	PasswordRegExps []string // Slice of rules to pass, because Go's regex doesn't support backtracking.
	UsernameRegExps []string
}

type CacheConfig struct {
	Host     string
	Port     int
	Password string
}

type Config struct {
	HTTP        HTTPConfig
	Database    postgresql.Config
	Logging     logging.Config
	Environment string
	Version     string
	Email       EmailConfig
	Cache       CacheConfig
	Validation  ValidationConfig
}
