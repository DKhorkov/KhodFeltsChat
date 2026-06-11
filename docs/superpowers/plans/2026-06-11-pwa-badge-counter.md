# PWA Badge Counter — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Показывать счётчик непрочитанных сообщений на иконке PWA (iOS 16.4+, Android Chrome, десктоп) и число непрочитанных на каждом элементе списка чатов. Источник истины — `messages_statuses`, синхронизация — через `loadChats` на клиенте и через `setAppBadge` в `sw.js` push-обработчике.

**Architecture:** Per-chat `UnreadCount` зашивается в `domains.Chat` через скалярный подзапрос в `GetUserChats`. Total `unreadCount` юзера считается в воркере push'а перед циклом по подпискам и прокидывается в JSON payload. `sw.js` дёргает `navigator.setAppBadge(unreadCount)` из push-хендлера. `chat.js` дёргает `setAppBadge` в конце `loadChats` (сумма `chat.unreadCount`), что автоматически покрывает все живые сценарии (WS, открытие чата, polling).

**Tech Stack:** Go 1.24, PostgreSQL + squirrel, NATS, vanilla JS service worker, Badging API.

**Spec:** `docs/superpowers/specs/2026-06-11-pwa-badge-counter-design.md`

---

## File Map

| Action | File | Responsibility |
|---|---|---|
| Modify | `internal/domains/chat.go` | Поле `UnreadCount uint64` в `Chat` |
| Modify | `internal/repositories/chats/repository.go` | Скалярный подзапрос `unread_count` в `GetUserChats` |
| Modify | `internal/repositories/chats/repository_test.go` | Тесты на `UnreadCount` |
| Modify | `internal/repositories/chats/doc.md` | Описать новое поле |
| Modify | `internal/repositories/messages/repository.go` | Новый метод `GetUserUnreadCount` |
| Modify | `internal/repositories/messages/repository_test.go` | Тесты на `GetUserUnreadCount` |
| Modify | `internal/repositories/messages/trace_decorator.go` | Добавить `GetUserUnreadCount` |
| Modify | `internal/repositories/messages/trace_decorator_test.go` | Тест trace decorator'а |
| Modify | `internal/repositories/messages/doc.md` | Описать новый метод |
| Modify | `internal/interfaces/repositories.go` | `GetUserUnreadCount` в `MessagesRepository` + `unreadCount` в `WebPushRepository.SendNotification` |
| Modify | `internal/interfaces/services.go` | `GetUserUnreadCount` в `MessagesService` + `unreadCount` в `NotificationsService.SendNewMessageByWebPush` |
| Regenerate | `mocks/repositories/messages_service.go` | Моки `MessagesRepository`, `WebPushRepository` |
| Regenerate | `mocks/services/messages_service.go` | Мок `MessagesService` |
| Regenerate | `mocks/services/notifications_service.go` | Мок `NotificationsService` |
| Regenerate | `mocks/usecases/messages_usecases.go` | Мок `MessagesUseCases` (наследует `MessagesService`) |
| Modify | `internal/services/messages/service.go` | Реализация `GetUserUnreadCount` |
| Modify | `internal/services/messages/service_test.go` | Тест |
| Modify | `internal/services/messages/trace_decorator.go` | Trace для `GetUserUnreadCount` |
| Modify | `internal/services/messages/trace_decorator_test.go` | Тест trace |
| Modify | `internal/services/messages/doc.md` | Описать |
| Modify | `internal/services/notifications/service.go` | Прокинуть `unreadCount` в `SendNewMessageByWebPush` |
| Modify | `internal/services/notifications/service_test.go` | Тест |
| Modify | `internal/services/notifications/trace_decorator.go` | Подкорректировать сигнатуру |
| Modify | `internal/services/notifications/trace_decorator_test.go` | Тест |
| Modify | `internal/services/notifications/doc.md` | Описать |
| Modify | `internal/usecases/notifications/usecases.go` | Перед циклом — `GetUserUnreadCount`, передать в сервис |
| Modify | `internal/usecases/notifications/usecases_test.go` | Тест |
| Modify | `internal/usecases/notifications/trace_decorator_test.go` | Обновить ожидания |
| Modify | `internal/usecases/notifications/doc.md` | Описать |
| Modify | `internal/repositories/web_push/repository.go` | `unreadCount` в JSON payload + сигнатура |
| Modify | `internal/repositories/web_push/repository_test.go` | Тест нового payload-поля (если есть) |
| Modify | `internal/repositories/web_push/trace_decorator.go` | Сигнатура |
| Modify | `internal/repositories/web_push/trace_decorator_test.go` | Тест |
| Modify | `internal/repositories/web_push/doc.md` | Описать поле |
| Modify | `internal/controllers/http/schemas/chats.go` | Поле `UnreadCount` в HTTP schema |
| Modify | `internal/controllers/http/mappers/chats/chats.go` | Маппинг `UnreadCount` |
| Modify | `internal/controllers/http/mappers/chats/chats_test.go` | Тест маппера |
| Modify | `internal/controllers/http/handlers/web/static/sw.js` | `setAppBadge` в push-хендлере |
| Modify | `internal/controllers/http/handlers/web/static/js/chat.js` | `updateAppBadge` в `loadChats`, число вместо точки |
| Modify | `internal/controllers/http/handlers/web/static/css/chat.css` | Стиль `chat-item__unread-badge` (пилюля с числом) |
| Modify | `internal/controllers/http/handlers/web/static/doc.md` | Упомянуть бейдж |
| Modify | `internal/controllers/http/handlers/web/static/js/doc.md` | Упомянуть `updateAppBadge` |

