# Notification Consents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-user notification consent management (bitmask) with UI toggles, refactor workers to use unified email/web-push architecture.

**Architecture:** Extend existing `Settings` domain and DB table with two bitmask columns (`email_consents`, `web_push_consents`). Replace three separate NATS workers (verify-email, forget-password, web-push-notification) with two unified workers (email-notification, web-push-notification) that use typed DTO envelopes with `json.RawMessage` payloads. Add collapsible "Уведомления" section in profile modal with toggles per notification type and channel.

**Tech Stack:** Go 1.24, PostgreSQL (squirrel), NATS, gomail, gorilla/mux, gorilla/websocket, vanilla JS/CSS

---

## File Map

### Backend — Modified Files

| File | Change |
|------|--------|
| `migrations/` | New migration: add `email_consents`, `web_push_consents` columns |
| `internal/domains/settings.go` | Add `NotificationConsent` type, constants, `HasConsent` func, extend `Settings` |
| `internal/domains/notifications.go` | Replace DTOs with typed envelope DTOs for email and web-push |
| `internal/config/config.go` | Replace `NATSSubjects` and `NATSWorkers` (remove verify/forget, add email-notification) |
| `internal/repositories/settings/repository.go` | Add `email_consents` and `web_push_consents` to Create/Update queries |
| `internal/interfaces/content_builders.go` | Add `NewMessageContentBuilder` interface |
| `internal/interfaces/repositories.go` | Add `SendNewMessageNotification` to `EmailsRepository` |
| `internal/interfaces/services.go` | Add `SendNewMessageNotification` to `NotificationsService` |
| `internal/interfaces/usecases.go` | Add `SendNewMessageNotification` to `NotificationsUseCases` |
| `internal/repositories/emails/repository.go` | Add `SendNewMessageNotification` method |
| `internal/services/notifications/service.go` | Add `SendNewMessageNotification` method |
| `internal/usecases/notifications/usecases.go` | Add `SendNewMessageNotification` method |
| `internal/services/auth/service.go` | Update NATS publish to use `EmailNotificationDTO` envelope |
| `internal/controllers/http/handlers/api/ws/ws.go` | Update NATS publish to use `WebPushNotificationDTO` envelope |
| `internal/controllers/http/schemas/settings.go` | Add `EmailConsents`, `WebPushConsents` fields |
| `internal/controllers/http/mappers/settings/settings.go` | Map new consent fields |
| `cmd/main.go` | Replace 3 workers with 2, wire new contentbuilder and dependencies |

### Backend — New Files

| File | Purpose |
|------|---------|
| `internal/contentbuilders/new_message/builder.go` | Email content builder for new message notifications |
| `internal/workers/handlers/builders/email_notification/builder.go` | Unified email worker: dispatches by `Type`, checks consents for non-system types |
| `internal/workers/handlers/builders/web_push_notification/builder.go` | Refactored: uses typed envelope, checks consents before sending |

### Backend — Deleted Files

| File | Reason |
|------|--------|
| `internal/workers/handlers/builders/forget_password/` | Merged into unified email worker |
| `internal/workers/handlers/builders/verify_email/` | Merged into unified email worker |

### Frontend — Modified Files

| File | Change |
|------|--------|
| `internal/controllers/http/handlers/web/templates/navbar.html` | Replace push status row with collapsible notifications section with toggles |
| `internal/controllers/http/handlers/web/static/js/navbar.js` | Add notification toggles logic, auto-sync browser push subscription |
| `internal/controllers/http/handlers/web/static/css/navbar.css` | Add styles for notification section and toggles |

---

### Task 1: DB Migration — Add Consent Columns

**Files:**
- Create: `migrations/20260516000000_notification_consents.sql`

- [ ] **Step 1: Write the migration file**

```sql
-- +goose Up
ALTER TABLE settings
    ADD COLUMN email_consents     INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN web_push_consents  INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE settings
    DROP COLUMN IF EXISTS email_consents,
    DROP COLUMN IF EXISTS web_push_consents;
```

- [ ] **Step 2: Apply migration**

Run: `task migrate-up`
Expected: Migration applied, `settings` table now has `email_consents` and `web_push_consents` columns.

- [ ] **Step 3: Commit**

```
feat: add notification consent bitmask columns to settings
```

---

### Task 2: Domain — NotificationConsent Type and Updated Settings

**Files:**
- Modify: `internal/domains/settings.go`
- Modify: `internal/domains/notifications.go`

- [ ] **Step 1: Add NotificationConsent type to settings.go**

Add the `NotificationConsent` type, consent constants, and helper function. Extend `Settings` struct with two new fields:

```go
package domains

import "time"

type ThemeType int

const (
	ThemeLight ThemeType = iota
	ThemeDark
)

type NotificationConsent int

const (
	ConsentNewMessage NotificationConsent = 1 << iota
)

func HasConsent(mask, consent NotificationConsent) bool {
	return mask&consent != 0
}

type Settings struct {
	ID              uint64              `json:"id"`
	UserID          uint64              `json:"userId"`
	Theme           ThemeType           `json:"theme"`
	EmailConsents   NotificationConsent `json:"emailConsents"`
	WebPushConsents NotificationConsent `json:"webPushConsents"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
}
```

- [ ] **Step 2: Replace DTOs in notifications.go**

Replace the three separate DTOs with typed envelope DTOs:

```go
package domains

import "encoding/json"

// Email notification types
type EmailNotificationType string

const (
	EmailTypeVerifyEmail    EmailNotificationType = "verify_email"
	EmailTypeForgetPassword EmailNotificationType = "forget_password"
	EmailTypeNewMessage     EmailNotificationType = "new_message"
)

