# services/notifications — Сервис уведомлений

## Назначение

Тонкий фасад над `EmailsRepository`. Не содержит бизнес-логики.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SendVerifyEmailMessage(user)` | Делегирует в emailsRepository |
| `SendForgetPasswordMessage(user)` | Делегирует в emailsRepository |

## Зависимости

- `EmailsRepository` — SMTP отправка через gomail
