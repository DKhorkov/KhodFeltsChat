# Reply to Message + Context Menu + Delete Message

## Summary

Добавление функциональности "ответить на сообщение", контекстного меню сообщений (ответить, копировать, удалить) и удаления сообщений (для себя / для всех). Затрагивает бэкенд (KhodFeltsChat), веб-клиент (chat.js) и десктоп GUI (KhodFeltsChatGUI).

---

## 1. База данных

### 1.1 Миграция: `reply_to_message_id` в таблице `messages`

```sql
ALTER TABLE messages
ADD COLUMN reply_to_message_id INTEGER REFERENCES messages(id) ON DELETE SET NULL;
```

- `NULL` — сообщение не является ответом.
- `ON DELETE SET NULL` — если оригинальное сообщение удалено hard delete ("для всех"), ответ сохраняется, но теряет ссылку.

### 1.2 Миграция: `is_deleted` в таблице `messages_statuses`

```sql
ALTER TABLE messages_statuses
ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;
```

Персональный флаг soft delete. При удалении "для себя" — ставится `is_deleted = true` для одного пользователя. При удалении "для всех" — ставится `is_deleted = true` для всех участников.

### 1.3 Индекс

```sql
CREATE INDEX messages_reply_to_message_id_idx ON messages (reply_to_message_id);
```

---

## 2. Домены

### 2.1 `domains.Message` — добавление поля `ReplyToMessage`

Файл: `internal/domains/message.go`

```go
type Message struct {
    ID             uint64    `json:"id"`
    ChatID         uint64    `json:"chatId"`
    Sender         User      `json:"sender"`
    Text           string    `json:"text"`
    CreatedAt      time.Time `json:"createdAt"`
    UpdatedAt      time.Time `json:"updatedAt"`
    IsRead         bool      `json:"isRead"`
    ReplyToMessage *Message  `json:"replyToMessage,omitempty"`
}
```

- `*Message` — указатель, `nil` когда сообщение не является ответом.
- Содержит полные данные оригинального сообщения для отрисовки (ID, текст, отправитель).
- Вложенный `ReplyToMessage` сам не содержит своего `ReplyToMessage` (цепочка глубиной 1).

### 2.2 `domains.WSEvent` — envelope для WebSocket

Новый файл: `internal/domains/ws_event.go`

```go
type WSEventType string

const (
    WSEventNewMessage     WSEventType = "new_message"
    WSEventMessageDeleted WSEventType = "message_deleted"
)

type WSEvent struct {
    Type    WSEventType `json:"type"`
    Payload interface{} `json:"payload"`
}

type MessageDeletedPayload struct {
    MessageID uint64 `json:"messageId"`
    ChatID    uint64 `json:"chatId"`
}
```

### 2.3 `domains.DeleteMessageDTO`

```go
type DeleteMessageDTO struct {
    MessageID uint64 `json:"messageId"`
    UserID    uint64 `json:"-"` // из JWT-контекста, не из JSON
    ForAll    bool   `json:"forAll"`
}
```

HTTP-хендлер десериализует body (получает `ForAll`), затем дополняет `MessageID` из path-параметра и `UserID` из JWT-контекста.


---

## 3. Репозиторий (бэкенд KhodFeltsChat)

### 3.1 `MessagesRepository` — изменения интерфейса

Файл: `internal/interfaces/repositories.go`

Новые методы:

```go
DeleteMessageForUser(ctx context.Context, userID uint64, messageID uint64) error
DeleteMessageForAll(ctx context.Context, messageID uint64) error
```

### 3.2 `SaveMessage` — сохранение `reply_to_message_id`

Файл: `internal/repositories/messages/repository.go`

В `INSERT INTO messages` добавляется колонка `reply_to_message_id` со значением из `message.ReplyToMessage.ID` (или `NULL` если `ReplyToMessage == nil`).

### 3.3 `GetChatMessages` / `GetMessageByID` — LEFT JOIN для reply

Добавляются два LEFT JOIN:

```sql
LEFT JOIN messages AS reply
    ON messages.reply_to_message_id = reply.id
LEFT JOIN users AS reply_sender
    ON reply.sender_id = reply_sender.id
```

LEFT JOIN нужен потому что `reply_to_message_id` может быть NULL (сообщение не ответ), и INNER JOIN отбросил бы все такие строки.

В SELECT добавляются колонки: `reply.id`, `reply.text`, `reply.created_at`, `reply_sender.id`, `reply_sender.username`.

Структура `MessagePg` расширяется nullable-полями для reply:

