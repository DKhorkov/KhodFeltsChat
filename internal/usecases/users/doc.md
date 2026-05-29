# Пакет usecases/users

## Назначение

Бизнес-логика для операций с пользователями: получение списков, поиск по ID,
обновление профиля.

## Ключевые операции

### GetUsers
- Принимает необязательный фильтр по `username` и параметры пагинации.
- Делегирует запрос сервису пользователей.
- Возвращает срез `[]domains.User`.

### GetUserByID
- Получает одного пользователя по его ID.
- Возвращает доменный объект `domains.User`.

### UpdateUser
- Валидирует новое имя пользователя через regex.
- Вызывает сервис для применения изменений.

### UpdateAvatar
- Принимает ID пользователя и байты изображения.
- Делегирует сохранение файла в `FileStorageService`.
- Обновляет поле `avatar_path` пользователя через `UsersService`.
- При ошибке сохранения файл не сохраняется и профиль не изменяется.

### DeleteAvatar
- Принимает ID пользователя.
- Удаляет файл аватара через `FileStorageService`.
- Сбрасывает `avatar_path` пользователя (пустая строка) через `UsersService`.

## Зависимости

- `internal/interfaces` — `UsersService`, `FileStorageService`.
- `internal/common` — `DecodeImage`, `ResizeImage`, `ExtractUUIDFromURL` (утилиты для работы с изображениями).
- `internal/domains` — `User`, `Pagination`.
- `internal/errors` — ошибки валидации.
