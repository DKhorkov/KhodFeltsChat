# services/push_subscriptions — Сервис push-подписок

## Назначение

Управляет push-подписками пользователей через `PushSubscriptionsRepository` в рамках UoW-транзакции.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `CreatePushSubscription` | Создание подписки + возврат созданной записи. При отсутствии — `ErrPushSubscriptionNotFound` |
| `GetPushSubscriptionsByUserID` | Получение всех подписок пользователя по UserID |
| `DeletePushSubscription` | Удаление подписки по ID |
| `DeletePushSubscriptionByEndpoint` | Удаление подписки по endpoint URL |

## Зависимости

- Factory function для `PushSubscriptionsRepository`
- `UnitOfWork`
