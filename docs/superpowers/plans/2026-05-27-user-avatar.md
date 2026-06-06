# User Avatar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add user avatar upload/update/delete with server-side image processing (resize to 256x256 JPEG) and a universal file download endpoint.

**Architecture:** New FileStorage layer (repository for disk I/O, service wrapping repository + user update). UsersUseCases orchestrates avatar operations (validation, image processing, calling FileStorageService). Universal `GET /api/files/download/{uuid}.jpg` endpoint for serving files. Frontend renders avatar via `<img src>` with initials fallback on error.

**Tech Stack:** Go stdlib `image`, `image/jpeg`, `image/png`, `golang.org/x/image/webp`, `image/gif` for decoding; `golang.org/x/image/draw` for resize; `github.com/google/uuid` for file naming.

**Spec:** `docs/superpowers/specs/2026-05-27-user-avatar-design.md`

---

### Task 1: Add dependencies

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add required Go modules**

```bash
cd /Users/dskhorkov/GolandProjects/KhodFeltsChat && go get golang.org/x/image@latest && go get github.com/google/uuid@latest
```

- [ ] **Step 2: Verify imports resolve**

```bash
go mod tidy
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add golang.org/x/image and google/uuid for avatar feature"
```

---

### Task 2: Database migration — add avatar_path to users

**Files:**
- Create: `migrations/20260527000000_user_avatar_path.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN avatar_path TEXT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS avatar_path;
-- +goose StatementEnd
```

- [ ] **Step 2: Apply migration locally**

```bash
task migrate-up
```

- [ ] **Step 3: Commit**

```bash
git add migrations/20260527000000_user_avatar_path.sql
git commit -m "migration: add avatar_path column to users table"
```

---

### Task 3: Update domain User — add AvatarPath field

**Files:**
- Modify: `internal/domains/user.go`
- Modify: `internal/domains/user_test.go` (if exists and tests struct)

- [ ] **Step 1: Add AvatarPath to User struct**

In `internal/domains/user.go`, add `AvatarPath` field to `User`:

```go
type User struct {
	ID             uint64    `json:"id"             db:"id"`
	Username       string    `json:"username"       db:"username"`
	Email          string    `json:"email"          db:"email"`
	EmailConfirmed bool      `json:"emailConfirmed" db:"email_confirmed"`
	Password       string    `json:"password"       db:"password"`
	CreatedAt      time.Time `json:"createdAt"      db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt"      db:"updated_at"`
	AvatarPath     *string   `json:"avatarPath"     db:"avatar_path"`
}
```

- [ ] **Step 2: Run existing tests to verify nothing breaks**

```bash
go test ./internal/domains/...
```

Expected: PASS (the new nullable field defaults to nil, existing tests should still work).

- [ ] **Step 3: Commit**

```bash
git add internal/domains/user.go
git commit -m "domain: add AvatarPath field to User"
```

---

### Task 4: Update User schema, mapper — add AvatarPath

**Files:**
- Modify: `internal/controllers/http/schemas/users.go`
- Modify: `internal/controllers/http/mappers/users/users.go`
- Modify: `internal/controllers/http/mappers/users/users_test.go`

- [ ] **Step 1: Add AvatarPath to User schema**

In `internal/controllers/http/schemas/users.go`, add to `User` struct:

```go
// Avatar URL of the user.
// required: false
// nullable: true
// example: https://kfc.webtm.ru/api/files/download/550e8400-e29b-41d4-a716-446655440000.jpg
AvatarPath *string `json:"avatarPath"`
```

- [ ] **Step 2: Update MapUser mapper**

In `internal/controllers/http/mappers/users/users.go`, add `AvatarPath` to `MapUser`:

```go
func MapUser(user domains.User) schemas.User {
	return schemas.User{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		EmailConfirmed: user.EmailConfirmed,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
		AvatarPath:     user.AvatarPath,
	}
}
```

- [ ] **Step 3: Update mapper tests**

In `internal/controllers/http/mappers/users/users_test.go`, add `AvatarPath` to test data where `domains.User` and `schemas.User` are constructed. Verify both nil and non-nil cases are covered.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/controllers/http/mappers/users/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/controllers/http/schemas/users.go internal/controllers/http/mappers/users/
git commit -m "schema: add AvatarPath to User schema and mapper"
```

---

### Task 5: Change UsersRepository.UpdateUser to accept domains.User

This is a breaking change that ripples through service, usecases, trace decorators, and tests.

**Files:**
- Modify: `internal/interfaces/repositories.go`
- Modify: `internal/interfaces/services.go`
- Modify: `internal/interfaces/usecases.go`
- Modify: `internal/repositories/users/repository.go`
- Modify: `internal/services/users/service.go`
- Modify: `internal/services/users/trace_decorator.go`
- Modify: `internal/usecases/users/usecases.go`
- Modify: `internal/usecases/users/trace_decorator.go`
- Modify: `internal/controllers/http/handlers/api/users/update/handler.go`

- [ ] **Step 1: Update UsersRepository interface**

In `internal/interfaces/repositories.go`, change:

```go
UpdateUser(ctx context.Context, user domains.User) error
```

- [ ] **Step 2: Update UsersService interface**

In `internal/interfaces/services.go`, change `UsersService.UpdateUser`:

```go
UpdateUser(ctx context.Context, user domains.User) (*domains.User, error)
```

- [ ] **Step 3: Update UsersUseCases interface**

In `internal/interfaces/usecases.go`, change `UsersUseCases.UpdateUser`:

