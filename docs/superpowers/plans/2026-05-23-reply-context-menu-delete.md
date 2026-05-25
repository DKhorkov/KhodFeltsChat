# Reply to Message + Context Menu + Delete — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add reply-to-message, context menu (reply/copy/delete), and soft-delete messages in both the backend (KhodFeltsChat) and the desktop GUI (KhodFeltsChatGUI).

**Architecture:** Two DB migrations (reply column + soft delete flag). Envelope pattern for WebSocket events. Backend: new domain types, repository LEFT JOINs, delete endpoint, WS event dispatch. GUI: WS event deserialization, delete HTTP call, Vue context menu + reply UI. Web: vanilla JS context menu + reply UI.

**Tech Stack:** Go 1.24, PostgreSQL (squirrel), gorilla/websocket, gorilla/mux, Wails v2 + Vue 3, vanilla JS (web client).

**Spec:** `docs/superpowers/specs/2026-05-23-reply-and-context-menu-design.md`

---

## File Map

### Backend (KhodFeltsChat)

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `migrations/20260523000001_reply_to_message.sql` | Add `reply_to_message_id` to `messages` |
| Create | `migrations/20260523000002_messages_soft_delete.sql` | Add `is_deleted` to `messages_statuses` |
| Modify | `internal/domains/message.go` | Add `ReplyToMessage`, `ReplyToMessageID` fields |
| Create | `internal/domains/ws_event.go` | `WSEvent`, `WSEventType`, `MessageDeletedPayload`, `DeleteMessageDTO` |
| Create | `internal/errors/messages.go` | `ErrMessageNotFound`, `ErrNotMessageAuthor` |
| Modify | `internal/interfaces/repositories.go` | Add `DeleteMessageForUser`, `DeleteMessageForAll` |
| Modify | `internal/interfaces/services.go` | Add `DeleteMessage` to `MessagesService` |
| Modify | `internal/interfaces/usecases.go` | Add `DeleteMessage` to `MessagesUseCases` |
| Modify | `internal/repositories/messages/repository.go` | LEFT JOIN for reply, `is_deleted` filter, `SaveMessage` with `reply_to_message_id`, new delete methods |
| Modify | `internal/repositories/messages/trace_decorator.go` | Add `DeleteMessageForUser`, `DeleteMessageForAll` |
| Modify | `internal/services/messages/service.go` | Add `DeleteMessage` |
| Modify | `internal/services/messages/trace_decorator.go` | Add `DeleteMessage` |
| Modify | `internal/usecases/messages/usecases.go` | Add `DeleteMessage` with validation |
| Modify | `internal/usecases/messages/trace_decorator.go` | Add `DeleteMessage` |
| Create | `internal/controllers/http/handlers/api/messages/delete/handler.go` | `DELETE /api/messages/{id}` handler |
| Modify | `internal/controllers/http/handlers/api/setup.go` | Register delete route |
| Modify | `internal/controllers/http/handlers/api/ws/ws.go` | Envelope pattern, `ReplyToMessageID` handling |
| Modify | `internal/controllers/http/schemas/messages.go` | Add `ReplyMessage` schema, `ReplyToMessage` field |
| Modify | `internal/controllers/http/mappers/messages/messages.go` | Map `ReplyToMessage` |
| Modify | `internal/controllers/http/handlers/web/templates/chat.html` | Reply preview, context menu DOM |
| Modify | `internal/controllers/http/handlers/web/static/js/chat.js` | WS dispatch, context menu, reply, delete |
| Modify | `internal/controllers/http/handlers/web/static/css/chat.css` | Context menu, reply bubble, reply preview styles |

### GUI (KhodFeltsChatGUI)

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/domains/message.go` | Add `ReplyToMessage` field |
| Create | `internal/domains/ws_event.go` | `WSEvent`, `WSEventType`, `MessageDeletedPayload` |
| Modify | `internal/interfaces/repositories.go` | `ReadMessage` → `ReadEvent` |
| Modify | `internal/interfaces/usecases.go` | `ReadMessage` → `ReadEvent`, add `DeleteMessage` |
| Modify | `internal/repositories/ws/repository.go` | `ReadEvent` returning `*domains.WSEvent` |
| Modify | `internal/usecases/usecases.go` | `ReadEvent`, `DeleteMessage` (HTTP call) |
| Modify | `internal/v2/handlers/chat/handler.go` | WS event dispatch, `SendMessage` with replyToMessageID, `DeleteMessage` |
| Modify | `frontend/src/constants/index.js` | Add `MESSAGE_DELETED` event |
| Modify | `frontend/src/components/ChatView/ChatView.js` | Context menu, reply, delete, WS event handling |
| Modify | `frontend/src/components/ChatView/ChatView.vue` | Context menu template, reply preview, reply bubble |
| Modify | `frontend/src/components/ChatView/ChatView.css` | Context menu, reply styles |

---

## Task 1: Database Migrations

**Files:**
- Create: `migrations/20260523000001_reply_to_message.sql`
- Create: `migrations/20260523000002_messages_soft_delete.sql`

- [ ] **Step 1: Create reply_to_message_id migration**

```sql
-- +goose Up
ALTER TABLE messages
ADD COLUMN reply_to_message_id INTEGER REFERENCES messages(id) ON DELETE SET NULL;

CREATE INDEX messages_reply_to_message_id_idx ON messages (reply_to_message_id);

