# uow — Unit of Work

## Назначение

Реализация паттерна Unit of Work для управления PostgreSQL транзакциями.

## Ключевые компоненты

### UoW (`uow.go`)

Оборачивает PostgreSQL коннектор. Метод `Do()`:
1. Начинает транзакцию (`BEGIN`)
2. Запускает переданную функцию в goroutine
3. Обрабатывает три сценария:
   - **Context cancelled** — rollback
   - **Ошибка функции** — rollback (с wrapping ошибки rollback если он тоже упал)
   - **Успех** — commit

Использует buffered channel для предотвращения goroutine leaks при timeout контекста.

### Trace Decorator (`trace_decorator.go`)

Оборачивает `UnitOfWork` OpenTelemetry спаном вокруг каждого вызова `Do()`.

## Использование

```go
err := uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
    repo := repoFactory(tx)
    return repo.SomeOperation(ctx, ...)
})
```

## Зависимости

- `github.com/DKhorkov/libs` — PostgreSQL connector
- OpenTelemetry SDK — трассировка