type EmailNotificationDTO struct {
	Type       EmailNotificationType `json:"type"`
	ReceiverID uint64                `json:"receiverId"`
	Payload    json.RawMessage       `json:"payload"`
}

// Web push notification types
type WebPushNotificationType string

const (
	WebPushTypeNewMessage WebPushNotificationType = "new_message"
)

type WebPushNotificationDTO struct {
	Type       WebPushNotificationType `json:"type"`
	ReceiverID uint64                  `json:"receiverId"`
	Payload    json.RawMessage         `json:"payload"`
}

// Type-specific payloads
type NewMessagePayload struct {
	MessageID uint64 `json:"messageId"`
	ChatID    uint64 `json:"chatId"`
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/domains/...`
Expected: All existing domain tests pass (settings_test.go may need updates if it references old struct).

- [ ] **Step 4: Commit**

```
feat: add NotificationConsent bitmask type and unified notification DTOs
```

---

### Task 3: Config — Update NATS Subjects and Workers

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Update NATSSubjects and NATSWorkers**

In `config.go`, replace `NATSSubjects`:

```go
type NATSSubjects struct {
	EmailNotification   string
	WebPushNotification string
}
```

Replace `NATSWorkers`:

```go
type NATSWorkers struct {
	EmailNotification   NATSWorker
	WebPushNotification NATSWorker
}
```

- [ ] **Step 2: Update the config initialization** (around line 172)

Replace the subjects initialization:

```go
Subjects: NATSSubjects{
    EmailNotification: loadenv.GetEnv(
        "NATS_EMAIL_NOTIFICATION_SUBJECT",
        "email-notification",
    ),
    WebPushNotification: loadenv.GetEnv(
        "NATS_PUSH_NOTIFICATION_SUBJECT",
        "web-push-notification",
    ),
},
```

Update workers initialization similarly — replace `VerifyEmail` and `ForgetPassword` workers with single `EmailNotification` worker.

- [ ] **Step 3: Fix compilation errors across the codebase**

Grep for `Subjects.VerifyEmail`, `Subjects.ForgetPassword`, `Workers.VerifyEmail`, `Workers.ForgetPassword` and update all references. Key files:
- `internal/services/auth/service.go` (publishes to verify-email and forget-password subjects)
- `cmd/main.go` (creates workers)
- Tests referencing old config fields

- [ ] **Step 4: Run compilation check**

Run: `go build ./...`
Expected: Build succeeds (some tests may still fail — will fix in later tasks).

- [ ] **Step 5: Commit**

```
refactor: consolidate NATS subjects into email-notification and web-push-notification
```

---

### Task 4: Repository — Settings CRUD with Consent Columns

**Files:**
- Modify: `internal/repositories/settings/repository.go`

- [ ] **Step 1: Add column constants**

Add to the const block:

```go
emailConsentsColumnName   = "email_consents"
webPushConsentsColumnName = "web_push_consents"
```

- [ ] **Step 2: Update CreateSettings**

Add the new columns to the INSERT:

```go
func (repo *Repository) CreateSettings(ctx context.Context, settings domains.Settings) error {
	stmt, params, err := sq.
		Insert(settingsTableName).
		Columns(
			userIDColumnName,
			themeColumnName,
			emailConsentsColumnName,
			webPushConsentsColumnName,
		).
		Values(
			settings.UserID,
			settings.Theme,
			settings.EmailConsents,
			settings.WebPushConsents,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
```

- [ ] **Step 3: Update UpdateSettings**

Add the new columns to the UPDATE:

```go
func (repo *Repository) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) error {
	builder := sq.
		Update(settingsTableName).
		Where(sq.Eq{userIDColumnName: settings.UserID}).
		Set(themeColumnName, settings.Theme).
		Set(emailConsentsColumnName, settings.EmailConsents).
		Set(webPushConsentsColumnName, settings.WebPushConsents).
		Set(updatedAtColumnName, time.Now()).
		PlaceholderFormat(sq.Dollar)

	stmt, params, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
```

Note: `GetSettingsByUserID` uses `SELECT *` and `pg.GetEntityColumns(settings)` — the new fields will be scanned automatically because they match the `Settings` struct field order after migration.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/repositories/settings/...`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat: add consent columns to settings repository CRUD
```

---

### Task 5: Schema, Mapper — Add Consent Fields to API Response

**Files:**
- Modify: `internal/controllers/http/schemas/settings.go`
- Modify: `internal/controllers/http/mappers/settings/settings.go`

- [ ] **Step 1: Update Settings schema**

```go
package schemas

// Settings represents user settings.
// swagger:model
type Settings struct {
	// Theme of the user interface. 0 = Light, 1 = Dark.
	// required: true
	// nullable: false
	// minimum: 0
	// maximum: 1
	Theme int `json:"theme"`

	// Email notification consents bitmask.
	// required: true
	// nullable: false
	EmailConsents int `json:"emailConsents"`

	// Web push notification consents bitmask.
	// required: true
	// nullable: false
	WebPushConsents int `json:"webPushConsents"`
}

// UpdateSettingsInput
// swagger:parameters UpdateSettings
type UpdateSettingsInput struct {
	// Settings data to update
	// required: true
	// nullable: false
	// in: body
	Body Settings
}
```

- [ ] **Step 2: Update mapper**

```go
func MapSettings(settings domains.Settings) schemas.Settings {
	return schemas.Settings{
		Theme:           int(settings.Theme),
		EmailConsents:   int(settings.EmailConsents),
		WebPushConsents: int(settings.WebPushConsents),
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/controllers/http/mappers/settings/...`
Expected: PASS (or fix test to include new fields).

- [ ] **Step 4: Commit**

```
feat: expose consent bitmasks in settings API response
```

---

### Task 6: New Message Content Builder for Email

**Files:**
- Create: `internal/contentbuilders/new_message/builder.go`
- Modify: `internal/interfaces/content_builders.go`

- [ ] **Step 1: Add NewMessageContentBuilder interface**

In `internal/interfaces/content_builders.go`, add:

```go
//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/new_message_content_builder.go -package=mockcontentbuilders -exclude_interfaces=VerifyEmailContentBuilder,ForgetPasswordContentBuilder
type NewMessageContentBuilder interface {
	Subject() string
	Body(ctx context.Context, senderUsername, chatName string) (string, error)
}
```

Add to the `ContentBuilders` struct:

```go
type ContentBuilders struct {
	VerifyEmail    VerifyEmailContentBuilder
	ForgetPassword ForgetPasswordContentBuilder
	NewMessage     NewMessageContentBuilder
}
```

- [ ] **Step 2: Create the content builder**

```go
package new_message

import (
	"context"
	"fmt"
)

type ContentBuilder struct{}

func New() *ContentBuilder {
	return &ContentBuilder{}
}

func (b *ContentBuilder) Subject() string {
	return "Новое сообщение в чате"
}

func (b *ContentBuilder) Body(_ context.Context, senderUsername, chatName string) (string, error) {
	template := `<p>Добрый день!</p>
<p>Вы получили новое сообщение от пользователя <b>%s</b> в чате <b>%s</b>.</p>
<p>Перейдите в приложение, чтобы прочитать его.</p>
<p>С уважением,<br>
команда KFC Chat.</p>
`
	return fmt.Sprintf(template, senderUsername, chatName), nil
}
```

- [ ] **Step 3: Commit**

```
feat: add new message email content builder
```

---

### Task 7: Add SendNewMessageNotification to Email Chain

**Files:**
- Modify: `internal/interfaces/repositories.go` — add `SendNewMessageNotification` to `EmailsRepository`
- Modify: `internal/interfaces/services.go` — add to `NotificationsService`
- Modify: `internal/interfaces/usecases.go` — add to `NotificationsUseCases`
- Modify: `internal/repositories/emails/repository.go` — implement
- Modify: `internal/services/notifications/service.go` — implement
- Modify: `internal/usecases/notifications/usecases.go` — implement

- [ ] **Step 1: Update EmailsRepository interface**

Add to `EmailsRepository`:

```go
type EmailsRepository interface {
	SendVerifyEmailMessage(ctx context.Context, user domains.User) error
	SendForgetPasswordMessage(ctx context.Context, user domains.User) error
	SendNewMessageNotification(ctx context.Context, recipientEmail, senderUsername, chatName string) error
}
```

- [ ] **Step 2: Update NotificationsService interface**

```go
type NotificationsService interface {
	EmailsRepository
}
```

This already embeds `EmailsRepository`, so the new method is inherited automatically.

- [ ] **Step 3: Update NotificationsUseCases interface**

Add to `NotificationsUseCases`:

```go
type NotificationsUseCases interface {
	SendForgetPasswordMessage(ctx context.Context, userID uint64) error
	SendVerifyEmailMessage(ctx context.Context, userID uint64) error
	SendNewMessageNotification(ctx context.Context, receiverID uint64, senderUsername, chatName string) error
}
```

- [ ] **Step 4: Implement in emails repository**

Add to `internal/repositories/emails/repository.go`:

```go
func (repo *Repository) SendNewMessageNotification(
	ctx context.Context,
	recipientEmail, senderUsername, chatName string,
) error {
	body, err := repo.contentBuilders.NewMessage.Body(ctx, senderUsername, chatName)
	if err != nil {
		return err
	}

	return repo.send(
		ctx,
		repo.contentBuilders.NewMessage.Subject(),
		body,
		[]string{recipientEmail},
	)
}
```

- [ ] **Step 5: Implement in notifications service**

Add to `internal/services/notifications/service.go`:

```go
func (s *Service) SendNewMessageNotification(
	ctx context.Context,
	recipientEmail, senderUsername, chatName string,
) error {
	return s.emailsRepository.SendNewMessageNotification(ctx, recipientEmail, senderUsername, chatName)
}
```

- [ ] **Step 6: Implement in notifications usecases**

Add to `internal/usecases/notifications/usecases.go`:

```go
func (u *UseCases) SendNewMessageNotification(
	ctx context.Context,
	receiverID uint64,
	senderUsername, chatName string,
) error {
	user, err := u.usersService.GetUserByID(ctx, receiverID)
	if err != nil {
		return err
	}

	return u.notificationsService.SendNewMessageNotification(ctx, user.Email, senderUsername, chatName)
}
```

- [ ] **Step 7: Run build**

Run: `go build ./...`
Expected: Build succeeds.

- [ ] **Step 8: Commit**

```
feat: add SendNewMessageNotification through email chain
```

---

### Task 8: Unified Email Worker

**Files:**
- Create: `internal/workers/handlers/builders/email_notification/builder.go`
- Delete: `internal/workers/handlers/builders/forget_password/builder.go`
- Delete: `internal/workers/handlers/builders/verify_email/builder.go`

- [ ] **Step 1: Create unified email notification worker**

```go
package email_notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	"github.com/nats-io/nats.go"
)

type Builder struct {
	notificationsUseCases interfaces.NotificationsUseCases
	settingsUseCases      interfaces.SettingsUseCases
	logger                logging.Logger
}

func New(
	notificationsUseCases interfaces.NotificationsUseCases,
	settingsUseCases interfaces.SettingsUseCases,
	logger logging.Logger,
) *Builder {
	return &Builder{
		notificationsUseCases: notificationsUseCases,
		settingsUseCases:      settingsUseCases,
		logger:                logger,
	}
}

func (b *Builder) MessageHandler(ctx context.Context) interfaces.MessageHandler {
	return func(message *nats.Msg) {
		var dto domains.EmailNotificationDTO
		if err := json.Unmarshal(message.Data, &dto); err != nil {
			logging.LogError(b.logger, "Failed to unmarshal email notification message", err)

			return
		}

		switch dto.Type {
		case domains.EmailTypeVerifyEmail:
			b.handleVerifyEmail(ctx, dto)
		case domains.EmailTypeForgetPassword:
			b.handleForgetPassword(ctx, dto)
		case domains.EmailTypeNewMessage:
			b.handleNewMessage(ctx, dto)
		default:
			logging.LogError(
				b.logger,
				fmt.Sprintf("Unknown email notification type: %s", dto.Type),
				nil,
			)
		}
	}
}

func (b *Builder) handleVerifyEmail(ctx context.Context, dto domains.EmailNotificationDTO) {
	if err := b.notificationsUseCases.SendVerifyEmailMessage(ctx, dto.ReceiverID); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send verify-email to User with ID=%d", dto.ReceiverID),
			err,
		)
	}
}

func (b *Builder) handleForgetPassword(ctx context.Context, dto domains.EmailNotificationDTO) {
	if err := b.notificationsUseCases.SendForgetPasswordMessage(ctx, dto.ReceiverID); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send forget-password to User with ID=%d", dto.ReceiverID),
			err,
		)
	}
}