```go
UpdateUser(ctx context.Context, userData domains.UpdateUserDTO) (*domains.User, error)
```

(This stays as UpdateUserDTO — the usecases layer still receives the DTO and maps it internally.)

- [ ] **Step 4: Update repository implementation**

In `internal/repositories/users/repository.go`, change `UpdateUser` to accept `domains.User`:

```go
func (repo *Repository) UpdateUser(
	ctx context.Context,
	user domains.User,
) error {
	builder := sq.
		Update(usersTableName).
		Where(sq.Eq{idColumnName: user.ID}).
		Set(usernameColumnName, user.Username).
		Set(emailColumnName, user.Email).
		Set(updatedAtColumnName, time.Now()).
		PlaceholderFormat(sq.Dollar)

	if user.AvatarPath != nil {
		builder = builder.Set(avatarPathColumnName, *user.AvatarPath)
	} else {
		builder = builder.Set(avatarPathColumnName, nil)
	}

	stmt, params, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
```

Add `avatarPathColumnName = "avatar_path"` to the constants block.

- [ ] **Step 5: Update service implementation**

In `internal/services/users/service.go`, change `UpdateUser` to fetch current user, patch fields from DTO, then pass full `domains.User`:

```go
func (s *Service) UpdateUser(
	ctx context.Context,
	userData domains.UpdateUserDTO,
) (*domains.User, error) {
	var (
		user *domains.User
		err  error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)

			user, err = usersRepository.GetUserByID(ctx, userData.ID)
			if err != nil {
				return err
			}

			if userData.Username != nil {
				user.Username = *userData.Username
			}

			err = usersRepository.UpdateUser(ctx, *user)
			if err != nil {
				return err
			}

			if user, err = usersRepository.GetUserByID(ctx, userData.ID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}
```

- [ ] **Step 6: Update service trace decorator**

In `internal/services/users/trace_decorator.go`, change `UpdateUser` signature:

```go
func (d *TraceDecorator) UpdateUser(
	ctx context.Context,
	userData domains.UpdateUserDTO,
) (*domains.User, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateUser(ctx, userData)
}
```

(Service interface still takes `UpdateUserDTO`, trace decorator wraps service.)

- [ ] **Step 7: Verify usecases don't need changes**

`internal/usecases/users/usecases.go` and its trace decorator already call `u.usersService.UpdateUser(ctx, UpdateUserDTO)` — the service handles the mapping now. No changes needed unless current code passes DTO differently. Verify.

- [ ] **Step 8: Regenerate mocks**

```bash
go generate ./internal/interfaces/...
```

- [ ] **Step 9: Update existing tests**

Fix all tests that construct `UpdateUser` calls with `UpdateUserDTO` at the repository level. Key files:
- `internal/controllers/http/handlers/api/users/update/handler_test.go`
- Any service-level tests
- Any repository-level tests

Run:

```bash
go test ./...
```

Fix until all pass.

- [ ] **Step 10: Commit**

```bash
git add internal/interfaces/ internal/repositories/users/ internal/services/users/ internal/usecases/users/ mocks/ internal/controllers/http/handlers/api/users/update/
git commit -m "refactor: UsersRepository.UpdateUser accepts domains.User instead of UpdateUserDTO"
```

---

### Task 6: Add FileStorage errors

**Files:**
- Create: `internal/errors/file_storage.go`

- [ ] **Step 1: Create error sentinels**

```go
package errors

import "errors"

var (
	ErrFileNotFound       = errors.New("file not found")
	ErrInvalidImageFormat = errors.New("invalid image format: supported formats are JPEG, PNG, WebP, GIF")
	ErrFileTooLarge       = errors.New("file too large")
)
```

- [ ] **Step 2: Commit**

```bash
git add internal/errors/file_storage.go
git commit -m "errors: add file storage error sentinels"
```

---

### Task 7: Add FileStorageConfig

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add FileStorageConfig struct**

Add after the existing config types:

```go
type FileStorageConfig struct {
	BasePath        string
	BaseUploadURL   string
	BaseDownloadURL string
	MaxSize         int64
}
```

- [ ] **Step 2: Add FileStorage field to Config**

```go
type Config struct {
	...existing fields...
	FileStorage FileStorageConfig
}
```

- [ ] **Step 3: Add FileStorage config initialization in New()**

In the `New()` function, add:

```go
FileStorage: FileStorageConfig{
	BasePath:        loadenv.GetEnv("FILE_STORAGE_BASE_PATH", "uploads"),
	BaseUploadURL:   loadenv.GetEnv("FILE_STORAGE_BASE_UPLOAD_URL", "http://localhost:8080/api/files/upload"),
	BaseDownloadURL: loadenv.GetEnv("FILE_STORAGE_BASE_DOWNLOAD_URL", "http://localhost:8080/api/files/download"),
	MaxSize:         int64(loadenv.GetEnvAsInt("FILE_STORAGE_MAX_SIZE", 20*1024*1024)),
},
```

- [ ] **Step 4: Add FileStorage tracing spans**

Add `FileStorage tracing.SpanConfig` to `SpanRepositories`, `SpanServices`.

- [ ] **Step 5: Initialize the span configs in New()**

Add the corresponding span config entries following the existing pattern in `New()`.

- [ ] **Step 6: Run build**

```bash
go build ./...
```

Expected: compiles.

- [ ] **Step 7: Commit**

```bash
git add internal/config/config.go
git commit -m "config: add FileStorageConfig with env vars"
```

---

### Task 8: Add FileStorage interfaces

