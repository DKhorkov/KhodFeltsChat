# Web Push Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Send native push notifications to offline users when they receive new chat messages.

**Architecture:** New push_subscriptions table in PostgreSQL, new NATS subject + worker for push delivery, `webpush-go` library for VAPID-based Web Push, Service Worker on the frontend. Follows the existing clean architecture pattern (repository → service → usecases → controller).

**Tech Stack:** Go 1.24, webpush-go, NATS, PostgreSQL (squirrel), gorilla/mux, Service Worker API, Push API

**Spec:** `docs/superpowers/specs/2026-05-15-web-push-notifications-design.md`

---

### Task 1: Add webpush-go dependency and generate VAPID keys

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add webpush-go dependency**

```bash
go get github.com/SherClockHolmes/webpush-go
```

- [ ] **Step 2: Generate VAPID key pair**

```bash
go run github.com/AlfredBerg/webpush-go/v2/cmd/vapid-keygen@latest
```

Save the output (public + private key) — they will be used in `.env` / config. Example output:

```
Private Key: sGk...
Public Key: BEl...
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "добавлена зависимость webpush-go для Web Push уведомлений"
```

---

### Task 2: Database migration — push_subscriptions table

**Files:**
- Create: `migrations/20260515000000_web_push_subscriptions.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
CREATE TABLE push_subscriptions (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT         NOT NULL UNIQUE,
    p256dh     TEXT         NOT NULL,
    auth       TEXT         NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);

-- +goose Down
DROP TABLE IF EXISTS push_subscriptions;
```

- [ ] **Step 2: Apply migration**

```bash
task migrate-up
```

- [ ] **Step 3: Commit**

```bash
git add migrations/20260515000000_web_push_subscriptions.sql
git commit -m "добавлена миграция для таблицы push_subscriptions"
```

---

### Task 3: Domain model — PushSubscription

**Files:**
- Create: `internal/domains/push_subscription.go`

- [ ] **Step 1: Create domain model**

```go
package domains

import "time"

type PushSubscription struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"userId"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"createdAt"`
}
```

- [ ] **Step 2: Add PushNotificationDTO to notifications.go**

Add to `internal/domains/notifications.go`:

```go
type PushNotificationDTO struct {
	UserID    uint64 `json:"userId"`
	MessageID uint64 `json:"messageId"`
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/domains/push_subscription.go internal/domains/notifications.go
git commit -m "добавлены доменные модели PushSubscription и PushNotificationDTO"
```

---

### Task 4: Error sentinel

**Files:**
- Create: `internal/errors/push_subscriptions.go`

- [ ] **Step 1: Create error file**

```go
package customerrors

import "errors"

var ErrPushSubscriptionNotFound = errors.New("push subscription not found")
```

- [ ] **Step 2: Commit**

```bash
git add internal/errors/push_subscriptions.go
git commit -m "добавлена sentinel ошибка для push-подписок"
```

---

### Task 5: Interface — PushSubscriptionsRepository

**Files:**
- Modify: `internal/interfaces/repositories.go`

- [ ] **Step 1: Add interface**

Add to the end of `internal/interfaces/repositories.go`:

```go
//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/push_subscriptions_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository
type PushSubscriptionsRepository interface {
	CreatePushSubscription(ctx context.Context, subscription domains.PushSubscription) (uint64, error)
	GetPushSubscriptionsByUserID(ctx context.Context, userID uint64) ([]domains.PushSubscription, error)
	DeletePushSubscription(ctx context.Context, id uint64) error
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
}
```

Примечание: `DeletePushSubscriptionByEndpoint` нужен для worker'а — когда push-сервер возвращает 410/404, worker знает только endpoint подписки, но не её ID.

- [ ] **Step 2: Generate mocks**

```bash
go generate ./internal/interfaces/repositories.go
```

- [ ] **Step 3: Commit**

```bash
git add internal/interfaces/repositories.go mocks/repositories/push_subscriptions_repository.go
git commit -m "добавлен интерфейс PushSubscriptionsRepository"
```

---

### Task 6: Interface — PushSubscriptionsService

**Files:**
- Modify: `internal/interfaces/services.go`

- [ ] **Step 1: Add interface**

Add to the end of `internal/interfaces/services.go`:

```go
//go:generate mockgen -source=services.go -destination=../../mocks/services/push_subscriptions_service.go -package=mockservices -exclude_interfaces=UsersService,AuthService,ChatsService,MessagesService,NotificationsService,SettingsService
type PushSubscriptionsService interface {
	CreatePushSubscription(ctx context.Context, subscription domains.PushSubscription) (*domains.PushSubscription, error)
	GetPushSubscriptionsByUserID(ctx context.Context, userID uint64) ([]domains.PushSubscription, error)
	DeletePushSubscription(ctx context.Context, id uint64) error
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
}
```

- [ ] **Step 2: Generate mocks**

```bash
go generate ./internal/interfaces/services.go
```

- [ ] **Step 3: Commit**

```bash
git add internal/interfaces/services.go mocks/services/push_subscriptions_service.go
git commit -m "добавлен интерфейс PushSubscriptionsService"
```

---

### Task 7: Interface — PushSubscriptionsUseCases

**Files:**
- Modify: `internal/interfaces/usecases.go`

- [ ] **Step 1: Add interface**

Add to the end of `internal/interfaces/usecases.go`:

```go
//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/push_subscriptions_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases,SettingsUseCases
type PushSubscriptionsUseCases interface {
	CreatePushSubscription(ctx context.Context, subscription domains.PushSubscription) (*domains.PushSubscription, error)
	GetPushSubscriptionsByUserID(ctx context.Context, userID uint64) ([]domains.PushSubscription, error)
	DeletePushSubscription(ctx context.Context, id uint64) error
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
	SendPushNotification(ctx context.Context, subscription domains.PushSubscription, message domains.Message) error
}
```

- [ ] **Step 2: Generate mocks**

```bash
go generate ./internal/interfaces/usecases.go
```

- [ ] **Step 3: Commit**

```bash
git add internal/interfaces/usecases.go mocks/usecases/push_subscriptions_usecases.go
git commit -m "добавлен интерфейс PushSubscriptionsUseCases"
```

---

### Task 8: Repository — push_subscriptions

**Files:**
- Create: `internal/repositories/push_subscriptions/repository.go`
- Create: `internal/repositories/push_subscriptions/repository_test.go`
- Create: `internal/repositories/push_subscriptions/trace_decorator.go`
- Create: `internal/repositories/push_subscriptions/trace_decorator_test.go`

- [ ] **Step 1: Write repository tests**

Create `internal/repositories/push_subscriptions/repository_test.go`. Tests follow the same pattern as `internal/repositories/settings/repository_test.go` — testing SQL builder output. Since repository uses UoW with real transactions, unit tests here verify the squirrel SQL generation is correct.

- [ ] **Step 2: Write repository implementation**

Create `internal/repositories/push_subscriptions/repository.go`:

```go
package push_subscriptions

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	sq "github.com/Masterminds/squirrel"
)

