package config

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/libs/cookies"
	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func New() Config {
	return Config{
		Environment: loadenv.GetEnv("ENVIRONMENT", "local"),
		Version:     loadenv.GetEnv("VERSION", "latest"),
		HTTP: HTTPConfig{
			Host: loadenv.GetEnv("HOST", "0.0.0.0"),
			Port: loadenv.GetEnv("PORT", "8080"),
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
		Websocket: WebsocketConfig{
			HandshakeTimeout: time.Second * time.Duration(
				loadenv.GetEnvAsInt("WEBSOCKET_HANDSHAKE_TIMEOUT", 2),
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
			Level: logging.Levels.DEBUG,
			LogFilePath: fmt.Sprintf(
				common.LogsPath,
				time.Now().In(common.Timezone).Format(common.DateFormat),
			),
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
					".{8,}",     // больше или равно 8 символам в длину
					"[a-z]",     // должно содержать букву на латиницу в нижнем регистре
					"[A-Z]",     // должно содержать букву на латиницу в верхнем регистре
					"[0-9]",     // должно содержать цифру
					"[^\\d\\w]", // должно содержать спецсимвол
				},
				";",
			),
			UsernameRegExps: loadenv.GetEnvAsSlice(
				"USERNAME_REGEXPS",
				[]string{
					`^.{5,70}$`,      // длина 5-70 символов
					`^[A-Za-z0-9]+$`, // только латинница и цифры
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
		WebPush: WebPushConfig{
			VAPIDPublicKey:  loadenv.GetEnv("VAPID_PUBLIC_KEY", ""),
			VAPIDPrivateKey: loadenv.GetEnv("VAPID_PRIVATE_KEY", ""),
			VAPIDContact:    loadenv.GetEnv("VAPID_CONTACT", "admin@example.com"),
			TTL:             loadenv.GetEnvAsInt("WEB_PUSH_TTL", 86400),
		},
		NATS: NATSConfig{
			ClientURL: fmt.Sprintf(
				"nats://%s:%d",
				loadenv.GetEnv("NATS_HOST", "0.0.0.0"),
				loadenv.GetEnvAsInt("NATS_CLIENT_PORT", 4222),
			),
			Subjects: NATSSubjects{
				EmailNotification: loadenv.GetEnv(
					"NATS_EMAIL_NOTIFICATION_SUBJECT",
					"email-notification",
				),
				WebPushNotification: loadenv.GetEnv(
					"NATS_WEB_PUSH_NOTIFICATION_SUBJECT",
					"web-push-notification",
				),
			},
			Publisher: NATSPublisher{
				Name: loadenv.GetEnv("NATS_PUBLISHER_NAME", "kfc-publisher"),
			},
			MessageChannelBufferSize: loadenv.GetEnvAsInt("NATS_MESSAGE_CHANNEL_BUFFER_SIZE", 1),
			GoroutinesPoolSize:       loadenv.GetEnvAsInt("NATS_GOROUTINES_POOL_SIZE", 1),
			Workers: NATSWorkers{
				EmailNotification: NATSWorker{
					Name: loadenv.GetEnv(
						"NATS_EMAIL_NOTIFICATION_WORKER_NAME",
						"email-notification-worker",
					),
				},
				WebPushNotification: NATSWorker{
					Name: loadenv.GetEnv(
						"NATS_WEB_PUSH_NOTIFICATION_WORKER_NAME",
						"web-push-notification-worker",
					),
				},
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
					Name: "Root",
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
				UnitOfWork: tracing.SpanConfig{
					Name: "UnitOfWork",
					Opts: []trace.SpanStartOption{
						trace.WithAttributes(
							attribute.String("Environment", loadenv.GetEnv("ENVIRONMENT", "local")),
						),
					},
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{
							Name: "Calling UnitOfWork",
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
							Name: "Received response from UnitOfWork",
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
						Name: "Auth repository",
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
						Name: "Users repository",
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
						Name: "Emails repository",
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
					Chats: tracing.SpanConfig{
						Name: "Chats repository",
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
								Name: "Calling Chats Repository",
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
								Name: "Received response from Chats Repository",
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
					Messages: tracing.SpanConfig{
						Name: "Messages repository",
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
								Name: "Calling Messages Repository",
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
								Name: "Received response from Messages Repository",
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
					Settings: tracing.SpanConfig{
						Name: "Settings repository",
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
								Name: "Calling Settings Repository",
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
								Name: "Received response from Settings Repository",
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
					WebPushSubscriptions: tracing.SpanConfig{
						Name: "WebPushSubscriptions repository",
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
								Name: "Calling WebPushSubscriptions Repository",
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
								Name: "Received response from WebPushSubscriptions Repository",
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
					WebPush: tracing.SpanConfig{
						Name: "WebPush repository",
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
								Name: "Calling WebPush Repository",
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
								Name: "Received response from WebPush Repository",
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
				Services: SpanServices{
					Auth: tracing.SpanConfig{
						Name: "Auth service",
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
								Name: "Calling Auth service",
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
								Name: "Received response from Auth service",
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
						Name: "Users service",
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
								Name: "Calling Users service",
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
								Name: "Received response from Users service",
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
					Chats: tracing.SpanConfig{
						Name: "Chats service",
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
								Name: "Calling Chats service",
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
								Name: "Received response from Chats service",
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
					Messages: tracing.SpanConfig{
						Name: "Messages service",
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
								Name: "Calling Messages service",
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
								Name: "Received response from Messages service",
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
					Notifications: tracing.SpanConfig{
						Name: "Notifications service",
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
								Name: "Calling Notifications service",
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
								Name: "Received response from Notifications service",
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
					Settings: tracing.SpanConfig{
						Name: "Settings service",
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
								Name: "Calling Settings Service",
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
								Name: "Received response from Settings Service",
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
					WebPushSubscriptions: tracing.SpanConfig{
						Name: "WebPushSubscriptions service",
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
								Name: "Calling WebPushSubscriptions service",
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
								Name: "Received response from WebPushSubscriptions service",
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
				UseCases: SpanUseCases{
					Auth: tracing.SpanConfig{
						Name: "Auth useCases",
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
								Name: "Calling Auth useCases",
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
								Name: "Received response from Auth useCases",
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
						Name: "Users useCases",
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
								Name: "Calling Users useCases",
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
								Name: "Received response from Users useCases",
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
					Chats: tracing.SpanConfig{
						Name: "Chats useCases",
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
								Name: "Calling Chats useCases",
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
								Name: "Received response from Chats useCases",
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
					Messages: tracing.SpanConfig{
						Name: "Messages useCases",
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
								Name: "Calling Messages useCases",
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
								Name: "Received response from Messages useCases",
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
					Notifications: tracing.SpanConfig{
						Name: "Notifications useCases",
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
								Name: "Calling Notifications useCases",
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
								Name: "Received response from Notifications useCases",
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
					Settings: tracing.SpanConfig{
						Name: "Settings useCases",
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
								Name: "Calling Settings UseCases",
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
								Name: "Received response from Settings UseCases",
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
					WebPushSubscriptions: tracing.SpanConfig{
						Name: "WebPushSubscriptions useCases",
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
								Name: "Calling WebPushSubscriptions useCases",
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
								Name: "Received response from WebPushSubscriptions useCases",
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
				Handlers: SpanHandlers{
					EmailNotification: tracing.SpanConfig{
						Name: "EmailNotification worker handler",
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
								Name: "Calling email-notification worker handler",
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
								Name: "Received response from email-notification worker handler",
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
					WebPushNotification: tracing.SpanConfig{
						Name: "WebPushNotification worker handler",
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
								Name: "Calling web-push-notification worker handler",
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
								Name: "Received response from web-push-notification worker handler",
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
	Port         string
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type EmailConfig struct {
	SMTP           SMTPConfig
	VerifyEmailURL string
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
	Auth                 tracing.SpanConfig
	Users                tracing.SpanConfig
	Emails               tracing.SpanConfig
	Chats                tracing.SpanConfig
	Messages             tracing.SpanConfig
	Settings             tracing.SpanConfig
	WebPushSubscriptions tracing.SpanConfig
	WebPush              tracing.SpanConfig
}

type SpanServices struct {
	Auth                 tracing.SpanConfig
	Users                tracing.SpanConfig
	Chats                tracing.SpanConfig
	Messages             tracing.SpanConfig
	Notifications        tracing.SpanConfig
	Settings             tracing.SpanConfig
	WebPushSubscriptions tracing.SpanConfig
}

type SpanUseCases struct {
	Auth                 tracing.SpanConfig
	Users                tracing.SpanConfig
	Chats                tracing.SpanConfig
	Messages             tracing.SpanConfig
	Notifications        tracing.SpanConfig
	Settings             tracing.SpanConfig
	WebPushSubscriptions tracing.SpanConfig
}

type SpanHandlers struct {
	EmailNotification   tracing.SpanConfig
	WebPushNotification tracing.SpanConfig
}

type SpansConfig struct {
	Root         tracing.SpanConfig
	UnitOfWork   tracing.SpanConfig
	Repositories SpanRepositories
	Services     SpanServices
	UseCases     SpanUseCases
	Handlers     SpanHandlers
}

type TracingConfig struct {
	Server tracing.Config
	Spans  SpansConfig
}

type CookiesConfig struct {
	AccessToken  cookies.Config
	RefreshToken cookies.Config
}

type WebsocketConfig struct {
	HandshakeTimeout time.Duration
}

type NATSConfig struct {
	ClientURL                string
	MessageChannelBufferSize int
	GoroutinesPoolSize       int
	Subjects                 NATSSubjects
	Publisher                NATSPublisher
	Workers                  NATSWorkers
}

type NATSSubjects struct {
	EmailNotification   string
	WebPushNotification string
}

type NATSPublisher struct {
	Name string
}
type NATSWorkers struct {
	EmailNotification   NATSWorker
	WebPushNotification NATSWorker
}

type NATSWorker struct {
	Name string
}

type WebPushConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDContact    string
	TTL             int
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
	Websocket   WebsocketConfig
	NATS        NATSConfig
	WebPush     WebPushConfig
}
