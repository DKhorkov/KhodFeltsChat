# Пакет internal/config

## Назначение

Читает настройки приложения из переменных окружения (с дефолтными значениями) и формирует единую структуру `Config`.

## Функция

### New() Config

Создаёт и возвращает заполненный `Config`. Использует `loadenv.GetEnv*` для чтения каждого параметра.

## Корневая структура

```go
type Config struct {
    HTTP        HTTPConfig
    Security    security.Config   // JWT TTL, алгоритм, bcrypt HashCost
    Database    postgresql.Config // хост, порт, credentials, пул соединений
    Logging     logging.Config    // уровень, путь к файлу логов
    Environment string
    Version     string
    Email       EmailConfig       // SMTP + URL для подтверждения email
    Cache       CacheConfig       // Redis host/port/password
    Validation  ValidationConfig  // regexp для email, пароля, username
    CORS        CORSConfig
    Docs        DocsConfig        // директория и файл Swagger
    Cookies     CookiesConfig     // настройки access- и refresh-token cookies
    Tracing     TracingConfig     // Jaeger URL + SpanConfig для каждого слоя
    Websocket   WebsocketConfig   // HandshakeTimeout
    NATS        NATSConfig        // URL, subjects, worker/publisher names, pool size
    WebPush     WebPushConfig     // VAPID ключи для Web Push уведомлений
}
```

## Вложенные конфиги

| Тип | Ключевые env-переменные |
|---|---|
| `HTTPConfig` | `HOST`, `PORT`, `HTTP_*_TIMEOUT` |
| `postgresql.Config` + `PoolConfig` | `POSTGRES_*`, `MAX_*_CONNECTIONS`, `MAX_CONNECTION_*` |
| `EmailConfig` / `SMTPConfig` | `EMAIL_SMTP_*`, `VERIFY_EMAIL_URL` |
| `CacheConfig` | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` |
| `ValidationConfig` | `EMAIL_REGEXP`, `PASSWORD_REGEXPS`, `USERNAME_REGEXPS` |
| `security.Config` | `HASH_COST`, `JWT_*`, `REFRESH/ACCESS_TOKEN_JWT_TTL` |
| `CookiesConfig` | `COOKIES_ACCESS_TOKEN_*`, `COOKIES_REFRESH_TOKEN_*` |
| `NATSConfig` | `NATS_HOST`, `NATS_CLIENT_PORT`, `NATS_*_SUBJECT`, `NATS_*_WORKER_NAME` |
| `TracingConfig` | `TRACING_SERVICE_NAME`, `TRACING_JAEGER_HOST`, `TRACING_API_TRACES_PORT` |
| `WebsocketConfig` | `WEBSOCKET_HANDSHAKE_TIMEOUT` |
| `WebPushConfig` | `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_CONTACT` |

Путь к файлу логов формируется как `logs/<дата>.log` с использованием `common.Timezone` (Europe/Moscow).

## Зависимости

- `github.com/DKhorkov/kfc/internal/common`
- `github.com/DKhorkov/libs/loadenv`, `logging`, `security`, `tracing`, `cookies`, `db/postgresql`
