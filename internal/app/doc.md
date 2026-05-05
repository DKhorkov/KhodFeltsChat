# Пакет internal/app

## Назначение

Запускает приложение и обеспечивает graceful shutdown.

## Типы

### App

```go
type App struct {
    controller interfaces.Controller
}
```

Обёртка над `interfaces.Controller`. Создаётся через `New(controller interfaces.Controller) *App`.

## Методы

### Run()

- Запускает `controller.Run()` в отдельной горутине.
- Блокирует основную горутину через канал сигналов ОС.
- При получении `SIGINT` или `SIGTERM` вызывает `controller.Stop()` и завершает процесс.

## Зависимости

- `github.com/DKhorkov/kfc/internal/interfaces` — интерфейс `Controller`.
- Стандартная библиотека: `os/signal`, `syscall`.
