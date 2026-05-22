# Multi-Session Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Поддержка нескольких одновременных сессий (refresh tokens) для одного пользователя, позволяющая быть залогиненным с нескольких устройств одновременно.

**Architecture:** Замена `GetRefreshTokenByUserID` на `GetRefreshTokenByValue` во всех слоях (repository → service → usecases). Добавление `ExpireAllUserRefreshTokens` для logout со всех устройств. Упрощение `RefreshTokens` flow — поиск токена по значению вместо парсинга вложенного JWT. Новый endpoint `DELETE /api/sessions/all`.

**Tech Stack:** Go 1.24, PostgreSQL (squirrel), gorilla/mux, mockgen, testify, gomock

**Spec:** `docs/superpowers/specs/2026-05-22-multi-session-design.md`

---

### Task 1: Обновить интерфейс AuthRepository

**Files:**
- Modify: `internal/interfaces/repositories.go:36-48`

- [ ] **Step 1: Заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue` и добавить `ExpireAllUserRefreshTokens`**

В файле `internal/interfaces/repositories.go` заменить блок `AuthRepository`:

```go
type AuthRepository interface {
	RegisterUser(ctx context.Context, userData domains.RegisterDTO) (userID uint64, err error)
	CreateRefreshToken(
		ctx context.Context,
		userID uint64,
		value string,
		ttl time.Duration,
	) (refreshTokenID uint64, err error)
	GetRefreshTokenByValue(ctx context.Context, value string) (*domains.RefreshToken, error)
	ExpireRefreshToken(ctx context.Context, refreshTokenID uint64) error
	ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error
	VerifyEmail(ctx context.Context, userID uint64) error
	ChangePassword(ctx context.Context, userID uint64, newPassword string) error
}
```

Изменения по сравнению с текущим:
- `GetRefreshTokenByUserID(ctx, userID uint64)` → `GetRefreshTokenByValue(ctx, value string)`
- Добавлен `ExpireAllUserRefreshTokens(ctx, userID uint64) error`

- [ ] **Step 2: Убедиться что компиляция ломается в ожидаемых местах**

Run: `go build ./...`
Expected: FAIL — ошибки компиляции в repository, service, usecases, trace decorators, mocks. Это ожидаемо — остальные задачи исправят это.

- [ ] **Step 3: Commit**

```bash
git add internal/interfaces/repositories.go
git commit -m "refactor: update AuthRepository interface for multi-session support"
```

---

### Task 2: Обновить интерфейс AuthService

**Files:**
- Modify: `internal/interfaces/services.go:24-39`

- [ ] **Step 1: Заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue` и добавить `ExpireAllUserRefreshTokens`**

В файле `internal/interfaces/services.go` заменить блок `AuthService`:

```go
type AuthService interface {
	RegisterUser(ctx context.Context, userData domains.RegisterDTO) (*domains.User, error)
	CreateRefreshToken(
		ctx context.Context,
		userID uint64,
		value string,
		ttl time.Duration,
	) (*domains.RefreshToken, error)
	GetRefreshTokenByValue(ctx context.Context, value string) (*domains.RefreshToken, error)
	ExpireRefreshToken(ctx context.Context, refreshTokenID uint64) error
	ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error
	VerifyEmail(ctx context.Context, userID uint64) error
	ForgetPassword(ctx context.Context, userID uint64, newPassword string) error
	ChangePassword(ctx context.Context, userID uint64, newPassword string) error
	SendForgetPasswordMessage(ctx context.Context, email string) error
	SendVerifyEmailMessage(ctx context.Context, email string) error
}
```

Изменения:
- `GetRefreshTokenByUserID(ctx, userID uint64)` → `GetRefreshTokenByValue(ctx, value string)`
- Добавлен `ExpireAllUserRefreshTokens(ctx, userID uint64) error`

- [ ] **Step 2: Commit**

```bash
git add internal/interfaces/services.go
git commit -m "refactor: update AuthService interface for multi-session support"
```

---

### Task 3: Обновить интерфейс AuthUseCases

**Files:**
- Modify: `internal/interfaces/usecases.go:21-31`

- [ ] **Step 1: Изменить `LogoutUser` и добавить `LogoutUserFromAllSessions`**

В файле `internal/interfaces/usecases.go` заменить блок `AuthUseCases`:

```go
type AuthUseCases interface {
	RegisterUser(ctx context.Context, dto domains.RegisterDTO) (*domains.User, error)
	LoginUser(ctx context.Context, dto domains.LoginDTO) (*domains.TokensDTO, error)
	LogoutUser(ctx context.Context, refreshToken string) error
	LogoutUserFromAllSessions(ctx context.Context, userID uint64) error
	RefreshTokens(ctx context.Context, refreshToken string) (*domains.TokensDTO, error)
	VerifyEmail(ctx context.Context, verifyEmailToken string) error
	ForgetPassword(ctx context.Context, forgetPasswordToken, newPassword string) error
	SendForgetPasswordMessage(ctx context.Context, email string) error
	ChangePassword(ctx context.Context, dto domains.ChangePasswordDTO) error
	SendVerifyEmailMessage(ctx context.Context, email string) error
}
```

