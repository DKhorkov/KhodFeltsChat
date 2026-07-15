# Khod Felts Chat

## Usage

Before usage need to create network for correct dependencies work:

```shell
task -d scripts network -v
```

To stop all docker containers,
use next command:

```bash
task -d scripts docker_stop -v
```

To clean up all created dirs and docker containers,
use next command:

```bash
task -d scripts clean_up -v
```

Run via IDE:

```shell
task -d scripts local 
go run cmd/main.go
```

Run via Docker:

```shell
task -d scripts prod 
```

## Prometheus

To see Prometheus metrics open
next [link](http://localhost:9090) in browser.

## Grafana

To see Grafana Dashboard open
next [link](http://localhost:3000) in browser.

Source URL:

```shell
http://prometheus:9090
```

## Tracing

To see tracing open
next [link](http://localhost:16686) in browser.

## Swagger

To see Swagger Documentation open
next [link](http://localhost:8080/docs) in browser.

## Linters

To run linters, use next command:

```shell
 task -d scripts linters -v
```

## Tests

To run test, use next commands. Coverage info will be
recorded to ```coverage``` folder:

```shell
task -d scripts tests -v
```

To include integration tests, add `integration` flag:

```shell
task -d scripts tests integration=true -v
```

## Benchmarks

To run benchmarks, use next command:

```shell
task -d scripts bench -v
```

## Redis

To stop redis server, use next command:

```shell
task -d scripts stop_redis
```

## Database

To connect to database container, use next command:

```shell
task -d scripts connect_to_database
```

To connect to DB inside database container, use next command:

```shell
psql -U $POSTGRES_USER
```

To create backup of database, use next command:

```shell
task -d scripts backup
```

To restore database from latest backup, use next command:

```shell
task -d scripts restore
```

To restore database from specific backup, use next command:

```shell
task -d scripts restore BACKUP_FILENAME={{backup_filename}}
```

## Migrations

To create migration file, use next command:

```shell
task -d scripts makemigrations NAME={{migration name}}
```

To apply all available migrations, use next command:

```shell
task -d scripts migrate
```

To migrate up to a specific version, use next command:

```shell
task -d scripts migrate_to VERSION={{migration version}}
```

To rollback migrations to a specific version, use next command:

```shell
task -d scripts downgrade_to VERSION={{migration version}}
```

To rollback all migrations (careful!), use next command:

```shell
task -d scripts downgrade_to_base
```

To print status of all migrations, use next command:

```shell
task -d scripts migrations_status
```

## Web Push Notifications

To enable push notifications for offline users, generate a VAPID key pair:

```shell
task -d scripts vapid
```

Add the output to `.env` / `.env.local` / `.env.prod`:

```env
VAPID_PUBLIC_KEY="<generated public key>"
VAPID_PRIVATE_KEY="<generated private key>"
VAPID_CONTACT="mailto:admin@kfc.com"
```

## Websockets

To connect via websocket, use next command:
```shell
websocat ws://localhost:8080/ws -H "Cookie: accessToken=<TOKEN_VALUE>"
```

Message structure for sending:
```
{"chatId": 1, "text": "some message for user"}
```

## Server disk cleanup (VPS)

Гайд по диагностике и очистке диска на сервере (например, Timeweb VPS).

### 1. Диагностика

```bash
df -h                                            # общая занятость разделов
df -i                                            # inodes (иногда упирается в них)

# Топ директорий верхнего уровня
sudo du -h --max-depth=1 / 2>/dev/null | sort -hr | head -20

# Углубиться в конкретную папку
sudo du -h --max-depth=1 /var 2>/dev/null | sort -hr | head -10
sudo du -h --max-depth=1 /var/lib 2>/dev/null | sort -hr | head -10
sudo du -h --max-depth=1 /var/log 2>/dev/null | sort -hr | head -10

# Интерактивный анализ (удобнее всего)
sudo apt install ncdu && sudo ncdu /
```

### 2. Docker (самый частый виновник)

```bash
docker system df                                 # общая картина по Docker
docker system df -v                              # детально по каждому объекту
docker volume ls -f dangling=true                # неиспользуемые volumes (сироты)
```

Безопасная чистка (не тронет данные запущенных контейнеров):

```bash
docker container prune -f                        # удалить остановленные контейнеры
docker image prune -a -f                         # удалить неиспользуемые образы
docker builder prune -a -f                       # удалить build-кэш
docker volume prune -f                           # удалить dangling volumes
```

Одной командой:

```bash
docker system prune -a -f                        # всё выше, БЕЗ volumes
docker system prune -a -f --volumes              # ВКЛЮЧАЯ volumes (осторожно с БД!)
```

> **Внимание про volumes:** флаг `--volumes` снесёт все volumes, не подключённые
> к запущенным контейнерам. Если сервисы с БД сейчас остановлены — их данные
> могут быть удалены. В этом проекте данные Postgres/Redis/Grafana/Prometheus
> лежат в bind mounts (`postgres_data`, `redis_data`, `grafana`, `prometheus_data`
> в корне проекта), а не в Docker volumes — так что `docker volume prune`
> безопасен.

### 3. Почему копятся анонимные volumes

Официальные образы (`postgres`, `redis`, `prom/prometheus`, `grafana/grafana`
и т.д.) объявляют директиву `VOLUME` в своих Dockerfile. Если в
`docker-compose.yml` этот путь **не замаплен** ни на bind mount, ни на
именованный volume — при каждом пересоздании контейнера Docker создаёт новый
анонимный volume, а старый остаётся сиротой.

В этом проекте все stateful-пути замаплены на bind mounts в
`build/package/{prod,local}/docker-compose.yml`. Если добавляешь новый сервис
с состоянием — обязательно замапь его data-путь.

### 4. Логи systemd (journald)

```bash
sudo journalctl --disk-usage                     # текущий размер
sudo journalctl --vacuum-time=3d                 # оставить только за 3 дня
sudo journalctl --vacuum-size=500M               # или ограничить размером
```

Закрепить лимит навсегда:

```bash
sudo sed -i 's/#SystemMaxUse=.*/SystemMaxUse=500M/' /etc/systemd/journald.conf
sudo systemctl restart systemd-journald
```

### 5. Go кэш (если на сервере собирается Go)

```bash
go clean -modcache                               # /root/go/pkg/mod
go clean -cache                                  # /root/.cache/go-build
```

### 6. Пакетный менеджер

```bash
# Debian/Ubuntu
sudo apt clean
sudo apt autoremove --purge
```

### 7. Периодическая чистка (cron)

`cron` — встроенный планировщик Linux, в Ubuntu уже установлен и запущен.
Проверить, что работает:

```bash
systemctl status cron
```

Открыть crontab пользователя root (Docker/journald требуют root-прав):

```bash
sudo crontab -e
```

Добавить строку:

```
0 3 1 * * docker system prune -f >> /var/log/docker-cleanup.log 2>&1 && journalctl --vacuum-time=7d >> /var/log/docker-cleanup.log 2>&1
```

**Расшифровка:**

| Часть | Значение |
|-------|----------|
| `0 3 1 * *` | 1-го числа каждого месяца в 03:00 (мин, час, день месяца, месяц, день недели) |
| `docker system prune -f` | Удаляет остановленные контейнеры, неиспользуемые образы, build-кэш и сети. БЕЗ `--volumes` — данные Postgres/Redis/Grafana/Prometheus не тронет |
| `journalctl --vacuum-time=7d` | Оставляет только systemd-логи за последние 7 дней |
| `>> /var/log/docker-cleanup.log 2>&1` | Дописывает stdout и stderr в лог-файл, чтобы можно было проверить что чистка отработала |

**Формат cron-расписания:**

```
┌── минута (0-59)
│ ┌── час (0-23)
│ │ ┌── день месяца (1-31)
│ │ │ ┌── месяц (1-12)
│ │ │ │ ┌── день недели (0-7, 0 и 7 = вс)
│ │ │ │ │
* * * * * команда
```

Примеры: `0 3 * * *` — каждый день в 03:00; `*/15 * * * *` — каждые 15 минут;
`0 */6 * * *` — каждые 6 часов.

**Полезные команды:**

```bash
sudo crontab -l                          # показать задачи
cat /var/log/docker-cleanup.log          # что было почищено и когда
grep CRON /var/log/syslog | tail -20     # факт запуска cron-задач
journalctl -u cron --since "1 hour ago"  # логи сервиса cron
```

### 8. Лимит journald на будущее

Чтобы логи снова не разрослись до нескольких гигабайт, поставь потолок в
конфиге systemd-journald:

```bash
sudo sed -i 's/#SystemMaxUse=.*/SystemMaxUse=500M/' /etc/systemd/journald.conf
sudo systemctl restart systemd-journald
```

После этого journald сам будет держать себя в пределах 500M —
`--vacuum-time=7d` в cron станет подстраховкой.
