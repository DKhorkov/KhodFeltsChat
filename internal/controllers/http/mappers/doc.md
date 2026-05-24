# mappers — Маппинг domain → schema

## Назначение

Тонкие конвертеры доменных моделей в HTTP schema структуры. Без бизнес-логики.

## Содержимое

| Пакет | Функции |
|-------|---------|
| `chats/` | `MapChat`, `MapChats` — конвертация Chat (делегирует members в users mapper, messages в messages mapper) |
| `messages/` | `MapMessage`, `MapMessages` — конвертация Message (включая `ReplyToMessage` → `ReplyMessage`) |
| `settings/` | `MapSettings` — конвертация Settings (включая `EmailConsents`, `WebPushConsents`) |
| `users/` | `MapUser`, `MapUsers` — конвертация User |
| `web_push_subscriptions/` | `MapCreateResponse` — конвертация WebPushSubscription → CreateWebPushSubscriptionResponse (возвращает ID созданной подписки) |