const (
	tableName = "push_subscriptions"

	idColumnName        = "id"
	userIDColumnName    = "user_id"
	endpointColumnName  = "endpoint"
	p256dhColumnName    = "p256dh"
	authColumnName      = "auth"
	createdAtColumnName = "created_at"

	selectAllColumns = "*"
)

type Repository struct {
	tx pg.Transaction
}

func New(tx pg.Transaction) *Repository {
	return &Repository{tx: tx}
}

func (repo *Repository) CreatePushSubscription(
	ctx context.Context,
	subscription domains.PushSubscription,
) (uint64, error) {
	stmt, params, err := sq.
		Insert(tableName).
		Columns(
			userIDColumnName,
			endpointColumnName,
			p256dhColumnName,
			authColumnName,
		).
		Values(
			subscription.UserID,
			subscription.Endpoint,
			subscription.P256dh,
			subscription.Auth,
		).
		Suffix("RETURNING " + idColumnName).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, err
	}

	var id uint64
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (repo *Repository) GetPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.PushSubscription, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(tableName).
		Where(sq.Eq{userIDColumnName: userID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := repo.tx.QueryContext(ctx, stmt, params...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subscriptions []domains.PushSubscription

	for rows.Next() {
		var sub domains.PushSubscription
		columns := pg.GetEntityColumns(&sub)

		if err = rows.Scan(columns...); err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, rows.Err()
}

func (repo *Repository) DeletePushSubscription(
	ctx context.Context,
	id uint64,
) error {
	stmt, params, err := sq.
		Delete(tableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) DeletePushSubscriptionByEndpoint(
	ctx context.Context,
	endpoint string,
) error {
	stmt, params, err := sq.
		Delete(tableName).
		Where(sq.Eq{endpointColumnName: endpoint}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/repositories/push_subscriptions/... -v
```

- [ ] **Step 4: Write trace decorator**

Create `internal/repositories/push_subscriptions/trace_decorator.go`, following the pattern from `internal/repositories/settings/trace_decorator.go`:

```go
package push_subscriptions

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.PushSubscriptionsRepository
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.PushSubscriptionsRepository,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) CreatePushSubscription(
	ctx context.Context,
	subscription domains.PushSubscription,
) (uint64, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.CreatePushSubscription(ctx, subscription)
}

func (d *TraceDecorator) GetPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.PushSubscription, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetPushSubscriptionsByUserID(ctx, userID)
}

func (d *TraceDecorator) DeletePushSubscription(ctx context.Context, id uint64) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeletePushSubscription(ctx, id)
}

func (d *TraceDecorator) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeletePushSubscriptionByEndpoint(ctx, endpoint)
}
```

- [ ] **Step 5: Write trace decorator tests**

Follow the pattern from `internal/repositories/settings/trace_decorator_test.go`.

- [ ] **Step 6: Run all tests**

```bash
go test ./internal/repositories/push_subscriptions/... -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/repositories/push_subscriptions/
git commit -m "добавлен репозиторий push_subscriptions с trace decorator"
```

---

### Task 9: Service — push_subscriptions

**Files:**
- Create: `internal/services/push_subscriptions/service.go`
- Create: `internal/services/push_subscriptions/service_test.go`
- Create: `internal/services/push_subscriptions/trace_decorator.go`
- Create: `internal/services/push_subscriptions/trace_decorator_test.go`

- [ ] **Step 1: Write service tests**

Follow the pattern from `internal/services/settings/service_test.go`.

- [ ] **Step 2: Write service implementation**

Create `internal/services/push_subscriptions/service.go`:

```go
package push_subscriptions

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                                interfaces.UnitOfWork
	newPushSubscriptionsRepositoryFunc func(tx pg.Transaction) interfaces.PushSubscriptionsRepository
}

func New(
	uow interfaces.UnitOfWork,
	newPushSubscriptionsRepositoryFunc func(tx pg.Transaction) interfaces.PushSubscriptionsRepository,
) *Service {
	return &Service{
		uow:                                uow,
		newPushSubscriptionsRepositoryFunc: newPushSubscriptionsRepositoryFunc,
	}
}

func (s *Service) CreatePushSubscription(
	ctx context.Context,
	subscription domains.PushSubscription,
) (*domains.PushSubscription, error) {
	var (
		result *domains.PushSubscription
		err    error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newPushSubscriptionsRepositoryFunc(tx)

			id, createErr := repo.CreatePushSubscription(ctx, subscription)
			if createErr != nil {
				return createErr
			}

			subs, getErr := repo.GetPushSubscriptionsByUserID(ctx, subscription.UserID)
			if getErr != nil {
				return getErr
			}

			for i := range subs {
				if subs[i].ID == id {
					result = &subs[i]

					return nil
				}
			}

			return fmt.Errorf("%w: id=%d", customerrors.ErrPushSubscriptionNotFound, id)
		},
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) GetPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.PushSubscription, error) {
	var (
		subscriptions []domains.PushSubscription
		err           error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newPushSubscriptionsRepositoryFunc(tx)
			subscriptions, err = repo.GetPushSubscriptionsByUserID(ctx, userID)

			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (s *Service) DeletePushSubscription(ctx context.Context, id uint64) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newPushSubscriptionsRepositoryFunc(tx)

			return repo.DeletePushSubscription(ctx, id)
		},
	)
}