```go
ReplyToMessageID       *uint64
ReplyToMessageText     *string
ReplyToMessageCreatedAt *time.Time
ReplyToSenderID        *uint64
ReplyToSenderUsername   *string
```

### 3.4 Фильтрация удалённых сообщений

Во все запросы `GetChatMessages` и `GetMessageByID` добавляется условие:

```sql
WHERE messages_statuses.is_deleted = false
```

### 3.5 `DeleteMessageForUser`

```sql
UPDATE messages_statuses
SET is_deleted = true
WHERE message_id = $1 AND user_id = $2
```

### 3.6 `DeleteMessageForAll`

```sql
UPDATE messages_statuses
SET is_deleted = true
WHERE message_id = $1
```

---

## 4. Сервисы (бэкенд)

### 4.1 `MessagesService` — новый метод

Файл: `internal/interfaces/services.go`

```go
DeleteMessage(ctx context.Context, dto domains.DeleteMessageDTO) error
```

Реализация в `internal/services/messages/service.go`:

Сервис отвечает только за вызов репозитория:
- Если `dto.ForAll` — вызывает `DeleteMessageForAll(ctx, dto.MessageID)`.
- Если не `ForAll` — вызывает `DeleteMessageForUser(ctx, dto.UserID, dto.MessageID)`.

### 4.2 `MessagesUseCases` — валидация и orchestration

Файл: `internal/interfaces/usecases.go`

```go
DeleteMessage(ctx context.Context, dto domains.DeleteMessageDTO) error
```

Usecase отвечает за валидацию и бизнес-правила:

1. Проверяет, что сообщение существует.
2. Если `ForAll` — проверяет, что `dto.UserID == message.Sender.ID` (только автор может удалить для всех).
3. Делегирует в `MessagesService.DeleteMessage`.

---

## 5. WebSocket — envelope pattern (бэкенд)

### 5.1 Отправка сообщений (сервер → клиент)

Файл: `internal/controllers/http/handlers/api/ws/ws.go`

**Текущее поведение:** `conn.WriteJSON(messageToSend)` — отправляет голый `schemas.Message`.

**Новое поведение:** оборачивает в `WSEvent`:

```go
event := domains.WSEvent{
    Type:    domains.WSEventNewMessage,
    Payload: messageToSend,
}
conn.WriteJSON(event)
```

### 5.2 Получение сообщений (клиент → сервер)

**Текущее поведение:** `conn.ReadJSON(message)` — читает напрямую в `domains.Message`.

**Новое поведение:** десериализация по-прежнему идёт в `domains.Message` (как сейчас). Клиент добавляет поле `replyToMessageId` в JSON. Для этого в `domains.Message` добавляется вспомогательное поле:

```go
ReplyToMessageID *uint64 `json:"replyToMessageId,omitempty"`
```

Это поле заполняется при десериализации входящего WS-сообщения от клиента. В WS-хендлере (`listen`) перед сохранением преобразуется:

```go
if message.ReplyToMessageID != nil {
    message.ReplyToMessage = &domains.Message{ID: *message.ReplyToMessageID}
}
```

Таким образом не нужна промежуточная структура — десериализация остаётся в домен, как и сейчас.

### 5.3 Удаление — WS-событие

При удалении "для всех" — отправляем всем участникам чата (кроме инициатора):

```go
event := domains.WSEvent{
    Type: domains.WSEventMessageDeleted,
    Payload: domains.MessageDeletedPayload{
        MessageID: messageID,
        ChatID:    chatID,
    },
}
```

При удалении "для себя" — WS-событие не отправляется (локальная операция).

---

## 6. HTTP API — новый endpoint

### 6.1 `DELETE /api/messages/{messageId}`

Файл: `internal/controllers/http/handlers/api/messages/delete/handler.go`

Request body:

```json
{"forAll": true}
```

- Авторизация через JWT cookie.
- `messageId` из path-параметра, `userID` из JWT-контекста, `forAll` из body — собираются в `DeleteMessageDTO`.
- `forAll: true` — только автор сообщения может вызвать.
- `forAll: false` — любой участник чата.
- При `forAll: true` — отправка WS-события `message_deleted` всем участникам чата (chatID берётся из сообщения в usecase).
- Response: `204 No Content`.

---

## 7. HTTP-схемы

### 7.1 `schemas.Message` — добавление reply

Файл: `internal/controllers/http/schemas/messages.go`

