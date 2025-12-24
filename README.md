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


