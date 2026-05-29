# User Avatar — Design Spec

## Overview

Добавление возможности загрузки, обновления, удаления и отображения аватара пользователя. Файлы хранятся на диске сервера, обработка (ресайз + конвертация) выполняется при загрузке. Аватар путешествует вместе с объектом пользователя как `avatarPath`.

## Домен и БД

### Миграция

Новая колонка в таблице `users`:

```sql
ALTER TABLE users ADD COLUMN avatar_path TEXT NULL;
```

### Домен User

```go
type User struct {
    ...existing fields...
    AvatarPath *string `json:"avatarPath"`
}
```

`AvatarPath` хранит полный URL вида `https://kfc.webtm.ru/api/files/download/{uuid}.jpg`. Nullable — `nil` означает отсутствие аватара.

`UpdateUserDTO` не меняется — аватар обновляется через отдельный эндпоинт, а `avatar_path` в БД обновляется внутри `FileStorageService`.

### Изменение UsersRepository

`UpdateUser` принимает `domains.User` вместо `domains.UpdateUserDTO`:

```go
UpdateUser(ctx context.Context, user domains.User) error
```

`UsersService.UpdateUser` маппит `UpdateUserDTO` → загружает текущего `User` → патчит нужные поля → вызывает `repository.UpdateUser(ctx, patchedUser)`.

Это позволяет `FileStorageService` переиспользовать тот же репозиторий для обновления `avatar_path`.

## FileStorage — интерфейсы

### FileStorageRepository

Работа с файловой системой (диск):

```go
type FileStorageRepository interface {
    Upload(ctx context.Context, path string, data io.Reader) error
    Download(ctx context.Context, path string) ([]byte, error)
    Delete(ctx context.Context, path string) error
}
```

### FileStorageService

Тонкая обёртка над репозиторием + обновление пользователя. Получает `FileStorageRepository` и `newUsersRepositoryFunc` как зависимости:

```go
type FileStorageService interface {
    Upload(ctx context.Context, path string, data io.Reader) error
    Download(ctx context.Context, path string) ([]byte, error)
    Delete(ctx context.Context, path string) error
}
```

После `Upload` — загружает пользователя по ID, ставит `AvatarPath`, вызывает `repository.UpdateUser`.
После `Delete` — обнуляет `AvatarPath`.

### UsersUseCases

Новые методы:

```go
type UsersUseCases interface {
    ...existing methods...
    UpdateAvatar(ctx context.Context, userID uint64, data io.Reader) (string, error)  // returns avatarPath URL
    DeleteAvatar(ctx context.Context, userID uint64) error
}
```

**Поток `UpdateAvatar`:**
1. Валидация формата (JPEG/PNG/WebP/GIF)
2. Декодирование → ресайз 256x256 → кодирование в JPEG quality 85
3. Генерация UUID для имени файла
4. Если у пользователя уже есть аватар — удалить старый файл через `fileStorageService.Delete`
5. `fileStorageService.Upload(ctx, path, processedData)` — сохраняет файл + обновляет `avatar_path` в БД
6. Возврат URL (`BaseDownloadURL + "/" + uuid + ".jpg"`)

**Поток `DeleteAvatar`:**
1. Получение текущего пользователя
2. Если `avatarPath` == nil → early return (204)
3. Извлечение UUID из `avatarPath`
4. `fileStorageService.Delete(ctx, path)` — удаляет файл + обнуляет `avatar_path` в БД

## Обработка изображений

- **Входные форматы:** JPEG, PNG, WebP, GIF
- **GIF:** берётся только первый кадр
- **Выходной формат:** JPEG 256x256, quality 85
- **Лимит входящего файла:** 20 МБ (отсекает неадекватные файлы, при этом пропускает фотки с iPhone в полном разрешении)
- Браузер на iPhone при `<input type="file" accept="image/*">` автоматически конвертирует HEIC в JPEG

## HTTP эндпоинты