func (b *Builder) handleNewMessage(ctx context.Context, dto domains.EmailNotificationDTO) {
	settings, err := b.settingsUseCases.GetSettingsByUserID(ctx, dto.ReceiverID)
	if err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to get settings for User with ID=%d", dto.ReceiverID),
			err,
		)

		return
	}

	if !domains.HasConsent(settings.EmailConsents, domains.ConsentNewMessage) {
		return
	}

	var payload domains.NewMessagePayload
	if err = json.Unmarshal(dto.Payload, &payload); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal new message payload", err)

		return
	}

	if err = b.notificationsUseCases.SendNewMessageNotification(
		ctx,
		dto.ReceiverID,
		"", // senderUsername — will need to be resolved or included in payload
		"", // chatName — same
	); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send new-message email to User with ID=%d", dto.ReceiverID),
			err,
		)
	}
}
```

**Note:** The `handleNewMessage` needs sender/chat info. Two options:
1. Add `SenderUsername` and `ChatName` to `NewMessagePayload` (recommended — publisher already has this data)
2. Resolve from IDs in the worker

Update `NewMessagePayload` to include these fields:

```go
type NewMessagePayload struct {
	MessageID      uint64 `json:"messageId"`
	ChatID         uint64 `json:"chatId"`
	SenderUsername string `json:"senderUsername"`
	ChatName       string `json:"chatName"`
}
```

Then the handler becomes:

```go
func (b *Builder) handleNewMessage(ctx context.Context, dto domains.EmailNotificationDTO) {
	settings, err := b.settingsUseCases.GetSettingsByUserID(ctx, dto.ReceiverID)
	if err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to get settings for User with ID=%d", dto.ReceiverID),
			err,
		)

		return
	}

	if !domains.HasConsent(settings.EmailConsents, domains.ConsentNewMessage) {
		return
	}

	var payload domains.NewMessagePayload
	if err = json.Unmarshal(dto.Payload, &payload); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal new message payload", err)

		return
	}

	if err = b.notificationsUseCases.SendNewMessageNotification(
		ctx,
		dto.ReceiverID,
		payload.SenderUsername,
		payload.ChatName,
	); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send new-message email to User with ID=%d", dto.ReceiverID),
			err,
		)
	}
}
```

- [ ] **Step 2: Delete old workers**

Delete directories:
- `internal/workers/handlers/builders/forget_password/`
- `internal/workers/handlers/builders/verify_email/`

- [ ] **Step 3: Run build**

Run: `go build ./...`
Expected: Build succeeds (cmd/main.go will be updated in Task 10).

- [ ] **Step 4: Commit**

```
refactor: replace separate email workers with unified email notification worker
```

---

### Task 9: Refactor Web Push Worker — Add Consent Check

**Files:**
- Modify: `internal/workers/handlers/builders/web_push_notification/builder.go`

- [ ] **Step 1: Add SettingsUseCases dependency and consent check**

The web push worker now needs to:
1. Deserialize the new `WebPushNotificationDTO` envelope
2. Check `WebPushConsents` before sending
3. Deserialize payload for type-specific data

```go
package web_push_notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	"github.com/nats-io/nats.go"
)

