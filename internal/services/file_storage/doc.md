# file_storage (service)

Тонкая обёртка над `FileStorageRepository` для работы с файлами.

## Структуры

- **Service** — основная реализация. Делегирует вызовы `FileStorageRepository`.
- **TraceDecorator** — обёртка для трассировки (OpenTelemetry) над `FileStorageService`.

## Методы

- `Upload(ctx, path, data io.Reader) error` — загружает файл через репозиторий.
- `Download(ctx, path) ([]byte, error)` — скачивает файл через репозиторий.
- `Delete(ctx, path) error` — удаляет файл через репозиторий.
