# Пакет handlers/api/auth

## Назначение

Группирующая директория для обработчиков аутентификации и управления аккаунтом.

## Подпакеты

| Пакет | Метод | Путь | Описание |
|-------|-------|------|----------|
| `login` | POST | `/api/sessions` | Вход — устанавливает access/refresh токены в cookies |
| `logout` | DELETE | `/api/sessions` | Выход из текущей сессии |
| `logout_all` | DELETE | `/api/sessions/all` | Выход из всех сессий |
| `refresh_tokens` | PUT | `/api/sessions` | Обновление access/refresh токенов |
| `register` | POST | `/api/users` | Регистрация нового пользователя |
| `verify_email` | GET | `/api/users/email/verify/{token}` | Подтверждение email по токену |
| `send_verify_email_message` | POST | `/api/users/email/verify` | Отправка письма для подтверждения email |
| `change_password` | POST | `/api/users/password/change` | Смена пароля (требуется старый пароль) |
| `forget_password` | POST | `/api/users/password/forget/{token}` | Сброс пароля по токену из письма |
| `send_forget_password_message` | POST | `/api/users/password/forget` | Отправка письма для сброса пароля |

## Зависимости

Все подпакеты используют `interfaces.AuthUseCases`. `login`, `logout`, `logout_all`, `refresh_tokens` дополнительно принимают `config.CookiesConfig`.
