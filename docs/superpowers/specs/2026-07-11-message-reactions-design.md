# Реакции на сообщения — дизайн

## 1. Цель

Дать пользователям возможность ставить emoji-реакции на сообщения в чате. Один пользователь может поставить несколько разных реакций на одно сообщение (Slack-стиль). Повторная установка уже существующей реакции запрещена и возвращает ошибку. При загрузке сообщений реакции подтягиваются вместе с ними. Изменения распространяются в реальном времени через WebSocket всем участникам чата.

## 2. Требования

- **Multi-реакции на юзера**: один юзер — несколько разных emoji на одно сообщение. Ту же реакцию дважды поставить нельзя — попытка возвращает ошибку (`ErrReactionAlreadyExists`).
- **Справочник emoji**: разрешённый набор реакций хранится в БД, наполняется миграциями. API-эндпоинтов на управление справочником нет.
- **Отдача при чтении чата**: сообщения возвращаются с массивом агрегированных реакций (emoji, список userId — count вычисляется клиентом как длина списка).
- **Real-time**: события `reaction.added` / `reaction.removed` фан-аутятся всем участникам чата (включая отправителя) существующим WS-механизмом.
- **Не влияем на существующие фичи**: `messages_statuses` (is_read, is_deleted), reply, редактирование текста — работают как раньше.

## 3. Модель данных

