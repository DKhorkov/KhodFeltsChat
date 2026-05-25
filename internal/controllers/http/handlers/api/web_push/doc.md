# Пакет handlers/api/web_push

## Назначение

Группирующая директория для обработчиков Web Push подписок.

## Подпакеты

| Пакет | Метод | Путь | Описание |
|-------|-------|------|----------|
| `subscribe` | POST | `/api/web-push/subscribe` | Создание web-push подписки |
| `unsubscribe` | DELETE | `/api/web-push/subscriptions/{id}` | Удаление web-push подписки |
| `vapid_key` | GET | `/api/web-push/vapid-key` | Получение VAPID публичного ключа |

## Зависимости

`subscribe` и `unsubscribe` используют `interfaces.WebPushSubscriptionsUseCases`. `vapid_key` принимает строку `vapidPublicKey`.
