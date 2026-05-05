# repositories/users

## Назначение

Репозиторий для CRUD-операций над пользователями. Работает исключительно
с таблицей `users`.

## Таблицы

- `users` — основная таблица пользователей

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `GetUserByID(ctx, id)` | Поиск пользователя по первичному ключу |
| `GetUserByUsername(ctx, username)` | Поиск пользователя по имени (точное совпадение) |
| `GetUserByEmail(ctx, email)` | Поиск пользователя по email (точное совпадение) |
| `GetUsers(ctx, filters, pagination)` | Список пользователей с постраничной выдачей; если передан `filters.Username`, применяется фильтр `ILIKE` |
| `UpdateUser(ctx, UpdateUserDTO)` | Обновляет изменяемые поля пользователя (`username`, `updated_at`) |

### GetUsers — детали

- Сортировка: `id DESC`.
- Фильтрация: `ILIKE '%<username>%'` по полю `username` (регистронезависимо).
- Пагинация: `LIMIT` / `OFFSET` применяются только при наличии соответствующих полей в `Pagination`.

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/libs/logging` — логирование ошибок закрытия rows
- `github.com/DKhorkov/kfc/internal/domains` — типы `User`, `UpdateUserDTO`, `UsersFilters`, `Pagination`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