---

## Phase 1: Domain + per-chat `UnreadCount`

### Task 1: Поле `UnreadCount` в `domains.Chat`

**Files:**
- Modify: `internal/domains/chat.go`

- [ ] **Step 1: Добавить поле в структуру `Chat`**

В `internal/domains/chat.go` после поля `IsRead`:

```go
type Chat struct {
    ID          uint64    `json:"id"`
    Title       *string   `json:"title,omitempty"`
    Description *string   `json:"description,omitempty"`
    Type        ChatType  `json:"type"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
    IsRead      bool      `json:"isRead"` // TODO добавить ручку MarkChatRead
    UnreadCount uint64    `json:"unreadCount"`
    Members     []User    `json:"members,omitempty"`
    Messages    []Message `json:"messages,omitempty"`
}
```

- [ ] **Step 2: Прогнать сборку (тесты ещё не трогаем)**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 3: НЕ коммитим — поле без чтения из БД бесполезно. Идём к Task 2.**

---

### Task 2: SQL-подзапрос `unread_count` в `GetUserChats`

**Files:**
- Modify: `internal/repositories/chats/repository.go:144-287`

- [ ] **Step 1: Сразу после построения `unreadSubquery` (строка ~187) добавить второй скалярный подзапрос — count**

Заменить блок с одним `unreadSQL`:

```go
unreadSQL, unreadArgs, err := unreadSubquery.ToSql()
if err != nil {
    return nil, err
}

isReadColumn := fmt.Sprintf("NOT EXISTS (%s) AS %s", unreadSQL, isReadColumnName)
```

На:

```go
unreadSQL, unreadArgs, err := unreadSubquery.ToSql()
if err != nil {
    return nil, err
}

isReadColumn := fmt.Sprintf("NOT EXISTS (%s) AS %s", unreadSQL, isReadColumnName)

unreadCountSubquery := sq.
    Select("COUNT(*)").
    From(messagesStatusesTableName).
    Join(
        fmt.Sprintf(
            "%s ON %s.%s = %s.%s",
            messagesTableName,
            messagesStatusesTableName,
            messageIDColumnName,
            messagesTableName,
            idColumnName,
        ),
    ).
    Where(
        sq.Expr(
            fmt.Sprintf(
                "%s.%s = %s.%s",
                messagesTableName,
                chatIDColumnName,
                chatsTableName,
                idColumnName,
            ),
        ),
    ).
    Where(
        sq.Eq{
            fmt.Sprintf("%s.%s", messagesStatusesTableName, userIDColumnName): userID,
        },
    ).
    Where(
        sq.Eq{
            fmt.Sprintf("%s.%s", messagesStatusesTableName, isReadColumnName): false,
        },
    ).
    Where(
        sq.Eq{
            fmt.Sprintf("%s.%s", messagesStatusesTableName, isDeletedColumnName): false,
        },
    )

unreadCountSQL, unreadCountArgs, err := unreadCountSubquery.ToSql()
if err != nil {
    return nil, err
}

unreadCountColumn := fmt.Sprintf("(%s) AS %s", unreadCountSQL, unreadCountColumnName)
```

- [ ] **Step 2: Добавить константу `unreadCountColumnName` в блок констант (строки 22-37)**

В блоке `const (...)` добавить:

```go
unreadCountColumnName = "unread_count"
```

- [ ] **Step 3: Подцепить колонку к билдеру (строка ~205)**

Заменить:

```go
builder := sq.
    Select(columnsForSelect...).
    Column(isReadColumn, unreadArgs...).
    From(chatsTableName).
```

На:

```go
builder := sq.
    Select(columnsForSelect...).
    Column(isReadColumn, unreadArgs...).
    Column(unreadCountColumn, unreadCountArgs...).
    From(chatsTableName).
