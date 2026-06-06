# services/users — Сервис пользователей

## Назначение

CRUD-сервис для пользователей, обёрнутый в Unit of Work транзакции.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `GetUserByID` | Поиск по ID, обёртка ошибки в `ErrUserNotFound` |
| `GetUserByEmail` | Поиск по email |
| `GetUserByUsername` | Поиск по username |
| `GetUsers` | Список с фильтрами (username) и пагинацией |
| `UpdateUser` | Обновление + re-fetch для возврата актуальных данных; поддерживает поле `AvatarPath` — пустая строка означает сброс аватара (очистку пути) |

## Зависимости

- Factory function для `UsersRepository`
- `UnitOfWork`