```go
type Message struct {
    // ... существующие поля ...
    ReplyToMessage *ReplyMessage `json:"replyToMessage,omitempty"`
}

type ReplyMessage struct {
    ID        uint64    `json:"id"`
    Sender    Sender    `json:"sender"`
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"createdAt"`
}
```

`ReplyMessage` — облегчённая версия `Message`, без вложенного reply (глубина 1).

### 7.2 Маппер

Файл: `internal/controllers/http/mappers/messages/messages.go`

`MapMessage` обновляется: если `message.ReplyToMessage != nil` — маппит в `schemas.ReplyMessage`.

---

## 8. Веб-клиент (chat.js)

### 8.1 WebSocket — dispatch по типу события

```js
ws.onmessage = (event) => {
    const wsEvent = JSON.parse(event.data);
    switch (wsEvent.type) {
        case 'new_message':
            handleNewMessage(wsEvent.payload);
            break;
        case 'message_deleted':
            handleMessageDeleted(wsEvent.payload);
            break;
    }
};
```

`handleNewMessage` — текущая логика из `ws.onmessage` (дедупликация, append bubble, toast, loadChats).

`handleMessageDeleted` — удаляет сообщение из массива `messages` и из DOM по `messageId`.

### 8.2 Отправка сообщений — `replyToMessageId`

```js
ws.send(JSON.stringify({
    chatId: selectedChatId,
    text,
    replyToMessageId: replyingToMessageId || undefined,
}));
```

`replyingToMessageId` — глобальная переменная, устанавливается при выборе "Ответить" в контекстном меню, сбрасывается при отправке или отмене.

### 8.3 `createMessageBubble` — отрисовка reply

Если `message.replyToMessage` существует, перед текстом сообщения вставляется плашка:

```html
<div class="message-bubble__reply" data-reply-id="42">
    <span class="message-bubble__reply-sender">Username</span>
    <span class="message-bubble__reply-text">Обрезанный текст оригинала...</span>
</div>
```

Клик по плашке → скролл к оригинальному сообщению. Если сообщение ещё не подгружено — подгружать порциями вверх (пагинацией), пока не найдётся сообщение с нужным ID, затем скролл к нему.

Текст оригинала обрезается до ~100 символов.

### 8.4 Контекстное меню

Новый элемент DOM — `div.context-menu` с опциями:

- **Ответить** — всегда видно
- **Копировать текст** — всегда видно
- **Удалить** — только для своих сообщений (`message.sender.id === currentUser.id`)

При выборе "Удалить" для собственного сообщения — показывается подменю:
- "Удалить для себя"
- "Удалить для всех"

При выборе "Удалить" для чужого сообщения — пункт не отображается (нельзя удалить чужое для всех, но можно для себя). Уточнение: для чужих сообщений показывается только "Удалить для себя".

**Desktop:** появляется по правому клику на `.message-bubble`, позиционируется у курсора.

**Mobile:** появляется по long tap (touchstart/touchend, ~500ms) на `.message-bubble`, позиционируется по центру экрана или над сообщением.

Закрывается по клику вне меню или по Escape.

### 8.5 Composer — reply preview

При выборе "Ответить" над textarea появляется плашка:

```html
<div class="conversation__reply-preview" id="reply-preview" style="display: none;">
    <div class="conversation__reply-preview-content">
        <span class="conversation__reply-preview-sender"></span>
        <span class="conversation__reply-preview-text"></span>
    </div>
    <button class="conversation__reply-preview-close" aria-label="Отменить ответ">&times;</button>
</div>
```

- Показывается при выборе "Ответить", скрывается при отправке или клике на `×`.
- Устанавливает `replyingToMessageId`.
- Фокус переходит в textarea.

---

## 9. GUI (KhodFeltsChatGUI)

### 9.1 `domains.Message` — синхронизация

Файл: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/domains/message.go`

Добавить `ReplyToMessage *Message` — аналогично бэкенду.

### 9.2 `domains.WSEvent` — новый тип

Новый файл: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/domains/ws_event.go`

```go
type WSEventType string

const (
    WSEventNewMessage     WSEventType = "new_message"
    WSEventMessageDeleted WSEventType = "message_deleted"
)

