# scripts — Скрипты и таски

## Содержимое

| Файл/Директория | Описание |
|----------------|----------|
| `Taskfile.yml` | Task runner — команды сборки, запуска, тестирования, линтинга, миграций |
| `postgres/` | Вспомогательные скрипты для PostgreSQL (бэкап, восстановление) |

## Основные таски

```bash
task local        # Поднять инфраструктуру для локальной разработки
task prod         # Production деплой
task lint         # golangci-lint
task test         # go test ./...
task migrate-up   # Применить миграции
task migrate-down # Откатить миграцию
```
