# Пакет handlers/api/users

## Назначение

Группирующая директория для обработчиков пользователей.

## Подпакеты

| Пакет | Метод | Путь | Описание |
|-------|-------|------|----------|
| `me` | GET | `/api/users/me` | Текущий авторизованный пользователь |
| `update` | PUT | `/api/users/me` | Обновление текущего пользователя |
| `user_by_id` | GET | `/api/users/{id}` | Пользователь по ID |
| `users` | GET | `/api/users` | Список пользователей с фильтрацией и пагинацией |
| `update_avatar` | PUT | `/api/users/me/avatar` | Загрузка/обновление аватара (multipart/form-data, resize 256×256 JPEG) |
| `delete_avatar` | DELETE | `/api/users/me/avatar` | Удаление аватара текущего пользователя |

## Зависимости

Все подпакеты используют `interfaces.UsersUseCases`.