```

- [ ] **Step 4: Обновить срез колонок при сканировании (строка ~272)**

Текущая строка:

```go
columns = columns[:len(columns)-2] // Not to paste Members, Messages fields to Scan function.
```

`UnreadCount` идёт в Chat **между** `IsRead` и `Members`. Текущий `GetEntityColumns` отражает порядок полей в struct. Мы добавили одно поле перед `Members`/`Messages` → срез `len-2` корректно отрезает только `Members` и `Messages`, и `UnreadCount` остаётся среди сканируемых.

Убедиться, что комментарий верный, и проверить, что порядок `SELECT` колонок (chats.id, title, description, type, created_at, updated_at, is_read, unread_count) соответствует порядку полей в `domains.Chat` (ID, Title, Description, Type, CreatedAt, UpdatedAt, IsRead, UnreadCount).

- [ ] **Step 5: Собрать**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 6: Запустить существующие тесты chats репозитория**

```bash
go test ./internal/repositories/chats/... -run TestRepositorySuite/TestGetUserChats -v
```

Expected: PASS (старые тесты не должны сломаться — поле просто сканируется как 0 или N, IsRead продолжает работать).

- [ ] **Step 7: Коммит**

```bash
git add internal/domains/chat.go internal/repositories/chats/repository.go
git commit -m "feat(chats): add UnreadCount field to Chat domain"
```

---

### Task 3: Тесты на `UnreadCount` в `GetUserChats`

**Files:**
- Modify: `internal/repositories/chats/repository_test.go`

- [ ] **Step 1: Найти `TestGetUserChats_IsReadFalse_WhenUnreadMessagesExist` (строка ~289) — взять его как образец сетапа БД**

Образец показывает, как создаются user, chat, message и `messages_statuses` через хелперы тест-сьюита.

- [ ] **Step 2: Добавить новые тесты после блока `IsRead*` тестов**

```go
func (s *RepositoryTestSuite) TestGetUserChats_UnreadCount_Zero_WhenAllRead() {
    userID := s.createTestUser()
    chatID := s.createTestPrivateChat(userID)
    msgID := s.createTestMessage(chatID, userID)
    s.markMessageStatus(msgID, userID, true, false)

    chats, err := s.repository.GetUserChats(s.ctx, userID, nil)
    s.Require().NoError(err)
    s.Require().Len(chats, 1)
    s.Equal(uint64(0), chats[0].UnreadCount)
}

func (s *RepositoryTestSuite) TestGetUserChats_UnreadCount_CountsUnreadOnly() {
    otherID := s.createTestUser()
    userID := s.createTestUser()
    chatID := s.createTestPrivateChatWithUsers(otherID, userID)

    msg1 := s.createTestMessage(chatID, otherID)
    s.markMessageStatus(msg1, userID, false, false)

    msg2 := s.createTestMessage(chatID, otherID)
    s.markMessageStatus(msg2, userID, false, false)

    msg3 := s.createTestMessage(chatID, otherID)
    s.markMessageStatus(msg3, userID, true, false) // прочитанное

    chats, err := s.repository.GetUserChats(s.ctx, userID, nil)
    s.Require().NoError(err)
    s.Require().Len(chats, 1)
    s.Equal(uint64(2), chats[0].UnreadCount)
}

func (s *RepositoryTestSuite) TestGetUserChats_UnreadCount_ExcludesDeletedStatuses() {
    otherID := s.createTestUser()
    userID := s.createTestUser()
    chatID := s.createTestPrivateChatWithUsers(otherID, userID)

    msg1 := s.createTestMessage(chatID, otherID)
    s.markMessageStatus(msg1, userID, false, false)

    msg2 := s.createTestMessage(chatID, otherID)
    s.markMessageStatus(msg2, userID, false, true) // удалено для юзера

    chats, err := s.repository.GetUserChats(s.ctx, userID, nil)
    s.Require().NoError(err)
    s.Require().Len(chats, 1)
    s.Equal(uint64(1), chats[0].UnreadCount)
}
```

Прежде чем писать — посмотреть существующие хелперы в `repository_test.go` (имена `createTestUser`, `createTestPrivateChat`, `markMessageStatus` — это предположения, **скорректировать под реальные имена хелперов сьюта**).

- [ ] **Step 3: Запустить тесты**

```bash
go test ./internal/repositories/chats/... -run "TestRepositorySuite/TestGetUserChats_UnreadCount" -v
```

Expected: PASS.

- [ ] **Step 4: Коммит**

```bash
git add internal/repositories/chats/repository_test.go
git commit -m "test(chats): cover UnreadCount in GetUserChats"
```

---

### Task 4: Обновить `doc.md` репозитория chats

**Files:**
- Modify: `internal/repositories/chats/doc.md`

- [ ] **Step 1: Прочитать текущий `doc.md`, найти описание `GetUserChats`**

- [ ] **Step 2: Добавить упоминание `UnreadCount`**

В описание `GetUserChats` добавить:

> Каждый чат включает `UnreadCount` — число непрочитанных и неудалённых сообщений для текущего пользователя (скалярный подзапрос по `messages_statuses`).

- [ ] **Step 3: Коммит**

```bash
git add internal/repositories/chats/doc.md
git commit -m "docs(chats): mention UnreadCount in GetUserChats"
```

---

### Task 5: HTTP schema и mapper — поле `UnreadCount`

**Files:**
- Modify: `internal/controllers/http/schemas/chats.go`
- Modify: `internal/controllers/http/mappers/chats/chats.go`
- Modify: `internal/controllers/http/mappers/chats/chats_test.go`

- [ ] **Step 1: Добавить поле в schema**

В `internal/controllers/http/schemas/chats.go` после `IsRead`:

```go
// Number of unread non-deleted messages in the chat for the current user.
// required: true
// nullable: false
// minimum: 0
// example: 3
UnreadCount uint64 `json:"unreadCount"`
```

- [ ] **Step 2: Добавить маппинг**

В `internal/controllers/http/mappers/chats/chats.go`:

```go
func MapChat(chat domains.Chat) schemas.Chat {
    return schemas.Chat{
        ID:          chat.ID,
        Title:       chat.Title,
        Description: chat.Description,
        Type:        string(chat.Type),
        CreatedAt:   chat.CreatedAt,
        UpdatedAt:   chat.UpdatedAt,
        IsRead:      chat.IsRead,
        UnreadCount: chat.UnreadCount,
        Members:     users.MapUsers(chat.Members),
        Messages:    messages.MapMessages(chat.Messages),
    }
}
```

- [ ] **Step 3: Обновить существующие тесты маппера**

Открыть `internal/controllers/http/mappers/chats/chats_test.go`. Найти все `domains.Chat{...}` фикстуры и `schemas.Chat{...}` expected — добавить `UnreadCount: <value>` в обоих местах.

Добавить один новый тест:

```go
func TestMapChat_PreservesUnreadCount(t *testing.T) {
    chat := domains.Chat{
        ID:          1,
        Type:        domains.ChatTypePrivate,
        UnreadCount: 42,
    }
    got := chats.MapChat(chat)
    assert.Equal(t, uint64(42), got.UnreadCount)
}
```

- [ ] **Step 4: Запустить тесты**

```bash
go test ./internal/controllers/http/mappers/chats/... -v
```

Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/controllers/http/schemas/chats.go internal/controllers/http/mappers/chats/
git commit -m "feat(http): expose UnreadCount in chats schema"
```