func (s *Service) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newPushSubscriptionsRepositoryFunc(tx)

			return repo.DeletePushSubscriptionByEndpoint(ctx, endpoint)
		},
	)
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/services/push_subscriptions/... -v
```

- [ ] **Step 4: Write trace decorator**

Follow the pattern from `internal/services/settings/trace_decorator.go`. Implement `PushSubscriptionsService` interface, delegating each method with span creation.

- [ ] **Step 5: Write trace decorator tests**

- [ ] **Step 6: Run all tests**

```bash
go test ./internal/services/push_subscriptions/... -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/services/push_subscriptions/
git commit -m "добавлен сервис push_subscriptions с trace decorator"
```

---

### Task 10: UseCases — push_subscriptions

**Files:**
- Create: `internal/usecases/push_subscriptions/usecases.go`
- Create: `internal/usecases/push_subscriptions/usecases_test.go`
- Create: `internal/usecases/push_subscriptions/trace_decorator.go`
- Create: `internal/usecases/push_subscriptions/trace_decorator_test.go`

- [ ] **Step 1: Write usecases tests**

- [ ] **Step 2: Write usecases implementation**

Create `internal/usecases/push_subscriptions/usecases.go`:

```go
package push_subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	webpush "github.com/SherClockHolmes/webpush-go"
)

type UseCases struct {
	pushSubscriptionsService interfaces.PushSubscriptionsService
	webPushConfig            config.WebPushConfig
	logger                   logging.Logger
}

func New(
	pushSubscriptionsService interfaces.PushSubscriptionsService,
	webPushConfig config.WebPushConfig,
	logger logging.Logger,
) *UseCases {
	return &UseCases{
		pushSubscriptionsService: pushSubscriptionsService,
		webPushConfig:            webPushConfig,
		logger:                   logger,
	}
}

func (u *UseCases) CreatePushSubscription(
	ctx context.Context,
	subscription domains.PushSubscription,
) (*domains.PushSubscription, error) {
	return u.pushSubscriptionsService.CreatePushSubscription(ctx, subscription)
}

func (u *UseCases) GetPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.PushSubscription, error) {
	return u.pushSubscriptionsService.GetPushSubscriptionsByUserID(ctx, userID)
}

func (u *UseCases) DeletePushSubscription(ctx context.Context, id uint64) error {
	return u.pushSubscriptionsService.DeletePushSubscription(ctx, id)
}

func (u *UseCases) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return u.pushSubscriptionsService.DeletePushSubscriptionByEndpoint(ctx, endpoint)
}

func (u *UseCases) SendPushNotification(
	ctx context.Context,
	subscription domains.PushSubscription,
	message domains.Message,
) error {
	payload, err := json.Marshal(map[string]any{
		"title":  message.Sender.Username,
		"body":   message.Text,
		"chatId": message.ChatID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal push payload: %w", err)
	}

	resp, err := webpush.SendNotification(
		payload,
		&webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				P256dh: subscription.P256dh,
				Auth:   subscription.Auth,
			},
		},
		&webpush.Options{
			VAPIDPublicKey:  u.webPushConfig.VAPIDPublicKey,
			VAPIDPrivateKey: u.webPushConfig.VAPIDPrivateKey,
			Subscriber:      u.webPushConfig.VAPIDContact,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		logging.LogInfo(
			u.logger,
			fmt.Sprintf(
				"Push subscription expired (status %d), deleting endpoint=%s",
				resp.StatusCode,
				subscription.Endpoint,
			),
		)

		return u.pushSubscriptionsService.DeletePushSubscriptionByEndpoint(ctx, subscription.Endpoint)
	}

	return nil
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/usecases/push_subscriptions/... -v
```

- [ ] **Step 4: Write trace decorator**

Follow the pattern from `internal/usecases/settings/trace_decorator.go`. Note: `SendPushNotification` also gets a trace span.

- [ ] **Step 5: Write trace decorator tests**

- [ ] **Step 6: Run all tests**

```bash
go test ./internal/usecases/push_subscriptions/... -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/usecases/push_subscriptions/
git commit -m "добавлены usecases push_subscriptions с отправкой Web Push и trace decorator"
```

---

### Task 11: Config — WebPush + NATS push-notification subject/worker

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add WebPushConfig type and field**

Add to `internal/config/config.go` the new struct:

```go
type WebPushConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDContact    string
}
```

Add field to `Config` struct:

```go
type Config struct {
	// ...existing fields...
	WebPush WebPushConfig
}
```

- [ ] **Step 2: Add WebPush config loading in New()**

Add to the `New()` function:

```go
WebPush: WebPushConfig{
	VAPIDPublicKey:  loadenv.GetEnv("VAPID_PUBLIC_KEY", ""),
	VAPIDPrivateKey: loadenv.GetEnv("VAPID_PRIVATE_KEY", ""),
	VAPIDContact:    loadenv.GetEnv("VAPID_CONTACT", "mailto:admin@kfc.com"),
},
```

- [ ] **Step 3: Add NATS push-notification subject and worker config**

Add to `NATSSubjects`:

```go
type NATSSubjects struct {
	VerifyEmail      string
	ForgetPassword   string
	PushNotification string
}
```

Add to `NATSWorkers`:

```go
type NATSWorkers struct {
	VerifyEmail      NATSWorker
	ForgetPassword   NATSWorker
	PushNotification NATSWorker
}
```

Add loading in `New()`:

```go
Subjects: NATSSubjects{
	VerifyEmail:      loadenv.GetEnv("NATS_VERIFY_EMAIL_SUBJECT", "verify-email"),
	ForgetPassword:   loadenv.GetEnv("NATS_FORGET_PASSWORD_SUBJECT", "forget-password"),
	PushNotification: loadenv.GetEnv("NATS_PUSH_NOTIFICATION_SUBJECT", "push-notification"),
},
// ...
Workers: NATSWorkers{
	VerifyEmail: NATSWorker{
		Name: loadenv.GetEnv("NATS_VERIFY_EMAIL_WORKER_NAME", "verify-email-worker"),
	},
	ForgetPassword: NATSWorker{
		Name: loadenv.GetEnv("NATS_FORGET_PASSWORD_WORKER_NAME", "forget-password-worker"),
	},
	PushNotification: NATSWorker{
		Name: loadenv.GetEnv("NATS_PUSH_NOTIFICATION_WORKER_NAME", "push-notification-worker"),
	},
},
```

- [ ] **Step 4: Add tracing span configs**

Add `PushSubscriptions` to `SpanRepositories`, `SpanServices`, `SpanUseCases`:

```go
type SpanRepositories struct {
	// ...existing...
	PushSubscriptions tracing.SpanConfig
}