**Files:**
- Modify: `internal/interfaces/repositories.go`
- Modify: `internal/interfaces/services.go`
- Modify: `internal/interfaces/usecases.go`

- [ ] **Step 1: Add FileStorageRepository interface**

In `internal/interfaces/repositories.go`, add:

```go
//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/file_storage_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository
type FileStorageRepository interface {
	Upload(ctx context.Context, path string, data io.Reader) error
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
}
```

Add `"io"` to the imports.

- [ ] **Step 2: Add FileStorageService interface**

In `internal/interfaces/services.go`, add:

```go
//go:generate mockgen -source=services.go -destination=../../mocks/services/file_storage_service.go -package=mockservices -exclude_interfaces=UsersService,AuthService,ChatsService,MessagesService,NotificationsService,SettingsService,WebPushSubscriptionsService
type FileStorageService interface {
	Upload(ctx context.Context, path string, data io.Reader) error
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
}
```

Add `"io"` to the imports.

- [ ] **Step 3: Add avatar methods to UsersUseCases**

In `internal/interfaces/usecases.go`, add to `UsersUseCases`:

```go
UpdateAvatar(ctx context.Context, userID uint64, data io.Reader) (string, error)
DeleteAvatar(ctx context.Context, userID uint64) error
```

Add `"io"` to the imports.

- [ ] **Step 4: Regenerate mocks**

```bash
go generate ./internal/interfaces/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/interfaces/ mocks/
git commit -m "interfaces: add FileStorageRepository, FileStorageService, avatar methods to UsersUseCases"
```

---

### Task 9: Implement FileStorageRepository (disk storage)

**Files:**
- Create: `internal/repositories/file_storage/repository.go`
- Create: `internal/repositories/file_storage/repository_test.go`
- Create: `internal/repositories/file_storage/trace_decorator.go`
- Create: `internal/repositories/file_storage/trace_decorator_test.go`
- Create: `internal/repositories/file_storage/doc.md`

- [ ] **Step 1: Write failing tests for Upload/Download/Delete**

In `internal/repositories/file_storage/repository_test.go`:

```go
package file_storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	filestorage "github.com/DKhorkov/kfc/internal/repositories/file_storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Upload(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	repo := filestorage.New(basePath)

	data := []byte("test file content")
	err := repo.Upload(context.Background(), "test.jpg", bytes.NewReader(data))
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(basePath, "test.jpg"))
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestRepository_Download(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	repo := filestorage.New(basePath)

	expected := []byte("test file content")
	err := os.WriteFile(filepath.Join(basePath, "test.jpg"), expected, 0644)
	require.NoError(t, err)

	result, err := repo.Download(context.Background(), "test.jpg")
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestRepository_Download_NotFound(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	repo := filestorage.New(basePath)

	_, err := repo.Download(context.Background(), "nonexistent.jpg")
	assert.Error(t, err)
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	repo := filestorage.New(basePath)

	filePath := filepath.Join(basePath, "test.jpg")
	err := os.WriteFile(filePath, []byte("content"), 0644)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), "test.jpg")
	require.NoError(t, err)

	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	basePath := t.TempDir()
	repo := filestorage.New(basePath)

	err := repo.Delete(context.Background(), "nonexistent.jpg")
	assert.NoError(t, err, "deleting non-existent file should not error")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/repositories/file_storage/...
```

Expected: FAIL (package does not exist yet)

- [ ] **Step 3: Implement repository**

In `internal/repositories/file_storage/repository.go`:

```go
package file_storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
)

type Repository struct {
	basePath string
}

func New(basePath string) *Repository {
	return &Repository{basePath: basePath}
}

func (r *Repository) Upload(_ context.Context, path string, data io.Reader) error {
	fullPath := filepath.Join(r.basePath, path)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, data)

	return err
}

func (r *Repository) Download(_ context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(r.basePath, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", customerrors.ErrFileNotFound, path)
		}

		return nil, err
	}

	return data, nil
}

func (r *Repository) Delete(_ context.Context, path string) error {
	fullPath := filepath.Join(r.basePath, path)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/repositories/file_storage/...
```

Expected: PASS

- [ ] **Step 5: Add trace decorator**

In `internal/repositories/file_storage/trace_decorator.go`, follow the pattern from `internal/services/users/trace_decorator.go`:

```go
package file_storage

import (
	"context"
	"io"

	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.FileStorageRepository
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.FileStorageRepository,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) Upload(ctx context.Context, path string, data io.Reader) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.Upload(ctx, path, data)
}

func (d *TraceDecorator) Download(ctx context.Context, path string) ([]byte, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.Download(ctx, path)
}

func (d *TraceDecorator) Delete(ctx context.Context, path string) (error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.Delete(ctx, path)
}
```

- [ ] **Step 6: Write trace decorator tests**

Follow the pattern from existing trace decorator tests in the project.

- [ ] **Step 7: Write doc.md**

Create `internal/repositories/file_storage/doc.md` documenting the package.

- [ ] **Step 8: Run all tests**

```bash
go test ./internal/repositories/file_storage/...
```

Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/repositories/file_storage/ internal/errors/file_storage.go
git commit -m "feat: implement FileStorageRepository with disk storage"
```

---

### Task 10: Implement FileStorageService

**Files:**
- Create: `internal/services/file_storage/service.go`
- Create: `internal/services/file_storage/service_test.go`
- Create: `internal/services/file_storage/trace_decorator.go`
- Create: `internal/services/file_storage/trace_decorator_test.go`
- Create: `internal/services/file_storage/doc.md`

- [ ] **Step 1: Write failing tests**

In `internal/services/file_storage/service_test.go`, test Upload/Download/Delete using mockgen mocks for `FileStorageRepository` and `UsersRepository`. Verify that after Upload, the user's `AvatarPath` is updated via `UsersRepository.UpdateUser`. After Delete, verify `AvatarPath` is set to nil.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/services/file_storage/...
```