---

## Phase 2: Total `UnreadCount` для воркера push'а

### Task 6: Метод `GetUserUnreadCount` в `MessagesRepository`

**Files:**
- Modify: `internal/repositories/messages/repository.go`

- [ ] **Step 1: Добавить метод в конец файла перед закрывающими типами**

```go
func (repo *Repository) GetUserUnreadCount(
    ctx context.Context,
    userID uint64,
) (uint64, error) {
    stmt, params, err := sq.
        Select("COUNT(*)").
        From(messagesStatusesTableName).
        Where(sq.Eq{userIDColumnName: userID}).
        Where(sq.Eq{isReadColumnName: false}).
        Where(sq.Eq{isDeletedColumnName: false}).
        PlaceholderFormat(sq.Dollar).
        ToSql()
    if err != nil {
        return 0, err
    }

    var count uint64
    if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&count); err != nil {
        return 0, err
    }

    return count, nil
}
```

- [ ] **Step 2: Сборка**

```bash
go build ./...
```

Expected: PASS (интерфейс ещё не обновлён — пока что метод есть только на конкретном типе).

- [ ] **Step 3: НЕ коммитим — без обновления интерфейса и моков не пройдёт компиляция в местах использования. Идём к Task 7.**

---

### Task 7: Расширить интерфейсы `MessagesRepository` и `MessagesService`

**Files:**
- Modify: `internal/interfaces/repositories.go`
- Modify: `internal/interfaces/services.go`

- [ ] **Step 1: Добавить `GetUserUnreadCount` в `MessagesRepository`**

В `internal/interfaces/repositories.go` в блок `MessagesRepository`:

```go
type MessagesRepository interface {
    SaveMessage(ctx context.Context, message domains.Message) (uint64, error)
    GetChatMessages(...) ...
    GetMessageByID(...) ...
    ChangeMessagesIsReadStatus(...) ...
    ReadAllChatMessages(ctx context.Context, userID uint64, chatID uint64) error
    GetUserUnreadCount(ctx context.Context, userID uint64) (uint64, error)
    DeleteMessageForUser(...) ...
    DeleteMessageForAll(...) ...
    UpdateMessage(...) ...
}
```

- [ ] **Step 2: Добавить `GetUserUnreadCount` в `MessagesService`**

В `internal/interfaces/services.go` в блок `MessagesService`:

```go
type MessagesService interface {
    SaveMessage(...) ...
    GetChatMessages(...) ...
    GetMessageByID(...) ...
    GetUserUnreadCount(ctx context.Context, userID uint64) (uint64, error)
    DeleteMessage(...) ...
    UpdateMessage(...) ...
}
```

`MessagesUseCases` — это alias на `MessagesService`, дополнительно ничего не нужно.

- [ ] **Step 3: Перегенерировать моки**

```bash
go generate ./internal/interfaces/...
```

Expected: моки `mocks/repositories/messages_service.go`, `mocks/services/messages_service.go`, `mocks/usecases/messages_usecases.go` обновлены.