type SpanServices struct {
	// ...existing...
	PushSubscriptions tracing.SpanConfig
}

type SpanUseCases struct {
	// ...existing...
	PushSubscriptions tracing.SpanConfig
}

type SpanHandlers struct {
	// ...existing...
	PushNotification tracing.SpanConfig
}
```

And initialize them in `New()` following the existing pattern with environment-based `tracing.SpanConfig` for each (same structure as Settings/Notifications spans, with names like "PushSubscriptions repository", "PushSubscriptions service", "PushSubscriptions useCases", "PushNotification worker handler").

- [ ] **Step 5: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "добавлена конфигурация WebPush, NATS push-notification и tracing spans"
```

---

### Task 12: Schemas and mappers for push subscriptions API

**Files:**
- Modify: `internal/controllers/http/schemas/push_subscriptions.go` (create)
- Create: `internal/controllers/http/mappers/push_subscriptions/push_subscriptions.go`

- [ ] **Step 1: Create schemas**

Create `internal/controllers/http/schemas/push_subscriptions.go`:

```go
package schemas

// PushSubscriptionKeys represents the browser push subscription keys.
// swagger:model
type PushSubscriptionKeys struct {
	// P256dh key
	// required: true
	P256dh string `json:"p256dh"`
	// Auth key
	// required: true
	Auth string `json:"auth"`
}

// CreatePushSubscriptionRequest represents the request body for creating a push subscription.
// swagger:parameters CreatePushSubscription
type CreatePushSubscriptionRequest struct {
	// Push subscription data
	// required: true
	// in: body
	Body struct {
		// Push endpoint URL
		// required: true
		Endpoint string `json:"endpoint"`
		// Push subscription keys
		// required: true
		Keys PushSubscriptionKeys `json:"keys"`
	}
}

// CreatePushSubscriptionResponse represents the response for a created push subscription.
// swagger:model
type CreatePushSubscriptionResponse struct {
	// ID of the created subscription
	ID uint64 `json:"id"`
}

// VAPIDKeyResponse represents the VAPID public key response.
// swagger:model
type VAPIDKeyResponse struct {
	// VAPID public key
	PublicKey string `json:"publicKey"`
}
```

- [ ] **Step 2: Create mappers**

Create `internal/controllers/http/mappers/push_subscriptions/push_subscriptions.go`:

```go
package push_subscriptions

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapCreateResponse(subscription domains.PushSubscription) schemas.CreatePushSubscriptionResponse {
	return schemas.CreatePushSubscriptionResponse{
		ID: subscription.ID,
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/controllers/http/schemas/push_subscriptions.go internal/controllers/http/mappers/push_subscriptions/
git commit -m "добавлены schemas и mappers для push subscriptions API"
```

---

### Task 13: HTTP handlers — subscribe, unsubscribe, vapid-key

**Files:**
- Create: `internal/controllers/http/handlers/api/push/subscribe/handler.go`
- Create: `internal/controllers/http/handlers/api/push/subscribe/handler_test.go`
- Create: `internal/controllers/http/handlers/api/push/unsubscribe/handler.go`
- Create: `internal/controllers/http/handlers/api/push/unsubscribe/handler_test.go`
- Create: `internal/controllers/http/handlers/api/push/vapid_key/handler.go`
- Create: `internal/controllers/http/handlers/api/push/vapid_key/handler_test.go`

- [ ] **Step 1: Write subscribe handler tests**

Follow the pattern from `internal/controllers/http/handlers/api/settings/get/handler_test.go`. Test cases:
- Successful subscription creation (201)
- Unauthorized — no userID in context (401)
- Invalid JSON body (400)
- Internal server error (500)

- [ ] **Step 2: Write subscribe handler**

Create `internal/controllers/http/handlers/api/push/subscribe/handler.go`:

```go
package subscribe

import (
	"encoding/json"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/push_subscriptions"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route POST /api/push/subscribe web-pushes CreatePushSubscription
//
// CreatePushSubscription
//
// Creates a push subscription for the current authorized User.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	201: CreatePushSubscriptionResponse
//	400: BadRequest
//	401: Unauthorized
//	500: InternalServerError

type requestBody struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// Handler creates a push subscription for the current user.
func Handler(u interfaces.PushSubscriptionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		var body requestBody
		if err = json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		subscription, err := u.CreatePushSubscription(r.Context(), domains.PushSubscription{
			UserID:   userID,
			Endpoint: body.Endpoint,
			P256dh:   body.Keys.P256dh,
			Auth:     body.Keys.Auth,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusCreated)

		if err = json.NewEncoder(w).Encode(mappers.MapCreateResponse(*subscription)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
```