| Метод | URL | Auth | Описание |
|-------|-----|------|----------|
| `PUT` | `/api/users/me/avatar` | да | Загрузка/обновление аватара. `multipart/form-data`, поле `avatar`. Возвращает URL аватара. |
| `DELETE` | `/api/users/me/avatar` | да | Удаление аватара. |
| `GET` | `/api/files/download/{uuid}` | нет | Скачивание файла по UUID. Отдаёт JPEG с `Content-Type: image/jpeg`. |

`GET /api/files/download/{uuid}` добавляется в `IgnoreURL` для auth middleware (публичный).

### Обработка ошибок

**PUT /api/users/me/avatar:**
- Файл не передан → 400
- Невалидный формат → 400
- Размер > 20 МБ → 400
- Ошибка декодирования/ресайза → 500
- Пользователь не найден → 404

**DELETE /api/users/me/avatar:**
- Аватара нет → 204 (idempotent)
- Пользователь не найден → 404

**GET /api/files/download/{uuid}:**
- Файл не найден → 404

## Конфигурация

```go
type FileStorageConfig struct {
    BasePath        string // директория хранения файлов, e.g. "uploads"
    BaseUploadURL   string // e.g. "https://kfc.webtm.ru/api/files/upload"
    BaseDownloadURL string // e.g. "https://kfc.webtm.ru/api/files/download"
    MaxSize         int64  // макс. размер входящего файла в байтах (20 МБ)
}
```

`BaseUploadURL` заложен на будущее для универсального эндпоинта загрузки файлов.

## Структура файлов на диске

```
uploads/
  {uuid1}.jpg
  {uuid2}.jpg
  ...
```

Плоская структура. При обновлении аватара — старый файл удаляется, новый создаётся с новым UUID.

## Фронтенд

### Рендеринг аватара (единая логика)

Применяется во всех местах: chat list, navbar, member profile, group chat members, search users.

- `user.avatarPath` есть → `<img src="user.avatarPath">`, `onerror` → fallback на инициал
- `user.avatarPath` нет → div с инициалом (как сейчас)

### Свой профиль (navbar modal)

Аватар-div кликабельный. При клике — контекстное меню:
- **«Изменить фото»** → открывает `<input type="file" accept="image/*">` → `PUT /api/users/me/avatar`
- **«Удалить фото»** → `DELETE /api/users/me/avatar` (показывается только если аватар есть)

Если аватара нет — контекстное меню только «Изменить фото».

### Профиль другого пользователя (member profile modal)

- Клик на аватар в чат-листе или шапке открытого чата → открывается модалка профиля (как сейчас)
- В модалке профиля клик на аватар (если есть картинка) → увеличенный просмотр аватара (оверлей) + кнопка закрыть
- Клик вне увеличенного аватара → возврат к обычному размеру
- Если аватара нет (fallback с инициалом) → клик ничего не делает

## Swagger

Добавить тег `files` в `scripts/Taskfile.yml` (после строки 356):
```yaml
- name: files
  description: "Операции с файлами"
```

## Тестирование

### Новые unit-тесты
- `FileStorageRepository` — upload/download/delete файлов (через temp dir)
- `FileStorageService` — вызовы репозитория + обновление пользователя (mockgen)
- `UsersUseCases.UpdateAvatar/DeleteAvatar` — валидация формата, вызов сервиса, ошибки (mockgen)
- Хэндлеры — multipart/form-data парсинг, коды ответов
- Domain validation — `AvatarPath` nullable поведение
- Trace decorator тесты для новых методов FileStorage

## Документация

- Обновить `doc.md` в каждой затронутой директории (согласно правилу проекта)
- Обновить `docs/architecture.md` — добавить FileStorage слой
- Обновить `docs/modules.md` — добавить новые пакеты (file_storage repository/service, хэндлеры avatars и files)
- Обновить OpenAPI/Swagger spec — новые эндпоинты и тег `files`

### Обновление существующих тестов
- Все тесты, затронутые изменением `UsersRepository.UpdateUser` (принимает `domains.User` вместо `UpdateUserDTO`)
- Тесты маппинга User (добавлено поле `AvatarPath`)
- Тесты хэндлеров, возвращающих User (новое поле в ответе)