Изменения:
- `LogoutUser(ctx, userID uint64)` → `LogoutUser(ctx, refreshToken string)`
- Добавлен `LogoutUserFromAllSessions(ctx, userID uint64) error`

- [ ] **Step 2: Commit**

```bash
git add internal/interfaces/usecases.go
git commit -m "refactor: update AuthUseCases interface for multi-session support"
```

---

### Task 4: Перегенерировать моки

**Files:**
- Modify: `mocks/repositories/auth_repository.go`
- Modify: `mocks/services/auth_service.go`
- Modify: `mocks/usecases/auth_usecases.go`

- [ ] **Step 1: Перегенерировать все моки**

Run:
```bash
go generate ./internal/interfaces/...
```

- [ ] **Step 2: Проверить что моки сгенерировались**

Run: `grep -l "GetRefreshTokenByValue\|ExpireAllUserRefreshTokens\|LogoutUserFromAllSessions" mocks/`
Expected: файлы `auth_repository.go`, `auth_service.go`, `auth_usecases.go` содержат новые методы.

- [ ] **Step 3: Commit**

```bash
git add mocks/
git commit -m "chore: regenerate mocks for updated auth interfaces"
```

---

### Task 5: Реализовать `GetRefreshTokenByValue` и `ExpireAllUserRefreshTokens` в репозитории

**Files:**
- Modify: `internal/repositories/auth/repository.go:106-133` (заменить `GetRefreshTokenByUserID`)
- Modify: `internal/repositories/auth/repository.go:135-152` (после `ExpireRefreshToken`)

