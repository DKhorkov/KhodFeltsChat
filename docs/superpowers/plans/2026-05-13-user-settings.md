# User Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Store user settings (theme) server-side so any client can sync them.

**Architecture:** New `Settings` domain + full Clean Architecture stack (repository -> service -> usecases -> HTTP handlers). Settings created automatically during user registration inside AuthService's UoW transaction. Two endpoints: GET and PUT `/api/users/me/settings`.

**Tech Stack:** PostgreSQL (squirrel), OpenTelemetry tracing, gorilla/mux, existing UoW pattern.

---

### Task 1: Migration

**Files:**
- Create: `migrations/20260513000000_settings.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS user_settings
(
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER   NOT NULL UNIQUE,
    theme      INTEGER   NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX user_settings_user_id_idx ON user_settings (user_id);

INSERT INTO user_settings (user_id)
SELECT id FROM users;

-- +goose Down
DROP TABLE IF EXISTS user_settings;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/20260513000000_settings.sql
git commit -m "feat: add user_settings migration"
```

---

### Task 2: Domain model

**Files:**
- Create: `internal/domains/settings.go`

- [ ] **Step 1: Create the domain file**

```go
package domains

import "time"

type ThemeType int

const (
	ThemeLight ThemeType = iota
	ThemeDark
)

type Settings struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"userID"`
	Theme     ThemeType `json:"theme"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/domains/settings.go
git commit -m "feat: add Settings domain model"
```

---

### Task 3: Interfaces

**Files:**
- Modify: `internal/interfaces/repositories.go`
- Modify: `internal/interfaces/services.go`
- Modify: `internal/interfaces/usecases.go`

- [ ] **Step 1: Add SettingsRepository interface**

Append to `internal/interfaces/repositories.go`:

```go
//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/settings_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository
type SettingsRepository interface {
	CreateSettings(ctx context.Context, settings domains.Settings) error
	GetSettingsByUserID(ctx context.Context, userID uint64) (*domains.Settings, error)
	UpdateSettings(ctx context.Context, settings domains.Settings) error
}
```

- [ ] **Step 2: Add SettingsService interface**

Append to `internal/interfaces/services.go`:

```go
//go:generate mockgen -source=services.go -destination=../../mocks/services/settings_service.go -package=mockservices -exclude_interfaces=AuthService,UsersService,ChatsService,MessagesService,NotificationsService
type SettingsService interface {
	GetSettingsByUserID(ctx context.Context, userID uint64) (*domains.Settings, error)
	UpdateSettings(ctx context.Context, settings domains.Settings) (*domains.Settings, error)
}
```

- [ ] **Step 3: Add SettingsUseCases interface**

Append to `internal/interfaces/usecases.go`:

```go
//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/settings_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases
type SettingsUseCases interface {
	GetSettingsByUserID(ctx context.Context, userID uint64) (*domains.Settings, error)
	UpdateSettings(ctx context.Context, settings domains.Settings) (*domains.Settings, error)
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/interfaces/repositories.go internal/interfaces/services.go internal/interfaces/usecases.go
git commit -m "feat: add Settings interfaces for all layers"
```

---

### Task 4: Settings Repository

**Files:**
- Create: `internal/repositories/settings/repository.go`
- Create: `internal/repositories/settings/trace_decorator.go`

- [ ] **Step 1: Create repository**

```go
package settings

import (
	"context"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	sq "github.com/Masterminds/squirrel"
)

const (
	tableName = "user_settings"

	idColumnName        = "id"
	userIDColumnName    = "user_id"
	themeColumnName     = "theme"
	createdAtColumnName = "created_at"
	updatedAtColumnName = "updated_at"

	selectAllColumns = "*"
)

type Repository struct {
	tx pg.Transaction
}

func New(tx pg.Transaction) *Repository {
	return &Repository{tx: tx}
}

func (repo *Repository) CreateSettings(ctx context.Context, settings domains.Settings) error {
	stmt, params, err := sq.
		Insert(tableName).
		Columns(userIDColumnName, themeColumnName).
		Values(settings.UserID, settings.Theme).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)
	return err
}

func (repo *Repository) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(tableName).
		Where(sq.Eq{userIDColumnName: userID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	settings := &domains.Settings{}
	columns := pg.GetEntityColumns(settings)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return settings, nil
}

func (repo *Repository) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) error {
	builder := sq.
		Update(tableName).
		Where(sq.Eq{userIDColumnName: settings.UserID}).
		Set(themeColumnName, settings.Theme).
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

- [ ] **Step 2: Create trace decorator**

```go
package settings

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.SettingsRepository
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.SettingsRepository,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) CreateSettings(ctx context.Context, settings domains.Settings) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.CreateSettings(ctx, settings)
}