type WSEvent struct {
    Type    WSEventType     `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

type MessageDeletedPayload struct {
    MessageID uint64 `json:"messageId"`
    ChatID    uint64 `json:"chatId"`
}
```

`json.RawMessage` для payload — чтобы десериализовать в конкретный тип после определения `Type`.

### 9.3 WebSocket Repository — envelope

Файл: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/repositories/ws/repository.go`

**Изменение интерфейса `WebSocketsRepository`:**

```go
type WebSocketsRepository interface {
    Connect(ctx context.Context, accessToken string) error
    Close() error
    ReadEvent(ctx context.Context) (*domains.WSEvent, error)    // было ReadMessage
    WriteMessage(ctx context.Context, message domains.Message) error
}
```

`readLoop` меняется: `ReadJSON(&event)` вместо `ReadJSON(&msg)`, канал меняется на `chan *domains.WSEvent`.

### 9.4 Chat Handler — dispatch по типу

Файл: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/v2/handlers/chat/handler.go`

Новые константы и логика:

```go
const (
    newMessageEventName      = "new_message"
    messageDeletedEventName  = "message_deleted"
)
```

В `readMessages()`:

```go
event, err := h.useCases.ReadEvent(h.goroutinesCtx)
// ...
switch event.Type {
case domains.WSEventNewMessage:
    var message domains.Message
    json.Unmarshal(event.Payload, &message)
    runtime.EventsEmit(h.wailsCtx, newMessageEventName, message)
case domains.WSEventMessageDeleted:
    var payload domains.MessageDeletedPayload
    json.Unmarshal(event.Payload, &payload)
    runtime.EventsEmit(h.wailsCtx, messageDeletedEventName, payload)
}
```

### 9.5 `SendMessage` — `replyToMessageId`

Файл: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/v2/handlers/chat/handler.go`

Обновить сигнатуру:

```go
func (h *Handler) SendMessage(chatID uint64, text string, replyToMessageID *uint64) error
```

Если `replyToMessageID != nil` — заполнить `message.ReplyToMessage = &domains.Message{ID: *replyToMessageID}`.

### 9.6 `DeleteMessage` — новый метод хендлера

```go
func (h *Handler) DeleteMessage(messageID uint64, forAll bool) error
```

Вызывает HTTP API: `DELETE /api/messages/{messageId}` с body `{"forAll": true/false}`.

### 9.7 Frontend (Vue) — контекстное меню и reply

Файл: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/components/ChatView/`

Изменения аналогичны веб-версии:

- Контекстное меню (правый клик / long tap) с опциями "Ответить", "Копировать", "Удалить".
- Reply preview в composer.
- Отрисовка reply-плашки в пузыре сообщения.
- Обработка события `message_deleted` — удаление из `messages` реактивного массива.
- Скролл к оригинальному сообщению при клике на reply-плашку (подгрузка порциями).

Фронтенд Wails-событие:

```js
window.runtime.EventsOn('message_deleted', (payload) => {
    messages.value = messages.value.filter(m => m.id !== payload.messageId);
});
```

---

## 10. CSS

### 10.1 Веб-версия

Файл: `internal/controllers/http/handlers/web/static/css/chat.css`

Новые классы:

- `.context-menu` — абсолютно позиционированное меню с тенью.
- `.context-menu__item` — элемент меню с hover-эффектом.
- `.context-menu__item--danger` — красный цвет для "Удалить".
- `.message-bubble__reply` — плашка ответа внутри пузыря (левая цветная полоска, уменьшенный шрифт, курсор pointer).
- `.conversation__reply-preview` — полоска над composer.

### 10.2 GUI (Vue)

Аналогичные стили в scoped CSS компонента ChatView.

---

## 11. Порядок реализации

1. Миграции БД (`reply_to_message_id`, `is_deleted`)
2. Домены бэкенда (`Message.ReplyToMessage`, `WSEvent`, `DeleteMessageDTO`)
3. Репозиторий бэкенда (LEFT JOIN для reply, фильтр `is_deleted`, `DeleteMessageForUser`, `DeleteMessageForAll`)
4. Сервисы + UseCases бэкенда (`DeleteMessage`)
5. HTTP endpoint (`DELETE /api/chats/{chatId}/messages/{messageId}`)
6. WebSocket бэкенда (envelope pattern, отправка `message_deleted`)
7. Схемы + мапперы (`ReplyMessage`)
8. Веб-клиент: WS dispatch, контекстное меню, reply UI, delete UI
9. GUI домены (`Message.ReplyToMessage`, `WSEvent`)
10. GUI WS repository (`ReadEvent` вместо `ReadMessage`)
11. GUI handler (dispatch по типу, `DeleteMessage`)
12. GUI frontend (Vue — контекстное меню, reply, delete, `message_deleted` event)
13. Тесты на всех слоях
14. Обновление `doc.md` в затронутых директориях
