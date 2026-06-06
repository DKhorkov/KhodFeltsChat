# Пакет usecases/file_storage

## Назначение

Юзкейсы для работы с файловым хранилищем: загрузка, скачивание и удаление файлов.
Оборачивает `FileStorageService` и предоставляет единый интерфейс `FileStorageUseCases`.

## Структуры

- **UseCases** — основная реализация, делегирует все операции в `FileStorageService`.
- **TraceDecorator** — обёртка для трассировки (OpenTelemetry) над `FileStorageUseCases`.

## Методы

### Upload
- Загружает файл по указанному пути, принимает `io.Reader` с данными.
- Делегирует вызов в `FileStorageService.Upload`.
- Возвращает URL для скачивания файла (`BaseDownloadURL + "/" + path`).

### Download
- Скачивает файл по пути, возвращает `[]byte`.
- Делегирует вызов в `FileStorageService.Download`.

### Delete
- Удаляет файл по указанному пути.
- Делегирует вызов в `FileStorageService.Delete`.

## Зависимости

- `internal/interfaces` — `FileStorageService`, `FileStorageUseCases`.
- `internal/config` — `FileStorageConfig` (для формирования URL скачивания).
- `github.com/DKhorkov/libs/tracing` — OpenTelemetry трассировка (TraceDecorator).
