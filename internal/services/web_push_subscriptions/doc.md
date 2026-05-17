# services/web_push_subscriptions — Сервис push-подписок

## Назначение

Управляет push-подписками пользователей через `WebPushSubscriptionsRepository` в рамках UoW-транзакции.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `CreateWebPushSubscription` | Создание подписки + возврат созданной записи. При отсутствии — `ErrWebPushSubscriptionNotFound` |
| `GetWebPushSubscriptionsByUserID` | Получение всех подписок пользователя по UserID |
| `DeleteWebPushSubscription` | Удаление подписки по ID |

## Зависимости

- Factory function для `WebPushSubscriptionsRepository`
- `UnitOfWork`