func (d *TraceDecorator) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetSettingsByUserID(ctx, userID)
}

func (d *TraceDecorator) UpdateSettings(ctx context.Context, settings domains.Settings) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateSettings(ctx, settings)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/repositories/settings/
git commit -m "feat: add Settings repository with trace decorator"
```

---

### Task 5: Settings Service

**Files:**
- Create: `internal/services/settings/service.go`
- Create: `internal/services/settings/trace_decorator.go`

- [ ] **Step 1: Create service**

```go
package settings

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                       interfaces.UnitOfWork
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository
}

func New(
	uow interfaces.UnitOfWork,
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository,
) *Service {
	return &Service{
		uow:                       uow,
		newSettingsRepositoryFunc: newSettingsRepositoryFunc,
	}
}

func (s *Service) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	var (
		settings *domains.Settings
		err      error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			settingsRepository := s.newSettingsRepositoryFunc(tx)
			if settings, err = settingsRepository.GetSettingsByUserID(ctx, userID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", customerrors.ErrSettingsNotFound, err)
	}

	return settings, nil
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	settingsData domains.Settings,
) (*domains.Settings, error) {
	var (
		settings *domains.Settings
		err      error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			settingsRepository := s.newSettingsRepositoryFunc(tx)

			if err = settingsRepository.UpdateSettings(ctx, settingsData); err != nil {
				return err
			}

			if settings, err = settingsRepository.GetSettingsByUserID(ctx, settingsData.UserID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return settings, nil
}
```

- [ ] **Step 2: Create trace decorator**

```go
package settings

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.SettingsService
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.SettingsService,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetSettingsByUserID(ctx, userID)
}

func (d *TraceDecorator) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) (*domains.Settings, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateSettings(ctx, settings)
}
```

- [ ] **Step 3: Add sentinel error**

Add to `internal/errors/settings.go`:

```go
package errors

import "errors"

var ErrSettingsNotFound = errors.New("settings not found")
```

- [ ] **Step 4: Commit**

```bash
git add internal/services/settings/ internal/errors/settings.go
git commit -m "feat: add Settings service with trace decorator"
```

---

### Task 6: Settings UseCases

**Files:**
- Create: `internal/usecases/settings/usecases.go`
- Create: `internal/usecases/settings/trace_decorator.go`

- [ ] **Step 1: Create usecases**

```go
package settings

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	settingsService interfaces.SettingsService
}

func New(settingsService interfaces.SettingsService) *UseCases {
	return &UseCases{settingsService: settingsService}
}

func (u *UseCases) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	return u.settingsService.GetSettingsByUserID(ctx, userID)
}

func (u *UseCases) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) (*domains.Settings, error) {
	return u.settingsService.UpdateSettings(ctx, settings)
}
```

- [ ] **Step 2: Create trace decorator**

```go
package settings

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.SettingsUseCases
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.SettingsUseCases,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetSettingsByUserID(ctx, userID)
}