- [ ] **Step 4: Сборка**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/interfaces/ internal/repositories/messages/repository.go mocks/
git commit -m "feat(messages): add GetUserUnreadCount method"
```

---

### Task 8: Trace decorator для `GetUserUnreadCount`

**Files:**
- Modify: `internal/repositories/messages/trace_decorator.go`

- [ ] **Step 1: Найти пример trace-метода в файле (например, `ReadAllChatMessages`)**

- [ ] **Step 2: Добавить аналогичный метод**

```go
func (d *TraceDecorator) GetUserUnreadCount(
    ctx context.Context,
    userID uint64,
) (uint64, error) {
    ctx, span := d.tracer.Start(ctx, tracing.SpanName(spanPrefix, "GetUserUnreadCount"))
    defer span.End()

    return d.base.GetUserUnreadCount(ctx, userID)
}
```

(Точные имена `spanPrefix`/`tracing.SpanName` подсмотреть в соседних методах файла.)

- [ ] **Step 3: Сборка**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Коммит**

```bash
git add internal/repositories/messages/trace_decorator.go
git commit -m "feat(messages): trace decorator for GetUserUnreadCount"
```

---

### Task 9: Тест trace decorator'а

**Files:**
- Modify: `internal/repositories/messages/trace_decorator_test.go`

- [ ] **Step 1: Найти образец — `TestTraceDecorator_ReadAllChatMessages` или похожий**

- [ ] **Step 2: Скопировать паттерн, заменить на новый метод**

```go
func TestTraceDecorator_GetUserUnreadCount(t *testing.T) {
    tests := []struct {
        name          string
        userID        uint64
        mockSetup     func(repo *mockrepositories.MockMessagesRepository)
        expectedCount uint64
        expectedError error
    }{
        {
            name:   "successful unread count with tracing",
            userID: 42,
            mockSetup: func(repo *mockrepositories.MockMessagesRepository) {
                repo.EXPECT().
                    GetUserUnreadCount(gomock.Any(), uint64(42)).
                    Return(uint64(7), nil)
            },
            expectedCount: 7,
            expectedError: nil,
        },
        {
            name:   "error from base repository",
            userID: 99,
            mockSetup: func(repo *mockrepositories.MockMessagesRepository) {
                repo.EXPECT().
                    GetUserUnreadCount(gomock.Any(), uint64(99)).
                    Return(uint64(0), errors.New("db error"))
            },
            expectedCount: 0,
            expectedError: errors.New("db error"),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            base := mockrepositories.NewMockMessagesRepository(ctrl)
            tt.mockSetup(base)

            decorator := messages.NewTraceDecorator(base, /* tracer */ ...)
            count, err := decorator.GetUserUnreadCount(context.Background(), tt.userID)

            if tt.expectedError != nil {
                require.EqualError(t, err, tt.expectedError.Error())
            } else {
                require.NoError(t, err)
            }
            assert.Equal(t, tt.expectedCount, count)
        })
    }
}
```

Скорректировать конструктор `NewTraceDecorator` и импорты под актуальные имена в файле.

- [ ] **Step 3: Тесты**

```bash
go test ./internal/repositories/messages/... -run TestTraceDecorator_GetUserUnreadCount -v
```

Expected: PASS.

- [ ] **Step 4: Коммит**

```bash
git add internal/repositories/messages/trace_decorator_test.go
git commit -m "test(messages): cover GetUserUnreadCount trace decorator"
```

---

### Task 10: Реализация `GetUserUnreadCount` в сервисе

**Files:**
- Modify: `internal/services/messages/service.go`

- [ ] **Step 1: Добавить метод по образцу `GetMessageByID`**

```go
func (s *Service) GetUserUnreadCount(
    ctx context.Context,
    userID uint64,
) (uint64, error) {
    var count uint64

    err := s.uow.Do(
        ctx,
        func(ctx context.Context, tx pg.Transaction) error {
            messagesRepository := s.newMessagesRepositoryFunc(tx)

            var err error
            count, err = messagesRepository.GetUserUnreadCount(ctx, userID)

            return err
        },
    )
    if err != nil {
        return 0, err
    }

    return count, nil
}
```

- [ ] **Step 2: Сборка**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Не коммитим — добавим вместе с тестом и trace-декоратором.**

---

### Task 11: Trace decorator сервиса + тесты сервиса

**Files:**
- Modify: `internal/services/messages/trace_decorator.go`
- Modify: `internal/services/messages/trace_decorator_test.go`
- Modify: `internal/services/messages/service_test.go`

- [ ] **Step 1: Добавить trace-метод (по образцу из файла)**

```go
func (d *TraceDecorator) GetUserUnreadCount(
    ctx context.Context,
    userID uint64,
) (uint64, error) {
    ctx, span := d.tracer.Start(ctx, tracing.SpanName(spanPrefix, "GetUserUnreadCount"))
    defer span.End()

    return d.base.GetUserUnreadCount(ctx, userID)
}
```

- [ ] **Step 2: Тест для trace-декоратора (по образцу TestTraceDecorator_GetMessageByID)**

```go
func TestTraceDecorator_GetUserUnreadCount(t *testing.T) {
    // success + error branches с проверкой проброса в base
}
```

- [ ] **Step 3: Тест для сервиса (по образцу TestService_GetMessageByID)**

```go
func TestService_GetUserUnreadCount(t *testing.T) {
    // мокаем UoW + MessagesRepository, проверяем что count прокинут
}
```

- [ ] **Step 4: Тесты**

```bash
go test ./internal/services/messages/... -v
```

Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/services/messages/
git commit -m "feat(messages-service): add GetUserUnreadCount"
```

---

### Task 12: Расширить `NotificationsService.SendNewMessageByWebPush` параметром `unreadCount`

**Files:**
- Modify: `internal/interfaces/services.go`
- Modify: `internal/interfaces/repositories.go` (WebPushRepository)
- Modify: `internal/services/notifications/service.go`
- Modify: `internal/services/notifications/trace_decorator.go`
- Modify: `internal/repositories/web_push/repository.go`
- Modify: `internal/repositories/web_push/trace_decorator.go`

- [ ] **Step 1: `interfaces/services.go` — добавить `unreadCount uint64` в `NotificationsService.SendNewMessageByWebPush`**

```go
SendNewMessageByWebPush(
    ctx context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64,
) error
```

- [ ] **Step 2: `interfaces/repositories.go` — то же в `WebPushRepository.SendNotification`**