Expected: FAIL

- [ ] **Step 3: Implement service**

In `internal/services/file_storage/service.go`:

```go
package file_storage

import (
	"context"
	"io"

	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                           interfaces.UnitOfWork
	newFileStorageRepositoryFunc   func() interfaces.FileStorageRepository
	newUsersRepositoryFunc         func(tx pg.Transaction) interfaces.UsersRepository
}

func New(
	uow interfaces.UnitOfWork,
	newFileStorageRepositoryFunc func() interfaces.FileStorageRepository,
	newUsersRepositoryFunc func(tx pg.Transaction) interfaces.UsersRepository,
) *Service {
	return &Service{
		uow:                         uow,
		newFileStorageRepositoryFunc: newFileStorageRepositoryFunc,
		newUsersRepositoryFunc:       newUsersRepositoryFunc,
	}
}

func (s *Service) Upload(ctx context.Context, path string, data io.Reader) error {
	fileStorageRepository := s.newFileStorageRepositoryFunc()

	return fileStorageRepository.Upload(ctx, path, data)
}

func (s *Service) Download(ctx context.Context, path string) ([]byte, error) {
	fileStorageRepository := s.newFileStorageRepositoryFunc()

	return fileStorageRepository.Download(ctx, path)
}

func (s *Service) Delete(ctx context.Context, path string) error {
	fileStorageRepository := s.newFileStorageRepositoryFunc()

	return fileStorageRepository.Delete(ctx, path)
}
```

Note: Per the spec, the service also handles user avatar_path updates in DB. The `Upload` and `Delete` methods shown above are for generic file operations. Add avatar-specific methods or have the usecases layer call both `fileStorageService.Upload` and then update the user via `usersService.UpdateUser`. The exact split depends on implementation — the key contract is:
- After avatar upload: file saved to disk + `avatar_path` updated in DB
- After avatar delete: file removed from disk + `avatar_path` set to nil in DB

- [ ] **Step 4: Run tests**

```bash
go test ./internal/services/file_storage/...
```

Expected: PASS

- [ ] **Step 5: Add trace decorator**

Follow the same pattern as `internal/repositories/file_storage/trace_decorator.go` but wrapping `interfaces.FileStorageService`.

- [ ] **Step 6: Write trace decorator tests and doc.md**

- [ ] **Step 7: Run all tests**

```bash
go test ./internal/services/file_storage/...
```

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/services/file_storage/
git commit -m "feat: implement FileStorageService"
```

---

### Task 11: Implement UsersUseCases.UpdateAvatar and DeleteAvatar

**Files:**
- Modify: `internal/usecases/users/usecases.go`
- Modify: `internal/usecases/users/trace_decorator.go`
- Create or modify: `internal/usecases/users/usecases_test.go`

- [ ] **Step 1: Write failing tests for UpdateAvatar**

Test cases:
1. Successful upload — valid JPEG data → returns URL string
2. Invalid image format → returns `ErrInvalidImageFormat`
3. File too large → returns `ErrFileTooLarge`
4. User already has avatar → old file deleted before new upload
5. FileStorageService.Upload error → propagated

- [ ] **Step 2: Write failing tests for DeleteAvatar**

Test cases:
1. Successful delete — user has avatar → file deleted, `AvatarPath` nullified
2. User has no avatar (`AvatarPath` is nil) → no-op, no error
3. User not found → returns error

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/usecases/users/...
```

Expected: FAIL

- [ ] **Step 4: Add dependencies to UseCases struct**

In `internal/usecases/users/usecases.go`, add `fileStorageService` and `fileStorageConfig` to the struct:

```go
type UseCases struct {
	usersService      interfaces.UsersService
	fileStorageService interfaces.FileStorageService
	securityConfig    security.Config
	validationConfig  config.ValidationConfig
	fileStorageConfig config.FileStorageConfig
}
```

Update `New()` to accept and store these new dependencies.

- [ ] **Step 5: Implement UpdateAvatar**

```go
func (u *UseCases) UpdateAvatar(
	ctx context.Context,
	userID uint64,
	data io.Reader,
) (string, error) {
	// Read all data to validate size and format
	rawData, err := io.ReadAll(io.LimitReader(data, u.fileStorageConfig.MaxSize+1))
	if err != nil {
		return "", err
	}

	if int64(len(rawData)) > u.fileStorageConfig.MaxSize {
		return "", customerrors.ErrFileTooLarge
	}

	// Decode image (validates format: JPEG, PNG, WebP, GIF)
	img, err := decodeImage(rawData)
	if err != nil {
		return "", fmt.Errorf("%w: %w", customerrors.ErrInvalidImageFormat, err)
	}

	// Resize to 256x256 and encode as JPEG
	resized := resizeImage(img, 256, 256)

	var buf bytes.Buffer
	if err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}

	// Delete old avatar if exists
	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}

	if user.AvatarPath != nil {
		oldUUID := extractUUIDFromURL(*user.AvatarPath)
		if oldUUID != "" {
			_ = u.fileStorageService.Delete(ctx, oldUUID+".jpg")
		}
	}

	// Upload new file
	fileUUID := uuid.New().String()
	fileName := fileUUID + ".jpg"

	if err = u.fileStorageService.Upload(ctx, fileName, bytes.NewReader(buf.Bytes())); err != nil {
		return "", err
	}

	// Update user avatar path
	avatarURL := u.fileStorageConfig.BaseDownloadURL + "/" + fileName
	user.AvatarPath = &avatarURL

	if _, err = u.usersService.UpdateUser(ctx, domains.UpdateUserDTO{
		ID: userID,
	}); err != nil {
		return "", err
	}

	return avatarURL, nil
}
```