type Builder struct {
	webPushSubscriptionsUseCases interfaces.WebPushSubscriptionsUseCases
	messagesUseCases             interfaces.MessagesUseCases
	settingsUseCases             interfaces.SettingsUseCases
	logger                       logging.Logger
}

func New(
	webPushSubscriptionsUseCases interfaces.WebPushSubscriptionsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	settingsUseCases interfaces.SettingsUseCases,
	logger logging.Logger,
) *Builder {
	return &Builder{
		webPushSubscriptionsUseCases: webPushSubscriptionsUseCases,
		messagesUseCases:             messagesUseCases,
		settingsUseCases:             settingsUseCases,
		logger:                       logger,
	}
}

func (b *Builder) MessageHandler(ctx context.Context) interfaces.MessageHandler {
	return func(message *nats.Msg) {
		var dto domains.WebPushNotificationDTO
		if err := json.Unmarshal(message.Data, &dto); err != nil {
			logging.LogError(b.logger, "Failed to unmarshal web-push notification message", err)

			return
		}

		switch dto.Type {
		case domains.WebPushTypeNewMessage:
			b.handleNewMessage(ctx, dto)
		default:
			logging.LogError(
				b.logger,
				fmt.Sprintf("Unknown web-push notification type: %s", dto.Type),
				nil,
			)
		}
	}
}

