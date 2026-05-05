# Пакет internal/common

## Назначение

Общие константы и настройки, используемые в разных слоях приложения.

## Файлы

### const.go

| Константа / переменная | Значение | Назначение |
|---|---|---|
| `DateFormat` | `"02.01.2006"` | Формат дат в именах файлов логов |
| `SaltSeparator` | `":"` | Разделитель при генерации солей |
| `LoggingTraceSkipLevel` | `1` | Уровень пропуска стека при логировании трейсов |
| `Timezone` | `Europe/Moscow` | Временная зона; инициализируется через `init()` |

Пакет `time/tzdata` импортируется пустым импортом для корректной работы таймзон в Docker-контейнерах.

### paths.go

| Константа | Значение | Назначение |
|---|---|---|
| `LogsPath` | `"logs/%s.log"` | Шаблон пути к файлу логов (подставляется дата) |
| `BackupsDir` | `"postgres_backups"` | Директория резервных копий PostgreSQL |
| `BackupLogsPath` | `"postgres_backups/backup.log"` | Лог процесса резервного копирования |

### cache.go — ключи и лимиты для Redis

#### Rate limiting (защита от спама)

| Константа | Значение | Назначение |
|---|---|---|
| `VerifyEmailRateLimitPrefix` | `"email_verification"` | Префикс ключа счётчика |
| `VerifyEmailRateLimitCount` | `3` | Макс. число попыток |
| `VerifyEmailRateLimitTTL` | `3 мин` | Окно ограничения |
| `ForgetPasswordRateLimitPrefix` | `"forget_password"` | Префикс ключа счётчика |
| `ForgetPasswordRateLimitCount` | `3` | Макс. число попыток |
| `ForgetPasswordRateLimitTTL` | `3 мин` | Окно ограничения |

#### Токены подтверждения

| Константа | Значение | Назначение |
|---|---|---|
| `VerifyEmailTokenPrefix` | `"verify_email_token"` | Префикс ключа токена в Redis |
| `ForgetPasswordTokenPrefix` | `"forget_password_token"` | Префикс ключа токена в Redis |
| `TokenTTL` | `15 мин` | Время жизни одноразового токена |
| `InitCacheValue` | `1` | Начальное значение счётчика попыток |

## Зависимости

Только стандартная библиотека (`time`, `time/tzdata`).
