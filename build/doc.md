# build — Docker конфигурации

## Структура

| Файл/Директория | Описание |
|----------------|----------|
| `package/Dockerfile` | Образ основного приложения |
| `package/backup_cron_entrypoint.sh` | Entrypoint для cron бэкапов PostgreSQL |
| `package/local/docker-compose.yml` | Локальный dev стек (PostgreSQL, Redis, NATS, Jaeger, Prometheus, Grafana) |
| `package/prod/docker-compose.yml` | Production стек (всё + само приложение в контейнере) |

## Использование

```bash
task local   # Поднимает только инфраструктуру, приложение запускается отдельно
task prod    # Поднимает всё включая приложение
```