func (b *Builder) handleNewMessage(ctx context.Context, dto domains.WebPushNotificationDTO) {
	settings, err := b.settingsUseCases.GetSettingsByUserID(ctx, dto.ReceiverID)
	if err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to get settings for User with ID=%d", dto.ReceiverID),
			err,
		)

		return
	}

	if !domains.HasConsent(settings.WebPushConsents, domains.ConsentNewMessage) {
		return
	}

	var payload domains.NewMessagePayload
	if err = json.Unmarshal(dto.Payload, &payload); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal new message payload", err)

		return
	}

	msg, err := b.messagesUseCases.GetMessageByID(ctx, dto.ReceiverID, payload.MessageID)
	if err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to get message with ID=%d", payload.MessageID),
			err,
		)

		return
	}

	subscriptions, err := b.webPushSubscriptionsUseCases.GetWebPushSubscriptionsByUserID(
		ctx,
		dto.ReceiverID,
	)
	if err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to get push subscriptions for User with ID=%d", dto.ReceiverID),
			err,
		)

		return
	}

	for _, sub := range subscriptions {
		if err = b.webPushSubscriptionsUseCases.SendWebPushNotification(
			ctx,
			sub,
			*msg,
		); err != nil {
			logging.LogError(
				b.logger,
				fmt.Sprintf(
					"Failed to send push notification to endpoint=%s for User with ID=%d",
					sub.Endpoint,
					dto.ReceiverID,
				),
				err,
			)
		}
	}
}
```

- [ ] **Step 2: Run build**

Run: `go build ./internal/workers/...`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```
refactor: add consent check to web-push worker with typed envelope DTO
```

---

### Task 10: Update NATS Publishers — Auth Service and WS Handler

**Files:**
- Modify: `internal/services/auth/service.go`
- Modify: `internal/controllers/http/handlers/api/ws/ws.go`

- [ ] **Step 1: Update auth service — RegisterUser NATS publish**

In `RegisterUser` (around line 91-105), replace:

```go
verifyEmailDTO := &domains.VerifyEmailNotificationDTO{
    UserID: user.ID,
}

content, err := json.Marshal(verifyEmailDTO)
```

With:

```go
emailDTO := &domains.EmailNotificationDTO{
    Type:       domains.EmailTypeVerifyEmail,
    ReceiverID: user.ID,
}