-- +goose Down
DROP INDEX IF EXISTS messages_reply_to_message_id_idx;
ALTER TABLE messages DROP COLUMN IF EXISTS reply_to_message_id;
```

- [ ] **Step 2: Create is_deleted migration**

```sql
-- +goose Up
ALTER TABLE messages_statuses
ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE messages_statuses DROP COLUMN IF EXISTS is_deleted;
```

- [ ] **Step 3: Apply migrations**

Run: `task migrate-up`
Expected: Both migrations applied successfully.

---

## Task 2: Backend Domains

**Files:**
- Modify: `internal/domains/message.go`
- Create: `internal/domains/ws_event.go`
- Create: `internal/errors/messages.go`

- [ ] **Step 1: Update Message domain**

In `internal/domains/message.go`, add two fields to `Message`:

```go
type Message struct {
	ID               uint64    `json:"id"`
	ChatID           uint64    `json:"chatId"`
	Sender           User      `json:"sender"`
	Text             string    `json:"text"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	IsRead           bool      `json:"isRead"`
	ReplyToMessage   *Message  `json:"replyToMessage,omitempty"`
	ReplyToMessageID *uint64   `json:"replyToMessageId,omitempty"`
}
```

`ReplyToMessage` — populated when reading from DB (full reply data for UI).
`ReplyToMessageID` — populated when deserializing incoming WS message from client (just the ID).

- [ ] **Step 2: Create WSEvent domain**

Create `internal/domains/ws_event.go`:

```go
package domains

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

type DeleteMessageDTO struct {
	MessageID uint64 `json:"messageId"`
	UserID    uint64 `json:"-"`
	ForAll    bool   `json:"forAll"`
}
```

- [ ] **Step 3: Create message errors**

Create `internal/errors/messages.go`:

```go
package errors

import "errors"

var (
	ErrMessageNotFound  = errors.New("message not found")
	ErrNotMessageAuthor = errors.New("only message author can delete for all")
)
```

---

## Task 3: Backend Repository — Reply & Delete Methods

**Files:**
- Modify: `internal/interfaces/repositories.go`
- Modify: `internal/repositories/messages/repository.go`
- Modify: `internal/repositories/messages/trace_decorator.go`

- [ ] **Step 1: Update MessagesRepository interface**

In `internal/interfaces/repositories.go`, add two methods to `MessagesRepository`:

```go
MessagesRepository interface {
	SaveMessage(ctx context.Context, message domains.Message) (uint64, error)
	GetChatMessages(ctx context.Context, userID uint64, chatID uint64, pagination *domains.Pagination) ([]domains.Message, error)
	GetMessageByID(ctx context.Context, userID uint64, messageID uint64) (*domains.Message, error)
	ChangeMessagesIsReadStatus(ctx context.Context, userID uint64, messages []domains.Message, isRead bool) error
	ReadAllChatMessages(ctx context.Context, userID uint64, chatID uint64) error
	DeleteMessageForUser(ctx context.Context, userID uint64, messageID uint64) error
	DeleteMessageForAll(ctx context.Context, messageID uint64) error
}
```

- [ ] **Step 2: Update MessagePg struct and mapper**

In `internal/repositories/messages/repository.go`, add constants:

```go
replyToMessageIDColumnName = "reply_to_message_id"
isDeletedColumnName        = "is_deleted"
```

Update `MessagePg` to include nullable reply fields:

```go
type MessagePg struct {
	ID                      uint64
	ChatID                  uint64
	SenderID                uint64
	SenderUsername           string
	SenderEmail             string
	SenderEmailConfirmed    bool
	SenderPassword          string
	SenderCreatedAt         time.Time
	SenderUpdatedAt         time.Time
	Text                    string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	IsRead                  bool
	ReplyToMessageID        *uint64
	ReplyToMessageText      *string
	ReplyToMessageCreatedAt *time.Time
	ReplyToSenderID         *uint64
	ReplyToSenderUsername    *string
}
```

Update `pgMessageToDomainMessage`:

```go
func pgMessageToDomainMessage(messagePg MessagePg) *domains.Message {
	msg := &domains.Message{
		ID:     messagePg.ID,
		ChatID: messagePg.ChatID,
		Sender: domains.User{
			ID:             messagePg.SenderID,
			Username:       messagePg.SenderUsername,
			Email:          messagePg.SenderEmail,
			EmailConfirmed: messagePg.SenderEmailConfirmed,
			Password:       messagePg.SenderPassword,
			CreatedAt:      messagePg.SenderCreatedAt,
			UpdatedAt:      messagePg.SenderUpdatedAt,
		},
		Text:      messagePg.Text,
		CreatedAt: messagePg.CreatedAt,
		UpdatedAt: messagePg.UpdatedAt,
		IsRead:    messagePg.IsRead,
	}

	if messagePg.ReplyToMessageID != nil {
		msg.ReplyToMessage = &domains.Message{
			ID:   *messagePg.ReplyToMessageID,
			Text: *messagePg.ReplyToMessageText,
			Sender: domains.User{
				ID:       *messagePg.ReplyToSenderID,
				Username: *messagePg.ReplyToSenderUsername,
			},
			CreatedAt: *messagePg.ReplyToMessageCreatedAt,
		}
	}

	return msg
}
```

- [ ] **Step 3: Update SaveMessage — insert reply_to_message_id**

In `SaveMessage`, update the INSERT to include `reply_to_message_id`:

```go
func (repo *Repository) SaveMessage(
	ctx context.Context,
	message domains.Message,
) (uint64, error) {
	var replyToMessageID interface{}
	if message.ReplyToMessage != nil {
		replyToMessageID = message.ReplyToMessage.ID
	}

	stmt, params, err := sq.
		Insert(messagesTableName).
		Columns(
			chatIDColumnName,
			senderIDColumnName,
			textColumnName,
			replyToMessageIDColumnName,
		).
		Values(
			message.ChatID,
			message.Sender.ID,
			message.Text,
			replyToMessageID,
		).
		Suffix(returningIDSuffix).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, err
	}

	// ... rest of SaveMessage stays the same (fetch members, insert statuses) ...
```

- [ ] **Step 4: Update GetChatMessages — LEFT JOIN reply + is_deleted filter**

Add reply table aliases as constants:

```go
const (
	replyTableAlias       = "reply"
	replySenderTableAlias = "reply_sender"
)
```

In `GetChatMessages`, add the reply columns to `columnsForSelect`:

```go
columnsForSelect := []string{
	fmt.Sprintf("%s.%s", messagesTableName, idColumnName),
	fmt.Sprintf("%s.%s", messagesTableName, chatIDColumnName),
	fmt.Sprintf("%s.%s", usersTableName, idColumnName),
	fmt.Sprintf("%s.%s", usersTableName, usernameColumnName),
	fmt.Sprintf("%s.%s", usersTableName, emailColumnName),
	fmt.Sprintf("%s.%s", usersTableName, emailConfirmedColumnName),
	fmt.Sprintf("%s.%s", usersTableName, passwordColumnName),
	fmt.Sprintf("%s.%s", usersTableName, createdAtColumnName),
	fmt.Sprintf("%s.%s", usersTableName, updatedAtColumnName),
	fmt.Sprintf("%s.%s", messagesTableName, textColumnName),
	fmt.Sprintf("%s.%s", messagesTableName, createdAtColumnName),
	fmt.Sprintf("%s.%s", messagesTableName, updatedAtColumnName),
	fmt.Sprintf("%s.%s", messagesStatusesTableName, isReadColumnName),
	// Reply fields:
	fmt.Sprintf("%s.%s", replyTableAlias, idColumnName),
	fmt.Sprintf("%s.%s", replyTableAlias, textColumnName),
	fmt.Sprintf("%s.%s", replyTableAlias, createdAtColumnName),
	fmt.Sprintf("%s.%s", replySenderTableAlias, idColumnName),
	fmt.Sprintf("%s.%s", replySenderTableAlias, usernameColumnName),
}
```

Add LEFT JOINs and `is_deleted` filter:

```go
builder := sq.
	Select(columnsForSelect...).
	From(messagesTableName).
	Join(
		fmt.Sprintf(
			"%s ON %s.%s = %s.%s",
			usersTableName, usersTableName, idColumnName,
			messagesTableName, senderIDColumnName,
		),
	).
	Join(
		fmt.Sprintf(
			"%s ON %s.%s = %s.%s",
			messagesStatusesTableName, messagesStatusesTableName, messageIDColumnName,
			messagesTableName, idColumnName,
		),
	).
	LeftJoin(
		fmt.Sprintf(
			"%s AS %s ON %s.%s = %s.%s",
			messagesTableName, replyTableAlias,
			messagesTableName, replyToMessageIDColumnName,
			replyTableAlias, idColumnName,
		),
	).
	LeftJoin(
		fmt.Sprintf(
			"%s AS %s ON %s.%s = %s.%s",
			usersTableName, replySenderTableAlias,
			replyTableAlias, senderIDColumnName,
			replySenderTableAlias, idColumnName,
		),
	).
	Where(
		sq.And{
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesTableName, chatIDColumnName): chatID,
			},
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesStatusesTableName, userIDColumnName): userID,
			},
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesStatusesTableName, isDeletedColumnName): false,
			},
		},
	).
	OrderBy(fmt.Sprintf("%s.%s %s", messagesTableName, idColumnName, desc)).
	PlaceholderFormat(sq.Dollar)
```

- [ ] **Step 5: Update GetMessageByID — same LEFT JOINs + is_deleted filter**

Apply the same changes to `GetMessageByID` as in Step 4: add the 5 reply columns to `columnsForSelect`, add the two `LeftJoin` clauses, add `is_deleted = false` to the WHERE clause.

- [ ] **Step 6: Implement DeleteMessageForUser and DeleteMessageForAll**

```go
func (repo *Repository) DeleteMessageForUser(
	ctx context.Context,
	userID uint64,
	messageID uint64,
) error {
	stmt, params, err := sq.
		Update(messagesStatusesTableName).
		Set(isDeletedColumnName, true).
		Where(sq.Eq{
			userIDColumnName:    userID,
			messageIDColumnName: messageID,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) DeleteMessageForAll(
	ctx context.Context,
	messageID uint64,
) error {
	stmt, params, err := sq.
		Update(messagesStatusesTableName).
		Set(isDeletedColumnName, true).
		Where(sq.Eq{
			messageIDColumnName: messageID,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
```

- [ ] **Step 7: Update repository trace decorator**

In `internal/repositories/messages/trace_decorator.go`, add two methods:

```go
func (d *TraceDecorator) DeleteMessageForUser(
	ctx context.Context,
	userID uint64,
	messageID uint64,
) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeleteMessageForUser(ctx, userID, messageID)
}

func (d *TraceDecorator) DeleteMessageForAll(
	ctx context.Context,
	messageID uint64,
) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeleteMessageForAll(ctx, messageID)
}
```

- [ ] **Step 8: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds (may fail on service/usecase — that's Task 4).

---

## Task 4: Backend Service & UseCase — DeleteMessage

**Files:**
- Modify: `internal/interfaces/services.go`
- Modify: `internal/interfaces/usecases.go`
- Modify: `internal/services/messages/service.go`
- Modify: `internal/services/messages/trace_decorator.go`
- Modify: `internal/usecases/messages/usecases.go`
- Modify: `internal/usecases/messages/trace_decorator.go`

- [ ] **Step 1: Update MessagesService interface**

In `internal/interfaces/services.go`, add to `MessagesService`:

```go
DeleteMessage(ctx context.Context, dto domains.DeleteMessageDTO) error
```

- [ ] **Step 2: Update MessagesUseCases interface**

In `internal/interfaces/usecases.go`, `MessagesUseCases` currently embeds `MessagesService`. Add the method explicitly or keep embedding — either way, `DeleteMessage` is available. Since `MessagesUseCases` embeds `MessagesService`, the method is inherited automatically. No change needed here unless the usecase has different signature — but per spec, usecase does validation before delegating, so it wraps the service. Add explicitly:

```go
type MessagesUseCases interface {
	MessagesService
	DeleteMessage(ctx context.Context, dto domains.DeleteMessageDTO) error
}
```

Wait — `MessagesService` already has `DeleteMessage` and `MessagesUseCases` embeds `MessagesService`, so it's already there. The usecase implementation will just add its own `DeleteMessage` that does validation before calling the service. Since Go embedding works this way, the interface is satisfied. No change needed to `interfaces/usecases.go`.

- [ ] **Step 3: Implement service DeleteMessage**

In `internal/services/messages/service.go`, add:

```go
func (s *Service) DeleteMessage(
	ctx context.Context,
	dto domains.DeleteMessageDTO,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			messagesRepository := s.newMessagesRepositoryFunc(tx)

			if dto.ForAll {
				return messagesRepository.DeleteMessageForAll(ctx, dto.MessageID)
			}

			return messagesRepository.DeleteMessageForUser(ctx, dto.UserID, dto.MessageID)
		},
	)
}
```

- [ ] **Step 4: Implement usecase DeleteMessage**

In `internal/usecases/messages/usecases.go`, add:

```go
func (u *UseCases) DeleteMessage(
	ctx context.Context,
	dto domains.DeleteMessageDTO,
) error {
	if dto.ForAll {
		message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
		if err != nil {
			return fmt.Errorf("%w: %w", customerrors.ErrMessageNotFound, err)
		}

		if message.Sender.ID != dto.UserID {
			return customerrors.ErrNotMessageAuthor
		}
	}

	return u.messagesService.DeleteMessage(ctx, dto)
}
```

Add `"fmt"` to imports if not present.

- [ ] **Step 5: Update service trace decorator**

In `internal/services/messages/trace_decorator.go`, add:

```go
func (d *TraceDecorator) DeleteMessage(
	ctx context.Context,
	dto domains.DeleteMessageDTO,
) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeleteMessage(ctx, dto)
}
```

- [ ] **Step 6: Update usecase trace decorator**

In `internal/usecases/messages/trace_decorator.go`, add:

```go
func (d *TraceDecorator) DeleteMessage(
	ctx context.Context,
	dto domains.DeleteMessageDTO,
) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeleteMessage(ctx, dto)
}
```

- [ ] **Step 7: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds.

---

## Task 5: HTTP Delete Endpoint

**Files:**
- Create: `internal/controllers/http/handlers/api/messages/delete/handler.go`
- Modify: `internal/controllers/http/handlers/api/setup.go`
- Modify: `internal/controllers/http/handlers/common/route.go`

- [ ] **Step 1: Add MessageIDRouteKey**

In `internal/controllers/http/handlers/common/route.go`:

```go
package common

const (
	IDRouteKey        = "id"
	MessageIDRouteKey = "messageId"
)
```

- [ ] **Step 2: Create delete handler**

Create `internal/controllers/http/handlers/api/messages/delete/handler.go`:

```go
package delete

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
)

// swagger:route DELETE /api/messages/{messageId} messages DeleteMessage
//
// DeleteMessage
//
// Deletes a message by ID. If forAll is true, deletes for all chat members (author only).
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	401: Unauthorized
//	403: Forbidden
//	404: NotFound
//	500: InternalServerError

// Handler deletes a message.
func Handler(u interfaces.MessagesUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		messageIDStr := mux.Vars(r)[common.MessageIDRouteKey]

		messageID, err := strconv.ParseUint(messageIDStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.DeleteMessageDTO
		if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto.MessageID = messageID
		dto.UserID = userID

		err = u.DeleteMessage(r.Context(), dto)

		switch {
		case errors.Is(err, customerrors.ErrMessageNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.Is(err, customerrors.ErrNotMessageAuthor):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 3: Register route in setup.go**

In `internal/controllers/http/handlers/api/setup.go`, add import:

```go
delete_message "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/delete"
```

Add URL constant:

```go
DeleteMessageURL = "/messages/{%s}"
```

In `SetupHandlers`, add to `deleteMux`:

```go
deleteMux.Handle(
	fmt.Sprintf(DeleteMessageURL, common.MessageIDRouteKey),
	delete_message.Handler(messagesUseCases),
)
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds.

---

## Task 6: Backend WebSocket — Envelope Pattern + Reply

**Files:**
- Modify: `internal/controllers/http/handlers/api/ws/ws.go`
- Modify: `internal/controllers/http/schemas/messages.go`
- Modify: `internal/controllers/http/mappers/messages/messages.go`

- [ ] **Step 1: Update schemas — add ReplyMessage**

In `internal/controllers/http/schemas/messages.go`, add the `ReplyToMessage` field to `Message` and new `ReplyMessage` type:

```go
type Message struct {
	ID        uint64    `json:"id"`
	ChatID    uint64    `json:"chatId"`
	Sender    Sender    `json:"sender"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsRead    bool      `json:"isRead"`

	// Message that this message is a reply to.
	// required: false
	// nullable: true
	ReplyToMessage *ReplyMessage `json:"replyToMessage,omitempty"`
}

// ReplyMessage represents an abbreviated message that was replied to.
// swagger:model
type ReplyMessage struct {
	// Unique identifier of the original message.
	// required: true
	ID uint64 `json:"id"`

	// Sender of the original message.
	// required: true
	Sender Sender `json:"sender"`

	// Text of the original message.
	// required: true
	Text string `json:"text"`

	// When the original message was sent.
	// required: true
	// format: date-time
	CreatedAt time.Time `json:"createdAt"`
}
```

- [ ] **Step 2: Update mapper**

In `internal/controllers/http/mappers/messages/messages.go`, update `MapMessage`:

```go
func MapMessage(message domains.Message) schemas.Message {
	mapped := schemas.Message{
		ID:     message.ID,
		ChatID: message.ChatID,
		Sender: schemas.Sender{
			ID:       message.Sender.ID,
			Username: message.Sender.Username,
		},
		Text:      message.Text,
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
		IsRead:    message.IsRead,
	}

	if message.ReplyToMessage != nil {
		mapped.ReplyToMessage = &schemas.ReplyMessage{
			ID: message.ReplyToMessage.ID,
			Sender: schemas.Sender{
				ID:       message.ReplyToMessage.Sender.ID,
				Username: message.ReplyToMessage.Sender.Username,
			},
			Text:      message.ReplyToMessage.Text,
			CreatedAt: message.ReplyToMessage.CreatedAt,
		}
	}

	return mapped
}
```

- [ ] **Step 3: Update WS handler — envelope + replyToMessageID**

In `internal/controllers/http/handlers/api/ws/ws.go`, update `listen`:

Change the message reading to handle `ReplyToMessageID`:

```go
func (h *Handler) listen(conn *websocket.Conn, user *domains.User) {
	for {
		ctx := context.Background()
		message := domains.NewMessage().From(*user).Received()

		if err := conn.ReadJSON(message); err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logging.LogErrorContext(
					ctx,
					h.logger,
					"Failed to read message",
					err,
					"Sender", message.Sender,
				)
			}

			return
		}

		// Convert ReplyToMessageID to ReplyToMessage:
		if message.ReplyToMessageID != nil {
			message.ReplyToMessage = &domains.Message{ID: *message.ReplyToMessageID}
		}

		chatMembers, err := h.chatsUseCases.GetChatMembers(ctx, message.ChatID)
		// ... existing validation ...

		var savedMessage *domains.Message

		if savedMessage, err = h.messagesUseCases.SaveMessage(ctx, *message); err != nil {
			// ... existing error handling ...
		}

		messageToSend := messages.MapMessage(*savedMessage)
		messageToSend.IsRead = false

		// Wrap in envelope:
		event := domains.WSEvent{
			Type:    domains.WSEventNewMessage,
			Payload: messageToSend,
		}

		for _, member := range chatMembers {
			if member.ID == user.ID {
				continue
			}

			value, exists := h.connections.Load(member.ID)
			if !exists {
				h.publishNewMessageNotifications(ctx, member.ID, savedMessage.ID)

				continue
			}

			connection, ok := value.(*websocket.Conn)
			if !ok {
				// ... existing error handling ...
				continue
			}

			if err = connection.WriteJSON(event); err != nil {
				// ... existing error handling ...
			}
		}
	}
}
```

- [ ] **Step 4: Add BroadcastMessageDeleted method to WS handler**

Add a public method that the delete HTTP handler can call (or expose `connections` through an interface). A simpler approach — add a method to `Handler` that can be called to broadcast deletions:

```go
func (h *Handler) BroadcastMessageDeleted(ctx context.Context, chatID uint64, messageID uint64, excludeUserID uint64) {
	event := domains.WSEvent{
		Type: domains.WSEventMessageDeleted,
		Payload: domains.MessageDeletedPayload{
			MessageID: messageID,
			ChatID:    chatID,
		},
	}

	chatMembers, err := h.chatsUseCases.GetChatMembers(ctx, chatID)
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to get chat members for delete broadcast", err)

		return
	}

	for _, member := range chatMembers {
		if member.ID == excludeUserID {
			continue
		}

		value, exists := h.connections.Load(member.ID)
		if !exists {
			continue
		}

		connection, ok := value.(*websocket.Conn)
		if !ok {
			h.connections.Delete(member.ID)

			continue
		}

		if err = connection.WriteJSON(event); err != nil {
			logging.LogErrorContext(ctx, h.logger, "Failed to broadcast message_deleted", err)

			if err = connection.Close(); err != nil {
				logging.LogErrorContext(ctx, h.logger, "Failed to close connection", err)
			}

			h.connections.Delete(member.ID)
		}
	}
}
```

Note: The delete HTTP handler will need access to this `ws.Handler` instance. Wire it through `SetupHandlers` — the `websocketHandler` is already created there. Pass it to the delete handler factory.

- [ ] **Step 5: Update delete handler to accept WS handler and broadcast**

Update `internal/controllers/http/handlers/api/messages/delete/handler.go` — change `Handler` factory to accept a broadcaster:

```go
// WSBroadcaster broadcasts WebSocket events to chat members.
type WSBroadcaster interface {
	BroadcastMessageDeleted(ctx context.Context, chatID uint64, messageID uint64, excludeUserID uint64)
}