- [ ] **Step 3: Run subscribe handler tests**

```bash
go test ./internal/controllers/http/handlers/api/push/subscribe/... -v
```

- [ ] **Step 4: Write unsubscribe handler tests**

Test cases:
- Successful deletion (204)
- Unauthorized (401)
- Invalid ID (400)
- Internal server error (500)

- [ ] **Step 5: Write unsubscribe handler**

Create `internal/controllers/http/handlers/api/push/unsubscribe/handler.go`:

```go
package unsubscribe

import (
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
)

// swagger:route DELETE /api/push/subscribe/{id} web-pushes DeletePushSubscription
//
// DeletePushSubscription
//
// Deletes a push subscription by ID.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	401: Unauthorized
//	500: InternalServerError

// Handler deletes a push subscription by ID.
func Handler(u interfaces.PushSubscriptionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		vars := mux.Vars(r)

		id, err := strconv.ParseUint(vars[common.IDRouteKey], 10, 64)
		if err != nil {
			http.Error(w, "invalid subscription id", http.StatusBadRequest)

			return
		}

		if err = u.DeletePushSubscription(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 6: Run unsubscribe handler tests**

```bash
go test ./internal/controllers/http/handlers/api/push/unsubscribe/... -v
```

- [ ] **Step 7: Write vapid-key handler tests and implementation**

Create `internal/controllers/http/handlers/api/push/vapid_key/handler.go`:

```go
package vapid_key

import (
	"encoding/json"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
)

// swagger:route GET /api/push/vapid-key web-pushes GetVAPIDKey
//
// GetVAPIDKey
//
// Returns the VAPID public key for push subscription.
//
// Responses:
//	200: VAPIDKeyResponse

// Handler returns the VAPID public key.
func Handler(vapidPublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusOK)

		response := schemas.VAPIDKeyResponse{
			PublicKey: vapidPublicKey,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
```

- [ ] **Step 8: Run all handler tests**

```bash
go test ./internal/controllers/http/handlers/api/push/... -v
```

- [ ] **Step 9: Commit**

```bash
git add internal/controllers/http/handlers/api/push/
git commit -m "добавлены HTTP handlers для push subscribe, unsubscribe и vapid-key"
```

---

### Task 14: Register push routes in API setup

**Files:**
- Modify: `internal/controllers/http/handlers/api/setup.go`
- Modify: `internal/controllers/http/handlers/setup.go`
- Modify: `internal/controllers/http/controller.go`
- Modify: `cmd/main.go`

- [ ] **Step 1: Add URL constants and handlers to api/setup.go**

Add constants:

```go
PushURL          = "/push"
PushSubscribeURL = PushURL + "/subscribe"
PushUnsubscribeURL = PushSubscribeURL + "/{%s}"
PushVAPIDKeyURL  = PushURL + "/vapid-key"
```

Update `SetupHandlers` signature to accept `pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases` and `vapidPublicKey string`.

Register routes:

```go
// GET /api/push/vapid-key (no auth required — handled in controller.go IgnoreURL)
getMux.Handle(PushVAPIDKeyURL, vapid_key.Handler(vapidPublicKey))

// POST /api/push/subscribe
postMux.Handle(PushSubscribeURL, subscribe.Handler(pushSubscriptionsUseCases))

// DELETE /api/push/subscribe/{id}
deleteMux.Handle(
	fmt.Sprintf(PushUnsubscribeURL, common.IDRouteKey),
	unsubscribe.Handler(pushSubscriptionsUseCases),
)
```

- [ ] **Step 2: Update handlers/setup.go**

Add `pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases` and `vapidPublicKey string` to `SetupHandlers` and pass them through to `api.SetupHandlers`.

- [ ] **Step 3: Update controller.go**

Add `pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases` and `vapidPublicKey string` parameters to `New()`. Pass them to `handlers.SetupHandlers`.

Add `PushVAPIDKeyURL` to the auth middleware ignore list:

```go
{
	Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.PushVAPIDKeyURL + `$`),
	Methods: []string{http.MethodGet},
},
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/controllers/http/handlers/api/setup.go internal/controllers/http/handlers/setup.go internal/controllers/http/controller.go
git commit -m "зарегистрированы push-маршруты в API роутере"
```

---

### Task 15: NATS worker — push_notification

**Files:**
- Create: `internal/workers/handlers/builders/push_notification/builder.go`
- Create: `internal/workers/handlers/builders/push_notification/builder_test.go`

- [ ] **Step 1: Write builder tests**

Follow the pattern from `internal/workers/handlers/builders/verify_email/builder_test.go`.

- [ ] **Step 2: Write builder implementation**

Create `internal/workers/handlers/builders/push_notification/builder.go`:

```go
package push_notification

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
	pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases
	messagesUseCases          interfaces.MessagesUseCases
	logger                    logging.Logger
}

func New(
	pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	logger logging.Logger,
) *Builder {
	return &Builder{
		pushSubscriptionsUseCases: pushSubscriptionsUseCases,
		messagesUseCases:          messagesUseCases,
		logger:                    logger,
	}
}