Новая миграция (например `20260711000000_message_reactions.sql`):

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS reactions (
    id         SERIAL PRIMARY KEY,
    emoji      VARCHAR(16) NOT NULL UNIQUE,
    sort_order INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS message_reactions (
    id          SERIAL PRIMARY KEY,
    message_id  INTEGER   NOT NULL REFERENCES messages(id)  ON DELETE CASCADE,
    user_id     INTEGER   NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
    reaction_id INTEGER   NOT NULL REFERENCES reactions(id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (message_id, user_id, reaction_id)
);

CREATE INDEX message_reactions_message_id_idx ON message_reactions (message_id);
CREATE INDEX message_reactions_user_id_idx    ON message_reactions (user_id);

INSERT INTO reactions (emoji, sort_order) VALUES
  ('👍', 10),
  ('❤️', 20),
  ('🔥', 30),
  ('💯', 40),
  ('😂', 50),
  ('😮', 60),
  ('😢', 70),
  ('😡', 80);

-- +goose Down
DROP TABLE IF EXISTS message_reactions;
DROP TABLE IF EXISTS reactions;
```

### Обоснование FK-политик

- `message_reactions.message_id ON DELETE CASCADE` — hard-delete сообщения уносит реакции. Soft-delete (`messages_statuses.is_deleted`) реакции не трогает — сообщение технически остаётся.
- `message_reactions.user_id ON DELETE CASCADE` — при удалении пользователя его реакции исчезают.
- `message_reactions.reaction_id ON DELETE CASCADE` — если реакция удаляется из справочника (миграцией), она удаляется и из всех сообщений. Единый жизненный цикл сущности, полу-состояния не допускаются.

### Уникальность

`UNIQUE (message_id, user_id, reaction_id)` — гарантирует «одна реакция от юзера на сообщение — максимум один раз». Даёт удобный `INSERT ... ON CONFLICT DO NOTHING` для идемпотентности.

## 4. Domain

`internal/domains/reaction.go`:

```go
package domains

import "time"

type Reaction struct {
    ID    uint64 `json:"id"`
    Emoji string `json:"emoji"`
}

// Агрегат для отображения на сообщении
type MessageReactionSummary struct {
    Reaction Reaction `json:"reaction"`
    UserIDs  []uint64 `json:"userIds"` // count = len(UserIDs), считается на клиенте
}

type MessageReactionDTO struct {
    MessageID  uint64 `json:"-"`
    ReactionID uint64 `json:"reactionId"`
    UserID     uint64 `json:"-"`
}
```

Расширение `Message` (`internal/domains/message.go`):

```go
type Message struct {
    // ... существующие поля
    Reactions []MessageReactionSummary `json:"reactions,omitempty"`
}
```

Поле опциональное — на сообщениях без реакций отдаётся `null`/отсутствует, клиент трактует как пустой массив.

## 5. Слои и пакеты

По конвенции проекта — один пакет `reactions/` на каждом слое, обслуживает и словарь, и M2M-таблицу.

### 5.1 Repository (`internal/repositories/reactions/repository.go`)

Методы:

```go
type Repository struct {
    tx     pg.Transaction
    logger logging.Logger
}

func New(tx pg.Transaction, logger logging.Logger) *Repository

// Справочник
func (r *Repository) ListReactions(ctx context.Context) ([]domains.Reaction, error)
func (r *Repository) GetReactionByID(ctx context.Context, id uint64) (*domains.Reaction, error)

// M2M
func (r *Repository) AddMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) error
func (r *Repository) RemoveMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) error
func (r *Repository) ListReactionsForMessages(
    ctx context.Context,
    messageIDs []uint64,
) (map[uint64][]domains.MessageReactionSummary, error)
```

**`AddMessageReaction`** — не-идемпотентная семантика с явной ошибкой на дубликат:

```sql
INSERT INTO message_reactions (message_id, user_id, reaction_id)
VALUES ($1, $2, $3)
ON CONFLICT (message_id, user_id, reaction_id) DO NOTHING
RETURNING id;
```

- Если `RETURNING id` вернул строку — INSERT прошёл, репо возвращает `nil`.
- Если `RETURNING` пуст (конфликт по UNIQUE) — репо возвращает sentinel `ErrReactionAlreadyExists` из `internal/errors/`.

Почему `ON CONFLICT DO NOTHING + RETURNING`, а не голый INSERT с отловом `pq.Error{Code: "23505"}`: не завязываемся на драйверские коды, логика читается на уровне SQL, поведение одинаковое для любого Postgres-драйвера.

**`RemoveMessageReaction`** — идемпотентный DELETE. Клиенту всегда 200 (нет смысла ругать на «уже нет»), но нужно понять, реально ли что-то удалили, чтобы **не спамить WS**.

Сигнатура:

```go
// deleted = true, если строка реально была удалена; false — если ничего не совпало.
func (r *Repository) RemoveMessageReaction(
    ctx context.Context,
    dto domains.MessageReactionDTO,
) (deleted bool, err error)
```

Реализация:

```sql
DELETE FROM message_reactions
WHERE message_id = $1 AND user_id = $2 AND reaction_id = $3;
```

```go
res, err := tx.ExecContext(ctx, stmt, args...)
if err != nil { return false, err }
n, err := res.RowsAffected()
if err != nil { return false, err }
return n > 0, nil
```

Usecase публикует `reaction.removed` только при `deleted == true`.

**`ListReactionsForMessages`** — один SQL с JOIN на справочник:

```sql
SELECT mr.message_id, mr.reaction_id, r.emoji, mr.user_id
FROM message_reactions mr
JOIN reactions r ON r.id = mr.reaction_id
WHERE mr.message_id = ANY($1)
ORDER BY mr.message_id, r.sort_order, mr.created_at;
```

Группировка в Go: `map[messageID]map[reactionID]*MessageReactionSummary` → раскатываем во внешнюю `map[uint64][]MessageReactionSummary`. Порядок реакций внутри сообщения сохраняется через `ORDER BY r.sort_order`.

Новые sentinel-ошибки в `internal/errors/`:

```go
var (
    ErrReactionAlreadyExists = errors.New("reaction already exists")
    ErrReactionNotFound      = errors.New("reaction does not exist in dictionary")
    ErrReactionNotSet        = errors.New("reaction was not set on this message for this user")
)
```

### 5.2 Service (`internal/services/reactions/service.go`)

Тонкая обёртка: принимает `UoW`, инжектит репо, оборачивает вызовы в `uow.Do(...)` где нужно.

### 5.3 Usecase (`internal/usecases/reactions/usecase.go`)

Оркестрация и валидация:

- **`AddReaction(ctx, dto)`**:
  1. Юзер — участник чата, где лежит сообщение (`chats_members` lookup через messages репо).
  2. Сообщение не soft-deleted для этого юзера (`messages_statuses.is_deleted = false`).
  3. Реакция существует в справочнике (`GetReactionByID` → `ErrReactionNotFound` если нет).
  4. `reactionsService.AddMessageReaction(dto)` → может вернуть `ErrReactionAlreadyExists`.
  5. Публикация WS-события `reaction.added` всем участникам чата (только при успехе п.4).
- **`RemoveReaction(ctx, dto)`**:
  1. Те же проверки (member + not soft-deleted).
  2. `deleted, err := reactionsService.RemoveMessageReaction(dto)`.
  3. Публикация `reaction.removed` только если `deleted == true`. Клиенту всегда возвращаем успех.
- **`ListReactions(ctx)`** — проксирует справочник для UI-пикера.

Интеграция с чтением сообщений — отдельная функция-хелпер, чтобы не размазывать логику по вызывающим методам:

```go
// AttachReactions — обогащает список сообщений реакциями.
// Возвращает те же сообщения с заполненным полем Reactions.
func (u *Usecase) AttachReactions(
    ctx context.Context,
    msgs []domains.Message,
) ([]domains.Message, error) {
    if len(msgs) == 0 {
        return msgs, nil
    }

    ids := make([]uint64, 0, len(msgs))
    for i := range msgs {
        ids = append(ids, msgs[i].ID)
    }

    reactionsByMsg, err := u.reactionsService.ListReactionsForMessages(ctx, ids)
    if err != nil {
        return nil, err
    }

    for i := range msgs {
        msgs[i].Reactions = reactionsByMsg[msgs[i].ID]
    }

    return msgs, nil
}
```

Вызов из `usecases/messages`:

```go
msgs, err := u.messagesService.GetChatMessages(...)
if err != nil { return nil, err }
return u.reactionsUsecase.AttachReactions(ctx, msgs)
```

Для одиночного `GetMessageByID` — обёртка `AttachReactions(ctx, []Message{msg})[0]` или отдельный `AttachReaction(ctx, msg)`.

### 5.4 Controller (`internal/controllers/http/`)

Новые handlers + schemas + mappers:

- `GET /reactions` — список из справочника для UI-пикера.
- `POST /messages/{id}/reactions` — **сохранить** реакцию юзера на сообщение. Body `{"reactionId": N}`. 200 при успехе, 409 если такая реакция уже стоит (`ErrReactionAlreadyExists`), 404 если реакция не в справочнике или сообщение недоступно, 403 если юзер не член чата.
- `DELETE /messages/{id}/reactions/{reactionId}` — **снять** реакцию. 200 всегда (идемпотентно: 200 и если реакции не было). 403 если не член чата, 404 если сообщение недоступно юзеру (soft-deleted).

Мапперы сообщений расширяются: если у `domains.Message.Reactions != nil`, кладём в HTTP-схему.

### 5.5 Interfaces (`internal/interfaces/`)

Интерфейсы `ReactionsRepository`, `ReactionsService`, `ReactionsUsecase` с `mockgen` директивами по существующему шаблону.

## 6. WebSocket события

Новые типы в `internal/domains/ws_event.go`:

- `reaction.added`:
  ```json
  {
    "type": "reaction.added",
    "payload": {
      "messageId": 123,
      "chatId": 45,
      "userId": 7,
      "reactionId": 2,
      "emoji": "❤️"
    }
  }
  ```
- `reaction.removed`:
  ```json
  {
    "type": "reaction.removed",
    "payload": {
      "messageId": 123,
      "chatId": 45,
      "userId": 7,
      "reactionId": 2
    }
  }
  ```

Fan-out — по существующему паттерну (`sync.Map[userID → *userConnections]`), всем участникам чата, включая инициатора.

## 7. Edge cases

- **Дубликат ADD**: репо возвращает `ErrReactionAlreadyExists`, usecase пробрасывает наружу → HTTP 409. WS не публикуется.
- **REMOVE несуществующего**: репо возвращает `ErrReactionNotSet`. Handler ловит и отдаёт 200 (идемпотентно), broadcast НЕ вызывается — цикл "usecase → broadcaster" отсутствует, broadcast делает handler после успешного usecase-вызова.
- **Неизвестный `reactionId`** (нет в справочнике): usecase проверяет через `GetReactionByID` до вызова репо → `ErrReactionNotFound` → HTTP 404.
- **Сообщение soft-deleted для юзера**: usecase блокирует и add, и remove — 403/404.
- **Юзер не в чате**: 403.
- **Редактирование текста сообщения**: реакции остаются.
- **Hard-delete сообщения/юзера**: реакции уходят по CASCADE.
- **Удаление записи из `reactions` (миграцией)**: все ссылающиеся `message_reactions` уходят по CASCADE.

## 8. Загрузка справочника

Простой подход: `GET /reactions` каждый раз бьёт по БД. Таблица маленькая (~10 строк), запрос быстрый. In-memory кэш **не делаем** в MVP — вернёмся, если появится реальная нагрузка. Это осознанное решение, а не пропуск.

## 9. Тесты

По правилу `CLAUDE.md` (все изменения кода → тесты):

- **Repository**: моки `pg.Transaction`, проверяем SQL (add/remove/list dict/list for messages/группировка).
- **Usecase**: моки repo/service/ws-publisher — валидация member/soft-deleted/dict lookup, публикация событий, идемпотентность (add существующей → нет WS).
- **Domain**: валидация DTO (positive IDs).
- **Trace decorators** на всех слоях — по существующему шаблону.
- **Controller**: happy path + auth + 404/403.

## 10. Что НЕ входит в scope

- API управления справочником реакций (только через миграции).
- Rate limiting на реакции (не нужно в MVP; `UNIQUE` защищает от дублей).
- Аналитика/метрики топ-реакций.
- Кастомные emoji (загружаемые картинки).
- Реакции на что-то кроме сообщений (чаты, юзеры и т.д.).

## 11. Обновления doc.md

По правилу `CLAUDE.md`, обновляем:

- `internal/domains/doc.md` — новая сущность Reaction / MessageReactionSummary.
- `internal/repositories/doc.md` — новый пакет `reactions`.
- `internal/services/doc.md` — то же.
- `internal/usecases/doc.md` — то же.
- `internal/controllers/http/doc.md` — новые эндпоинты и WS-события.
- `migrations/doc.md` — новая миграция.
- `docs/modules.md` — таблица пакетов.
- `docs/architecture.md` — упомянуть реакции в разделе про сообщения.