Note: The actual implementation needs refinement — `UpdateUser` currently only handles `Username` in the DTO. Since we changed the repository to accept `domains.User`, we need the service to also pass `AvatarPath`. This may require adding `AvatarPath *string` to `UpdateUserDTO` or having a separate method. Adjust based on final service implementation.

- [ ] **Step 6: Implement helper functions**

Add `decodeImage`, `resizeImage`, and `extractUUIDFromURL` as private functions in a separate file `internal/usecases/users/image.go`:

```go
package users

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/url"
	"path"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

func decodeImage(data []byte) (image.Image, error) {
	reader := bytes.NewReader(data)

	// Try standard formats first
	img, _, err := image.Decode(reader)
	if err == nil {
		return img, nil
	}

	// Try WebP explicitly
	reader.Reset(data)

	img, err = webp.Decode(reader)
	if err == nil {
		return img, nil
	}

	return nil, err
}

func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	return dst
}

func extractUUIDFromURL(avatarURL string) string {
	u, err := url.Parse(avatarURL)
	if err != nil {
		return ""
	}

	base := path.Base(u.Path)

	return strings.TrimSuffix(base, ".jpg")
}
```

Register standard decoders in an `init()` or import for side effects:

```go
import (
	_ "image/jpeg"
	_ "image/png"
	_ "image/gif"
)
```

- [ ] **Step 7: Implement DeleteAvatar**

```go
func (u *UseCases) DeleteAvatar(ctx context.Context, userID uint64) error {
	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.AvatarPath == nil {
		return nil
	}

	oldUUID := extractUUIDFromURL(*user.AvatarPath)
	if oldUUID != "" {
		if err = u.fileStorageService.Delete(ctx, oldUUID+".jpg"); err != nil {
			return err
		}
	}

	user.AvatarPath = nil

	_, err = u.usersService.UpdateUser(ctx, domains.UpdateUserDTO{
		ID: userID,
	})

	return err
}
```

- [ ] **Step 8: Update trace decorator**

In `internal/usecases/users/trace_decorator.go`, add `UpdateAvatar` and `DeleteAvatar` methods:

```go
func (d *TraceDecorator) UpdateAvatar(
	ctx context.Context,
	userID uint64,
	data io.Reader,
) (string, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateAvatar(ctx, userID, data)
}

func (d *TraceDecorator) DeleteAvatar(ctx context.Context, userID uint64) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeleteAvatar(ctx, userID)
}
```

- [ ] **Step 9: Run tests**

```bash
go test ./internal/usecases/users/...
```

Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/usecases/users/
git commit -m "feat: implement UpdateAvatar and DeleteAvatar in UsersUseCases"
```

---

### Task 12: Add HTTP handlers — upload avatar, delete avatar, download file

**Files:**
- Create: `internal/controllers/http/handlers/api/users/update_avatar/handler.go`
- Create: `internal/controllers/http/handlers/api/users/update_avatar/handler_test.go`
- Create: `internal/controllers/http/handlers/api/users/delete_avatar/handler.go`
- Create: `internal/controllers/http/handlers/api/users/delete_avatar/handler_test.go`
- Create: `internal/controllers/http/handlers/api/files/download/handler.go`
- Create: `internal/controllers/http/handlers/api/files/download/handler_test.go`

- [ ] **Step 1: Write failing tests for upload avatar handler**

Test cases: successful multipart upload (200 + URL in body), missing file field (400), unauthorized (401), usecase error (500).

- [ ] **Step 2: Implement upload avatar handler**

In `internal/controllers/http/handlers/api/users/update_avatar/handler.go`:

```go
package update_avatar

import (
	"errors"
	"net/http"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

const (
	formFileKey = "avatar"
	maxMemory   = 20 << 20 // 20 MB
)

func Handler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		if err = r.ParseMultipartForm(maxMemory); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		file, _, err := r.FormFile(formFileKey)
		if err != nil {
			http.Error(w, "avatar file is required", http.StatusBadRequest)

			return
		}
		defer file.Close()

		avatarURL, err := u.UpdateAvatar(r.Context(), userID, file)

		switch {
		case errors.Is(err, customerrors.ErrInvalidImageFormat):
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		case errors.Is(err, customerrors.ErrFileTooLarge):
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(avatarURL))
	}
}
```

- [ ] **Step 3: Run upload avatar tests**

```bash
go test ./internal/controllers/http/handlers/api/users/update_avatar/...
```

Expected: PASS

- [ ] **Step 4: Write failing tests for delete avatar handler**

Test cases: successful delete (204), unauthorized (401), user not found (404).

- [ ] **Step 5: Implement delete avatar handler**

```go
package delete_avatar

import (
	"errors"
	"net/http"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
)