```go
SendNotification(
    ctx context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64,
) error
```

- [ ] **Step 3: Перегенерировать моки**

```bash
go generate ./internal/interfaces/...
```

- [ ] **Step 4: `services/notifications/service.go:42` — пробросить вниз**

```go
func (s *Service) SendNewMessageByWebPush(
    ctx context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64,
) error {
    return s.webPushRepository.SendNotification(ctx, subscription, message, unreadCount)
}
```

- [ ] **Step 5: `services/notifications/trace_decorator.go` — поправить сигнатуру**

```go
func (d *TraceDecorator) SendNewMessageByWebPush(
    ctx context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64,
) error {
    ctx, span := d.tracer.Start(ctx, tracing.SpanName(spanPrefix, "SendNewMessageByWebPush"))
    defer span.End()

    return d.base.SendNewMessageByWebPush(ctx, subscription, message, unreadCount)
}
```

- [ ] **Step 6: `repositories/web_push/repository.go:32` — принять параметр и положить в payload**

```go
func (repo *Repository) SendNotification(
    _ context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64,
) error {
    payload, err := json.Marshal(map[string]any{
        "title":       message.Sender.Username,
        "body":        message.Text,
        "chatId":      message.ChatID,
        "timestamp":   message.CreatedAt.UnixMilli(),
        "unreadCount": unreadCount,
    })
    // ... остальное без изменений
}
```

- [ ] **Step 7: `repositories/web_push/trace_decorator.go` — поправить сигнатуру**

```go
func (d *TraceDecorator) SendNotification(
    ctx context.Context,
    subscription domains.WebPushSubscription,
    message domains.Message,
    unreadCount uint64,
) error {
    ctx, span := d.tracer.Start(...)
    defer span.End()
    return d.base.SendNotification(ctx, subscription, message, unreadCount)
}
```

- [ ] **Step 8: Сборка**

```bash
go build ./...
```

Expected: компилятор подсветит вызывающие места (usecase, тесты). Это ожидаемо, чиним в следующих задачах.

- [ ] **Step 9: НЕ коммитим — фиксим вызовы.**

---

### Task 13: Использовать `unreadCount` в usecase

**Files:**
- Modify: `internal/usecases/notifications/usecases.go`

- [ ] **Step 1: Перед циклом по подпискам получить count**

В `SendNewMessageByWebPush` после получения `subscriptions` (строка ~106) добавить:

```go
subscriptions, err := u.webPushSubscriptionsService.GetWebPushSubscriptionsByUserID(
    ctx,
    userID,
)
if err != nil {
    return err
}

unreadCount, err := u.messagesService.GetUserUnreadCount(ctx, userID)
if err != nil {
    return err
}

for _, sub := range subscriptions {
    if err = u.notificationsService.SendNewMessageByWebPush(ctx, sub, *message, unreadCount); err != nil {
        // ... существующая логика обработки ошибок
    }
}
```

- [ ] **Step 2: Сборка**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Не коммитим — починим тесты в следующих задачах.**

---

### Task 14: Обновить тесты (сервис, репозиторий, trace, usecase)

**Files:**
- Modify: `internal/services/notifications/service_test.go`
- Modify: `internal/services/notifications/trace_decorator_test.go`
- Modify: `internal/repositories/web_push/repository_test.go` (если существует)
- Modify: `internal/repositories/web_push/trace_decorator_test.go`
- Modify: `internal/usecases/notifications/usecases_test.go`
- Modify: `internal/usecases/notifications/trace_decorator_test.go`

- [ ] **Step 1: Прогнать тесты и собрать список падений**

```bash
go test ./... 2>&1 | grep -E "FAIL|wrong number of args|undefined" | head -50
```

- [ ] **Step 2: В `services/notifications/service_test.go` — добавить `unreadCount` параметр во все вызовы `SendNewMessageByWebPush` и ожидаемые `webPushRepository.SendNotification(...)`**

Например:

```go
mockWebPushRepo.EXPECT().
    SendNotification(gomock.Any(), subscription, message, uint64(5)).
    Return(nil)

err := service.SendNewMessageByWebPush(ctx, subscription, message, uint64(5))
```

- [ ] **Step 3: Аналогично в `trace_decorator_test.go` нотификаций и web_push.**

- [ ] **Step 4: В `usecases/notifications/usecases_test.go` `SendNewMessageByWebPush`-тестах**

Добавить ожидание вызова `messagesService.GetUserUnreadCount(...)` перед циклом + передачу значения в `notificationsService.SendNewMessageByWebPush`:

```go
mockMessagesService.EXPECT().
    GetMessageByID(gomock.Any(), userID, payload.MessageID).
    Return(&message, nil)

mockMessagesService.EXPECT().
    GetUserUnreadCount(gomock.Any(), userID).
    Return(uint64(3), nil)

mockWebPushSubscriptionsService.EXPECT().
    GetWebPushSubscriptionsByUserID(gomock.Any(), userID).
    Return(subscriptions, nil)

mockNotificationsService.EXPECT().
    SendNewMessageByWebPush(gomock.Any(), gomock.Any(), message, uint64(3)).
    Return(nil).
    Times(len(subscriptions))
```

Добавить новый тест: `SendNewMessageByWebPush_ErrorOnGetUserUnreadCount` — `GetUserUnreadCount` возвращает ошибку, цикл не вызывается, ошибка пробрасывается.