content, err := json.Marshal(emailDTO)
```

And update the subject:

```go
if err = s.natsPublisher.Publish(
    s.natsConfig.Subjects.EmailNotification,
    content,
); err != nil {
```

- [ ] **Step 2: Update auth service — SendForgetPasswordMessage NATS publish**

In `SendForgetPasswordMessage` (around line 259-271), replace:

```go
forgetPasswordDTO := &domains.ForgetPasswordNotificationDTO{
    UserID: user.ID,
}

content, err := json.Marshal(forgetPasswordDTO)
```

With:

```go
emailDTO := &domains.EmailNotificationDTO{
    Type:       domains.EmailTypeForgetPassword,
    ReceiverID: user.ID,
}

content, err := json.Marshal(emailDTO)
```

And update the subject to `s.natsConfig.Subjects.EmailNotification`.

- [ ] **Step 3: Update auth service — SendVerifyEmailMessage NATS publish**

In `SendVerifyEmailMessage` (around line 291-303), apply the same pattern:

```go
emailDTO := &domains.EmailNotificationDTO{
    Type:       domains.EmailTypeVerifyEmail,
    ReceiverID: user.ID,
}
```

Subject: `s.natsConfig.Subjects.EmailNotification`.

- [ ] **Step 4: Update WS handler — publishWebPushNotification**

In `internal/controllers/http/handlers/api/ws/ws.go`, method `publishWebPushNotification` (line 243), replace:

```go
func (h *Handler) publishWebPushNotification(ctx context.Context, userID, messageID uint64) {
	payload, err := json.Marshal(domains.NewMessagePayload{
		MessageID: messageID,
	})
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal push notification payload", err)

		return
	}

	pushDTO := domains.WebPushNotificationDTO{
		Type:       domains.WebPushTypeNewMessage,
		ReceiverID: userID,
		Payload:    payload,
	}

	content, err := json.Marshal(pushDTO)
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal push notification DTO", err)

		return
	}

	if err = h.natsPublisher.Publish(
		h.natsConfig.Subjects.WebPushNotification,
		content,
	); err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to publish push notification", err)
	}
}
```

**Also add email notification publishing for new messages** — in the same place where web-push is published (when a message is sent to an offline user). This requires publishing to both subjects:

```go
func (h *Handler) publishNewMessageNotifications(
	ctx context.Context,
	receiverID, messageID, chatID uint64,
	senderUsername, chatName string,
) {
	// Web push notification
	webPushPayload, err := json.Marshal(domains.NewMessagePayload{
		MessageID: messageID,
		ChatID:    chatID,
	})
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal web-push payload", err)

		return
	}

	webPushDTO := domains.WebPushNotificationDTO{
		Type:       domains.WebPushTypeNewMessage,
		ReceiverID: receiverID,
		Payload:    webPushPayload,
	}

	content, err := json.Marshal(webPushDTO)
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal web-push DTO", err)

		return
	}

	if err = h.natsPublisher.Publish(
		h.natsConfig.Subjects.WebPushNotification,
		content,
	); err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to publish web-push notification", err)
	}

	// Email notification
	emailPayload, err := json.Marshal(domains.NewMessagePayload{
		MessageID:      messageID,
		ChatID:         chatID,
		SenderUsername: senderUsername,
		ChatName:       chatName,
	})
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal email payload", err)

		return
	}

	emailDTO := domains.EmailNotificationDTO{
		Type:       domains.EmailTypeNewMessage,
		ReceiverID: receiverID,
		Payload:    emailPayload,
	}

	content, err = json.Marshal(emailDTO)
	if err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to marshal email DTO", err)

		return
	}

	if err = h.natsPublisher.Publish(
		h.natsConfig.Subjects.EmailNotification,
		content,
	); err != nil {
		logging.LogErrorContext(ctx, h.logger, "Failed to publish email notification", err)
	}
}
```

Update the call site in WS handler to use `publishNewMessageNotifications` instead of `publishWebPushNotification`, passing sender username and chat name.

- [ ] **Step 5: Run build and tests**

Run: `go build ./... && go test ./internal/services/auth/...`
Expected: Build succeeds, auth tests pass (may need mock updates for new config fields).

- [ ] **Step 6: Commit**

```
refactor: update NATS publishers to use unified notification DTOs
```

---

### Task 11: Wire Up in cmd/main.go

**Files:**
- Modify: `cmd/main.go`

- [ ] **Step 1: Add new_message content builder**

After the existing content builders initialization (around line 119-127), add:

```go
import newmessage "github.com/DKhorkov/kfc/internal/contentbuilders/new_message"
```

Update `contentBuilders`:

```go
contentBuilders := interfaces.ContentBuilders{
    VerifyEmail: verify_email.New(
        cfg.Email.VerifyEmailURL,
        cacheProvider,
    ),
    ForgetPassword: forget_password.New(
        cacheProvider,
    ),
    NewMessage: newmessage.New(),
}
```

- [ ] **Step 2: Replace three workers with two**

Remove the `verifyEmailWorker` and `forgetPasswordWorker` blocks (lines 319-393).

Replace with single email notification worker:

```go
import emailnotificationmessagehandlerbuilder "github.com/DKhorkov/kfc/internal/workers/handlers/builders/email_notification"
```

```go
emailNotificationWorker, err := customnats.NewConsumer(
    cfg.NATS.ClientURL,
    cfg.NATS.Subjects.EmailNotification,
    customnats.WithGoroutinesPoolSize(cfg.NATS.GoroutinesPoolSize),
    customnats.WithMessageChannelBufferSize(cfg.NATS.MessageChannelBufferSize),
    customnats.WithNatsOptions(nats.Name(cfg.NATS.Workers.EmailNotification.Name)),
    customnats.WithMessageHandler(
        messagehandlerbuildertracingdecorator.New(
            traceProvider,
            cfg.Tracing.Spans.Handlers.EmailNotification,
            emailnotificationmessagehandlerbuilder.New(
                notificationsUseCases,
                settingsUseCases,
                logger,
            ),
        ).MessageHandler(context.Background()),
    ),
)
if err != nil {
    panic(err)
}

if err = emailNotificationWorker.Run(); err != nil {
    panic(err)
}

defer func() {
    if err = emailNotificationWorker.Stop(); err != nil {
        logging.LogError(
            logger,
            fmt.Sprintf("Error shutting down %q worker", cfg.NATS.Workers.EmailNotification.Name),
            err,
        )
    }
}()
```

- [ ] **Step 3: Update web push worker — add settingsUseCases dependency**

Update the web push worker builder call to pass `settingsUseCases`:

```go
webpushnotificationmessagehandlerbuilder.New(
    webPushSubscriptionsUseCases,
    messagesUseCases,
    settingsUseCases,
    logger,
),
```

- [ ] **Step 4: Remove old imports**

Remove:
```go
forgetpasswordmessagehandlerbuilder "github.com/DKhorkov/kfc/internal/workers/handlers/builders/forget_password"
verifyemailmessagehandlerbuilder "github.com/DKhorkov/kfc/internal/workers/handlers/builders/verify_email"
```

- [ ] **Step 5: Update tracing spans config if needed**

Check if `cfg.Tracing.Spans.Handlers` has `VerifyEmail` and `ForgetPassword` fields — update to `EmailNotification`. This may require updating the tracing config struct in `config.go`.

- [ ] **Step 6: Run full build and tests**

Run: `go build ./... && go test ./...`
Expected: All pass.

- [ ] **Step 7: Commit**

```
refactor: wire up unified email worker and consent dependencies in main
```

---

### Task 12: Frontend — Collapsible Notifications Section in Profile Modal

**Files:**
- Modify: `internal/controllers/http/handlers/web/templates/navbar.html`
- Modify: `internal/controllers/http/handlers/web/static/css/navbar.css`

- [ ] **Step 1: Replace push-уведомления section in navbar.html**

Replace lines 72-77 (the push toggle section) with:

```html
<div class="profile-modal__section">
    <div class="profile-modal__toggle" id="my-profile-toggle-notifications">
        <span class="profile-modal__label">Уведомления</span>
        <span class="profile-modal__chevron">&#9654;</span>
    </div>
    <div id="my-profile-notifications-panel" class="profile-modal__form" style="display: none;">
        <div class="notifications-group">
            <div class="notifications-group__title">Новые сообщения</div>
            <div class="notifications-group__item">
                <span class="notifications-group__label">Email</span>
                <div class="theme-switch" id="toggle-email-new-message">
                    <div class="theme-switch__track">
                        <div class="theme-switch__thumb"></div>
                    </div>
                </div>
            </div>
            <div class="notifications-group__item">
                <span class="notifications-group__label">Web Push</span>
                <div class="theme-switch" id="toggle-webpush-new-message">
                    <div class="theme-switch__track">
                        <div class="theme-switch__thumb"></div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>
