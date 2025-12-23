package config

import (
	"fmt"
	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/libs/cookies"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"net/http"
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
			IdleTimeout: time.Second * time.Duration(
				loadenv.GetEnvAsInt("HTTP_IDLE_TIMEOUT", 10),
			),
			WriteTimeout: time.Second * time.Duration(
				loadenv.GetEnvAsInt("HTTP_WRITE_TIMEOUT", 2),
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
				"http://localhost:443/verify-email",
			),
			ForgetPasswordURL: loadenv.GetEnv(
				"FORGET_PASSWORD_URL",
				"http://localhost:443/forget-password",
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
		CORS: CORSConfig{
			AllowedOrigins:   loadenv.GetEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"*"}, ", "),
			AllowedMethods:   loadenv.GetEnvAsSlice("CORS_ALLOWED_METHODS", []string{"*"}, ", "),
			AllowedHeaders:   loadenv.GetEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"*"}, ", "),
			AllowCredentials: loadenv.GetEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           loadenv.GetEnvAsInt("CORS_MAX_AGE", 600),
		},
		Docs: DocsConfig{
			Dir:      loadenv.GetEnv("DOCS_DIR", "./"),
			Filepath: loadenv.GetEnv("DOCS_FILEPATH", "swagger.yaml"),
		},
		Security: security.Config{
			HashCost: loadenv.GetEnvAsInt("HASH_COST", 8), // Auth speed sensitive if large
			JWT: security.JWTConfig{
				RefreshTokenTTL: time.Hour * time.Duration(
					loadenv.GetEnvAsInt("REFRESH_TOKEN_JWT_TTL", 168),
				),
				AccessTokenTTL: time.Minute * time.Duration(
					loadenv.GetEnvAsInt("ACCESS_TOKEN_JWT_TTL", 15),
				),
				Algorithm: loadenv.GetEnv("JWT_ALGORITHM", "HS256"),
				SecretKey: loadenv.GetEnv("JWT_SECRET", "defaultSecret"),
			},
		},
		Cookies: CookiesConfig{
			AccessToken: cookies.Config{
				Path:   loadenv.GetEnv("COOKIES_ACCESS_TOKEN_PATH", "/"),
				Domain: loadenv.GetEnv("COOKIES_ACCESS_TOKEN_DOMAIN", ""),
				MaxAge: loadenv.GetEnvAsInt("COOKIES_ACCESS_TOKEN_MAX_AGE", 0),
				Expires: time.Minute * time.Duration(
					loadenv.GetEnvAsInt("COOKIES_ACCESS_TOKEN_EXPIRES", 15),
				),
				Secure:   loadenv.GetEnvAsBool("COOKIES_ACCESS_TOKEN_SECURE", false),
				HTTPOnly: loadenv.GetEnvAsBool("COOKIES_ACCESS_TOKEN_HTTP_ONLY", false),
				SameSite: http.SameSite(
					loadenv.GetEnvAsInt("COOKIES_ACCESS_TOKEN_SAME_SITE", 1),
				),
			},
			RefreshToken: cookies.Config{
				Path:   loadenv.GetEnv("COOKIES_REFRESH_TOKEN_PATH", "/"),
				Domain: loadenv.GetEnv("COOKIES_REFRESH_TOKEN_DOMAIN", ""),
				MaxAge: loadenv.GetEnvAsInt("COOKIES_REFRESH_TOKEN_MAX_AGE", 0),
				Expires: time.Hour * time.Duration(
					loadenv.GetEnvAsInt("COOKIES_REFRESH_TOKEN_EXPIRES", 24*7),
				),
				Secure:   loadenv.GetEnvAsBool("COOKIES_REFRESH_TOKEN_SECURE", false),
				HTTPOnly: loadenv.GetEnvAsBool("COOKIES_REFRESH_TOKEN_HTTP_ONLY", false),
				SameSite: http.SameSite(
					loadenv.GetEnvAsInt("COOKIES_REFRESH_TOKEN_SAME_SITE", 1),
				),
			},
		},
		Tracing: TracingConfig{
			Server: tracing.Config{
				ServiceName:    loadenv.GetEnv("TRACING_SERVICE_NAME", "kfc"),
				ServiceVersion: loadenv.GetEnv("VERSION", "latest"),
				JaegerURL: fmt.Sprintf(
					"http://%s:%d/api/traces",
					loadenv.GetEnv("TRACING_JAEGER_HOST", "0.0.0.0"),
					loadenv.GetEnvAsInt("TRACING_API_TRACES_PORT", 14268),
				),
			},
			Spans: SpansConfig{
				Root: tracing.SpanConfig{
					Opts: []trace.SpanStartOption{
						trace.WithAttributes(
							attribute.String("Environment", loadenv.GetEnv("ENVIRONMENT", "local")),
						),
					},
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{
							Name: "Calling handler",
							Opts: []trace.EventOption{
								trace.WithAttributes(
									attribute.String(
										"Environment",
										loadenv.GetEnv("ENVIRONMENT", "local"),
									),
								),
							},
						},
						End: tracing.SpanEventConfig{
							Name: "Received response from handler",
							Opts: []trace.EventOption{
								trace.WithAttributes(
									attribute.String(
										"Environment",
										loadenv.GetEnv("ENVIRONMENT", "local"),
									),
								),
							},
						},
					},
				},
				Repositories: SpanRepositories{
					Auth: tracing.SpanConfig{
						Opts: []trace.SpanStartOption{
							trace.WithAttributes(
								attribute.String(
									"Environment",
									loadenv.GetEnv("ENVIRONMENT", "local"),
								),
							),
						},
						Events: tracing.SpanEventsConfig{
							Start: tracing.SpanEventConfig{
								Name: "Calling Auth Repository",
								Opts: []trace.EventOption{
									trace.WithAttributes(
										attribute.String(
											"Environment",
											loadenv.GetEnv("ENVIRONMENT", "local"),
										),
									),
								},
							},
							End: tracing.SpanEventConfig{
								Name: "Received response from Auth Repository",
								Opts: []trace.EventOption{
									trace.WithAttributes(
										attribute.String(
											"Environment",
											loadenv.GetEnv("ENVIRONMENT", "local"),
										),
									),
								},
							},
						},
					},
					Users: tracing.SpanConfig{
						Opts: []trace.SpanStartOption{
							trace.WithAttributes(
								attribute.String(
									"Environment",
									loadenv.GetEnv("ENVIRONMENT", "local"),
								),
							),
						},
						Events: tracing.SpanEventsConfig{
							Start: tracing.SpanEventConfig{
								Name: "Calling Users Repository",
								Opts: []trace.EventOption{
									trace.WithAttributes(
										attribute.String(
											"Environment",
											loadenv.GetEnv("ENVIRONMENT", "local"),
										),
									),
								},
							},
							End: tracing.SpanEventConfig{
								Name: "Received response from Users Repository",
								Opts: []trace.EventOption{
									trace.WithAttributes(
										attribute.String(
											"Environment",
											loadenv.GetEnv("ENVIRONMENT", "local"),
										),
									),
								},
							},
						},
					},
					Emails: tracing.SpanConfig{
						Opts: []trace.SpanStartOption{
							trace.WithAttributes(
								attribute.String(
									"Environment",
									loadenv.GetEnv("ENVIRONMENT", "local"),
								),
							),
						},
						Events: tracing.SpanEventsConfig{
							Start: tracing.SpanEventConfig{
								Name: "Calling Emails Repository",
								Opts: []trace.EventOption{
									trace.WithAttributes(
										attribute.String(
											"Environment",
											loadenv.GetEnv("ENVIRONMENT", "local"),
										),
									),
								},
							},
							End: tracing.SpanEventConfig{
								Name: "Received response from Emails Repository",
								Opts: []trace.EventOption{
									trace.WithAttributes(
										attribute.String(
											"Environment",
											loadenv.GetEnv("ENVIRONMENT", "local"),
										),
									),
								},
							},
						},
					},
				},
			},
		},
	}
}

type HTTPConfig struct {
	Host         string
	Port         int
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
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

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	MaxAge           int
	AllowCredentials bool
}

type DocsConfig struct {
	Dir      string
	Filepath string
}

type SpanRepositories struct {
	Auth   tracing.SpanConfig
	Users  tracing.SpanConfig
	Emails tracing.SpanConfig
}

type SpansConfig struct {
	Root         tracing.SpanConfig
	Repositories SpanRepositories
}

type TracingConfig struct {
	Server tracing.Config
	Spans  SpansConfig
}

type CookiesConfig struct {
	AccessToken  cookies.Config
	RefreshToken cookies.Config
}

type Config struct {
	HTTP        HTTPConfig
	Security    security.Config
	Database    postgresql.Config
	Logging     logging.Config
	Environment string
	Version     string
	Email       EmailConfig
	Cache       CacheConfig
	Validation  ValidationConfig
	CORS        CORSConfig
	Docs        DocsConfig
	Cookies     CookiesConfig
	Tracing     TracingConfig
}
