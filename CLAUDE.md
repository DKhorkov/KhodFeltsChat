# KhodFeltsChat — Справка для Claude Code

## Быстрый старт

- **Модуль:** `github.com/DKhorkov/kfc`
- **Go:** 1.24
- **Архитектура:** Clean Architecture (domains → repositories → services → usecases → controllers)

## Документация

- [Архитектура проекта](docs/architecture.md) — общее описание, слои, инфраструктура, API
- [Индекс модулей](docs/modules.md) — таблица всех пакетов с описаниями
- `doc.md` в каждой директории с кодом — детали конкретного модуля

> **Правило:** при изменении кода в директории **обязательно** обнови `doc.md` в этой же директории, чтобы документация соответствовала актуальному состоянию кода.

## Ключевые команды

```bash
task local          # Поднять инфраструктуру (docker) + запуск приложения
task prod           # Полный деплой через docker-compose
go test ./...       # Запуск тестов
task lint           # Линтер (golangci-lint)
task migrate-up     # Применить миграции
```

## Ключевые URL (локально)

| Сервис | URL |
|--------|-----|
| API + Swagger | http://localhost:8080/docs |
| Jaeger (трассировка) | http://localhost:16686 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |

## Структура проекта

```
cmd/                    — точка входа, DI, запуск серверов
internal/
  app/                  — обёртка запуска с graceful shutdown
  config/               — конфигурация из env
  domains/              — доменные модели и DTO
  common/               — константы, пути, настройки кэша
  errors/               — sentinel ошибки
  interfaces/           — все интерфейсы (mockgen)
  repositories/         — слой данных (PostgreSQL + SMTP)
  services/             — бизнес-логика над репозиториями (UoW)
  usecases/             — orchestration, валидация, кэш-декораторы
  controllers/http/     — HTTP/WS контроллер, handlers, schemas, mappers
  uow/                  — Unit of Work (PostgreSQL транзакции)
  workers/              — NATS consumer workers
  contentbuilders/      — генерация email + токенов в Redis
mocks/                  — сгенерированные моки
migrations/             — Goose SQL миграции
build/                  — Docker конфигурации
api/                    — OpenAPI/Swagger spec
scripts/                — Taskfile, скрипты PostgreSQL
```

## Ключевые технологии

- **PostgreSQL** — основное хранилище (squirrel query builder)
- **Redis** — кэш, rate limiting (3 попытки/3 мин), одноразовые токены (TTL 15 мин)
- **NATS** — асинхронные уведомления (verify-email, forget-password)
- **OpenTelemetry + Jaeger** — распределённая трассировка
- **gorilla/mux + gorilla/websocket** — HTTP роутер + WebSocket
- **JWT** — access/refresh токены в cookies
- **gomail** — SMTP отправка

## Паттерны

- Все интерфейсы в `internal/interfaces/` с mockgen директивами
- UoW для транзакций (repositories вызываются внутри `uow.Do()`)
- Trace decorators на каждом слое (OpenTelemetry spans)
- Cache decorator в usecases/auth для rate limiting и token validation
- Factory functions для repositories (принимают `pg.Transaction`)
- WebSocket: sync.Map[userID → *websocket.Conn], fan-out сообщений

## Тестирование

- Unit-тесты с mockgen моками
- Trace decorator тесты
- Domain validation тесты
- `coverage/` — отчёты покрытия