func (d *TraceDecorator) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) (*domains.Settings, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateSettings(ctx, settings)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/usecases/settings/
git commit -m "feat: add Settings usecases with trace decorator"
```

---

### Task 7: AuthService — create settings on registration

**Files:**
- Modify: `internal/services/auth/service.go`

- [ ] **Step 1: Add `newSettingsRepositoryFunc` to AuthService**

In `internal/services/auth/service.go`, add the field to the `Service` struct and the `New` constructor:

```go
type Service struct {
	uow                       interfaces.UnitOfWork
	newAuthRepositoryFunc     func(tx pg.Transaction) interfaces.AuthRepository
	newUsersRepositoryFunc    func(tx pg.Transaction) interfaces.UsersRepository
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository
	natsPublisher             customnats.Publisher
	natsConfig                config.NATSConfig
}

func New(
	uow interfaces.UnitOfWork,
	newAuthRepositoryFunc func(tx pg.Transaction) interfaces.AuthRepository,
	newUsersRepositoryFunc func(tx pg.Transaction) interfaces.UsersRepository,
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository,
	natsPublisher customnats.Publisher,
	natsConfig config.NATSConfig,
) *Service {
	return &Service{
		uow:                       uow,
		newAuthRepositoryFunc:     newAuthRepositoryFunc,
		newUsersRepositoryFunc:    newUsersRepositoryFunc,
		newSettingsRepositoryFunc: newSettingsRepositoryFunc,
		natsPublisher:             natsPublisher,
		natsConfig:                natsConfig,
	}
}
```

- [ ] **Step 2: Create default settings inside RegisterUser**

In the `RegisterUser` method, after getting the user by email (line 75-78), add settings creation before the NATS publish:

```go
			user, err = usersRepository.GetUserByEmail(ctx, userData.Email)
			if err != nil {
				return err
			}

			settingsRepository := s.newSettingsRepositoryFunc(tx)
			if err = settingsRepository.CreateSettings(ctx, domains.Settings{
				UserID: user.ID,
				Theme:  domains.ThemeLight,
			}); err != nil {
				return err
			}

			verifyEmailDTO := &domains.VerifyEmailNotificationDTO{
```

- [ ] **Step 3: Commit**

```bash
git add internal/services/auth/service.go
git commit -m "feat: create default settings on user registration"
```

---

### Task 8: HTTP layer — schemas, mappers, handlers

**Files:**
- Create: `internal/controllers/http/schemas/settings.go`
- Create: `internal/controllers/http/mappers/settings/settings.go`
- Create: `internal/controllers/http/handlers/api/users/settings/get/handler.go`
- Create: `internal/controllers/http/handlers/api/users/settings/update/handler.go`

- [ ] **Step 1: Create schema**

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
}

// GetSettingsInput
// swagger:parameters GetSettings
type GetSettingsInput struct{}

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

- [ ] **Step 2: Create mapper**

```go
package settings

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapSettings(settings domains.Settings) schemas.Settings {
	return schemas.Settings{
		Theme: int(settings.Theme),
	}
}
```

- [ ] **Step 3: Create GET handler**

```go
package get

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/settings"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route GET /api/users/me/settings settings GetSettings
//
// GetSettings
//
// Provides settings of the current authorized User.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: Settings
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handler provides settings of the current authorized User.
func Handler(s interfaces.SettingsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		settings, err := s.GetSettingsByUserID(r.Context(), userID)

		switch {
		case errors.Is(err, customerrors.ErrSettingsNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusOK)

		if err = json.NewEncoder(w).Encode(mappers.MapSettings(*settings)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
```

- [ ] **Step 4: Create PUT handler**

```go
package update

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/settings"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route PUT /api/users/me/settings settings UpdateSettings
//
// UpdateSettings
//
// Updates settings of the current authorized User.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: Settings
//	400: BadRequest
//	401: Unauthorized
//	404: NotFound
//	500: InternalServerError

// Handler updates settings of the current authorized User.
func Handler(s interfaces.SettingsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var settingsData domains.Settings
		if err = json.Unmarshal(data, &settingsData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		settingsData.UserID = userID

		settings, err := s.UpdateSettings(r.Context(), settingsData)

		switch {
		case errors.Is(err, customerrors.ErrSettingsNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusOK)

		if err = json.NewEncoder(w).Encode(mappers.MapSettings(*settings)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/controllers/http/schemas/settings.go internal/controllers/http/mappers/settings/ internal/controllers/http/handlers/api/users/settings/
git commit -m "feat: add HTTP handlers for user settings"
```

---

### Task 9: Register routes and wire DI

**Files:**
- Modify: `internal/controllers/http/handlers/api/setup.go`
- Modify: `internal/controllers/http/handlers/setup.go`
- Modify: `internal/controllers/http/controller.go`
- Modify: `internal/config/config.go` (SpanRepositories, SpanServices, SpanUseCases + SpanConfig entries)
- Modify: `cmd/main.go`

- [ ] **Step 1: Add route constants and register handlers in `internal/controllers/http/handlers/api/setup.go`**

Add import:
```go
get_settings "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/settings/get"
update_settings "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/settings/update"
```

Add constant:
```go
SettingsURL = MeURL + "/settings"
```

Add `settingsUseCases interfaces.SettingsUseCases` parameter to `SetupHandlers`.

Add handler registrations:
```go
getMux.Handle(SettingsURL, get_settings.Handler(settingsUseCases))
```
```go
putMux.Handle(SettingsURL, update_settings.Handler(settingsUseCases))
```

- [ ] **Step 2: Pass `settingsUseCases` through `internal/controllers/http/handlers/setup.go`**

Add `settingsUseCases interfaces.SettingsUseCases` parameter to `SetupHandlers` and pass it to `api.SetupHandlers`.

- [ ] **Step 3: Pass `settingsUseCases` through `internal/controllers/http/controller.go`**

Add `settingsUseCases interfaces.SettingsUseCases` parameter to `New` and pass it to `handlers.SetupHandlers`.

- [ ] **Step 4: Add Settings SpanConfig entries to `internal/config/config.go`**

Add `Settings tracing.SpanConfig` field to `SpanRepositories`, `SpanServices`, and `SpanUseCases` structs.

Add the SpanConfig initializations in `New()` function, following the exact pattern of other spans. For repositories:

```go
Settings: tracing.SpanConfig{
	Name: "Settings repository",
	Opts: []trace.SpanStartOption{
		trace.WithAttributes(
			attribute.String(
				"Environment",
				loadenv.GetEnv("ENVIRONMENT", "local"),
			),
		),
	},
	Events: tracing.SpanEventsConfig{
		Start: tracing.SpanEventConfig{
			Name: "Calling Settings Repository",
			Opts: []trace.EventOption{
				trace.WithAttributes(
					attribute.String(
						"Environment",
						loadenv.GetEnv("ENVIRONMENT", "local"),
					),
				),
			},
		},
		End: tracing.SpanEventConfig{
			Name: "Received response from Settings Repository",
			Opts: []trace.EventOption{
				trace.WithAttributes(
					attribute.String(
						"Environment",
						loadenv.GetEnv("ENVIRONMENT", "local"),
					),
				),
			},
		},
	},
},
```

Same pattern for services (name: "Settings service", events: "Calling Settings Service" / "Received response from Settings Service") and usecases (name: "Settings useCases", events: "Calling Settings UseCases" / "Received response from Settings UseCases").

- [ ] **Step 5: Wire everything in `cmd/main.go`**

Add imports:
```go
settingsrepository "github.com/DKhorkov/kfc/internal/repositories/settings"
settingsservice "github.com/DKhorkov/kfc/internal/services/settings"
settingsusecases "github.com/DKhorkov/kfc/internal/usecases/settings"
```

Create settings service (after `usersService`):
```go
settingsService := settingsservice.NewTraceDecorator(
	traceProvider,
	cfg.Tracing.Spans.Services.Settings,
	settingsservice.New(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.SettingsRepository {
			return settingsrepository.NewTraceDecorator(
				traceProvider,
				cfg.Tracing.Spans.Repositories.Settings,
				settingsrepository.New(tx),
			)
		},
	),
)
```

Add `newSettingsRepositoryFunc` to `authService.New` call (the 4th parameter, after `newUsersRepositoryFunc`):
```go
authService := authservice.NewTraceDecorator(
	traceProvider,
	cfg.Tracing.Spans.Services.Auth,
	authservice.New(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.AuthRepository {
			return authrepository.NewTraceDecorator(
				traceProvider,
				cfg.Tracing.Spans.Repositories.Auth,
				authrepository.New(tx),
			)
		},
		func(tx postgresql.Transaction) interfaces.UsersRepository {
			return usersrepository.NewTraceDecorator(
				traceProvider,
				cfg.Tracing.Spans.Repositories.Users,
				usersrepository.New(tx, logger),
			)
		},
		func(tx postgresql.Transaction) interfaces.SettingsRepository {
			return settingsrepository.NewTraceDecorator(
				traceProvider,
				cfg.Tracing.Spans.Repositories.Settings,
				settingsrepository.New(tx),
			)
		},
		natsPublisher,
		cfg.NATS,
	),
)
```

Create settings usecases (after `usersUseCases`):
```go
settingsUseCases := settingsusecases.NewTraceDecorator(
	traceProvider,
	cfg.Tracing.Spans.UseCases.Settings,
	settingsusecases.New(settingsService),
)
```

Add `settingsUseCases` to `controllers.New` call (after `messagesUseCases`).

- [ ] **Step 6: Build and verify**

Run: `go build ./...`
Expected: no compilation errors.

- [ ] **Step 7: Commit**

```bash
git add internal/controllers/http/handlers/api/setup.go internal/controllers/http/handlers/setup.go internal/controllers/http/controller.go internal/config/config.go cmd/main.go
git commit -m "feat: wire Settings through DI, routes, and tracing config"
```