func (b *Builder) MessageHandler(ctx context.Context) interfaces.MessageHandler {
	return func(message *nats.Msg) {
		dto := b.natsMessageToDTO(message)
		if dto == nil {
			return
		}

		msg, err := b.messagesUseCases.GetMessageByID(ctx, dto.UserID, dto.MessageID)
		if err != nil {
			logging.LogError(
				b.logger,
				fmt.Sprintf("Failed to get message with ID=%d", dto.MessageID),
				err,
			)

			return
		}

		subscriptions, err := b.pushSubscriptionsUseCases.GetPushSubscriptionsByUserID(ctx, dto.UserID)
		if err != nil {
			logging.LogError(
				b.logger,
				fmt.Sprintf("Failed to get push subscriptions for User with ID=%d", dto.UserID),
				err,
			)

			return
		}

		for _, sub := range subscriptions {
			if err = b.pushSubscriptionsUseCases.SendPushNotification(ctx, sub, *msg); err != nil {
				logging.LogError(
					b.logger,
					fmt.Sprintf(
						"Failed to send push notification to endpoint=%s for User with ID=%d",
						sub.Endpoint,
						dto.UserID,
					),
					err,
				)
			}
		}
	}
}

func (b *Builder) natsMessageToDTO(message *nats.Msg) *domains.PushNotificationDTO {
	var dto domains.PushNotificationDTO
	if err := json.Unmarshal(message.Data, &dto); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal push-notification message", err)

		return nil
	}

	return &dto
}
```

**Примечание:** Worker использует `messagesUseCases.GetMessageByID(ctx, dto.UserID, dto.MessageID)`. Эта функция уже существует в интерфейсе `MessagesService` (в `internal/interfaces/services.go`), но нужно убедиться, что она также есть в `MessagesUseCases`. Если её нет — нужно добавить `GetMessageByID` в `MessagesUseCases` интерфейс и реализацию.

Проверьте `internal/interfaces/usecases.go`:

```go
type MessagesUseCases interface {
	MessagesService
}
```

`MessagesService` содержит `SaveMessage` и `GetChatMessages`. `GetMessageByID` есть в `MessagesRepository`, но не в `MessagesService`. Нужно протянуть `GetMessageByID` через service и usecases.

- [ ] **Step 3: Add GetMessageByID to MessagesService interface and implementation**

Add to `internal/interfaces/services.go` `MessagesService`:

```go
type MessagesService interface {
	SaveMessage(ctx context.Context, message domains.Message) (*domains.Message, error)
	GetChatMessages(
		ctx context.Context,
		userID uint64,
		chatID uint64,
		pagination *domains.Pagination,
	) ([]domains.Message, error)
	GetMessageByID(ctx context.Context, userID uint64, messageID uint64) (*domains.Message, error)
}
```

Implement `GetMessageByID` in `internal/services/messages/service.go` and its trace decorator. The implementation wraps the existing `MessagesRepository.GetMessageByID`.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/workers/handlers/builders/push_notification/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/workers/handlers/builders/push_notification/ internal/interfaces/services.go internal/services/messages/
git commit -m "добавлен NATS worker push_notification и GetMessageByID в messages service"
```

---

### Task 16: WS handler — publish NATS events for offline users

**Files:**
- Modify: `internal/controllers/http/handlers/api/ws/ws.go`

- [ ] **Step 1: Add NATS publisher dependency to WS Handler**

Add `natsPublisher` and `natsConfig` fields to `Handler` struct:

```go
type Handler struct {
	upgrader         interfaces.Upgrader
	usersUseCases    interfaces.UsersUseCases
	chatsUseCases    interfaces.ChatsUseCases
	messagesUseCases interfaces.MessagesUseCases
	logger           logging.Logger
	connections      *sync.Map
	natsPublisher    customnats.Publisher
	natsConfig       config.NATSConfig
}
```

Update `New()` to accept these parameters.

- [ ] **Step 2: Publish push-notification event for offline members**

In the `listen` method, in the loop over `chatMembers`, when `h.connections.Load(member.ID)` returns `false` (member is offline), add:

```go
if !exists {
	pushDTO := domains.PushNotificationDTO{
		UserID:    member.ID,
		MessageID: savedMessage.ID,
	}

	content, marshalErr := json.Marshal(pushDTO)
	if marshalErr != nil {
		logging.LogErrorContext(
			ctx,
			h.logger,
			"Failed to marshal push notification DTO",
			marshalErr,
		)

		continue
	}

	if publishErr := h.natsPublisher.Publish(
		h.natsConfig.Subjects.PushNotification,
		content,
	); publishErr != nil {
		logging.LogErrorContext(
			ctx,
			h.logger,
			"Failed to publish push notification",
			publishErr,
			"UserID", member.ID,
		)
	}

	continue
}
```

- [ ] **Step 3: Update api/setup.go**

Update `ws.New()` call to pass `natsPublisher` and `natsConfig`.

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/controllers/http/handlers/api/ws/ws.go internal/controllers/http/handlers/api/setup.go
git commit -m "WS handler публикует NATS push-notification события для офлайн-участников"
```

---

### Task 17: Wire everything in cmd/main.go

**Files:**
- Modify: `cmd/main.go`

- [ ] **Step 1: Create push_subscriptions service and usecases**

Add to `main()` after existing service/usecase creation:

```go
pushSubscriptionsService := pushsubscriptionsservice.NewTraceDecorator(
	traceProvider,
	cfg.Tracing.Spans.Services.PushSubscriptions,
	pushsubscriptionsservice.New(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.PushSubscriptionsRepository {
			return pushsubscriptionsrepository.NewTraceDecorator(
				traceProvider,
				cfg.Tracing.Spans.Repositories.PushSubscriptions,
				pushsubscriptionsrepository.New(tx),
			)
		},
	),
)

