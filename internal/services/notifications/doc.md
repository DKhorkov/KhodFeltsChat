# services/notifications — Сервис уведомлений

## Назначение

Тонкий фасад над репозиториями уведомлений. Не содержит бизнес-логики.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SendVerifyEmailMessage(user)` | Делегирует в emailsRepository |
| `SendForgetPasswordMessage(user)` | Делегирует в emailsRepository |
| `SendNewMessageByEmail(recipient, message, chat)` | Делегирует в emailsRepository |
| `SendNewMessageByWebPush(subscription, message, unreadCount)` | Делегирует в webPushRepository. `unreadCount` — текущее число непрочитанных сообщений пользователя, кладётся в push payload для service worker'а |

## Зависимости

- `EmailsRepository` — SMTP отправка через gomail
- `WebPushRepository` — отправка Web Push уведомлений через VAPID