func Handler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		err = u.DeleteAvatar(r.Context(), userID)

		switch {
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 6: Run delete avatar tests**

```bash
go test ./internal/controllers/http/handlers/api/users/delete_avatar/...
```

Expected: PASS

- [ ] **Step 7: Write failing tests for file download handler**

Test cases: successful download (200, `Content-Type: image/jpeg`), file not found (404).

- [ ] **Step 8: Implement file download handler**

```go
package download

import (
	"errors"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/gorilla/mux"
)

const FileRouteKey = "file"

func Handler(fileStorageService interfaces.FileStorageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileName := mux.Vars(r)[FileRouteKey]
		if fileName == "" {
			http.Error(w, "file name is required", http.StatusBadRequest)

			return
		}

		data, err := fileStorageService.Download(r.Context(), fileName)

		switch {
		case errors.Is(err, customerrors.ErrFileNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
```

- [ ] **Step 9: Run download tests**

```bash
go test ./internal/controllers/http/handlers/api/files/download/...
```

Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/controllers/http/handlers/api/users/update_avatar/ internal/controllers/http/handlers/api/users/delete_avatar/ internal/controllers/http/handlers/api/files/download/
git commit -m "feat: add HTTP handlers for avatar upload/delete and file download"
```

---

### Task 13: Register routes and wire DI

**Files:**
- Modify: `internal/controllers/http/handlers/api/setup.go`
- Modify: `internal/controllers/http/controller.go`
- Modify: `cmd/main.go`

- [ ] **Step 1: Add route constants in setup.go**

In `internal/controllers/http/handlers/api/setup.go`, add:

```go
AvatarURL       = MeURL + "/avatar"
FilesURL        = "/files"
FileDownloadURL = FilesURL + "/download/{%s}"
```

- [ ] **Step 2: Update SetupHandlers signature**

Add `fileStorageService interfaces.FileStorageService` parameter to `SetupHandlers`.

- [ ] **Step 3: Register routes**

In `SetupHandlers`:

```go
// Avatar
putMux.Handle(AvatarURL, update_avatar.Handler(usersUseCases))
deleteMux.Handle(AvatarURL, delete_avatar.Handler(usersUseCases))

// File download (public, registered on getMux)
getMux.Handle(
    fmt.Sprintf(FileDownloadURL, download.FileRouteKey),
    download.Handler(fileStorageService),
)
```

- [ ] **Step 4: Add file download URL to auth middleware IgnoreURL**

In `internal/controllers/http/controller.go`, add to the `IgnoreURL` list:

```go
{
    Path: regexp.MustCompile(
        `^` + handlers.APIPrefix + strings.ReplaceAll(
            api.FileDownloadURL,
            "{%s}",
            "",
        ) + `(.+)$`,
    ),
    Methods: []string{http.MethodGet},
},
```

- [ ] **Step 5: Wire FileStorageService in controller.go New()**

Add `fileStorageService interfaces.FileStorageService` parameter to `controllers.New()` and pass it to `SetupHandlers`.

- [ ] **Step 6: Wire everything in cmd/main.go**

Create `FileStorageRepository`, `FileStorageService`, pass `fileStorageConfig` to `UsersUseCases`, and pass `FileStorageService` to the controller:

```go
fileStorageRepository := filestoragerepository.NewTraceDecorator(
    traceProvider,
    cfg.Tracing.Spans.Repositories.FileStorage,
    filestoragerepository.New(cfg.FileStorage.BasePath),
)

fileStorageService := filestorageservice.NewTraceDecorator(
    traceProvider,
    cfg.Tracing.Spans.Services.FileStorage,
    filestorageservice.New(
        unitOfWork,
        func() interfaces.FileStorageRepository {
            return fileStorageRepository
        },
        func(tx postgresql.Transaction) interfaces.UsersRepository {
            return usersrepository.NewTraceDecorator(
                traceProvider,
                cfg.Tracing.Spans.Repositories.Users,
                usersrepository.New(tx, logger),
            )
        },
    ),
)

usersUseCases := usersusecases.NewTraceDecorator(
    traceProvider,
    cfg.Tracing.Spans.UseCases.Users,
    usersusecases.New(
        usersService,
        fileStorageService,
        cfg.Security,
        cfg.Validation,
        cfg.FileStorage,
    ),
)
```

- [ ] **Step 7: Ensure uploads directory is created**

Add to `cmd/main.go` before wiring:

```go
if err = os.MkdirAll(cfg.FileStorage.BasePath, 0755); err != nil {
    panic(err)
}
```

- [ ] **Step 8: Build and run**

```bash
go build ./... && task local
```

Verify the server starts without errors.

- [ ] **Step 9: Commit**

```bash
git add internal/controllers/http/handlers/api/setup.go internal/controllers/http/controller.go cmd/main.go
git commit -m "feat: register avatar and file download routes, wire DI"
```

---

### Task 14: Add Swagger tag for files

**Files:**
- Modify: `scripts/Taskfile.yml`

- [ ] **Step 1: Add files tag**

In `scripts/Taskfile.yml`, after line 356 (after `web-pushes` tag), add:

```yaml
        - name: files
          description: "Операции с файлами"
```

- [ ] **Step 2: Commit**

```bash
git add scripts/Taskfile.yml
git commit -m "docs: add files swagger tag"
```

---

### Task 15: Frontend — avatar rendering helper

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`

- [ ] **Step 1: Add createAvatarElement helper function**

At the top of `chat.js` (in the utility section), add a shared helper:

```javascript
function createAvatarElement(className, user, titleOverride) {
    const displayName = titleOverride || user.username;

    if (user.avatarPath) {
        const img = document.createElement('img');
        img.className = className;
        img.src = user.avatarPath;
        img.alt = displayName;
        img.onerror = function () {
            const fallback = document.createElement('div');
            fallback.className = className;
            fallback.textContent = displayName.charAt(0).toUpperCase();
            img.replaceWith(fallback);
        };

        return img;
    }

    const div = document.createElement('div');
    div.className = className;
    div.textContent = displayName.charAt(0).toUpperCase();

    return div;
}
```

- [ ] **Step 2: Replace all avatar div creation in chat.js**

Replace all places where avatars are created with `createAvatarElement` calls:

1. `renderChatList` — chat item avatars
2. `showMemberProfile` — member profile avatar
3. `showGroupChatInfo` — group chat avatar + member list avatars
4. `renderSearchResults` — search user avatars
5. `renderCreateChatUsers` — create chat user avatars

For chat items, handle the case where the chat is private (use the other member's `avatarPath`) vs group (use the chat title initial).

- [ ] **Step 3: Test in browser**

```bash
task local
```

Open http://localhost:8080, verify all avatars render as initials (no `avatarPath` set yet). Verify no JS errors in console.

- [ ] **Step 4: Commit**

```bash
git add internal/controllers/http/handlers/web/static/js/chat.js
git commit -m "frontend: add avatar rendering helper with img/fallback logic"
```

---

### Task 16: Frontend — navbar avatar + avatar context menu

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/navbar.js`
- Modify: `internal/controllers/http/handlers/web/templates/navbar.html`
- Modify: `internal/controllers/http/handlers/web/static/css/navbar.css`

- [ ] **Step 1: Update navbar profile avatar rendering**

In `navbar.js`, update the code that creates the navbar profile avatar (around line 293) and `openMyProfileModal` (around line 328) to use `createAvatarElement` or equivalent logic with `<img>` + fallback.

- [ ] **Step 2: Add avatar context menu HTML**

In `navbar.html`, inside the my-profile modal, add a context menu near the avatar:

```html
<div class="avatar-context-menu" id="avatar-context-menu" style="display: none;">
    <button class="avatar-context-menu__item" id="btn-change-avatar">Изменить фото</button>
    <button class="avatar-context-menu__item avatar-context-menu__item--danger" id="btn-delete-avatar" style="display: none;">Удалить фото</button>
</div>
<input type="file" id="avatar-file-input" accept="image/*" style="display: none;">
```

- [ ] **Step 3: Add avatar context menu JS logic**

In `navbar.js`, add click handler on avatar div in my-profile modal:
- Click on avatar → show context menu
- «Изменить фото» → trigger hidden `<input type="file">` click
- File selected → `PUT /api/users/me/avatar` with `FormData`
- «Удалить фото» → `DELETE /api/users/me/avatar`
- After success → update avatar display + `currentUser.avatarPath`

```javascript
const avatarEl = document.getElementById('my-profile-avatar');
const contextMenu = document.getElementById('avatar-context-menu');
const fileInput = document.getElementById('avatar-file-input');
const deleteBtn = document.getElementById('btn-delete-avatar');

avatarEl.addEventListener('click', (e) => {
    e.stopPropagation();
    deleteBtn.style.display = currentUser.avatarPath ? '' : 'none';
    contextMenu.style.display = contextMenu.style.display === 'none' ? '' : 'none';
});

document.getElementById('btn-change-avatar').addEventListener('click', () => {
    contextMenu.style.display = 'none';
    fileInput.click();
});

fileInput.addEventListener('change', async () => {
    const file = fileInput.files[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('avatar', file);

    const resp = await fetchWithAuth('/api/users/me/avatar', {
        method: 'PUT',
        body: formData,
    });

    if (resp.ok) {
        const avatarURL = await resp.text();
        currentUser.avatarPath = avatarURL;
        // Re-render avatar in modal and navbar
        updateAvatarDisplay();
    }

    fileInput.value = '';
});

document.getElementById('btn-delete-avatar').addEventListener('click', async () => {
    contextMenu.style.display = 'none';

    const resp = await fetchWithAuth('/api/users/me/avatar', {
        method: 'DELETE',
    });

    if (resp.ok) {
        currentUser.avatarPath = null;
        updateAvatarDisplay();
    }
});

// Close context menu on outside click
document.addEventListener('click', () => {
    contextMenu.style.display = 'none';
});
```

- [ ] **Step 4: Add CSS for avatar context menu**

In `navbar.css`:

```css
.avatar-context-menu {
    position: absolute;
    z-index: 1000;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.15);
    padding: 4px 0;
}

.avatar-context-menu__item {
    display: block;
    width: 100%;
    padding: 8px 16px;
    border: none;
    background: none;
    cursor: pointer;
    text-align: left;
    font-size: 14px;
}

.avatar-context-menu__item:hover {
    background: var(--bg-hover);
}

.avatar-context-menu__item--danger {
    color: var(--color-danger);
}
```

- [ ] **Step 5: Test in browser**

Test avatar upload, change, delete in the profile modal. Verify fallback to initials after delete.

- [ ] **Step 6: Commit**

```bash
git add internal/controllers/http/handlers/web/static/js/navbar.js internal/controllers/http/handlers/web/templates/navbar.html internal/controllers/http/handlers/web/static/css/navbar.css
git commit -m "frontend: add avatar upload/change/delete in profile modal"
```

---

### Task 17: Frontend — avatar zoom in member profile

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`
- Modify: `internal/controllers/http/handlers/web/templates/chat.html`
- Modify: `internal/controllers/http/handlers/web/static/css/chat.css`

- [ ] **Step 1: Add zoom overlay HTML**

In `chat.html`, add before `</body>`:

```html
<div class="avatar-zoom-overlay" id="avatar-zoom-overlay" style="display: none;">
    <button class="avatar-zoom-overlay__close" id="btn-close-avatar-zoom" aria-label="Закрыть">&times;</button>
    <img class="avatar-zoom-overlay__img" id="avatar-zoom-img" src="" alt="Аватар">
</div>
```

- [ ] **Step 2: Add CSS for zoom overlay**

In `chat.css`:

```css
.avatar-zoom-overlay {
    position: fixed;
    inset: 0;
    z-index: 2000;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
}

.avatar-zoom-overlay__img {
    max-width: 90vw;
    max-height: 90vh;
    border-radius: 8px;
}

.avatar-zoom-overlay__close {
    position: absolute;
    top: 16px;
    right: 16px;
    background: none;
    border: none;
    color: white;
    font-size: 32px;
    cursor: pointer;
}
```

- [ ] **Step 3: Add JS logic for avatar zoom**

In `chat.js`, in `showMemberProfile`:

```javascript
const memberAvatarEl = document.getElementById('member-avatar');

// Make avatar clickable only if user has an avatar image
if (user.avatarPath) {
    memberAvatarEl.style.cursor = 'pointer';
    memberAvatarEl.onclick = () => {
        document.getElementById('avatar-zoom-img').src = user.avatarPath;
        document.getElementById('avatar-zoom-overlay').style.display = '';
    };
} else {
    memberAvatarEl.style.cursor = 'default';
    memberAvatarEl.onclick = null;
}
```

Add close handlers:

```javascript
document.getElementById('btn-close-avatar-zoom').addEventListener('click', () => {
    document.getElementById('avatar-zoom-overlay').style.display = 'none';
});

document.getElementById('avatar-zoom-overlay').addEventListener('click', (e) => {
    if (e.target === document.getElementById('avatar-zoom-overlay')) {
        document.getElementById('avatar-zoom-overlay').style.display = 'none';
    }
});
```

- [ ] **Step 4: Test in browser**

Upload an avatar for a user, open their profile from chat → click avatar → zoom overlay appears. Click outside or close button → overlay closes. Test with user without avatar → click does nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/controllers/http/handlers/web/static/js/chat.js internal/controllers/http/handlers/web/templates/chat.html internal/controllers/http/handlers/web/static/css/chat.css
git commit -m "frontend: add avatar zoom overlay in member profile"
```

---

### Task 18: Update documentation

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/modules.md`
- Create: `internal/usecases/users/doc.md` (update if exists)
- Update: `internal/controllers/http/handlers/api/users/update_avatar/doc.md`
- Update: `internal/controllers/http/handlers/api/users/delete_avatar/doc.md`
- Update: `internal/controllers/http/handlers/api/files/download/doc.md`
- Update all other `doc.md` files in directories that were modified

- [ ] **Step 1: Update docs/architecture.md**

Add FileStorage layer description.

- [ ] **Step 2: Update docs/modules.md**

Add new packages to the module index table.

- [ ] **Step 3: Create/update doc.md in every modified directory**

For each new or modified directory, create or update `doc.md` per the project rule.

- [ ] **Step 4: Commit**

```bash
git add docs/ internal/**/doc.md
git commit -m "docs: update architecture, modules, and doc.md for avatar feature"
```

---

### Task 19: Update existing tests affected by changes

**Files:**
- Modify: `internal/controllers/http/handlers/api/users/update/handler_test.go`
- Modify: any other tests referencing `domains.User` without `AvatarPath`
- Modify: `internal/controllers/http/controller_test.go`
- Modify: `internal/controllers/http/handlers/api/setup_test.go`

- [ ] **Step 1: Find all tests affected by User struct change**

```bash
go test ./... 2>&1 | head -50
```

Fix any compilation errors caused by adding `AvatarPath` to `domains.User`.

- [ ] **Step 2: Fix controller tests**

Update `controller_test.go` and `setup_test.go` to account for new `FileStorageService` parameter.

- [ ] **Step 3: Run full test suite**

```bash
go test ./...
```

Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add -u
git commit -m "test: fix existing tests for avatar feature changes"
```

---

### Task 20: End-to-end smoke test

- [ ] **Step 1: Start the application**

```bash
task local
```

- [ ] **Step 2: Test avatar upload**

Register/login, go to profile, click avatar → «Изменить фото» → select image → verify avatar appears.

- [ ] **Step 3: Test avatar in chat**

Open a chat → verify the other user's avatar shows (or initials if no avatar). Click avatar in chat list → profile opens. Click avatar in profile → zoom if avatar exists.

- [ ] **Step 4: Test avatar delete**

Profile → click avatar → «Удалить фото» → verify initials fallback.

- [ ] **Step 5: Test file download endpoint**

Open `http://localhost:8080/api/files/download/{uuid}.jpg` directly in browser → JPEG loads.

- [ ] **Step 6: Test edge cases**

- Upload a large PNG (>5 MB) → should succeed (under 20 MB limit)
- Upload a GIF → should succeed, displayed as static JPEG
- Upload a non-image file → should get 400 error
- Upload WebP → should succeed

- [ ] **Step 7: Run lint**

```bash
task lint
```

Fix any issues.

- [ ] **Step 8: Final commit if any fixes**

```bash
git add -u
git commit -m "fix: address issues found during smoke testing"
```
