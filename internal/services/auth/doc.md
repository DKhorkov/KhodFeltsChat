# services/auth — Сервис авторизации

## Назначение

Оркестрация операций авторизации с использованием Unit of Work и NATS публикации.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `RegisterUser` | Дедупликация по email/username, вставка пользователя, публикация `EmailNotificationDTO` в NATS |
| `CreateRefreshToken` | Создание + немедленный re-fetch токена в одной транзакции |
| `GetRefreshTokenByUserID` | Получение неистёкшего токена |
| `ExpireRefreshToken` | Удаление токена (инвалидация) |
| `VerifyEmail` | Установка `email_confirmed = true` |
| `ForgetPassword` | Обновление пароля + expire refresh token (принуждение к re-login) |
| `ChangePassword` | Обновление пароля без expire токена |
| `SendForgetPasswordMessage` | Поиск user по email → публикация `EmailNotificationDTO` в NATS (subject: `EmailNotification`) |
| `SendVerifyEmailMessage` | Поиск user по email → публикация `EmailNotificationDTO` в NATS (subject: `EmailNotification`) |

## Зависимости

- Factory functions для `AuthRepository` и `UsersRepository` (принимают `pg.Transaction`)
- `UnitOfWork` — управление транзакциями
- NATS Publisher — асинхронная публикация событий уведомлений
