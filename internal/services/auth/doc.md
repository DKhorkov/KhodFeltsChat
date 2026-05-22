# services/auth — Сервис авторизации

## Назначение

Оркестрация операций авторизации с использованием Unit of Work и NATS публикации.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `RegisterUser` | Дедупликация по email/username, вставка пользователя, публикация `EmailNotificationDTO` в NATS |
| `CreateRefreshToken` | Создание + немедленный re-fetch токена в одной транзакции |
| `GetRefreshTokenByValue` | Получение неистёкшего токена по значению |
| `ExpireRefreshToken` | Удаление токена (инвалидация) |
| `ExpireAllUserRefreshTokens` | Удаление всех токенов пользователя (logout из всех сессий) |
| `VerifyEmail` | Установка `email_confirmed = true` |
| `ForgetPassword` | Обновление пароля + expire всех refresh-токенов (принуждение к re-login на всех устройствах) |
| `ChangePassword` | Обновление пароля без expire токена |
| `SendForgetPasswordMessage` | Поиск user по email → публикация `EmailNotificationDTO` в NATS (subject: `EmailNotification`) |
| `SendVerifyEmailMessage` | Поиск user по email → публикация `EmailNotificationDTO` в NATS (subject: `EmailNotification`) |

## Зависимости

- Factory functions для `AuthRepository` и `UsersRepository` (принимают `pg.Transaction`)
- `UnitOfWork` — управление транзакциями
- NATS Publisher — асинхронная публикация событий уведомлений
