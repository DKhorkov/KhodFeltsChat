# PWA Badge Counter — Design

## Цель

Показывать счётчик непрочитанных сообщений на иконке PWA на главном экране (iOS 16.4+, Android Chrome) и в табе/доке десктопа через [Badging API](https://developer.mozilla.org/en-US/docs/Web/API/Badging_API). Без этого на iOS юзер вообще не видит, что прилетело новое сообщение, если приложение свёрнуто — push поступает, но цифры на иконке нет.

Дополнительно: в списке чатов вместо абстрактной красной точки показывать **число непрочитанных** для каждого чата (Telegram-стиль).

## Проблема

Сейчас:

- Web Push приходит, `sw.js` показывает уведомление — но `setAppBadge` нигде не вызывается, бейдж на иконке PWA не появляется и не уходит.
- На iOS бейдж = единственный пассивный сигнал «что-то прилетело», потому что WebSocket в фоне убит. На Android системный лаунчер сам ставит точку, но числа не показывает.
- На фронте есть бинарный `chat.isRead`, а количество непрочитанных нигде не считается.

## Принципы решения

**Единственный источник истины — БД.** Бейдж = `COUNT(*) FROM messages_statuses WHERE user_id=X AND is_read=false AND is_deleted=false`. Никаких локальных «было N стало N+1»: каждый раз клиент/SW ставит абсолютное число, пришедшее с сервера. Любые рассинхроны (между устройствами, при потере WS, при пропуске push'а) лечатся одним свежим запросом.

**`loadChats` — единственная точка обновления бейджа на клиенте.** В `chat.js` уже есть `loadChats()` + `debouncedLoadChats()`, который дёргается после каждого значимого события (WS new/delete/edit, отправка сообщения, открытие чата, polling каждые 5 сек). Если зашить вызов `setAppBadge` внутрь `loadChats`, бейдж синхронизируется во всех уже существующих местах бесплатно.

**В push payload — абсолютное число.** Worker до отправки push'а считает `unreadCount` для получателя и кладёт его в payload. `sw.js` в обработчике `push` вызывает `setAppBadge(unreadCount)`.

**Прочтение — там же, где уже есть.** `GET /api/chats/{id}/messages` уже помечает прочитанным через `ChangeMessagesIsReadStatus`. Отдельный endpoint не нужен.

## Источники счётчика

Два разных места, оба бьются друг в друга на одной и той же БД-агрегации:

| Источник | Кто считает | Где используется | Поле / payload |
|---|---|---|---|
| Per-chat `unreadCount` | Скалярный подзапрос в `GetUserChats` | `loadChats` на клиенте: сумма по чатам → `setAppBadge`. Также рендерится число на каждом элементе списка чатов. | `Chat.UnreadCount uint64` |
| Total `unreadCount` юзера | `MessagesRepository.GetUserUnreadCount(userID)` | NATS worker push'а до цикла по подпискам | поле `unreadCount` в JSON push payload |

Они вычисляют одно и то же абсолютное число для одного и того же состояния БД, но через разные запросы. Дубль оправдан: per-chat нужен для UI, total — для одного скаляра в push payload без загрузки списка чатов.

## Изменения в модели

### Домен `domains.Chat`

Добавить поле:

```go
type Chat struct {
    ID          uint64    `json:"id"`
    // ...
    IsRead      bool      `json:"isRead"`      // оставить — фронт использует для bold-заголовка
    UnreadCount uint64    `json:"unreadCount"` // новое
    // ...
}
```

`IsRead` оставляем — `chat.js` использует его для жирного заголовка непрочитанного чата. Семантически: `IsRead == (UnreadCount == 0)`. Можно было бы выпилить и заменить на сравнение на клиенте, но это не цель этой ветки.

## Изменения в SQL

### `chats.GetUserChats` — добавить колонку `unread_count`

В существующий запрос рядом со скалярным подзапросом `NOT EXISTS(...) AS is_read` добавить второй скалярный подзапрос:

```sql
(SELECT COUNT(*)
 FROM messages_statuses
 INNER JOIN messages ON messages_statuses.message_id = messages.id
 WHERE messages.chat_id = chats.id
   AND messages_statuses.user_id = $1
   AND messages_statuses.is_read = false
   AND messages_statuses.is_deleted = false
) AS unread_count
```

Без алиасов на таблицы. INNER JOIN внутри подзапроса корректен (для каждого сообщения есть строка статуса для каждого участника). Внешнего LEFT/INNER не нужно — скалярный подзапрос для чатов без непрочитанных вернёт 0.

### `messages.GetUserUnreadCount` — новый метод

```sql
SELECT COUNT(*)
FROM messages_statuses
WHERE user_id = $1
  AND is_read = false
  AND is_deleted = false
```

Возвращает `uint64`.

## Изменения в интерфейсах

### `MessagesRepository`

```go
GetUserUnreadCount(ctx context.Context, userID uint64) (uint64, error)
```

### `MessagesService` (и автоматически `MessagesUseCases`)

```go
GetUserUnreadCount(ctx context.Context, userID uint64) (uint64, error)
```

### `NotificationsService`

```go
SendNewMessageByWebPush(
    ctx context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64, // новое
) error
```

### `WebPushRepository`

```go
SendNotification(
    ctx context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64, // новое
) error
```

## Поток отправки push'а

```
NATS worker (builder.go)
  └─ notificationsUseCases.SendNewMessageByWebPush(ctx, userID, payload)
      ├─ message := messagesService.GetMessageByID(payload.MessageID)
      ├─ unreadCount := messagesService.GetUserUnreadCount(userID)   ← новое (один раз перед циклом)
      ├─ subs := webPushSubscriptionsService.GetWebPushSubscriptionsByUserID(userID)
      └─ for sub in subs:
           notificationsService.SendNewMessageByWebPush(ctx, sub, message, unreadCount)
             └─ webPushRepository.SendNotification(ctx, sub, message, unreadCount)
                  └─ payload["unreadCount"] = unreadCount  ← новое поле в JSON
```

## Изменения в JSON push payload

`repositories/web_push/repository.go:37`:

```go
payload, err := json.Marshal(map[string]any{
    "title":       message.Sender.Username,
    "body":        message.Text,
    "chatId":      message.ChatID,
    "timestamp":   message.CreatedAt.UnixMilli(),
    "unreadCount": unreadCount, // новое
})
```

## Изменения на фронте

### `sw.js`

В обработчике `push` внутри `event.waitUntil` вызывать `setAppBadge`:

```js
if ('setAppBadge' in self.navigator && typeof data.unreadCount === 'number') {
    await self.navigator.setAppBadge(data.unreadCount);
}
```

Важно: `setAppBadge` нужно вызывать **внутри `event.waitUntil`**, иначе iOS может прибить SW до резолва промиса.

### `chat.js`

`loadChats` после рендера списка вызывает `updateAppBadge(chats)`:

```js
function updateAppBadge(chats) {
    if (!('setAppBadge' in navigator)) return;
    const total = chats.reduce((sum, c) => sum + (c.unreadCount || 0), 0);
    total > 0 ? navigator.setAppBadge(total) : navigator.clearAppBadge();
}
```

Это покрывает все случаи:
- Открытие приложения → `loadChats` в `DOMContentLoaded` → бейдж синхронизируется с реальным состоянием.
- WS `new_message`/`message_deleted`/`message_edited` → `debouncedLoadChats` → бейдж обновляется.
- Отправка сообщения → `loadChats` → бейдж обновляется (для самого отправителя счётчик не меняется, его статус `is_read=true`).
- Открытие чата → `loadMessages` → бэк помечает прочитанным → `debouncedLoadChats` → бейдж уменьшается.
- Polling каждые 5 сек → ленивая синхронизация на случай, если что-то проскочило.

### Список чатов — число вместо точки

Сейчас в `chat.js` рисуется `.chat-item__unread-dot` (просто кружок). Заменить на число:

```js
if (chat.unreadCount > 0) {
    const badge = document.createElement('div');
    badge.className = 'chat-item__unread-badge';
    badge.textContent = chat.unreadCount > 99 ? '99+' : String(chat.unreadCount);
    item.appendChild(badge);
}
```

CSS: вместо `8px` круга — пилюля с цифрой по стилю кнопки scroll-down badge (используем тот же `--accent`).

## Поведение

| Сценарий | Что происходит с бейджем |
|---|---|
| PWA закрыта, прилетел push | SW в push-хендлере: `setAppBadge(unreadCount)` из payload. |
| PWA открыта (десктоп), приходит сообщение в другом чате | Push подавлен (sw.js уже подавляет). WS триггерит `loadChats` → `updateAppBadge`. |
| PWA открыта (iOS), приходит сообщение | Push НЕ подавлен (на iOS нельзя) + WS триггерит `loadChats`. Оба ставят одно и то же число. |
| Юзер открыл приложение, но в чаты не зашёл | Бейдж НЕ обнуляется. Сохраняется реальное число непрочитанных. |
| Юзер открыл чат с непрочитанными | `GET /api/chats/{id}/messages` помечает прочитанным → `loadChats` → бейдж уменьшается на число прочитанного. |
| Юзер прочитал на другом устройстве | На этом устройстве бейдж обновится при следующем `loadChats` (push или WS) либо через 5-сек polling. |
| Сообщение удалили «у всех», оно было непрочитанным | `is_deleted=true` в статусе → не считается → бейдж уменьшится при следующем `loadChats`. |

## Ограничения

- Badging API на iOS работает **только в PWA, добавленной на homescreen**, iOS 16.4+. В обычной вкладке Safari — нет.
- `setAppBadge` возвращает Promise — обязательно `await` внутри `event.waitUntil`.
- Подписки per-device, но счётчик per-user — так и должно быть, юзеру важно общее число непрочитанных на всех устройствах.

## Что НЕ делается в этой ветке

- Нет нового endpoint'а `GET /api/users/me/unread-count` — счётчик и так доступен через `/api/chats` (сумма `unreadCount`).
- Нет нового endpoint'а `POST /api/chats/{id}/read` — прочтение уже происходит через `GET /api/chats/{id}/messages`.
- Нет WS-события `chat_read` от других устройств — синхронизация идёт через 5-сек polling + push с актуальным числом.
- GUI (`KhodFeltsChatGUI`) — не трогается, это PWA-фича.

## Совместимость

Поле `Chat.UnreadCount` — добавление, фронт ничего не сломает. Старый GUI клиент проигнорирует новое поле.

Сигнатуры `SendNewMessageByWebPush` и `WebPushRepository.SendNotification` ломаются — потребуется обновить моки и тесты. Все вызывающие места в монорепо одни.