- [ ] **Step 5: `usecases/notifications/trace_decorator_test.go` — обновить ожидания под новый порядок вызовов (если есть проверки последовательности)**

- [ ] **Step 6: Если в `repositories/web_push/` есть тест на JSON-payload — добавить проверку, что `unreadCount` сериализуется.**

Если такого теста нет — добавить:

```go
func TestRepository_SendNotification_IncludesUnreadCountInPayload(t *testing.T) {
    // развернуть HTTP test server, перехватить тело, распарсить JSON, assert unreadCount == 7
}
```

(Если сетап такого теста слишком тяжёлый — отложить, минимум должен быть unit-тест на trace-декоратор с проверкой проброса.)

- [ ] **Step 7: Прогнать все тесты**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Коммит**

```bash
git add internal/services/notifications/ internal/repositories/web_push/ internal/usecases/notifications/ mocks/
git commit -m "feat(push): include unreadCount in web push payload"
```

---

### Task 15: Обновить doc.md затронутых модулей

**Files:**
- Modify: `internal/repositories/messages/doc.md`
- Modify: `internal/repositories/web_push/doc.md`
- Modify: `internal/services/messages/doc.md`
- Modify: `internal/services/notifications/doc.md`
- Modify: `internal/usecases/notifications/doc.md`

- [ ] **Step 1: В каждом `doc.md` упомянуть новый метод/поле/параметр**

Минимум — одна строка в описании соответствующего метода:

- `messages/doc.md` — добавить раздел про `GetUserUnreadCount` (что считает).
- `web_push/doc.md` — упомянуть поле `unreadCount` в payload и параметр в `SendNotification`.
- `services/messages/doc.md` — упомянуть `GetUserUnreadCount`.
- `services/notifications/doc.md` — упомянуть, что `SendNewMessageByWebPush` принимает `unreadCount`.
- `usecases/notifications/doc.md` — упомянуть, что перед циклом по подпискам тянется `GetUserUnreadCount`.

- [ ] **Step 2: Коммит**

```bash
git add internal/repositories/messages/doc.md internal/repositories/web_push/doc.md internal/services/messages/doc.md internal/services/notifications/doc.md internal/usecases/notifications/doc.md
git commit -m "docs: describe unread count flow for web push"
```

---

## Phase 3: Frontend (PWA badge + число в списке чатов)

### Task 16: `setAppBadge` в `sw.js` push-обработчике

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/sw.js`

- [ ] **Step 1: Заменить содержимое `push` listener**

Найти текущий блок (строки 9-41) и заменить на:

```js
self.addEventListener('push', (event) => {
    const data = event.data ? event.data.json() : {};

    const title = data.title || 'Новое сообщение';
    const options = {
        body: data.body || '',
        icon: '/web/static/assets/icon.png',
        timestamp: data.timestamp || Date.now(),
        data: {
            chatId: data.chatId,
        },
    };

    event.waitUntil((async () => {
        // Badge: всегда ставим присланное сервером абсолютное число (или сбрасываем).
        if ('setAppBadge' in self.navigator && typeof data.unreadCount === 'number') {
            try {
                if (data.unreadCount > 0) {
                    await self.navigator.setAppBadge(data.unreadCount);
                } else {
                    await self.navigator.clearAppBadge();
                }
            } catch (err) {
                // setAppBadge может бросать в редких случаях — не блокируем уведомление.
                console.warn('setAppBadge failed:', err);
            }
        }

        const windowClients = await clients.matchAll({ type: 'window', includeUncontrolled: true });
        // На десктопе не показываем уведомление, если чат в фокусе.
        // На iOS подавлять нельзя — iOS требует showNotification на каждый push,
        // иначе молча отбрасывает уведомление и может отозвать разрешение.
        const isIOS = /iP(hone|ad|od)/.test(self.navigator.userAgent);
        if (!isIOS) {
            const hasFocusedClient = windowClients.some(
                (client) => client.focused && client.url.includes('/web/chat')
            );
            if (hasFocusedClient) return;
        }

        await self.registration.showNotification(title, options);
    })());
});
```

- [ ] **Step 2: Проверить вручную в браузере**

Открыть DevTools → Application → Service Workers → Push → создать тестовый push с payload:

```json
{"title":"Test","body":"Hello","chatId":1,"unreadCount":5}
```

В Android Chrome или iOS PWA на homescreen — должен появиться бейдж `5`. В обычной вкладке Safari macOS бейдж не появится (это нормально).

- [ ] **Step 3: Коммит**

```bash
git add internal/controllers/http/handlers/web/static/sw.js
git commit -m "feat(sw): set app badge from web push payload"
```

---

### Task 17: `updateAppBadge` в `loadChats`

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`

- [ ] **Step 1: Найти функцию `loadChats` (строка ~259)**

Текущая:

```js
async function loadChats() {
    try {
        const resp = await fetchWithAuth('/api/chats');
        if (!resp.ok) return;

        const chats = await resp.json();
        renderChatList(chats);
    } catch (err) {
        console.error(err);
    }
}
```

Заменить на:

```js
async function loadChats() {
    try {
        const resp = await fetchWithAuth('/api/chats');
        if (!resp.ok) return;

        const chats = await resp.json();
        renderChatList(chats);
        updateAppBadge(chats);
    } catch (err) {
        console.error(err);
    }
}

function updateAppBadge(chats) {
    if (!('setAppBadge' in navigator)) return;
    const total = chats.reduce((sum, c) => sum + (c.unreadCount || 0), 0);
    try {
        if (total > 0) {
            navigator.setAppBadge(total);
        } else {
            navigator.clearAppBadge();
        }
    } catch (err) {
        console.warn('setAppBadge failed:', err);
    }
}
```

- [ ] **Step 2: Коммит**

```bash
git add internal/controllers/http/handlers/web/static/js/chat.js
git commit -m "feat(chat-web): sync app badge with loadChats"
```

---

### Task 18: Число вместо точки в списке чатов

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`
- Modify: `internal/controllers/http/handlers/web/static/css/chat.css`

- [ ] **Step 1: В `renderChatList` (строка ~325-372) заменить блок с точкой**

Найти:

```js
if (!chat.isRead) {
    const dot = document.createElement('div');
    dot.className = 'chat-item__unread-dot';
    item.appendChild(dot);
}
```

Заменить на:

```js
if (chat.unreadCount > 0) {
    const badge = document.createElement('div');
    badge.className = 'chat-item__unread-badge';
    badge.textContent = chat.unreadCount > 99 ? '99+' : String(chat.unreadCount);
    item.appendChild(badge);
}
```

- [ ] **Step 2: Заменить старое правило `.chat-item__unread-dot` (строки 139-146) на пилюлю с числом**

В `internal/controllers/http/handlers/web/static/css/chat.css`:

```css
.chat-item__unread-badge {
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    background: var(--accent);
    color: var(--accent-contrast, #fff);
    border-radius: var(--radius-full);
    font-size: var(--font-xs);
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    margin-left: var(--space-sm);
    line-height: 1;
}
```

Если переменная `--accent-contrast` не объявлена в `global.css` — заменить на хардкод `#fff`.

- [ ] **Step 3: Проверить в браузере**

Запустить локально (`task local`), залогиниться, открыть `/web/chat`, написать сообщение от другого юзера → убедиться, что у чата в списке появляется число `1`, после ещё одного сообщения — `2`, после открытия чата — пропадает.

- [ ] **Step 4: Коммит**

```bash
git add internal/controllers/http/handlers/web/static/js/chat.js internal/controllers/http/handlers/web/static/css/chat.css
git commit -m "feat(chat-web): show unread count badge instead of dot"
```

---

### Task 19: Обновить web-doc'и

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/doc.md`
- Modify: `internal/controllers/http/handlers/web/static/doc.md`

- [ ] **Step 1: В `js/doc.md` упомянуть `updateAppBadge` в разделе про `chat.js`**

> `loadChats` после рендера вызывает `updateAppBadge(chats)`, который суммирует `unreadCount` по всем чатам и вызывает `navigator.setAppBadge` / `clearAppBadge`. Это единственная точка обновления PWA бейджа на клиенте.

> Service worker (`sw.js`) дублирует то же действие на push-обработчике, используя `unreadCount` из payload.

- [ ] **Step 2: В корневом `static/doc.md` упомянуть, что `sw.js` ставит бейдж из push-payload.**

- [ ] **Step 3: Коммит**

```bash
git add internal/controllers/http/handlers/web/static/js/doc.md internal/controllers/http/handlers/web/static/doc.md
git commit -m "docs(web): describe PWA badge counter flow"
```

---

## Phase 4: Финал

### Task 20: Полная проверка

- [ ] **Step 1: Линт**

```bash
task lint
```

Expected: PASS.

- [ ] **Step 2: Полный прогон тестов**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Ручная проверка в браузере (golden path)**

Локально (`task local`):

1. Логин как user A в одной вкладке, как user B — в другой.
2. A пишет B пять сообщений.
3. У B в списке чатов на месте точки — число `5`. На вкладке Chrome — бейдж `5` в табе/доке.
4. B открывает чат → бейдж пропадает.
5. B закрывает вкладку. A пишет ещё. На устройстве с установленной PWA (Android/iOS) — бейдж `1` на иконке.

- [ ] **Step 4: Если все шаги зелёные — финальный empty-commit отсутствует, ничего не делаем.**

Если что-то не сошлось — фиксим точечно, коммитим отдельно.

---

## Self-Review Checklist

- [x] **Spec coverage:** каждое требование спеки покрыто: домен (Task 1), per-chat count (Task 2-4), HTTP (Task 5), total count (Task 6-11), push payload (Task 12-14), docs (Task 15), SW badge (Task 16), client loadChats badge (Task 17), число в списке (Task 18).
- [x] **Placeholder scan:** TODO/TBD не встречаются; в местах, где образец зависит от существующего файла (имена хелперов в тестах, конструктор trace decorator'а), явно сказано «подсмотреть в файле».
- [x] **Type consistency:** `GetUserUnreadCount(ctx, userID) (uint64, error)` одинаково везде. `unreadCount uint64` одинаково везде. `UnreadCount uint64` в домене, schema, JSON.
- [x] **Никаких новых endpoint'ов** — счётчик через `/api/chats`, прочтение через `GET /api/chats/{id}/messages`.