pushSubscriptionsUseCases := pushsubscriptionsusecases.NewTraceDecorator(
	traceProvider,
	cfg.Tracing.Spans.UseCases.PushSubscriptions,
	pushsubscriptionsusecases.New(
		pushSubscriptionsService,
		cfg.WebPush,
		logger,
	),
)
```

- [ ] **Step 2: Create push-notification NATS worker**

```go
pushNotificationWorker, err := customnats.NewConsumer(
	cfg.NATS.ClientURL,
	cfg.NATS.Subjects.PushNotification,
	customnats.WithGoroutinesPoolSize(cfg.NATS.GoroutinesPoolSize),
	customnats.WithMessageChannelBufferSize(cfg.NATS.MessageChannelBufferSize),
	customnats.WithNatsOptions(nats.Name(cfg.NATS.Workers.PushNotification.Name)),
	customnats.WithMessageHandler(
		messagehandlerbuildertracingdecorator.New(
			traceProvider,
			cfg.Tracing.Spans.Handlers.PushNotification,
			pushnotificationmessagehandlerbuilder.New(
				pushSubscriptionsUseCases,
				messagesUseCases,
				logger,
			),
		).MessageHandler(context.Background()),
	),
)
if err != nil {
	panic(err)
}

if err = pushNotificationWorker.Run(); err != nil {
	panic(err)
}

defer func() {
	if err = pushNotificationWorker.Stop(); err != nil {
		logging.LogError(
			logger,
			fmt.Sprintf(
				"Error shutting down %q worker",
				cfg.NATS.Workers.PushNotification.Name,
			),
			err,
		)
	}
}()
```

- [ ] **Step 3: Pass push dependencies to controller**

Update `controllers.New()` call to include `pushSubscriptionsUseCases` and `cfg.WebPush.VAPIDPublicKey`.

Pass `natsPublisher` and `cfg.NATS` down through the handler chain to `ws.New()`.

- [ ] **Step 4: Verify compilation and run**

```bash
go build ./cmd/main.go
```

- [ ] **Step 5: Commit**

```bash
git add cmd/main.go
git commit -m "подключены push_subscriptions service/usecases и push-notification worker в main.go"
```

---

### Task 18: Service Worker — sw.js

**Files:**
- Create: `internal/controllers/http/handlers/web/static/sw.js`

- [ ] **Step 1: Create Service Worker file**

Create `internal/controllers/http/handlers/web/static/sw.js`:

```js
self.addEventListener('push', (event) => {
    const data = event.data ? event.data.json() : {};

    const title = data.title || 'Новое сообщение';
    const options = {
        body: data.body || '',
        icon: '/web/static/img/icon-192.png',
        data: {
            chatId: data.chatId,
        },
    };

    event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
    event.notification.close();

    event.waitUntil(
        clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
            for (const client of windowClients) {
                if (client.url.includes('/web/chat') && 'focus' in client) {
                    return client.focus();
                }
            }

            return clients.openWindow('/web/chat');
        })
    );
});
```

- [ ] **Step 2: Serve sw.js from root scope**

The Service Worker must be served from `/sw.js` (or `/web/sw.js` at minimum) to control the app scope. Add a route in `internal/controllers/http/handlers/web/setup.go`:

```go
// Service Worker (served from /web/ scope):
getMux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, StaticFilePath+"sw.js")
})
```

Add `/web/sw.js` to the auth middleware ignore list in `controller.go`.

- [ ] **Step 3: Commit**

```bash
git add internal/controllers/http/handlers/web/static/sw.js internal/controllers/http/handlers/web/setup.go
git commit -m "добавлен Service Worker для Web Push уведомлений"
```

---

### Task 19: Frontend — push notification initialization in chat.js

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`

- [ ] **Step 1: Add initPushNotifications function**

Add at the end of `chat.js`:

```js
// ═══════════════════════════════════════
// Web Push уведомления
// ═══════════════════════════════════════
async function initPushNotifications() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
        console.log('Push notifications not supported');
        return;
    }

    try {
        const registration = await navigator.serviceWorker.register('/web/sw.js');

        const existingSubscription = await registration.pushManager.getSubscription();
        if (existingSubscription) {
            await sendSubscriptionToServer(existingSubscription);
            return;
        }

        if (Notification.permission === 'denied') {
            return;
        }

        if (Notification.permission === 'default') {
            const permission = await Notification.requestPermission();
            if (permission !== 'granted') {
                return;
            }
        }

        await subscribeToPush(registration);
    } catch (err) {
        console.log('Push init error:', err);
    }
}

async function subscribeToPush(registration) {
    try {
        const resp = await fetchWithAuth('/api/push/vapid-key');
        if (!resp.ok) return;

        const { publicKey } = await resp.json();

        const subscription = await registration.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey: urlBase64ToUint8Array(publicKey),
        });

        await sendSubscriptionToServer(subscription);
    } catch (err) {
        console.log('Push subscribe error:', err);
    }
}

async function sendSubscriptionToServer(subscription) {
    try {
        const resp = await fetchWithAuth('/api/push/subscribe', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                endpoint: subscription.endpoint,
                keys: {
                    p256dh: btoa(String.fromCharCode(...new Uint8Array(subscription.getKey('p256dh')))),
                    auth: btoa(String.fromCharCode(...new Uint8Array(subscription.getKey('auth')))),
                },
            }),
        });

        if (resp.ok) {
            const data = await resp.json();
            localStorage.setItem('pushSubscriptionId', data.id);
        }
    } catch (err) {
        console.log('Send subscription error:', err);
    }
}

function urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const rawData = atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
        outputArray[i] = rawData.charCodeAt(i);
    }

    return outputArray;
}
```

- [ ] **Step 2: Call initPushNotifications in DOMContentLoaded**

In the `DOMContentLoaded` handler, after `connectWebSocket();`, add:

```js
initPushNotifications();
```

- [ ] **Step 3: Test in browser**

1. Start the app: `task local`
2. Open http://localhost:8080/web/chat
3. Browser should ask for notification permission
4. Accept — verify subscription is created (check DB or network tab for POST /api/push/subscribe)
5. Open a second browser/incognito, send a message
6. Verify push notification appears when first browser tab is closed

