# file_storage (repository)

Реализация `FileStorageRepository` для хранения файлов на диске.

## Структуры

- **Repository** — основная реализация, работает с файловой системой. Принимает `basePath` — корневую директорию хранилища и `logger` — логгер для записи ошибок (например, при закрытии файлов).
- **TraceDecorator** — обёртка для трассировки (OpenTelemetry) над `FileStorageRepository`.

## Методы

- `Upload(ctx, path, data io.Reader) error` — сохраняет файл по указанному пути относительно `basePath`. Создаёт промежуточные директории при необходимости.
- `Download(ctx, path) ([]byte, error)` — читает файл по пути. Возвращает `ErrFileNotFound` если файл не существует.
- `Delete(ctx, path) error` — удаляет файл. Не возвращает ошибку если файл уже не существует.