func Handler(u interfaces.MessagesUseCases, broadcaster WSBroadcaster) http.HandlerFunc {
```

After successful delete, if `dto.ForAll`:

```go
		err = u.DeleteMessage(r.Context(), dto)
		// ... error handling ...

		if dto.ForAll {
			// Get message to find chatID for broadcasting:
			// Message is already soft-deleted, so we need the chatID.
			// We can get it from the message before deletion, but at this point it's deleted.
			// Solution: return chatID from the usecase/service, or fetch it beforehand.
			// Simplest: add ChatID to DeleteMessageDTO (set by handler from a pre-fetch).
		}

		w.WriteHeader(http.StatusNoContent)
```

Actually, since we removed `ChatID` from DTO, we need to fetch the message before deleting to get its `chatID`. Update the handler:

```go
func Handler(u interfaces.MessagesUseCases, broadcaster WSBroadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ... parse userID, messageID, decode dto ...

		dto.MessageID = messageID
		dto.UserID = userID

		// Fetch message before deletion to get chatID for WS broadcast:
		var chatID uint64
		if dto.ForAll {
			message, err := u.GetMessageByID(r.Context(), userID, messageID)
			if err != nil {
				http.Error(w, customerrors.ErrMessageNotFound.Error(), http.StatusNotFound)

				return
			}

			chatID = message.ChatID
		}

		err = u.DeleteMessage(r.Context(), dto)
		// ... error handling ...

		if dto.ForAll {
			broadcaster.BroadcastMessageDeleted(r.Context(), chatID, messageID, userID)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 6: Update setup.go to pass websocketHandler to delete handler**

In `SetupHandlers`, the `websocketHandler` is already created. Update the delete route registration:

```go
deleteMux.Handle(
	fmt.Sprintf(DeleteMessageURL, common.MessageIDRouteKey),
	delete_message.Handler(messagesUseCases, &websocketHandler),
)
```

- [ ] **Step 7: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds.

---

## Task 7: Backend — Regenerate Mocks

**Files:**
- Modify: `mocks/repositories/*.go`
- Modify: `mocks/services/*.go`
- Modify: `mocks/usecases/*.go`

- [ ] **Step 1: Regenerate all mocks**

Run: `go generate ./internal/interfaces/...`
Expected: Mocks regenerated with new methods.

- [ ] **Step 2: Verify tests compile**

Run: `go test ./... -count=1 -short`
Expected: All tests pass. Fix any compilation errors from updated interfaces.

---

## Task 8: Web Client — WS Envelope + Context Menu + Reply + Delete

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`
- Modify: `internal/controllers/http/handlers/web/templates/chat.html`
- Modify: `internal/controllers/http/handlers/web/static/css/chat.css`

- [ ] **Step 1: Update chat.html — add reply preview and context menu DOM**

In `internal/controllers/http/handlers/web/templates/chat.html`, add reply preview above the composer `<textarea>`:

```html
<div class="conversation__composer">
    <div class="conversation__reply-preview" id="reply-preview" style="display: none;">
        <div class="conversation__reply-preview-content">
            <span class="conversation__reply-preview-sender" id="reply-preview-sender"></span>
            <span class="conversation__reply-preview-text" id="reply-preview-text"></span>
        </div>
        <button class="conversation__reply-preview-close" id="btn-cancel-reply" aria-label="Отменить ответ">&times;</button>
    </div>
    <div class="conversation__composer-input">
```

Add context menu before closing `</body>`:

```html
<div class="context-menu" id="context-menu" style="display: none;">
    <div class="context-menu__item" id="ctx-reply">Ответить</div>
    <div class="context-menu__item" id="ctx-copy">Копировать текст</div>
    <div class="context-menu__item context-menu__item--danger" id="ctx-delete-self" style="display: none;">Удалить для себя</div>
    <div class="context-menu__item context-menu__item--danger" id="ctx-delete-all" style="display: none;">Удалить для всех</div>
</div>
```

- [ ] **Step 2: Update chat.css — styles**

In `internal/controllers/http/handlers/web/static/css/chat.css`, add:

```css
/* ═══ Context menu ═══ */
.context-menu {
    position: fixed;
    z-index: 1000;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
    min-width: 180px;
    padding: var(--space-xs) 0;
}

.context-menu__item {
    padding: var(--space-sm) var(--space-lg);
    cursor: pointer;
    font-size: var(--font-md);
    color: var(--text-primary);
}

.context-menu__item:hover {
    background: var(--bg-hover);
}

.context-menu__item--danger {
    color: var(--danger, #e74c3c);
}

/* ═══ Reply bubble ═══ */
.message-bubble__reply {
    padding: var(--space-xs) var(--space-sm);
    margin-bottom: var(--space-xs);
    border-left: 3px solid var(--accent);
    background: rgba(0, 0, 0, 0.05);
    border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
    cursor: pointer;
    font-size: var(--font-xs);
}

.message-bubble--own .message-bubble__reply {
    background: rgba(255, 255, 255, 0.15);
    border-left-color: rgba(255, 255, 255, 0.5);
}

.message-bubble__reply-sender {
    font-weight: 600;
    display: block;
    margin-bottom: 2px;
}

.message-bubble__reply-text {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 300px;
}

/* ═══ Reply preview in composer ═══ */
.conversation__reply-preview {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-sm) var(--space-md);
    background: var(--bg-card);
    border-left: 3px solid var(--accent);
    border-radius: var(--radius-sm);
    margin-bottom: var(--space-xs);
    font-size: var(--font-sm);
}

.conversation__reply-preview-sender {
    font-weight: 600;
    margin-right: var(--space-sm);
}

.conversation__reply-preview-text {
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
}

.conversation__reply-preview-close {
    background: none;
    border: none;
    font-size: var(--font-lg);
    cursor: pointer;
    color: var(--text-muted);
    padding: 0 var(--space-xs);
}
```

- [ ] **Step 3: Update chat.js — WS envelope dispatch**

Replace the `ws.onmessage` handler:

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

Extract existing `ws.onmessage` body into `handleNewMessage(message)`. Add:

```js
function handleMessageDeleted(payload) {
    const idx = messages.findIndex(m => m.id === payload.messageId);
    if (idx >= 0) {
        messages.splice(idx, 1);
    }

    if (selectedChatId === payload.chatId) {
        const bubble = document.querySelector(`.message-bubble[data-message-id="${payload.messageId}"]`);
        if (bubble) bubble.remove();
    }

    debouncedLoadChats();
}
```

- [ ] **Step 4: Update chat.js — data-message-id on bubbles**

In `createMessageBubble`, add `data-message-id` attribute:

```js
function createMessageBubble(message) {
    const isOwn = message.sender.id === currentUser.id;

    const bubble = document.createElement('div');
    bubble.className = 'message-bubble' + (isOwn ? ' message-bubble--own' : '');
    bubble.dataset.messageId = message.id;

    // Reply bubble:
    if (message.replyToMessage) {
        const replyDiv = document.createElement('div');
        replyDiv.className = 'message-bubble__reply';
        replyDiv.dataset.replyId = message.replyToMessage.id;

        const replySender = document.createElement('span');
        replySender.className = 'message-bubble__reply-sender';
        replySender.textContent = message.replyToMessage.sender.username;

        const replyText = document.createElement('span');
        replyText.className = 'message-bubble__reply-text';
        const maxLen = 100;
        replyText.textContent = message.replyToMessage.text.length > maxLen
            ? message.replyToMessage.text.slice(0, maxLen) + '...'
            : message.replyToMessage.text;

        replyDiv.appendChild(replySender);
        replyDiv.appendChild(replyText);
        replyDiv.addEventListener('click', () => scrollToMessage(message.replyToMessage.id));
        bubble.appendChild(replyDiv);
    }

    // ... existing header, text creation ...
```

- [ ] **Step 5: Update chat.js — sendMessage with replyToMessageId**

Add global state:

```js
let replyingToMessage = null; // {id, sender: {username}, text}
```

Update `sendMessage()`:

```js
function sendMessage() {
    const input = document.getElementById('message-input');
    const text = input.value.trim();
    if (!text || !selectedChatId || !ws || ws.readyState !== WebSocket.OPEN) return;

    const payload = {chatId: selectedChatId, text};
    if (replyingToMessage) {
        payload.replyToMessageId = replyingToMessage.id;
    }

    ws.send(JSON.stringify(payload));

    markAllAsRead();

    const optimisticMessage = {
        id: Date.now(),
        chatId: selectedChatId,
        text,
        createdAt: new Date().toISOString(),
        sender: {id: currentUser.id, username: currentUser.username},
        isRead: true,
        optimistic: true,
        replyToMessage: replyingToMessage ? {
            id: replyingToMessage.id,
            sender: replyingToMessage.sender,
            text: replyingToMessage.text,
        } : undefined,
    };

    messages.push(optimisticMessage);
    appendMessageBubble(optimisticMessage);
    scrollToBottom();
    cancelReply();

    input.value = '';
    document.getElementById('btn-send').disabled = true;
    input.focus();
    loadChats().catch(console.error);
}
```

- [ ] **Step 6: Update chat.js — reply preview functions**

```js
function setReply(message) {
    replyingToMessage = message;

    const preview = document.getElementById('reply-preview');
    document.getElementById('reply-preview-sender').textContent = message.sender.username;
    document.getElementById('reply-preview-text').textContent = message.text;
    preview.style.display = '';

    document.getElementById('message-input').focus();
}

function cancelReply() {
    replyingToMessage = null;

    document.getElementById('reply-preview').style.display = 'none';
}
```

Add listener in initialization:

```js
document.getElementById('btn-cancel-reply').addEventListener('click', cancelReply);
```

- [ ] **Step 7: Update chat.js — context menu**

```js
let contextMenuMessageId = null;

function setupContextMenu() {
    const menu = document.getElementById('context-menu');
    let longTapTimer = null;

    document.getElementById('messages-list').addEventListener('contextmenu', (e) => {
        const bubble = e.target.closest('.message-bubble');
        if (!bubble) return;

        e.preventDefault();
        showContextMenu(bubble, e.clientX, e.clientY);
    });

    // Long tap for mobile:
    document.getElementById('messages-list').addEventListener('touchstart', (e) => {
        const bubble = e.target.closest('.message-bubble');
        if (!bubble) return;

        longTapTimer = setTimeout(() => {
            e.preventDefault();
            const touch = e.touches[0];
            showContextMenu(bubble, touch.clientX, touch.clientY);
        }, 500);
    });

    document.getElementById('messages-list').addEventListener('touchend', () => {
        clearTimeout(longTapTimer);
    });

    document.getElementById('messages-list').addEventListener('touchmove', () => {
        clearTimeout(longTapTimer);
    });

    document.addEventListener('click', () => hideContextMenu());
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') hideContextMenu();
    });

    document.getElementById('ctx-reply').addEventListener('click', () => {
        const msg = messages.find(m => m.id === contextMenuMessageId);
        if (msg) setReply(msg);
        hideContextMenu();
    });

    document.getElementById('ctx-copy').addEventListener('click', () => {
        const msg = messages.find(m => m.id === contextMenuMessageId);
        if (msg) navigator.clipboard.writeText(msg.text);
        hideContextMenu();
    });

    document.getElementById('ctx-delete-self').addEventListener('click', () => {
        deleteMessage(contextMenuMessageId, false);
        hideContextMenu();
    });

    document.getElementById('ctx-delete-all').addEventListener('click', () => {
        deleteMessage(contextMenuMessageId, true);
        hideContextMenu();
    });
}

function showContextMenu(bubble, x, y) {
    const menu = document.getElementById('context-menu');
    contextMenuMessageId = Number(bubble.dataset.messageId);
    const msg = messages.find(m => m.id === contextMenuMessageId);

    // Show/hide delete options:
    const isOwn = msg && msg.sender.id === currentUser.id;
    document.getElementById('ctx-delete-self').style.display = '';
    document.getElementById('ctx-delete-all').style.display = isOwn ? '' : 'none';

    menu.style.left = x + 'px';
    menu.style.top = y + 'px';
    menu.style.display = '';

    // Keep menu within viewport:
    requestAnimationFrame(() => {
        const rect = menu.getBoundingClientRect();
        if (rect.right > window.innerWidth) {
            menu.style.left = (window.innerWidth - rect.width - 8) + 'px';
        }
        if (rect.bottom > window.innerHeight) {
            menu.style.top = (window.innerHeight - rect.height - 8) + 'px';
        }
    });
}

function hideContextMenu() {
    document.getElementById('context-menu').style.display = 'none';
    contextMenuMessageId = null;
}

async function deleteMessage(messageId, forAll) {
    try {
        const response = await fetch(`/api/messages/${messageId}`, {
            method: 'DELETE',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({forAll}),
        });

        if (!response.ok) return;

        // Remove locally:
        const idx = messages.findIndex(m => m.id === messageId);
        if (idx >= 0) messages.splice(idx, 1);

        const bubble = document.querySelector(`.message-bubble[data-message-id="${messageId}"]`);
        if (bubble) bubble.remove();

        debouncedLoadChats();
    } catch (err) {
        console.error('Failed to delete message:', err);
    }
}
```

- [ ] **Step 8: Update chat.js — scrollToMessage**

```js
async function scrollToMessage(messageId) {
    const container = document.getElementById('messages-list');

    let target = container.querySelector(`.message-bubble[data-message-id="${messageId}"]`);

    // Load more messages until we find the target:
    while (!target && hasMoreMessages) {
        await loadMoreMessages();
        target = container.querySelector(`.message-bubble[data-message-id="${messageId}"]`);
    }

    if (target) {
        target.scrollIntoView({behavior: 'smooth', block: 'center'});
        target.classList.add('message-bubble--highlight');
        setTimeout(() => target.classList.remove('message-bubble--highlight'), 2000);
    }
}
```

Add highlight style in CSS:

```css
.message-bubble--highlight {
    animation: highlight-fade 2s ease-out;
}

@keyframes highlight-fade {
    0% { box-shadow: 0 0 0 3px var(--accent); }
    100% { box-shadow: none; }
}
```

- [ ] **Step 9: Wire setupContextMenu in initialization**

In the `DOMContentLoaded` block of `chat.js`, add `setupContextMenu()` call alongside existing `setupSendMessage()`.

- [ ] **Step 10: Test in browser**

1. Start: `task local`
2. Open http://localhost:8080
3. Send a message → verify envelope format works (new_message event)
4. Right-click a message → verify context menu appears
5. Click "Ответить" → verify reply preview shows, send a reply → verify reply bubble renders
6. Click reply bubble → verify scroll to original
7. Click "Копировать текст" → verify clipboard
8. Click "Удалить для себя" on own message → verify removal
9. Click "Удалить для всех" on own message → verify removal for both users
10. Long tap on mobile → verify context menu

---

## Task 9: GUI — Domain & WS Changes

**Files:**
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/domains/message.go`
- Create: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/domains/ws_event.go`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/interfaces/repositories.go`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/interfaces/usecases.go`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/repositories/ws/repository.go`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/usecases/usecases.go`

- [ ] **Step 1: Update GUI Message domain**

In `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/domains/message.go`, add `ReplyToMessage` field:

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

- [ ] **Step 2: Create GUI WSEvent domain**

Create `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/domains/ws_event.go`:

```go
package domains

import "encoding/json"

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

- [ ] **Step 3: Update WebSocketsRepository interface**

In `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/interfaces/repositories.go`, change `ReadMessage` to `ReadEvent`:

```go
type WebSocketsRepository interface {
	Connect(ctx context.Context, accessToken string) error
	Close() error
	ReadEvent(ctx context.Context) (*domains.WSEvent, error)
	WriteMessage(ctx context.Context, message domains.Message) error
}
```

- [ ] **Step 4: Update WS repository implementation**

In `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/repositories/ws/repository.go`:

Change `messagesChan` to `eventsChan`:

```go
type Repository struct {
	baseURL    string
	logger     logging.Logger
	ws         *websocket.Conn
	mu         sync.Mutex
	eventsChan chan *domains.WSEvent
	errChan    chan error
}
```

Update `Connect`:

```go
r.eventsChan = make(chan *domains.WSEvent, readMessagesBufferSize)
```

Update `readLoop`:

```go
func (r *Repository) readLoop() {
	defer close(r.eventsChan)
	defer close(r.errChan)

	for {
		var event domains.WSEvent
		if err := r.ws.ReadJSON(&event); err != nil {
			if websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				r.errChan <- customerrors.ErrWebsocketClosed
			} else {
				r.errChan <- fmt.Errorf("%w: %w", customerrors.ErrWebsocket, err)
			}

			if err = r.Close(); err != nil {
				logging.LogError(r.logger, "failed to close ws connection", err)
			}

			return
		}

		r.eventsChan <- &event
	}
}
```

Rename `ReadMessage` to `ReadEvent`:

```go
func (r *Repository) ReadEvent(ctx context.Context) (*domains.WSEvent, error) {
	if r.ws == nil {
		return nil, fmt.Errorf("%w: connection was not established", customerrors.ErrWebsocket)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-r.errChan:
		return nil, err
	case event, ok := <-r.eventsChan:
		if !ok {
			select {
			case err := <-r.errChan:
				return nil, err
			default:
				return nil, customerrors.ErrWebsocketClosed
			}
		}

		return event, nil
	}
}
```

- [ ] **Step 5: Update UseCases interface**

In `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/interfaces/usecases.go`, replace `ReadMessage` with `ReadEvent` and add `DeleteMessage`:

```go
ReadEvent(ctx context.Context) (*domains.WSEvent, error)
DeleteMessage(ctx context.Context, messageID uint64, forAll bool) error
```

- [ ] **Step 6: Update UseCases implementation**

In `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/usecases/usecases.go`:

Replace `ReadMessage` with `ReadEvent`:

```go
func (u *UseCases) ReadEvent(ctx context.Context) (*domains.WSEvent, error) {
	tokens, err := u.tokensRepository.Load()
	if err != nil {
		return nil, err
	}

	if err = u.wsRepository.Connect(ctx, tokens.AccessToken); err != nil {
		return nil, err
	}

	return u.wsRepository.ReadEvent(ctx)
}
```

Add `DeleteMessage`:

```go
func (u *UseCases) DeleteMessage(ctx context.Context, messageID uint64, forAll bool) error {
	tokens, err := u.tokensRepository.Load()
	if err != nil {
		return err
	}

	return u.chatsRepository.DeleteMessage(ctx, tokens.AccessToken, messageID, forAll)
}
```

Note: This requires adding `DeleteMessage` to `ChatsRepository` interface and implementing the HTTP call. Add to the interface:

```go
type ChatsRepository interface {
	// ... existing methods ...
	DeleteMessage(ctx context.Context, accessToken string, messageID uint64, forAll bool) error
}
```

And implement in `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/repositories/chats/repository.go`:

```go
func (r *Repository) DeleteMessage(
	ctx context.Context,
	accessToken string,
	messageID uint64,
	forAll bool,
) error {
	body, err := json.Marshal(map[string]bool{"forAll": forAll})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/messages/%d", r.baseURL, messageID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: accessToken})

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			logging.LogErrorContext(ctx, r.logger, "Failed to close response body", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] **Step 7: Verify GUI compilation**

Run from KhodFeltsChatGUI: `go build ./...`
Expected: Build succeeds.

---

## Task 10: GUI — Handler & Frontend

**Files:**
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/v2/handlers/chat/handler.go`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/constants/index.js`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/components/ChatView/ChatView.js`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/components/ChatView/ChatView.vue`
- Modify: `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/components/ChatView/ChatView.css`

- [ ] **Step 1: Update handler — WS event dispatch**

In `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/internal/v2/handlers/chat/handler.go`:

Add import `"encoding/json"` and update constants:

```go
const (
	refreshTokensInterval = 1 * time.Minute
	updateChatsInterval   = 5 * time.Second

	chatsUpdatedEventName    = "chats_updated"
	newMessageEventName      = "new_message"
	messageDeletedEventName  = "message_deleted"
)
```

Update `readMessages()`:

```go
func (h *Handler) readMessages() {
	defer h.wg.Done()

	for {
		select {
		case <-h.goroutinesCtx.Done():
			return
		default:
			event, err := h.useCases.ReadEvent(h.goroutinesCtx)
			if err != nil {
				if errors.Is(err, customerrors.ErrWebsocketClosed) {
					logging.LogInfo(
						h.logger,
						"readMessages goroutine stopped: connection closed",
					)

					return
				}

				logging.LogErrorContext(
					h.goroutinesCtx,
					h.logger,
					"Failed to read event",
					err,
				)

				continue
			}

			switch event.Type {
			case domains.WSEventNewMessage:
				var message domains.Message
				if err = json.Unmarshal(event.Payload, &message); err != nil {
					logging.LogErrorContext(
						h.goroutinesCtx,
						h.logger,
						"Failed to unmarshal new message",
						err,
					)

					continue
				}

				message.CreatedAt = message.CreatedAt.In(common.Timezone)
				message.UpdatedAt = message.UpdatedAt.In(common.Timezone)

				runtime.EventsEmit(h.wailsCtx, newMessageEventName, message)
			case domains.WSEventMessageDeleted:
				var payload domains.MessageDeletedPayload
				if err = json.Unmarshal(event.Payload, &payload); err != nil {
					logging.LogErrorContext(
						h.goroutinesCtx,
						h.logger,
						"Failed to unmarshal message_deleted",
						err,
					)

					continue
				}

				runtime.EventsEmit(h.wailsCtx, messageDeletedEventName, payload)
			}
		}
	}
}
```

- [ ] **Step 2: Update handler — SendMessage with replyToMessageID**

```go
func (h *Handler) SendMessage(chatID uint64, text string, replyToMessageID uint64) error {
	ctx := context.Background()

	sender, err := h.useCases.GetCurrentUser(ctx)
	if err != nil {
		return err
	}

	message := domains.Message{
		ChatID: chatID,
		Text:   text,
		Sender: domains.User{
			ID: sender.ID,
		},
		CreatedAt: time.Now().In(common.Timezone),
		UpdatedAt: time.Now().In(common.Timezone),
		IsRead:    true,
	}

	if replyToMessageID > 0 {
		message.ReplyToMessage = &domains.Message{ID: replyToMessageID}
	}

	return h.useCases.SendMessage(ctx, message)
}
```

- [ ] **Step 3: Add DeleteMessage handler method**

```go
func (h *Handler) DeleteMessage(messageID uint64, forAll bool) error {
	ctx := context.Background()

	return h.useCases.DeleteMessage(ctx, messageID, forAll)
}
```

- [ ] **Step 4: Update frontend constants**

In `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/constants/index.js`:

```js
export const WAILS_EVENT = Object.freeze({
    NEW_MESSAGE: 'new_message',
    MESSAGE_DELETED: 'message_deleted',
    CHATS_UPDATED: 'chats_updated',
    OPEN_CHAT: 'open_chat',
})
```

- [ ] **Step 5: Update ChatView.vue — reply bubble, reply preview, context menu**

In the message bubble `v-for` loop, add reply section before the header:

```html
<div
    v-for="(message, index) in messages"
    :key="message.id"
    :class="['message-bubble', {'message-bubble--own': message.sender.id === currentUser.id}]"
    :data-message-id="message.id"
    @contextmenu.prevent="showContextMenu($event, message)"
    @touchstart="startLongPress($event, message)"
    @touchend="cancelLongPress"
    @touchmove="cancelLongPress"
>
    <!-- Reply bubble -->
    <div
        v-if="message.replyToMessage"
        class="message-bubble__reply"
        @click="scrollToMessage(message.replyToMessage.id)"
    >
        <span class="message-bubble__reply-sender">{{ message.replyToMessage.sender.username }}</span>
        <span class="message-bubble__reply-text">{{ truncateText(message.replyToMessage.text, 100) }}</span>
    </div>

    <!-- Unread divider -->
    <!-- ... existing ... -->
```

Add reply preview above composer textarea:

```html
<div v-if="replyingToMessage" class="conversation__reply-preview">
    <div class="conversation__reply-preview-content">
        <span class="conversation__reply-preview-sender">{{ replyingToMessage.sender.username }}</span>
        <span class="conversation__reply-preview-text">{{ replyingToMessage.text }}</span>
    </div>
    <button class="conversation__reply-preview-close" @click="cancelReply" aria-label="Отменить ответ">&times;</button>
</div>
```

Add context menu at the end of template:

```html
<div
    v-if="contextMenu.visible"
    class="context-menu"
    :style="{left: contextMenu.x + 'px', top: contextMenu.y + 'px'}"
>
    <div class="context-menu__item" @click="replyToMessage">Ответить</div>
    <div class="context-menu__item" @click="copyMessageText">Копировать текст</div>
    <div class="context-menu__item context-menu__item--danger" @click="deleteMessageForSelf">Удалить для себя</div>
    <div
        v-if="contextMenu.message && contextMenu.message.sender.id === currentUser.id"
        class="context-menu__item context-menu__item--danger"
        @click="deleteMessageForAll"
    >Удалить для всех</div>
</div>
```

- [ ] **Step 6: Update ChatView.js — context menu, reply, delete logic**

Add reactive state:

```js
const replyingToMessage = ref(null)
const contextMenu = reactive({visible: false, x: 0, y: 0, message: null})
let longPressTimer = null
```

Add functions:

```js
function showContextMenu(event, message) {
    contextMenu.x = event.clientX
    contextMenu.y = event.clientY
    contextMenu.message = message
    contextMenu.visible = true
}

function hideContextMenu() {
    contextMenu.visible = false
    contextMenu.message = null
}

function startLongPress(event, message) {
    longPressTimer = setTimeout(() => {
        const touch = event.touches[0]
        showContextMenu({clientX: touch.clientX, clientY: touch.clientY}, message)
    }, 500)
}

function cancelLongPress() {
    clearTimeout(longPressTimer)
}

function replyToMessage() {
    replyingToMessage.value = contextMenu.message
    hideContextMenu()
    nextTick(() => textareaRef.value?.focus())
}

function cancelReply() {
    replyingToMessage.value = null
}

function copyMessageText() {
    if (contextMenu.message) {
        navigator.clipboard.writeText(contextMenu.message.text)
    }
    hideContextMenu()
}

async function deleteMessageForSelf() {
    if (!contextMenu.message) return
    const msgId = contextMenu.message.id
    hideContextMenu()

    try {
        await window.go.chat.Handler.DeleteMessage(msgId, false)
        messages.value = messages.value.filter(m => m.id !== msgId)
    } catch (err) {
        console.error('Delete failed:', err)
    }
}

async function deleteMessageForAll() {
    if (!contextMenu.message) return
    const msgId = contextMenu.message.id
    hideContextMenu()

    try {
        await window.go.chat.Handler.DeleteMessage(msgId, true)
        messages.value = messages.value.filter(m => m.id !== msgId)
    } catch (err) {
        console.error('Delete failed:', err)
    }
}

function truncateText(text, maxLen) {
    return text.length > maxLen ? text.slice(0, maxLen) + '...' : text
}

async function scrollToMessage(messageId) {
    let target = document.querySelector(`.message-bubble[data-message-id="${messageId}"]`)

    while (!target && hasMoreMessages.value) {
        await loadMoreMessages()
        target = document.querySelector(`.message-bubble[data-message-id="${messageId}"]`)
    }

    if (target) {
        target.scrollIntoView({behavior: 'smooth', block: 'center'})
        target.classList.add('message-bubble--highlight')
        setTimeout(() => target.classList.remove('message-bubble--highlight'), 2000)
    }
}
```

Update `sendMessage`:

```js
async function sendMessage() {
    const text = newMessage.value.trim()
    if (!text || !selectedChat.value) return

    const replyId = replyingToMessage.value ? replyingToMessage.value.id : 0

    try {
        await window.go.chat.Handler.SendMessage(selectedChat.value.id, text, replyId)
    } catch (err) {
        console.error('Send failed:', err)
        return
    }

    const optimistic = {
        id: Date.now(),
        chatId: selectedChat.value.id,
        text,
        createdAt: new Date().toISOString(),
        sender: {id: currentUser.value.id, username: currentUser.value.username},
        isRead: true,
        replyToMessage: replyingToMessage.value ? {
            id: replyingToMessage.value.id,
            sender: replyingToMessage.value.sender,
            text: replyingToMessage.value.text,
        } : undefined,
    }

    messages.value.push(optimistic)
    cancelReply()
    newMessage.value = ''
    await nextTick()
    scrollToBottom()
}
```

Add `message_deleted` event listener in `onMounted`:

```js
window.runtime.EventsOn(WAILS_EVENT.MESSAGE_DELETED, (payload) => {
    messages.value = messages.value.filter(m => m.id !== payload.messageId)
})
```

Add click-outside handler for context menu in `onMounted`:

```js
document.addEventListener('click', hideContextMenu)
```

And cleanup in `onUnmounted`:

```js
document.removeEventListener('click', hideContextMenu)
```

Return new functions/state from `setup`:

```js
return {
    // ... existing ...
    replyingToMessage, contextMenu,
    showContextMenu, hideContextMenu, startLongPress, cancelLongPress,
    replyToMessage, cancelReply, copyMessageText,
    deleteMessageForSelf, deleteMessageForAll,
    truncateText, scrollToMessage,
}
```

- [ ] **Step 7: Update ChatView.css — context menu and reply styles**

Add the same CSS classes as in the web version (Task 8, Step 2), adapted for Vue scoped styles.

- [ ] **Step 8: Verify GUI build**

Run from KhodFeltsChatGUI: `wails build` or `go build ./...`
Expected: Build succeeds.

- [ ] **Step 9: Test GUI**

1. Launch the desktop app
2. Send a message → verify it arrives via envelope (new_message type)
3. Right-click a message → context menu appears
4. Reply → reply preview shows, send → reply bubble renders
5. Copy text → clipboard works
6. Delete for self → message disappears locally
7. Delete for all → message disappears for other user too

---

## Task 11: Backend Tests

**Files:**
- Modify: existing test files in `internal/repositories/messages/`, `internal/services/messages/`, `internal/usecases/messages/`, `internal/controllers/http/handlers/api/messages/`, `internal/controllers/http/mappers/messages/`

- [ ] **Step 1: Update mapper tests**

In `internal/controllers/http/mappers/messages/messages_test.go`, add test cases for messages with and without `ReplyToMessage`.

- [ ] **Step 2: Update handler tests**

Add tests for the delete handler in `internal/controllers/http/handlers/api/messages/delete/handler_test.go` — test 204 success, 404 not found, 403 not author, 400 bad request.

- [ ] **Step 3: Update usecase tests**

In `internal/usecases/messages/` tests — test `DeleteMessage`: success for self, success for all (as author), failure for all (not author).

- [ ] **Step 4: Run all tests**

Run: `go test ./... -count=1`
Expected: All tests pass.

---

## Task 12: Documentation

- [ ] **Step 1: Update doc.md files**

Update `doc.md` in every directory where code was changed:
- `internal/domains/doc.md`
- `internal/errors/doc.md`
- `internal/interfaces/doc.md`
- `internal/repositories/messages/doc.md`
- `internal/services/messages/doc.md`
- `internal/usecases/messages/doc.md`
- `internal/controllers/http/handlers/api/messages/doc.md` (or create if missing)
- `internal/controllers/http/schemas/doc.md`
- `internal/controllers/http/mappers/messages/doc.md`

Document the new `DeleteMessage` flow, `WSEvent` envelope pattern, `ReplyToMessage` field, and `is_deleted` soft delete.