- [ ] **Step 4: Commit**

```bash
git add internal/controllers/http/handlers/web/static/js/chat.js
git commit -m "добавлена инициализация Web Push подписки в chat.js"
```

---

### Task 20: Frontend — enable/disable push in profile settings

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/navbar.js`

- [ ] **Step 1: Add push notification toggle to profile modal**

In `navbar.js`, inside the `DOMContentLoaded` handler after the password form section, add a push notification toggle section.

Add a toggle button setup after `setupMyProfileToggle('my-profile-toggle-password', 'my-profile-password-form');`:

```js
// Push-уведомления
const pushToggle = document.getElementById('my-profile-toggle-push');
if (pushToggle) {
    updatePushToggleUI(pushToggle);

    pushToggle.addEventListener('click', async () => {
        if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
            showInfo('Push-уведомления не поддерживаются вашим браузером');
            return;
        }

        const registration = await navigator.serviceWorker.getRegistration('/web/sw.js');
        if (!registration) {
            showInfo('Service Worker не зарегистрирован. Обновите страницу.');
            return;
        }

        const subscription = await registration.pushManager.getSubscription();

        if (subscription) {
            // Отключаем
            const subId = localStorage.getItem('pushSubscriptionId');
            await subscription.unsubscribe();

            if (subId) {
                try {
                    await fetchWithAuth('/api/push/subscribe/' + subId, { method: 'DELETE' });
                } catch (err) {
                    console.log('Unsubscribe error:', err);
                }

                localStorage.removeItem('pushSubscriptionId');
            }

            showInfo('Push-уведомления отключены');
        } else {
            // Включаем
            if (Notification.permission === 'denied') {
                showInfo('Уведомления заблокированы в настройках браузера');
                return;
            }

            try {
                await subscribeToPush(registration);
                showInfo('Push-уведомления включены');
            } catch (err) {
                showError('Не удалось включить уведомления: ' + err.message);
            }
        }

        updatePushToggleUI(pushToggle);
    });
}

async function updatePushToggleUI(toggleEl) {
    const label = toggleEl.querySelector('.profile-modal__push-status');
    if (!label) return;

    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
        label.textContent = 'Не поддерживается';
        return;
    }

    const registration = await navigator.serviceWorker.getRegistration('/web/sw.js');
    if (!registration) {
        label.textContent = 'Отключены';
        return;
    }

    const subscription = await registration.pushManager.getSubscription();
    label.textContent = subscription ? 'Включены' : 'Отключены';
}
```

Note: `subscribeToPush` is defined in `chat.js`. Since both scripts are loaded on the chat page, this function is globally available. If navbar.js is also loaded on non-chat pages, wrap the call in `if (typeof subscribeToPush === 'function')`.

- [ ] **Step 2: Add HTML for push toggle in the profile modal template**

Add the push toggle button to the chat page HTML template (the same file that contains `modal-my-profile`). Add after the password change section:

```html
<div class="profile-modal__section" id="my-profile-toggle-push">
    <span class="profile-modal__section-title">Push-уведомления</span>
    <span class="profile-modal__push-status">...</span>
</div>
```

- [ ] **Step 3: Test in browser**

1. Open profile modal
2. Verify push status shows "Включены" or "Отключены"
3. Click to toggle — verify it works
4. Close and reopen — verify state persists

- [ ] **Step 4: Commit**

```bash
git add internal/controllers/http/handlers/web/static/js/navbar.js internal/controllers/http/handlers/web/templates/
git commit -m "добавлен переключатель push-уведомлений в профиле пользователя"
```

---

### Task 21: Update doc.md files

**Files:**
- Update all `doc.md` files in directories with changed code

Per project rules, update `doc.md` in each directory where code was changed:

- [ ] **Step 1: Update doc.md files**

Update the following `doc.md` files to reflect new/changed content:
- `internal/domains/doc.md` — add PushSubscription, PushNotificationDTO
- `internal/errors/doc.md` — add ErrPushSubscriptionNotFound
- `internal/interfaces/doc.md` — add PushSubscriptionsRepository, PushSubscriptionsService, PushSubscriptionsUseCases
- `internal/repositories/push_subscriptions/doc.md` — create, describe repository
- `internal/services/push_subscriptions/doc.md` — create, describe service
- `internal/usecases/push_subscriptions/doc.md` — create, describe usecases
- `internal/controllers/http/handlers/api/push/doc.md` — create, describe push handlers
- `internal/workers/handlers/builders/push_notification/doc.md` — create, describe worker
- `internal/config/doc.md` — add WebPushConfig, NATS push-notification config
- `migrations/doc.md` — add push_subscriptions migration

- [ ] **Step 2: Commit**

```bash
git add -A **/doc.md
git commit -m "обновлена документация doc.md для Web Push уведомлений"
```

---

### Task 22: End-to-end verification

- [ ] **Step 1: Set VAPID keys in environment**

Add to `.env` or environment:

```
VAPID_PUBLIC_KEY=<your-generated-public-key>
VAPID_PRIVATE_KEY=<your-generated-private-key>
VAPID_CONTACT=mailto:admin@kfc.com
```

- [ ] **Step 2: Run full application**

```bash
task local
```

- [ ] **Step 3: Test happy path**

1. Open browser A, log in as User A, allow notifications
2. Verify `push_subscriptions` table has a record for User A
3. Close browser A tab
4. Open browser B (or incognito), log in as User B
5. Send a message to a chat with User A
6. Verify native push notification appears on User A's device/browser
7. Click the notification — verify it opens the chat

- [ ] **Step 4: Test unsubscribe**

1. Open profile, click "Отключить уведомления"
2. Verify `push_subscriptions` record is deleted
3. Send another message — verify no push arrives
4. Click "Включить уведомления" — verify resubscription works

- [ ] **Step 5: Run all tests**

```bash
go test ./... -v
```