- [ ] **Step 1: Заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue`**

В файле `internal/repositories/auth/repository.go` заменить метод `GetRefreshTokenByUserID` (строки 106-133):

```go
func (repo *Repository) GetRefreshTokenByValue(
	ctx context.Context,
	value string,
) (*domains.RefreshToken, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(refreshTokensTableName).
		Where(sq.Eq{refreshTokenValueColumnName: value}).
		Where(
			sq.Expr(
				refreshTokenTTLColumnName + " > CURRENT_TIMESTAMP",
			),
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	refreshToken := &domains.RefreshToken{}

	columns := pg.GetEntityColumns(refreshToken)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return refreshToken, nil
}
```

- [ ] **Step 2: Добавить `ExpireAllUserRefreshTokens` после `ExpireRefreshToken`**

```go
func (repo *Repository) ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error {
	stmt, params, err := sq.
		Delete(refreshTokensTableName).
		Where(sq.Eq{userIDColumnName: userID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(
		ctx,
		stmt,
		params...,
	)

	return err
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/repositories/auth/repository.go
git commit -m "feat: implement GetRefreshTokenByValue and ExpireAllUserRefreshTokens in repository"
```

---

### Task 6: Обновить trace decorator репозитория

**Files:**
- Modify: `internal/repositories/auth/trace_decorator.go:58-69` (заменить `GetRefreshTokenByUserID`)
- Modify: `internal/repositories/auth/trace_decorator.go` (добавить новый метод после `ExpireRefreshToken`)

- [ ] **Step 1: Заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue`**

В файле `internal/repositories/auth/trace_decorator.go` заменить метод (строки 58-69):

```go
func (d *TraceDecorator) GetRefreshTokenByValue(
	ctx context.Context,
	value string,
) (*domains.RefreshToken, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetRefreshTokenByValue(ctx, value)
}
```

- [ ] **Step 2: Добавить `ExpireAllUserRefreshTokens` после `ExpireRefreshToken`**

```go
func (d *TraceDecorator) ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.ExpireAllUserRefreshTokens(ctx, userID)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/repositories/auth/trace_decorator.go
git commit -m "feat: update repository trace decorator for multi-session"
```

---

### Task 7: Обновить сервисный слой

**Files:**
- Modify: `internal/services/auth/service.go:121-159` (метод `CreateRefreshToken`)
- Modify: `internal/services/auth/service.go:161-186` (заменить `GetRefreshTokenByUserID`)
- Modify: `internal/services/auth/service.go:210-235` (метод `ForgetPassword`)
- Add new method: `ExpireAllUserRefreshTokens`

- [ ] **Step 1: Обновить `CreateRefreshToken` — заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue`**

В файле `internal/services/auth/service.go` заменить метод `CreateRefreshToken` (строки 121-159):

```go
func (s *Service) CreateRefreshToken(
	ctx context.Context,
	userID uint64,
	value string,
	ttl time.Duration,
) (*domains.RefreshToken, error) {
	var (
		refreshToken *domains.RefreshToken
		err          error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)

			_, err = authRepository.CreateRefreshToken(
				ctx,
				userID,
				value,
				ttl,
			)
			if err != nil {
				return err
			}

			if refreshToken, err = authRepository.GetRefreshTokenByValue(ctx, value); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}
```

- [ ] **Step 2: Заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue`**

Заменить метод `GetRefreshTokenByUserID` (строки 161-186):

```go
func (s *Service) GetRefreshTokenByValue(
	ctx context.Context,
	value string,
) (*domains.RefreshToken, error) {
	var (
		refreshToken *domains.RefreshToken
		err          error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			if refreshToken, err = authRepository.GetRefreshTokenByValue(ctx, value); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}
```

- [ ] **Step 3: Обновить `ForgetPassword` — заменить `GetRefreshTokenByUserID` + `ExpireRefreshToken` на `ExpireAllUserRefreshTokens`**

Заменить метод `ForgetPassword` (строки 210-235):

```go
func (s *Service) ForgetPassword(
	ctx context.Context,
	userID uint64,
	newPassword string,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			if err := authRepository.ChangePassword(ctx, userID, newPassword); err != nil {
				return err
			}

			return authRepository.ExpireAllUserRefreshTokens(ctx, userID)
		},
	)
}
```

- [ ] **Step 4: Добавить `ExpireAllUserRefreshTokens` после `ExpireRefreshToken`**

```go
func (s *Service) ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)

			return authRepository.ExpireAllUserRefreshTokens(ctx, userID)
		},
	)
}
```

- [ ] **Step 5: Удалить импорт `"database/sql"` и `"errors"` если они больше не используются**

Проверить что `errors.Is(err, sql.ErrNoRows)` больше нигде не используется в файле (был только в `ForgetPassword`). Если так — удалить импорты `"database/sql"` и `"errors"`.

- [ ] **Step 6: Commit**

```bash
git add internal/services/auth/service.go
git commit -m "feat: update auth service for multi-session support"
```

---

### Task 8: Обновить trace decorator сервиса

**Files:**
- Modify: `internal/services/auth/trace_decorator.go:58-69` (заменить `GetRefreshTokenByUserID`)
- Add new method after `ExpireRefreshToken`

- [ ] **Step 1: Заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue`**

В файле `internal/services/auth/trace_decorator.go` заменить метод (строки 58-69):

```go
func (d *TraceDecorator) GetRefreshTokenByValue(
	ctx context.Context,
	value string,
) (*domains.RefreshToken, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetRefreshTokenByValue(ctx, value)
}
```

- [ ] **Step 2: Добавить `ExpireAllUserRefreshTokens` после `ExpireRefreshToken`**

```go
func (d *TraceDecorator) ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.ExpireAllUserRefreshTokens(ctx, userID)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/services/auth/trace_decorator.go
git commit -m "feat: update service trace decorator for multi-session"
```

---

### Task 9: Обновить usecases

**Files:**
- Modify: `internal/usecases/auth/usecases.go:66-143` (метод `LoginUser`)
- Modify: `internal/usecases/auth/usecases.go:145-242` (метод `RefreshTokens`)
- Modify: `internal/usecases/auth/usecases.go:244-251` (метод `LogoutUser`)
- Add new method: `LogoutUserFromAllSessions`

- [ ] **Step 1: Упростить `LoginUser` — убрать удаление старого refresh token**

В файле `internal/usecases/auth/usecases.go` заменить метод `LoginUser` (строки 66-143):

```go
func (u *UseCases) LoginUser(
	ctx context.Context,
	dto domains.LoginDTO,
) (*domains.TokensDTO, error) {
	if dto.Login == "" {
		return nil, fmt.Errorf("%w: login is required", customerrors.ErrValidationFailed)
	}

	if dto.Password == "" {
		return nil, fmt.Errorf("%w: password is required", customerrors.ErrValidationFailed)
	}

	user, _ := u.usersService.GetUserByEmail(ctx, dto.Login)

	// Fallback логина по имени пользователя
	if user == nil {
		user, _ = u.usersService.GetUserByUsername(ctx, dto.Login)
	}

	// Если пользователь все еще не найден - значит такого пользователя не существует
	if user == nil {
		return nil, customerrors.ErrUserNotFound
	}

	if !user.EmailConfirmed {
		return nil, customerrors.ErrEmailNotConfirmed
	}

	if !security.ValidateHash(dto.Password, user.Password) {
		return nil, customerrors.ErrWrongPassword
	}

	// Create tokens:
	accessToken, err := security.GenerateJWT(
		user.ID,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.AccessTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := security.GenerateJWT(
		user.ID,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.RefreshTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	// Save token to Database:
	if _, err = u.authService.CreateRefreshToken(
		ctx,
		user.ID,
		refreshToken,
		u.securityConfig.JWT.RefreshTokenTTL,
	); err != nil {
		return nil, err
	}

	// Encoding refresh token for secure usage via internet:
	encodedRefreshToken := security.RawEncode([]byte(refreshToken))

	return &domains.TokensDTO{
		AccessToken:  accessToken,
		RefreshToken: encodedRefreshToken,
	}, nil
}
```

Ключевые изменения:
- Удалён блок `GetRefreshTokenByUserID` + `ExpireRefreshToken` (строки 98-103 текущего кода)
- Refresh token теперь подписывает `user.ID` вместо `accessToken` (убираем связку access↔refresh)

- [ ] **Step 2: Упростить `RefreshTokens` — поиск по значению вместо парсинга JWT**

Заменить метод `RefreshTokens` (строки 145-242):

```go
func (u *UseCases) RefreshTokens(
	ctx context.Context,
	refreshToken string,
) (*domains.TokensDTO, error) {
	// Decoding refresh token to get original JWT value:
	oldRefreshTokenBytes, err := security.RawDecode(refreshToken)
	if err != nil {
		return nil, customerrors.ErrInvalidJWT
	}

	oldRefreshToken := string(oldRefreshTokenBytes)

	// Find refresh token in database by value:
	dbRefreshToken, err := u.authService.GetRefreshTokenByValue(ctx, oldRefreshToken)
	if err != nil {
		return nil, customerrors.ErrInvalidJWT
	}

	userID := dbRefreshToken.UserID

	// Expire old refresh token:
	if err = u.authService.ExpireRefreshToken(ctx, dbRefreshToken.ID); err != nil {
		return nil, err
	}

	// Create new tokens:
	newAccessToken, err := security.GenerateJWT(
		userID,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.AccessTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := security.GenerateJWT(
		userID,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.RefreshTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	// Save new token to Database:
	if _, err = u.authService.CreateRefreshToken(
		ctx,
		userID,
		newRefreshToken,
		u.securityConfig.JWT.RefreshTokenTTL,
	); err != nil {
		return nil, err
	}

	// Encoding refresh token for secure usage via internet:
	encodedRefreshToken := security.RawEncode([]byte(newRefreshToken))

	return &domains.TokensDTO{
		AccessToken:  newAccessToken,
		RefreshToken: encodedRefreshToken,
	}, nil
}
```

Ключевые изменения:
- Убрано двойное парсирование JWT (refresh → access → userID)
- `GetRefreshTokenByUserID` → `GetRefreshTokenByValue`
- Убрана проверка `oldRefreshToken != dbRefreshToken.Value` (поиск по значению делает её ненужной)
- `userID` берётся из `dbRefreshToken.UserID`

- [ ] **Step 3: Обновить `LogoutUser` — принимать refreshToken вместо userID**

Заменить метод `LogoutUser` (строки 244-251):

```go
func (u *UseCases) LogoutUser(ctx context.Context, refreshToken string) error {
	decodedTokenBytes, err := security.RawDecode(refreshToken)
	if err != nil {
		return nil
	}

	dbRefreshToken, err := u.authService.GetRefreshTokenByValue(ctx, string(decodedTokenBytes))
	if err != nil {
		return nil
	}

	return u.authService.ExpireRefreshToken(ctx, dbRefreshToken.ID)
}
```

- [ ] **Step 4: Добавить `LogoutUserFromAllSessions`**

```go
func (u *UseCases) LogoutUserFromAllSessions(ctx context.Context, userID uint64) error {
	return u.authService.ExpireAllUserRefreshTokens(ctx, userID)
}
```

- [ ] **Step 5: Удалить неиспользуемые импорты**

Проверить что импорты `"strconv"`, `"strings"`, `"github.com/golang-jwt/jwt/v5"` и `"github.com/DKhorkov/kfc/internal/common"` больше не нужны в `usecases.go` (они использовались в старом `RefreshTokens`). Удалить если так.

Оставить: `"fmt"`, `"github.com/DKhorkov/kfc/internal/config"`, `"github.com/DKhorkov/kfc/internal/domains"`, `customerrors`, `"github.com/DKhorkov/kfc/internal/interfaces"`, `"github.com/DKhorkov/libs/security"`, `"github.com/DKhorkov/libs/validation"`.

Убрать: `"strconv"`, `"strings"`, `"github.com/golang-jwt/jwt/v5"`, `"github.com/DKhorkov/kfc/internal/common"`.

- [ ] **Step 6: Commit**

```bash
git add internal/usecases/auth/usecases.go
git commit -m "feat: implement multi-session logic in usecases"
```

---

### Task 10: Обновить trace decorator usecases

**Files:**
- Modify: `internal/usecases/auth/trace_decorator.go:68-76` (метод `LogoutUser`)
- Add new method: `LogoutUserFromAllSessions`

- [ ] **Step 1: Обновить `LogoutUser` — принимать `refreshToken string`**

В файле `internal/usecases/auth/trace_decorator.go` заменить метод `LogoutUser` (строки 68-76):

```go
func (d *TraceDecorator) LogoutUser(ctx context.Context, refreshToken string) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.LogoutUser(ctx, refreshToken)
}
```

- [ ] **Step 2: Добавить `LogoutUserFromAllSessions` после `LogoutUser`**

```go
func (d *TraceDecorator) LogoutUserFromAllSessions(ctx context.Context, userID uint64) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.LogoutUserFromAllSessions(ctx, userID)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/usecases/auth/trace_decorator.go
git commit -m "feat: update usecases trace decorator for multi-session"
```

---

### Task 11: Обновить cache decorator usecases

**Files:**
- Modify: `internal/usecases/auth/cache_decorator.go:57-59` (метод `LogoutUser`)
- Add new method: `LogoutUserFromAllSessions`

- [ ] **Step 1: Обновить `LogoutUser` — принимать `refreshToken string`**

В файле `internal/usecases/auth/cache_decorator.go` заменить метод `LogoutUser` (строки 57-59):

```go
func (d *CacheDecorator) LogoutUser(ctx context.Context, refreshToken string) error {
	return d.base.LogoutUser(ctx, refreshToken)
}
```

- [ ] **Step 2: Добавить `LogoutUserFromAllSessions` после `LogoutUser`**

```go
func (d *CacheDecorator) LogoutUserFromAllSessions(ctx context.Context, userID uint64) error {
	return d.base.LogoutUserFromAllSessions(ctx, userID)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/usecases/auth/cache_decorator.go
git commit -m "feat: update cache decorator for multi-session"
```

---

### Task 12: Обновить logout handler

**Files:**
- Modify: `internal/controllers/http/handlers/api/auth/logout/handler.go`

- [ ] **Step 1: Переписать handler — читать refresh token из cookie**

Заменить содержимое файла `internal/controllers/http/handlers/api/auth/logout/handler.go`:

```go
package logout

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/cookies"
)

// swagger:route DELETE /api/sessions sessions Logout
//
// Logout
//
// Logout User from the current session and deletes access and refresh tokens.
//
// Responses:
//	204: NoContent
//	401: Unauthorized
//	500: InternalServerError

// Handler logouts User from current session.
func Handler(u interfaces.AuthUseCases, cookiesConfig config.CookiesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refreshTokenCookie, err := r.Cookie(login.RefreshTokenCookieName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		if err = u.LogoutUser(r.Context(), refreshTokenCookie.Value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Deleting cookies:
		cookiesConfig.AccessToken.MaxAge = -1
		cookiesConfig.RefreshToken.MaxAge = -1
		cookies.Set(w, login.AccessTokenCookieName, "", cookiesConfig.AccessToken)
		cookies.Set(w, login.RefreshTokenCookieName, "", cookiesConfig.RefreshToken)

		w.WriteHeader(http.StatusNoContent)
	}
}
```

Ключевые изменения:
- Убрано чтение `userID` из контекста (auth middleware)
- Читаем refresh token из cookie
- Передаём значение cookie в `LogoutUser`
- Убраны импорты `contextlib` и `authmiddleware`

- [ ] **Step 2: Commit**

```bash
git add internal/controllers/http/handlers/api/auth/logout/handler.go
git commit -m "feat: update logout handler to use refresh token instead of userID"
```

---

### Task 13: Создать logout_all handler

**Files:**
- Create: `internal/controllers/http/handlers/api/auth/logout_all/handler.go`

- [ ] **Step 1: Создать handler**

Создать файл `internal/controllers/http/handlers/api/auth/logout_all/handler.go`:

```go
package logout_all

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	"github.com/DKhorkov/libs/cookies"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

// swagger:route DELETE /api/sessions/all sessions LogoutAll
//
// LogoutAll
//
// Logout User from all sessions and deletes access and refresh tokens.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	401: Unauthorized
//	500: InternalServerError

// Handler logouts User from all sessions.
func Handler(u interfaces.AuthUseCases, cookiesConfig config.CookiesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		if err = u.LogoutUserFromAllSessions(r.Context(), userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Deleting cookies:
		cookiesConfig.AccessToken.MaxAge = -1
		cookiesConfig.RefreshToken.MaxAge = -1
		cookies.Set(w, login.AccessTokenCookieName, "", cookiesConfig.AccessToken)
		cookies.Set(w, login.RefreshTokenCookieName, "", cookiesConfig.RefreshToken)

		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/controllers/http/handlers/api/auth/logout_all/handler.go
git commit -m "feat: add logout_all handler for logging out from all sessions"
```

---

### Task 14: Зарегистрировать новый роут в setup

**Files:**
- Modify: `internal/controllers/http/handlers/api/setup.go`

- [ ] **Step 1: Добавить импорт и URL**

В файле `internal/controllers/http/handlers/api/setup.go`:

1. Добавить импорт:
```go
"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/logout_all"
```

2. Добавить константу URL после `SessionsURL`:
```go
AllSessionsURL = SessionsURL + "/all"
```

3. В блоке `deleteMux` добавить новый handler после `SessionsURL`:
```go
deleteMux.Handle(AllSessionsURL, logout_all.Handler(authUseCases, cookiesConfig))
```

**Важно:** `AllSessionsURL` должен быть зарегистрирован **перед** `SessionsURL` в deleteMux, иначе gorilla/mux может матчить `/sessions/all` как `/sessions` с суффиксом. Порядок:

```go
deleteMux := apiMux.Methods(http.MethodDelete).Subrouter()
deleteMux.Handle(AllSessionsURL, logout_all.Handler(authUseCases, cookiesConfig))
deleteMux.Handle(SessionsURL, logout.Handler(authUseCases, cookiesConfig))
deleteMux.Handle(
    fmt.Sprintf(WebPushUnsubscribeURL, common.IDRouteKey),
    unsubscribe.Handler(webPushSubscriptionsUseCases),
)
```

- [ ] **Step 2: Commit**

```bash
git add internal/controllers/http/handlers/api/setup.go
git commit -m "feat: register logout_all route in API setup"
```

---

### Task 15: Проверить компиляцию

- [ ] **Step 1: Собрать проект**

Run: `go build ./...`
Expected: SUCCESS — проект компилируется без ошибок.

- [ ] **Step 2: Если есть ошибки — исправить**

Пройтись по каждой ошибке и исправить. Чаще всего это забытые места, которые использовали старые методы.

- [ ] **Step 3: Commit если были исправления**

```bash
git add -A
git commit -m "fix: resolve compilation errors after multi-session refactor"
```

---

### Task 16: Обновить тесты репозитория (trace decorator)

**Files:**
- Modify: `internal/repositories/auth/trace_decorator_test.go`

- [ ] **Step 1: Заменить `TestTraceDecorator_GetRefreshTokenByUserID` на `TestTraceDecorator_GetRefreshTokenByValue`**

В файле `internal/repositories/auth/trace_decorator_test.go` заменить тест `TestTraceDecorator_GetRefreshTokenByUserID` (строки 364-517):

```go
func TestTraceDecorator_GetRefreshTokenByValue(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		value         string
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedToken *domains.RefreshToken
		expectedError error
	}{
		{
			name:  "successful get refresh token by value with tracing",
			value: "refresh_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetRefreshTokenByValue(gomock.Any(), "refresh_token").
					Return(&domains.RefreshToken{
						ID:        1,
						UserID:    1,
						Value:     "refresh_token",
						TTL:       now.Add(24 * time.Hour),
						CreatedAt: now,
					}, nil)
			},
			expectedToken: &domains.RefreshToken{
				ID:        1,
				UserID:    1,
				Value:     "refresh_token",
				TTL:       now.Add(24 * time.Hour),
				CreatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:  "refresh token not found",
			value: "nonexistent_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetRefreshTokenByValue(gomock.Any(), "nonexistent_token").
					Return(nil, errors.New("refresh token not found"))
			},
			expectedToken: nil,
			expectedError: errors.New("refresh token not found"),
		},
		{
			name:  "database error",
			value: "some_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetRefreshTokenByValue(gomock.Any(), "some_token").
					Return(nil, errors.New("database connection failed"))
			},
			expectedToken: nil,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			token, err := decorator.GetRefreshTokenByValue(ctx, tt.value)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedToken, token)
		})
	}
}
```

- [ ] **Step 2: Добавить `TestTraceDecorator_ExpireAllUserRefreshTokens`**

```go
func TestTraceDecorator_ExpireAllUserRefreshTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful expire all user refresh tokens with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ExpireAllUserRefreshTokens(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ExpireAllUserRefreshTokens(gomock.Any(), uint64(1)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.ExpireAllUserRefreshTokens(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 3: Запустить тесты**

Run: `go test ./internal/repositories/auth/... -run TestTraceDecorator -v -count=1`
Expected: все тесты PASS

- [ ] **Step 4: Commit**

```bash
git add internal/repositories/auth/trace_decorator_test.go
git commit -m "test: update repository trace decorator tests for multi-session"
```

---

### Task 17: Обновить тесты сервиса (trace decorator)

**Files:**
- Modify: `internal/services/auth/trace_decorator_test.go`

- [ ] **Step 1: Заменить `TestTraceDecorator_GetRefreshTokenByUserID` на `TestTraceDecorator_GetRefreshTokenByValue`**

По аналогии с Task 16, но используя `mockservices.MockAuthService` вместо `mockrepositories.MockAuthRepository`. Заменить тест `TestTraceDecorator_GetRefreshTokenByUserID` (строки 347-481):

```go
func TestTraceDecorator_GetRefreshTokenByValue(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		value         string
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedToken *domains.RefreshToken
		expectedError error
	}{
		{
			name:  "successful get refresh token by value with tracing",
			value: "refresh_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetRefreshTokenByValue(gomock.Any(), "refresh_token").
					Return(&domains.RefreshToken{
						ID:        1,
						UserID:    1,
						Value:     "refresh_token",
						TTL:       now.Add(24 * time.Hour),
						CreatedAt: now,
					}, nil)
			},
			expectedToken: &domains.RefreshToken{
				ID:        1,
				UserID:    1,
				Value:     "refresh_token",
				TTL:       now.Add(24 * time.Hour),
				CreatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:  "refresh token not found",
			value: "nonexistent_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetRefreshTokenByValue(gomock.Any(), "nonexistent_token").
					Return(nil, errors.New("refresh token not found"))
			},
			expectedToken: nil,
			expectedError: errors.New("refresh token not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			token, err := decorator.GetRefreshTokenByValue(ctx, tt.value)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedToken, token)
		})
	}
}
```

- [ ] **Step 2: Добавить `TestTraceDecorator_ExpireAllUserRefreshTokens`**

По аналогии с Task 16 Step 2, но с `mockservices.MockAuthService`.

```go
func TestTraceDecorator_ExpireAllUserRefreshTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful expire all user refresh tokens with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ExpireAllUserRefreshTokens(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ExpireAllUserRefreshTokens(gomock.Any(), uint64(1)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.ExpireAllUserRefreshTokens(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 3: Запустить тесты**

Run: `go test ./internal/services/auth/... -run TestTraceDecorator -v -count=1`
Expected: все тесты PASS

- [ ] **Step 4: Commit**

```bash
git add internal/services/auth/trace_decorator_test.go
git commit -m "test: update service trace decorator tests for multi-session"
```

---

### Task 18: Обновить тесты usecases (trace decorator)

**Files:**
- Modify: `internal/usecases/auth/trace_decorator_test.go`

- [ ] **Step 1: Обновить `TestTraceDecorator_LogoutUser` — принимать string**

В файле `internal/usecases/auth/trace_decorator_test.go` заменить тест `TestTraceDecorator_LogoutUser` (строки 458-560):

```go
func TestTraceDecorator_LogoutUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		refreshToken  string
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:         "successful logout with tracing",
			refreshToken: "valid_refresh_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					LogoutUser(gomock.Any(), "valid_refresh_token").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:         "logout error",
			refreshToken: "some_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					LogoutUser(gomock.Any(), "some_token").
					Return(errors.New("logout failed"))
			},
			expectedError: errors.New("logout failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.LogoutUser(ctx, tt.refreshToken)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 2: Добавить `TestTraceDecorator_LogoutUserFromAllSessions`**

```go
func TestTraceDecorator_LogoutUserFromAllSessions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful logout from all sessions with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					LogoutUserFromAllSessions(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					LogoutUserFromAllSessions(gomock.Any(), uint64(1)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.LogoutUserFromAllSessions(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 3: Запустить тесты**

Run: `go test ./internal/usecases/auth/... -run TestTraceDecorator -v -count=1`
Expected: все тесты PASS

- [ ] **Step 4: Commit**

```bash
git add internal/usecases/auth/trace_decorator_test.go
git commit -m "test: update usecases trace decorator tests for multi-session"
```

---

### Task 19: Обновить тесты usecases (основные + cache decorator)

**Files:**
- Modify: `internal/usecases/auth/usecases_test.go` — обновить тесты для `LoginUser`, `RefreshTokens`, `LogoutUser`, добавить `LogoutUserFromAllSessions`
- Modify: `internal/usecases/auth/cache_decorator_test.go` — обновить тест `LogoutUser`, добавить `LogoutUserFromAllSessions`

- [ ] **Step 1: Обновить тесты `LoginUser` в `usecases_test.go`**

Найти тест `LoginUser` — убрать mock-вызовы `GetRefreshTokenByUserID` и `ExpireRefreshToken` из успешного сценария. Теперь при логине старый токен НЕ удаляется.

- [ ] **Step 2: Обновить тесты `RefreshTokens` в `usecases_test.go`**

Обновить мок-вызовы: заменить `GetRefreshTokenByUserID` на `GetRefreshTokenByValue`, убрать парсинг access token из refresh token. Мок должен вызывать `GetRefreshTokenByValue(ctx, oldRefreshTokenJWT)`.

- [ ] **Step 3: Обновить тесты `LogoutUser` в `usecases_test.go`**

Тест должен передавать encoded refresh token в `LogoutUser(ctx, encodedRefreshToken)` вместо `LogoutUser(ctx, userID)`. Мок вызовы: `GetRefreshTokenByValue` → `ExpireRefreshToken`.

- [ ] **Step 4: Добавить тест `LogoutUserFromAllSessions` в `usecases_test.go`**

Тест вызывает `LogoutUserFromAllSessions(ctx, userID)`, мок ожидает `ExpireAllUserRefreshTokens(ctx, userID)`.

- [ ] **Step 5: Обновить `LogoutUser` в `cache_decorator_test.go`**

Заменить вызовы с `uint64` на `string` (refresh token).

- [ ] **Step 6: Добавить `LogoutUserFromAllSessions` тест в `cache_decorator_test.go`**

Простой passthrough-тест, аналогично существующему `LogoutUser`.

- [ ] **Step 7: Запустить все тесты usecases**

Run: `go test ./internal/usecases/auth/... -v -count=1`
Expected: все тесты PASS

- [ ] **Step 8: Commit**

```bash
git add internal/usecases/auth/usecases_test.go internal/usecases/auth/cache_decorator_test.go
git commit -m "test: update usecases and cache decorator tests for multi-session"
```

---

### Task 20: Обновить тесты сервиса

**Files:**
- Modify: `internal/services/auth/service_test.go`

- [ ] **Step 1: Обновить тесты, использующие `GetRefreshTokenByUserID`**

Заменить мок-вызовы `GetRefreshTokenByUserID` на `GetRefreshTokenByValue` в тестах `CreateRefreshToken`. Заменить тесты `ForgetPassword` — вместо `GetRefreshTokenByUserID` + `ExpireRefreshToken` должен быть один вызов `ExpireAllUserRefreshTokens`.

- [ ] **Step 2: Добавить тест для `GetRefreshTokenByValue`**

- [ ] **Step 3: Добавить тест для `ExpireAllUserRefreshTokens`**

- [ ] **Step 4: Запустить тесты**

Run: `go test ./internal/services/auth/... -v -count=1`
Expected: все тесты PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/auth/service_test.go
git commit -m "test: update service tests for multi-session"
```

---

### Task 21: Обновить тесты handler-ов

**Files:**
- Modify: `internal/controllers/http/handlers/api/auth/logout/handler_test.go`
- Create: `internal/controllers/http/handlers/api/auth/logout_all/handler_test.go`

- [ ] **Step 1: Переписать logout handler_test.go**

Тесты должны:
- Создавать запросы с refresh token cookie вместо userID в контексте
- Мок вызывать `LogoutUser(ctx, refreshTokenCookieValue)` вместо `LogoutUser(ctx, userID)`
- Проверять 401 когда cookie отсутствует

- [ ] **Step 2: Создать logout_all handler_test.go**

Тесты аналогичны текущему logout handler (используют userID из контекста), но вызывают `LogoutUserFromAllSessions`. Кейсы:
- successful logout all
- unauthorized (no userID in context)
- internal server error

- [ ] **Step 3: Запустить тесты**

Run: `go test ./internal/controllers/http/handlers/api/auth/... -v -count=1`
Expected: все тесты PASS

- [ ] **Step 4: Commit**

```bash
git add internal/controllers/http/handlers/api/auth/logout/handler_test.go
git add internal/controllers/http/handlers/api/auth/logout_all/handler_test.go
git commit -m "test: update logout handler tests and add logout_all handler tests"
```

---

### Task 22: Запустить все тесты и lint

- [ ] **Step 1: Запустить все unit-тесты**

Run: `go test ./... -count=1`
Expected: все тесты PASS

- [ ] **Step 2: Запустить линтер**

Run: `task lint`
Expected: нет ошибок (или минимальные предупреждения, не связанные с нашими изменениями)

- [ ] **Step 3: Исправить ошибки если есть**

- [ ] **Step 4: Commit если были исправления**

```bash
git add -A
git commit -m "fix: resolve lint issues after multi-session refactor"
```

---

### Task 23: Обновить doc.md файлы

**Files:**
- Modify: `internal/repositories/auth/doc.md`
- Modify: `internal/services/auth/doc.md`
- Modify: `internal/usecases/auth/doc.md`
- Modify: `internal/controllers/http/handlers/api/setup.go` (doc comment if present)
- Modify: `internal/controllers/http/schemas/doc.md`

- [ ] **Step 1: Обновить doc.md в затронутых директориях**

В каждом doc.md обновить описания методов:
- `GetRefreshTokenByUserID` → `GetRefreshTokenByValue`
- Добавить описание `ExpireAllUserRefreshTokens`
- Добавить описание `LogoutUserFromAllSessions`
- Упомянуть мультисессионность

- [ ] **Step 2: Commit**

```bash
git add internal/repositories/auth/doc.md internal/services/auth/doc.md internal/usecases/auth/doc.md internal/controllers/http/schemas/doc.md
git commit -m "docs: update doc.md files for multi-session support"
```

---

### Task 24: Обновить Swagger

**Files:**
- Modify: `api/swagger.yaml`

- [ ] **Step 1: Добавить endpoint `DELETE /api/sessions/all`**

Добавить описание нового endpoint в swagger spec. Скопировать структуру из существующего `DELETE /api/sessions`, обновить описание.

- [ ] **Step 2: Обновить описание `DELETE /api/sessions`**

Обновить описание: теперь это logout из текущей сессии, не из всех.

- [ ] **Step 3: Commit**

```bash
git add api/swagger.yaml
git commit -m "docs: update swagger spec with logout_all endpoint"
```