```

- [ ] **Step 2: Add CSS for notification group**

Add to `navbar.css`:

```css
.notifications-group {
    padding: var(--space-xs) 0;
}

.notifications-group__title {
    font-weight: 600;
    font-size: 0.9rem;
    color: var(--text-secondary);
    margin-bottom: var(--space-xs);
}

.notifications-group__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-xs) 0;
}

.notifications-group__label {
    font-size: 0.9rem;
    color: var(--text-primary);
}

.theme-switch--disabled {
    opacity: 0.4;
    pointer-events: none;
}
```

- [ ] **Step 3: Commit**

```
feat: add collapsible notifications section with toggles in profile modal
```

---

### Task 13: Frontend — Notification Toggles Logic

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/navbar.js`

- [ ] **Step 1: Add consent bitmask constant**

At the top of the file (near `THEME_LIGHT`/`THEME_DARK` constants):

```js
const CONSENT_NEW_MESSAGE = 1;
```

- [ ] **Step 2: Add collapsible toggle for notifications section**

In the `DOMContentLoaded` handler, add panel toggle logic (similar to edit profile / change password toggles):

```js
const notificationsToggle = document.getElementById('my-profile-toggle-notifications');
const notificationsPanel = document.getElementById('my-profile-notifications-panel');
if (notificationsToggle && notificationsPanel) {
    notificationsToggle.addEventListener('click', () => {
        const isOpen = notificationsPanel.style.display !== 'none';
        notificationsPanel.style.display = isOpen ? 'none' : '';
        notificationsToggle.querySelector('.profile-modal__chevron').style.transform =
            isOpen ? '' : 'rotate(90deg)';
    });
}
```

- [ ] **Step 3: Add toggle state management functions**

```js
function updateToggleUI(toggleEl, isOn) {
    const track = toggleEl.querySelector('.theme-switch__track');
    const thumb = toggleEl.querySelector('.theme-switch__thumb');
    if (!track || !thumb) return;

    track.classList.toggle('theme-switch__track--on', isOn);
    thumb.classList.toggle('theme-switch__thumb--on', isOn);
}

function setToggleDisabled(toggleEl, disabled, tooltip) {
    if (disabled) {
        toggleEl.classList.add('theme-switch--disabled');
        toggleEl.title = tooltip || '';
    } else {
        toggleEl.classList.remove('theme-switch--disabled');
        toggleEl.title = '';
    }
}
```

- [ ] **Step 4: Initialize toggles from server settings**

After fetching settings (around line 122-126), extend with consent initialization:

```js
const settingsResp = await fetchWithAuth('/api/users/me/settings');
if (settingsResp.ok) {
    const settings = await settingsResp.json();
    applyTheme(settings.theme === THEME_DARK);

    // Initialize notification toggles
    const emailToggle = document.getElementById('toggle-email-new-message');
    const webPushToggle = document.getElementById('toggle-webpush-new-message');

    if (emailToggle) {
        const emailOn = (settings.emailConsents & CONSENT_NEW_MESSAGE) !== 0;
        updateToggleUI(emailToggle, emailOn);
    }

    if (webPushToggle) {
        await initWebPushToggle(webPushToggle, settings);
    }
}
```

- [ ] **Step 5: Implement email toggle click handler**

```js
const emailToggle = document.getElementById('toggle-email-new-message');
if (emailToggle) {
    emailToggle.addEventListener('click', async () => {
        const track = emailToggle.querySelector('.theme-switch__track');
        const isCurrentlyOn = track.classList.contains('theme-switch__track--on');
        const newConsents = isCurrentlyOn ? 0 : CONSENT_NEW_MESSAGE;

        try {
            const resp = await fetchWithAuth('/api/users/me/settings', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ emailConsents: newConsents }),
            });

            if (resp.ok) {
                updateToggleUI(emailToggle, !isCurrentlyOn);
            }
        } catch (err) {
            console.error('Failed to update email consents:', err);
        }
    });
}
```

- [ ] **Step 6: Implement web push toggle with browser subscription sync**

