# repositories/push_subscriptions

## Назначение

Репозиторий для работы с push-подписками пользователей. Обслуживает операции
создания, получения и удаления подписок на push-уведомления.

## Таблица

- `push_subscriptions` — подписки на push-уведомления (endpoint, encryption_key, auth), связана с `users` через `user_id`

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `CreatePushSubscription(ctx, subscription)` | Вставляет запись подписки, возвращает ID |
| `GetPushSubscriptionsByUserID(ctx, userID)` | Возвращает все подписки пользователя по его ID |
| `DeletePushSubscription(ctx, id)` | Удаляет подписку по ID |

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/kfc/internal/domains` — тип `PushSubscription`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
