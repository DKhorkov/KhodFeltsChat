# mappers — Маппинг domain → schema

## Назначение

Тонкие конвертеры доменных моделей в HTTP schema структуры. Без бизнес-логики.

## Содержимое

| Пакет | Функции |
|-------|---------|
| `chats/` | `MapChat`, `MapChats` — конвертация Chat (делегирует members в users mapper, messages в messages mapper) |
| `messages/` | `MapMessage`, `MapMessages` — конвертация Message |
| `users/` | `MapUser`, `MapUsers` — конвертация User |