```js
async function initWebPushToggle(toggleEl, settings) {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
        setToggleDisabled(toggleEl, true, 'Не поддерживается браузером');
        return;
    }

    const serverOn = (settings.webPushConsents & CONSENT_NEW_MESSAGE) !== 0;

    if (serverOn) {
        // Server says ON — ensure browser is subscribed
        const registration = await navigator.serviceWorker.getRegistration('/web/sw.js');
        const subscription = registration
            ? await registration.pushManager.getSubscription()
            : null;

        if (!subscription) {
            // Auto-sync: subscribe browser
            try {
                await subscribeToPush();
                updateToggleUI(toggleEl, true);
            } catch (err) {
                // User denied permission — reset server bit
                console.log('Browser push denied, resetting server consent');
                await fetchWithAuth('/api/users/me/settings', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ webPushConsents: 0 }),
                });
                updateToggleUI(toggleEl, false);
            }
        } else {
            updateToggleUI(toggleEl, true);
        }
    } else {
        updateToggleUI(toggleEl, false);
    }
}

async function subscribeToPush() {
    const registration = await navigator.serviceWorker.getRegistration('/web/sw.js')
        || await navigator.serviceWorker.register('/web/sw.js');

    const vapidKeyEl = document.querySelector('meta[name="vapid-public-key"]');
    const vapidPublicKey = vapidKeyEl ? vapidKeyEl.content : '';

    const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: vapidPublicKey,
    });

    const key = subscription.getKey('p256dh');
    const auth = subscription.getKey('auth');

    const resp = await fetchWithAuth('/api/web-push/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            endpoint: subscription.endpoint,
            encryptionKey: btoa(String.fromCharCode(...new Uint8Array(key))),
            auth: btoa(String.fromCharCode(...new Uint8Array(auth))),
        }),
    });

    if (resp.ok) {
        const data = await resp.json();
        localStorage.setItem('pushSubscriptionId', data.id);
    }

    return subscription;
}

async function unsubscribeFromPush() {
    const registration = await navigator.serviceWorker.getRegistration('/web/sw.js');
    if (!registration) return;

    const subscription = await registration.pushManager.getSubscription();
    if (subscription) {
        await subscription.unsubscribe();
    }

    const subId = localStorage.getItem('pushSubscriptionId');
    if (subId) {
        try {
            await fetchWithAuth('/api/web-push/subscribe/' + subId, { method: 'DELETE' });
        } catch (err) {
            console.log('Unsubscribe error:', err);
        }

        localStorage.removeItem('pushSubscriptionId');
    }
}
```

- [ ] **Step 7: Implement web push toggle click handler**

```js
const webPushToggle = document.getElementById('toggle-webpush-new-message');
if (webPushToggle) {
    webPushToggle.addEventListener('click', async () => {
        if (webPushToggle.classList.contains('theme-switch--disabled')) return;

        const track = webPushToggle.querySelector('.theme-switch__track');
        const isCurrentlyOn = track.classList.contains('theme-switch__track--on');

        try {
            if (isCurrentlyOn) {
                // Turn off
                await unsubscribeFromPush();
                await fetchWithAuth('/api/users/me/settings', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ webPushConsents: 0 }),
                });
                updateToggleUI(webPushToggle, false);
                showInfo('Web Push уведомления отключены');
            } else {
                // Turn on
                await subscribeToPush();
                await fetchWithAuth('/api/users/me/settings', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ webPushConsents: CONSENT_NEW_MESSAGE }),
                });
                updateToggleUI(webPushToggle, true);
                showInfo('Web Push уведомления включены');
            }
        } catch (err) {
            console.error('Failed to toggle web push:', err);
            showError('Не удалось переключить Web Push уведомления');
        }
    });
}
```

- [ ] **Step 8: Remove old push toggle code**

Remove the old `updatePushToggleUI` function and the `my-profile-toggle-push` click handler block.

- [ ] **Step 9: Test manually**

Run: `task local`

Test:
1. Open profile modal — «Уведомления» section is collapsed
2. Click «Уведомления» — section expands with toggles
3. Toggle email — saves consent, toggle reflects state
4. Toggle web push — requests browser permission, subscribes, saves consent
5. Refresh page — toggles restored from server settings
6. Turn off web push — unsubscribes, removes consent
7. Theme toggle still works

- [ ] **Step 10: Commit**

```
feat: add notification consent toggles with browser push sync in profile modal
```

---

### Task 14: Update doc.md Files

**Files:**
- Modify: `internal/domains/doc.md`
- Modify: `internal/workers/handlers/builders/doc.md` (if exists)
- Modify: `internal/contentbuilders/doc.md` (if exists)
- Modify: `internal/config/doc.md`
- Modify: `internal/controllers/http/handlers/web/templates/doc.md` (if exists)
- Create: `internal/contentbuilders/new_message/doc.md`
- Create: `internal/workers/handlers/builders/email_notification/doc.md`

Update all doc.md files in directories where code was changed, per the project rule in CLAUDE.md.

- [ ] **Step 1: Update each doc.md to reflect changes**

For each directory with modified code, update the corresponding doc.md to describe:
- New types and constants
- Changed structures
- New/removed files
- New behavior (consent checking, unified workers)

- [ ] **Step 2: Commit**

```
docs: update doc.md files for notification consents changes
```

---

### Task 15: Update Trace Decorators and Mocks

**Files:**
- Various trace decorator and mock files

- [ ] **Step 1: Regenerate mocks**

Run: `go generate ./internal/interfaces/...`

This regenerates all mocks to match updated interfaces.

- [ ] **Step 2: Update trace decorators**

Check and update trace decorators in:
- `internal/services/notifications/trace_decorator.go` — add `SendNewMessageNotification`
- `internal/usecases/notifications/trace_decorator.go` — add `SendNewMessageNotification`
- `internal/repositories/emails/trace_decorator.go` — add `SendNewMessageNotification`

Follow the existing pattern for trace decorators in these files.

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: All tests pass.

- [ ] **Step 4: Run linter**

Run: `task lint`
Expected: No new linting errors.

- [ ] **Step 5: Commit**

```
chore: regenerate mocks and update trace decorators for notification consents
```
