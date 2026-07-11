# Message Reactions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить emoji-реакции на сообщения: справочник emoji в БД, M2M-связь юзер↔сообщение, интеграция с чтением сообщений и WebSocket-фан-аутом всем участникам чата.

**Architecture:** Отдельный пакет `reactions/` на каждом слое (repository, service, usecase). Multi-реакции на юзера (Slack-стиль). ADD возвращает ошибку на дубликат (409), REMOVE идемпотентен (200 всегда, WS только при `deleted=true`). Загрузка реакций к сообщениям — отдельным вызовом в usecase (без разбухания join'ов существующего messages-репо).

**Tech Stack:** Go 1.24, PostgreSQL (squirrel query builder), goose migrations, gorilla/mux, mockgen, OpenTelemetry tracing, WebSocket (gorilla/websocket).

## Global Constraints

- Все интерфейсы — в `internal/interfaces/` с `//go:generate mockgen` директивами по существующему шаблону.
- Каждый репозиторий/сервис/usecase имеет `TraceDecorator` в отдельном файле.
- Все SQL-плейсхолдеры — `sq.Dollar` (`$1, $2, ...`) — драйвер `pq`.
- Новые sentinel-ошибки — в `internal/errors/`.
- Транзакции — через `interfaces.UnitOfWork` (`uow.Do(ctx, func(ctx, tx) error {...})`).
- Каждый пакет с кодом имеет `doc.md`, обновляемый при изменениях (правило проекта из CLAUDE.md).
- Каждое изменение Go-кода сопровождается тестами (правило проекта).
- Один коммит на задачу, коммит-сообщение — краткое описание в стиле существующих коммитов проекта (см. `git log`).

## Spec Reference

Спека: `docs/superpowers/specs/2026-07-11-message-reactions-design.md`. Все детали (SQL, семантика ошибок, HTTP-коды, WS-события) — оттуда.

---

## Task 1: Миграция схемы и seed

**Files:**
- Create: `migrations/20260711000000_message_reactions.sql`

**Interfaces:**
- Produces: таблицы `reactions`, `message_reactions`; seed 8 базовых emoji.

- [ ] **Step 1: Создать файл миграции**

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS reactions
(
    id         SERIAL PRIMARY KEY,
    emoji      VARCHAR(16) NOT NULL UNIQUE,
    sort_order INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS message_reactions
(
    id          SERIAL PRIMARY KEY,
    message_id  INTEGER   NOT NULL REFERENCES messages (id)  ON DELETE CASCADE,
    user_id     INTEGER   NOT NULL REFERENCES users (id)     ON DELETE CASCADE,
    reaction_id INTEGER   NOT NULL REFERENCES reactions (id) ON DELETE CASCADE,
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS message_reactions;
DROP TABLE IF EXISTS reactions;
-- +goose StatementEnd
```

- [ ] **Step 2: Проверить, что миграция применяется и откатывается**

Run: `task migrate-up` затем `task migrate-down` (или прямые команды goose из `Taskfile.yml`).
Expected: обе таблицы создаются, seed вставляется; down-миграция чисто откатывает.

- [ ] **Step 3: Обновить `migrations/doc.md`**

Добавить строку про новую миграцию с описанием создаваемых таблиц и seed'а.

- [ ] **Step 4: Commit**

```bash
git add migrations/20260711000000_message_reactions.sql migrations/doc.md
git commit -m "feat: миграция таблиц reactions и message_reactions"
```

---

## Task 2: Доменные типы, ошибки, WS-события

**Files:**
- Create: `internal/domains/reaction.go`
- Create: `internal/domains/reaction_test.go`
- Create: `internal/errors/reactions.go`
- Modify: `internal/domains/message.go` (добавить поле `Reactions`)
- Modify: `internal/domains/ws_event.go` (новые типы событий + payload'ы)

**Interfaces:**
- Produces:
  - `domains.Reaction{ID uint64, Emoji string}`
  - `domains.MessageReactionSummary{Reaction Reaction, UserIDs []uint64}`
  - `domains.MessageReactionDTO{MessageID, ReactionID, UserID uint64}`
  - `domains.Message.Reactions []MessageReactionSummary`
  - `customerrors.ErrReactionAlreadyExists`, `customerrors.ErrReactionNotFound`
  - `domains.WSEventReactionAdded`, `domains.WSEventReactionRemoved`
  - `domains.ReactionAddedPayload`, `domains.ReactionRemovedPayload`

- [ ] **Step 1: Создать `internal/domains/reaction.go`**

```go
package domains

type Reaction struct {
	ID    uint64 `json:"id"`
	Emoji string `json:"emoji"`
}

// MessageReactionSummary агрегирует реакцию на сообщение: сама реакция и
// список userId, кто её поставил. Клиент считает count как len(UserIDs).
type MessageReactionSummary struct {
	Reaction Reaction `json:"reaction"`
	UserIDs  []uint64 `json:"userIds"`
}

type MessageReactionDTO struct {
	MessageID  uint64 `json:"-"`
	ReactionID uint64 `json:"reactionId"`
	UserID     uint64 `json:"-"`
}
```

- [ ] **Step 2: Написать `internal/domains/reaction_test.go` (падающие тесты)**

```go
package domains_test

import (
	"encoding/json"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/stretchr/testify/assert"
)

func TestReaction_JSON(t *testing.T) {
	r := domains.Reaction{ID: 1, Emoji: "👍"}

	data, err := json.Marshal(r)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":1,"emoji":"👍"}`, string(data))
}

func TestMessageReactionSummary_JSON(t *testing.T) {
	s := domains.MessageReactionSummary{
		Reaction: domains.Reaction{ID: 1, Emoji: "👍"},
		UserIDs:  []uint64{10, 20},
	}

	data, err := json.Marshal(s)
	assert.NoError(t, err)
	assert.JSONEq(
		t,
		`{"reaction":{"id":1,"emoji":"👍"},"userIds":[10,20]}`,
		string(data),
	)
}

func TestMessageReactionDTO_JSONDecoding(t *testing.T) {
	// MessageID и UserID проставляются из URL/JWT и не сериализуются
	body := `{"reactionId":5}`

	var dto domains.MessageReactionDTO
	err := json.Unmarshal([]byte(body), &dto)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), dto.ReactionID)
	assert.Equal(t, uint64(0), dto.MessageID)
	assert.Equal(t, uint64(0), dto.UserID)
}
```

- [ ] **Step 3: Запустить — должны упасть на отсутствии типов**

Run: `go test ./internal/domains/... -run TestReaction -run TestMessageReaction -v`
Expected: FAIL (undefined types) — если файл `reaction.go` ещё не был создан. Если создан на Step 1 — тесты должны пройти.

- [ ] **Step 4: Расширить `internal/domains/message.go`**

Добавить в структуру `Message` (после существующего `ReplyToMessage`):

```go
Reactions []MessageReactionSummary `json:"reactions,omitempty"`
```

- [ ] **Step 5: Добавить sentinel-ошибки — создать `internal/errors/reactions.go`**

```go
package errors

import "errors"

var (
	ErrReactionAlreadyExists = errors.New("reaction already exists on this message for this user")
	ErrReactionNotFound      = errors.New("reaction not found in dictionary")
)
```

- [ ] **Step 6: Расширить `internal/domains/ws_event.go`**

Добавить константы после существующих `WSEventMessage*`:

```go
WSEventReactionAdded   WSEventType = "reaction_added"
WSEventReactionRemoved WSEventType = "reaction_removed"
```

И новые payload-структуры в конец файла:

```go
type ReactionAddedPayload struct {
	MessageID  uint64 `json:"messageId"`
	ChatID     uint64 `json:"chatId"`
	UserID     uint64 `json:"userId"`
	ReactionID uint64 `json:"reactionId"`
	Emoji      string `json:"emoji"`
}

type ReactionRemovedPayload struct {
	MessageID  uint64 `json:"messageId"`
	ChatID     uint64 `json:"chatId"`
	UserID     uint64 `json:"userId"`
	ReactionID uint64 `json:"reactionId"`
}
```

- [ ] **Step 7: Запустить все доменные тесты**

Run: `go test ./internal/domains/... -v`
Expected: PASS. Ничего в существующих тестах не должно сломаться (`Reactions` — опциональное поле).

- [ ] **Step 8: Обновить `internal/domains/doc.md` и `internal/errors/doc.md`**

Кратко описать новые типы и ошибки.

- [ ] **Step 9: Commit**

```bash
git add internal/domains/reaction.go internal/domains/reaction_test.go \
        internal/domains/message.go internal/domains/ws_event.go \
        internal/errors/reactions.go \
        internal/domains/doc.md internal/errors/doc.md
git commit -m "feat: доменные типы и sentinel-ошибки для реакций"
```

---

## Task 3: ReactionsRepository — интерфейс, mockgen, реализация, тесты

**Files:**
- Modify: `internal/interfaces/repositories.go` (добавить `ReactionsRepository`, обновить `exclude_interfaces` в других директивах)
- Create: `internal/repositories/reactions/repository.go`
- Create: `internal/repositories/reactions/repository_test.go`
- Create: `internal/repositories/reactions/trace_decorator.go`
- Create: `internal/repositories/reactions/trace_decorator_test.go`
- Create: `internal/repositories/reactions/doc.md`

**Interfaces:**
- Consumes: `pg.Transaction`, `logging.Logger`, `domains.MessageReactionDTO`, `domains.Reaction`, `domains.MessageReactionSummary`, sentinels из Task 2.
- Produces:
  ```go
  type ReactionsRepository interface {
      ListReactions(ctx context.Context) ([]domains.Reaction, error)
      GetReactionByID(ctx context.Context, id uint64) (*domains.Reaction, error)
      AddMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) error
      RemoveMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) (bool, error)
      ListReactionsForMessages(ctx context.Context, messageIDs []uint64) (map[uint64][]domains.MessageReactionSummary, error)
  }
  ```
  Функциональный конструктор: `reactionsrepository.New(tx pg.Transaction, logger logging.Logger) *Repository`.

- [ ] **Step 1: Добавить интерфейс в `internal/interfaces/repositories.go`**

Дополнить `exclude_interfaces` у **всех** существующих mockgen-директив словом `ReactionsRepository` (иначе моки других репо начнут его тянуть). Затем добавить в конец файла:

```go
//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/reactions_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository,FileStorageRepository
type ReactionsRepository interface {
	ListReactions(ctx context.Context) ([]domains.Reaction, error)
	GetReactionByID(ctx context.Context, id uint64) (*domains.Reaction, error)
	AddMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) error
	RemoveMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) (deleted bool, err error)
	ListReactionsForMessages(
		ctx context.Context,
		messageIDs []uint64,
	) (map[uint64][]domains.MessageReactionSummary, error)
}
```

- [ ] **Step 2: Сгенерировать моки**

Run: `go generate ./internal/interfaces/...`
Expected: создан `mocks/repositories/reactions_repository.go`.

- [ ] **Step 3: Создать `internal/repositories/reactions/repository.go`**

Скелет пакета — паттерн взят из `internal/repositories/messages/repository.go`:

```go
package reactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/logging"
	sq "github.com/Masterminds/squirrel"
)

const (
	reactionsTableName        = "reactions"
	messageReactionsTableName = "message_reactions"

	idColumnName         = "id"
	emojiColumnName      = "emoji"
	sortOrderColumnName  = "sort_order"
	messageIDColumnName  = "message_id"
	userIDColumnName     = "user_id"
	reactionIDColumnName = "reaction_id"
	createdAtColumnName  = "created_at"

	returningIDSuffix = "RETURNING id"
	asc               = "ASC"
)

type Repository struct {
	tx     pg.Transaction
	logger logging.Logger
}

func New(tx pg.Transaction, logger logging.Logger) *Repository {
	return &Repository{tx: tx, logger: logger}
}

func (r *Repository) ListReactions(ctx context.Context) ([]domains.Reaction, error) {
	stmt, params, err := sq.
		Select(idColumnName, emojiColumnName).
		From(reactionsTableName).
		OrderBy(fmt.Sprintf("%s %s", sortOrderColumnName, asc)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.tx.QueryContext(ctx, stmt, params...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			logging.LogErrorContext(ctx, r.logger, "Failed to close SQL rows", err)
		}
	}()

	var reactions []domains.Reaction
	for rows.Next() {
		var re domains.Reaction
		if err = rows.Scan(&re.ID, &re.Emoji); err != nil {
			return nil, err
		}
		reactions = append(reactions, re)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return reactions, nil
}

func (r *Repository) GetReactionByID(
	ctx context.Context,
	id uint64,
) (*domains.Reaction, error) {
	stmt, params, err := sq.
		Select(idColumnName, emojiColumnName).
		From(reactionsTableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	var re domains.Reaction
	if err = r.tx.QueryRowContext(ctx, stmt, params...).Scan(&re.ID, &re.Emoji); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, customerrors.ErrReactionNotFound
		}
		return nil, err
	}
	return &re, nil
}

func (r *Repository) AddMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	stmt, params, err := sq.
		Insert(messageReactionsTableName).
		Columns(messageIDColumnName, userIDColumnName, reactionIDColumnName).
		Values(dto.MessageID, dto.UserID, dto.ReactionID).
		Suffix(fmt.Sprintf(
			"ON CONFLICT (%s, %s, %s) DO NOTHING %s",
			messageIDColumnName, userIDColumnName, reactionIDColumnName,
			returningIDSuffix,
		)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	var id uint64
	if err = r.tx.QueryRowContext(ctx, stmt, params...).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return customerrors.ErrReactionAlreadyExists
		}
		return err
	}
	return nil
}

func (r *Repository) RemoveMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) (bool, error) {
	stmt, params, err := sq.
		Delete(messageReactionsTableName).
		Where(sq.Eq{
			messageIDColumnName:  dto.MessageID,
			userIDColumnName:     dto.UserID,
			reactionIDColumnName: dto.ReactionID,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return false, err
	}

	res, err := r.tx.ExecContext(ctx, stmt, params...)
	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *Repository) ListReactionsForMessages(
	ctx context.Context,
	messageIDs []uint64,
) (map[uint64][]domains.MessageReactionSummary, error) {
	if len(messageIDs) == 0 {
		return map[uint64][]domains.MessageReactionSummary{}, nil
	}

	stmt, params, err := sq.
		Select(
			fmt.Sprintf("%s.%s", messageReactionsTableName, messageIDColumnName),
			fmt.Sprintf("%s.%s", messageReactionsTableName, reactionIDColumnName),
			fmt.Sprintf("%s.%s", reactionsTableName, emojiColumnName),
			fmt.Sprintf("%s.%s", messageReactionsTableName, userIDColumnName),
		).
		From(messageReactionsTableName).
		Join(fmt.Sprintf(
			"%s ON %s.%s = %s.%s",
			reactionsTableName,
			reactionsTableName, idColumnName,
			messageReactionsTableName, reactionIDColumnName,
		)).
		Where(sq.Eq{
			fmt.Sprintf("%s.%s", messageReactionsTableName, messageIDColumnName): messageIDs,
		}).
		OrderBy(
			fmt.Sprintf("%s.%s %s", messageReactionsTableName, messageIDColumnName, asc),
			fmt.Sprintf("%s.%s %s", reactionsTableName, sortOrderColumnName, asc),
			fmt.Sprintf("%s.%s %s", messageReactionsTableName, createdAtColumnName, asc),
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.tx.QueryContext(ctx, stmt, params...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			logging.LogErrorContext(ctx, r.logger, "Failed to close SQL rows", err)
		}
	}()

	// map[messageID]map[reactionID]*MessageReactionSummary — для агрегации по одной реакции
	agg := make(map[uint64]map[uint64]*domains.MessageReactionSummary)
	// keepOrder[messageID] — порядок появления reactionID (сохраняем ORDER BY sort_order)
	keepOrder := make(map[uint64][]uint64)

	for rows.Next() {
		var (
			msgID, reactionID, userID uint64
			emoji                     string
		)
		if err = rows.Scan(&msgID, &reactionID, &emoji, &userID); err != nil {
			return nil, err
		}

		byReaction, ok := agg[msgID]
		if !ok {
			byReaction = make(map[uint64]*domains.MessageReactionSummary)
			agg[msgID] = byReaction
		}

		summary, ok := byReaction[reactionID]
		if !ok {
			summary = &domains.MessageReactionSummary{
				Reaction: domains.Reaction{ID: reactionID, Emoji: emoji},
				UserIDs:  []uint64{},
			}
			byReaction[reactionID] = summary
			keepOrder[msgID] = append(keepOrder[msgID], reactionID)
		}
		summary.UserIDs = append(summary.UserIDs, userID)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	result := make(map[uint64][]domains.MessageReactionSummary, len(agg))
	for msgID, order := range keepOrder {
		summaries := make([]domains.MessageReactionSummary, 0, len(order))
		for _, rid := range order {
			summaries = append(summaries, *agg[msgID][rid])
		}
		result[msgID] = summaries
	}
	return result, nil
}
```

- [ ] **Step 4: Написать unit-тесты `internal/repositories/reactions/repository_test.go`**

Пример на `AddMessageReaction` (успех + дубликат) через `github.com/DATA-DOG/go-sqlmock`. Паттерн подсмотреть в `internal/repositories/messages/repository_test.go` (если он использует `sqlmock` — использовать тот же подход; если это интеграционный тест с реальным Postgres под тегом `integration` — писать в том же стиле).

**ВАЖНО:** уже существующий `messages/repository_test.go` идёт под тегом `//go:build integration` и использует реальный Postgres через `pg.Transaction`. Значит и здесь пишем интеграционные тесты в том же стиле.

Ключевые кейсы (по одному тестовому методу на каждый):

1. `TestListReactions_ReturnsSeed` — после миграции `ListReactions` возвращает 8 seed'ов в порядке `sort_order ASC`, первый — `👍`.
2. `TestGetReactionByID_Found` — по ID из seed'а возвращается `Reaction`.
3. `TestGetReactionByID_NotFound` — по несуществующему ID возвращается `customerrors.ErrReactionNotFound`.
4. `TestAddMessageReaction_Success` — вставка новой реакции возвращает `nil`, строка появляется в БД.
5. `TestAddMessageReaction_Duplicate_ReturnsErrReactionAlreadyExists` — повторный `Add` того же triple возвращает `customerrors.ErrReactionAlreadyExists`.
6. `TestRemoveMessageReaction_ExistingRow_ReturnsTrue` — удаление существующего triple возвращает `(true, nil)`.
7. `TestRemoveMessageReaction_MissingRow_ReturnsFalse` — удаление того, чего нет, возвращает `(false, nil)`.
8. `TestListReactionsForMessages_EmptyInput_ReturnsEmptyMap` — на пустой список ID возвращается пустая мапа без запроса в БД.
9. `TestListReactionsForMessages_AggregatesByMessageAndReaction` — вставить 2 сообщения, 3 юзера, разные реакции; проверить, что `map[msgID]` содержит правильные группы, `UserIDs` набраны, порядок — по `sort_order` из справочника.

Полный код теста #5 как эталон:

```go
func (s *RepositoryTestSuite) TestAddMessageReaction_Duplicate_ReturnsErrReactionAlreadyExists() {
	// SetupTest уже создал тестового юзера, сообщение и т.п. — либо создаём здесь через
	// прямые SQL INSERT'ы в s.tx (по паттерну других тестов).
	// Предполагается: userID=1, messageID=1 существуют, seed справочника реакций уже применён.
	dto := domains.MessageReactionDTO{
		MessageID:  1,
		ReactionID: 1, // '👍' из seed'а
		UserID:     1,
	}

	err := s.repository.AddMessageReaction(s.ctx, dto)
	s.NoError(err)

	err = s.repository.AddMessageReaction(s.ctx, dto)
	s.ErrorIs(err, customerrors.ErrReactionAlreadyExists)
}
```

Для тестов, требующих предподготовку данных (юзер, чат, сообщение), заведи helper в `SetupTest`, аналогично тому, как это сделано в `messages/repository_test.go`.

- [ ] **Step 5: Прогнать тесты**

Run: `go test -tags integration ./internal/repositories/reactions/... -v`
Expected: PASS. Если какие-то тесты падают — исправить репо/тест, а не костыли.

- [ ] **Step 6: Создать `internal/repositories/reactions/trace_decorator.go`**

Скопировать паттерн с `internal/repositories/messages/trace_decorator.go`: декоратор оборачивает каждый публичный метод в span. Полный код одного метода как эталон:

```go
package reactions

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.ReactionsRepository
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.ReactionsRepository,
) *TraceDecorator {
	return &TraceDecorator{traceProvider: traceProvider, spanConfig: spanConfig, base: base}
}

func (d *TraceDecorator) ListReactions(ctx context.Context) ([]domains.Reaction, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.ListReactions(ctx)
}

// GetReactionByID, AddMessageReaction, RemoveMessageReaction, ListReactionsForMessages
// — по тому же шаблону: span + start/end events + делегирование в d.base.
```

Написать все 5 методов.

- [ ] **Step 7: Написать `internal/repositories/reactions/trace_decorator_test.go`**

Паттерн — как в `internal/repositories/messages/trace_decorator_test.go`. Использовать моки `interfaces.ReactionsRepository` (сгенерированные на Step 2) и мок `tracing.Provider`. Проверять, что каждый метод вызывает `d.base.<Method>` с теми же аргументами и возвращает тот же результат. По одному тесту на метод (5 тестов).

- [ ] **Step 8: Прогнать все тесты пакета**

Run: `go test -tags integration ./internal/repositories/reactions/... -v`
Expected: PASS для всех.

Также unit-тесты (декоратор):
Run: `go test ./internal/repositories/reactions/... -v`
Expected: PASS.

- [ ] **Step 9: Создать `internal/repositories/reactions/doc.md`**

Кратко: назначение пакета, публичные методы, семантика ошибок.

Также обновить `internal/repositories/doc.md` — добавить строку про новый пакет.

- [ ] **Step 10: Commit**

```bash
git add internal/interfaces/repositories.go \
        mocks/repositories/reactions_repository.go \
        internal/repositories/reactions/ \
        internal/repositories/doc.md
git commit -m "feat: репозиторий реакций (dict + m2m)"
```

---

## Task 4: ReactionsService — интерфейс, реализация, тесты, trace-decorator

**Files:**
- Modify: `internal/interfaces/services.go` (добавить `ReactionsService`, обновить `exclude_interfaces` в других директивах)
- Create: `internal/services/reactions/service.go`
- Create: `internal/services/reactions/service_test.go`
- Create: `internal/services/reactions/trace_decorator.go`
- Create: `internal/services/reactions/trace_decorator_test.go`
- Create: `internal/services/reactions/doc.md`

**Interfaces:**
- Consumes: `interfaces.UnitOfWork`, `interfaces.ReactionsRepository` (через фабрику `func(tx pg.Transaction) interfaces.ReactionsRepository`).
- Produces:
  ```go
  type ReactionsService interface {
      ListReactions(ctx context.Context) ([]domains.Reaction, error)
      GetReactionByID(ctx context.Context, id uint64) (*domains.Reaction, error)
      AddMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) error
      RemoveMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) (bool, error)
      ListReactionsForMessages(ctx context.Context, messageIDs []uint64) (map[uint64][]domains.MessageReactionSummary, error)
  }
  ```

- [ ] **Step 1: Добавить интерфейс в `internal/interfaces/services.go`**

Правило то же, что в Task 3 Step 1: обновить `exclude_interfaces` во всех существующих директивах, добавив `ReactionsService`, затем добавить в конец:

```go
//go:generate mockgen -source=services.go -destination=../../mocks/services/reactions_service.go -package=mockservices -exclude_interfaces=UsersService,AuthService,ChatsService,MessagesService,NotificationsService,SettingsService,WebPushSubscriptionsService,FileStorageService
type ReactionsService interface {
	ListReactions(ctx context.Context) ([]domains.Reaction, error)
	GetReactionByID(ctx context.Context, id uint64) (*domains.Reaction, error)
	AddMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) error
	RemoveMessageReaction(ctx context.Context, dto domains.MessageReactionDTO) (bool, error)
	ListReactionsForMessages(
		ctx context.Context,
		messageIDs []uint64,
	) (map[uint64][]domains.MessageReactionSummary, error)
}
```

- [ ] **Step 2: Сгенерировать моки**

Run: `go generate ./internal/interfaces/...`
Expected: `mocks/services/reactions_service.go` создан.

- [ ] **Step 3: Создать `internal/services/reactions/service.go`**

Паттерн — `internal/services/messages/service.go`. Каждая операция — в `s.uow.Do(...)`.

```go
package reactions

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                        interfaces.UnitOfWork
	newReactionsRepositoryFunc func(tx pg.Transaction) interfaces.ReactionsRepository
}

func New(
	uow interfaces.UnitOfWork,
	newReactionsRepositoryFunc func(tx pg.Transaction) interfaces.ReactionsRepository,
) *Service {
	return &Service{
		uow:                        uow,
		newReactionsRepositoryFunc: newReactionsRepositoryFunc,
	}
}

func (s *Service) ListReactions(ctx context.Context) ([]domains.Reaction, error) {
	var (
		reactions []domains.Reaction
		err       error
	)
	err = s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		reactions, err = repo.ListReactions(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return reactions, nil
}

func (s *Service) GetReactionByID(
	ctx context.Context,
	id uint64,
) (*domains.Reaction, error) {
	var (
		reaction *domains.Reaction
		err      error
	)
	err = s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		reaction, err = repo.GetReactionByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return reaction, nil
}

func (s *Service) AddMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	return s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		return repo.AddMessageReaction(ctx, dto)
	})
}

func (s *Service) RemoveMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) (bool, error) {
	var (
		deleted bool
		err     error
	)
	err = s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		deleted, err = repo.RemoveMessageReaction(ctx, dto)
		return err
	})
	if err != nil {
		return false, err
	}
	return deleted, nil
}

func (s *Service) ListReactionsForMessages(
	ctx context.Context,
	messageIDs []uint64,
) (map[uint64][]domains.MessageReactionSummary, error) {
	var (
		result map[uint64][]domains.MessageReactionSummary
		err    error
	)
	err = s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		result, err = repo.ListReactionsForMessages(ctx, messageIDs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
```

- [ ] **Step 4: Написать `internal/services/reactions/service_test.go`**

Паттерн — `internal/services/messages/service_test.go`. Мокаем `interfaces.UnitOfWork` и `interfaces.ReactionsRepository` (через фабричную функцию). По одному тесту на метод. Эталон:

```go
func TestService_AddMessageReaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	uow := mockinterfaces.NewMockUnitOfWork(ctrl)
	repo := mockrepositories.NewMockReactionsRepository(ctrl)

	factory := func(tx pg.Transaction) interfaces.ReactionsRepository { return repo }
	svc := reactions.New(uow, factory)

	ctx := context.Background()
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}

	uow.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(ctx context.Context, tx pg.Transaction) error) error {
			return fn(ctx, nil)
		})
	repo.EXPECT().AddMessageReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, svc.AddMessageReaction(ctx, dto))
}
```

Аналогичные тесты для остальных 4 методов + happy path и проброс ошибки.

- [ ] **Step 5: Прогнать тесты**

Run: `go test ./internal/services/reactions/... -v`
Expected: PASS.

- [ ] **Step 6: Создать `internal/services/reactions/trace_decorator.go`**

Паттерн — `internal/services/messages/trace_decorator.go` (посмотреть). Аналогично Task 3 Step 6, но реализуем `interfaces.ReactionsService`. 5 методов, каждый — span + start/end events + делегирование.

- [ ] **Step 7: Написать `internal/services/reactions/trace_decorator_test.go`**

Паттерн — `internal/services/messages/trace_decorator_test.go`. По тесту на метод.

- [ ] **Step 8: Прогнать все тесты пакета**

Run: `go test ./internal/services/reactions/... -v`
Expected: PASS.

- [ ] **Step 9: Создать `internal/services/reactions/doc.md` и обновить `internal/services/doc.md`**

- [ ] **Step 10: Commit**

```bash
git add internal/interfaces/services.go \
        mocks/services/reactions_service.go \
        internal/services/reactions/ \
        internal/services/doc.md
git commit -m "feat: сервис реакций"
```

---

## Task 5: ReactionsUseCases — интерфейс, реализация с валидацией, AttachReactions, тесты

**Files:**
- Modify: `internal/interfaces/usecases.go` (добавить `ReactionsUseCases`, обновить `exclude_interfaces`)
- Create: `internal/usecases/reactions/usecases.go`
- Create: `internal/usecases/reactions/usecases_test.go`
- Create: `internal/usecases/reactions/trace_decorator.go`
- Create: `internal/usecases/reactions/trace_decorator_test.go`
- Create: `internal/usecases/reactions/doc.md`

**Interfaces:**
- Consumes: `interfaces.ReactionsService`, `interfaces.ChatsService`, `interfaces.MessagesService`, `interfaces.WSBroadcaster` (из Task 6 — если Task 6 идёт после, здесь пока `WSBroadcaster` уже есть с методами про сообщения; методы для реакций мы **добавим в Task 6**, но интерфейс объявляем сейчас, а в Broadcaster уже добавим методы к моменту вызова из этого пакета в тестах / DI).
- Produces:
  ```go
  type ReactionsUseCases interface {
      ListReactions(ctx context.Context) ([]domains.Reaction, error)
      AddReaction(ctx context.Context, dto domains.MessageReactionDTO) error
      RemoveReaction(ctx context.Context, dto domains.MessageReactionDTO) error
      AttachReactions(ctx context.Context, msgs []domains.Message) ([]domains.Message, error)
      AttachReaction(ctx context.Context, msg *domains.Message) (*domains.Message, error)
  }
  ```

**Порядок:** Task 6 (WSBroadcaster для реакций) должна быть **до** этой либо параллельно, но с точки зрения плана удобнее сначала завести всё в usecase, потом расширить broadcaster. Здесь мы объявляем зависимость от `interfaces.WSBroadcaster` — этот интерфейс уже существует, мы просто расширим его в Task 6 методами `BroadcastReactionAdded/BroadcastReactionRemoved`. Тесты этого таска мокают broadcaster.

- [ ] **Step 1: Расширить `interfaces.WSBroadcaster` (декларативно, реализация — в Task 6)**

В `internal/interfaces/controllers.go` добавить в интерфейс `WSBroadcaster`:

```go
BroadcastReactionAdded(
	ctx context.Context,
	chatID, messageID, userID, reactionID uint64,
	emoji string,
)
BroadcastReactionRemoved(
	ctx context.Context,
	chatID, messageID, userID, reactionID uint64,
)
```

- [ ] **Step 2: Регенерировать моки контроллеров**

Run: `go generate ./internal/interfaces/...`
Expected: обновлён `mocks/controllers/ws_broadcaster.go` с новыми методами.

- [ ] **Step 3: Добавить `ReactionsUseCases` в `internal/interfaces/usecases.go`**

Обновить `exclude_interfaces` во всех директивах usecases, добавив `ReactionsUseCases`. Затем:

```go
//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/reactions_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases,SettingsUseCases,WebPushSubscriptionsUseCases,FileStorageUseCases
type ReactionsUseCases interface {
	ListReactions(ctx context.Context) ([]domains.Reaction, error)
	AddReaction(ctx context.Context, dto domains.MessageReactionDTO) error
	RemoveReaction(ctx context.Context, dto domains.MessageReactionDTO) error
	AttachReactions(ctx context.Context, msgs []domains.Message) ([]domains.Message, error)
	AttachReaction(ctx context.Context, msg *domains.Message) (*domains.Message, error)
}
```

Run: `go generate ./internal/interfaces/...`
Expected: `mocks/usecases/reactions_usecases.go` создан.

- [ ] **Step 4: Создать `internal/usecases/reactions/usecases.go`**

```go
package reactions

import (
	"context"
	"errors"
	"slices"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	reactionsService interfaces.ReactionsService
	messagesService  interfaces.MessagesService
	chatsService     interfaces.ChatsService
	broadcaster      interfaces.WSBroadcaster
}

func New(
	reactionsService interfaces.ReactionsService,
	messagesService interfaces.MessagesService,
	chatsService interfaces.ChatsService,
	broadcaster interfaces.WSBroadcaster,
) *UseCases {
	return &UseCases{
		reactionsService: reactionsService,
		messagesService:  messagesService,
		chatsService:     chatsService,
		broadcaster:      broadcaster,
	}
}

func (u *UseCases) ListReactions(ctx context.Context) ([]domains.Reaction, error) {
	return u.reactionsService.ListReactions(ctx)
}

func (u *UseCases) AddReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
	if err != nil {
		return err
	}

	if err = u.ensureUserIsChatMember(ctx, message.ChatID, dto.UserID); err != nil {
		return err
	}

	reaction, err := u.reactionsService.GetReactionByID(ctx, dto.ReactionID)
	if err != nil {
		return err
	}

	if err = u.reactionsService.AddMessageReaction(ctx, dto); err != nil {
		return err
	}

	u.broadcaster.BroadcastReactionAdded(
		ctx,
		message.ChatID, dto.MessageID, dto.UserID, dto.ReactionID,
		reaction.Emoji,
	)
	return nil
}

func (u *UseCases) RemoveReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
	if err != nil {
		return err
	}

	if err = u.ensureUserIsChatMember(ctx, message.ChatID, dto.UserID); err != nil {
		return err
	}

	deleted, err := u.reactionsService.RemoveMessageReaction(ctx, dto)
	if err != nil {
		return err
	}

	if deleted {
		u.broadcaster.BroadcastReactionRemoved(
			ctx,
			message.ChatID, dto.MessageID, dto.UserID, dto.ReactionID,
		)
	}
	return nil
}

func (u *UseCases) AttachReactions(
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

	byMsg, err := u.reactionsService.ListReactionsForMessages(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range msgs {
		msgs[i].Reactions = byMsg[msgs[i].ID]
	}
	return msgs, nil
}

func (u *UseCases) AttachReaction(
	ctx context.Context,
	msg *domains.Message,
) (*domains.Message, error) {
	if msg == nil {
		return nil, nil
	}
	byMsg, err := u.reactionsService.ListReactionsForMessages(ctx, []uint64{msg.ID})
	if err != nil {
		return nil, err
	}
	msg.Reactions = byMsg[msg.ID]
	return msg, nil
}

func (u *UseCases) ensureUserIsChatMember(
	ctx context.Context,
	chatID, userID uint64,
) error {
	members, err := u.chatsService.GetChatMembers(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(members, func(m domains.User) bool { return m.ID == userID }) {
		return customerrors.ErrUserIsNotChatMember
	}
	return nil
}

// компилятор проверит, что UseCases реализует ReactionsUseCases
var _ interfaces.ReactionsUseCases = (*UseCases)(nil)

// компилятор считает "errors" использованным
var _ = errors.Is
```

Убрать `errors` из импорта, если не используется в реальном коде.

- [ ] **Step 5: Написать `internal/usecases/reactions/usecases_test.go`**

Паттерн — `internal/usecases/messages/usecases_test.go`. Обязательные кейсы:

1. `TestAddReaction_HappyPath` — все моки OK → broadcaster.EXPECT().BroadcastReactionAdded вызван с правильными аргументами.
2. `TestAddReaction_MessageNotFound` — `messagesService.GetMessageByID` возвращает ошибку → пробрасывается наверх; broadcast НЕ вызывается.
3. `TestAddReaction_UserNotChatMember` — `chatsService.GetChatMembers` не содержит userID → `ErrUserIsNotChatMember`; broadcast НЕ вызывается.
4. `TestAddReaction_UnknownReaction` — `reactionsService.GetReactionByID` возвращает `ErrReactionNotFound` → пробрасывается; broadcast НЕ вызывается.
5. `TestAddReaction_Duplicate` — `reactionsService.AddMessageReaction` возвращает `ErrReactionAlreadyExists` → пробрасывается; broadcast НЕ вызывается.
6. `TestRemoveReaction_Deleted_BroadcastsEvent` — `RemoveMessageReaction` возвращает `(true, nil)` → broadcaster вызван.
7. `TestRemoveReaction_NothingDeleted_NoBroadcast` — `RemoveMessageReaction` возвращает `(false, nil)` → broadcaster НЕ вызван, юзкейс возвращает `nil`.
8. `TestRemoveReaction_UserNotChatMember` — 403-путь.
9. `TestAttachReactions_EmptyInput_NoServiceCall` — на пустом слайсе `reactionsService.ListReactionsForMessages` НЕ вызывается.
10. `TestAttachReactions_MapsReactionsByMessageID` — сервис возвращает мапу, реакции разложены по правильным сообщениям.

Эталон для #1 (полный код):

```go
func TestAddReaction_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	msgs := mockservices.NewMockMessagesService(ctrl)
	chats := mockservices.NewMockChatsService(ctrl)
	reacts := mockservices.NewMockReactionsService(ctrl)
	broadcaster := mockcontrollers.NewMockWSBroadcaster(ctrl)

	uc := reactions.New(reacts, msgs, chats, broadcaster)
	ctx := context.Background()
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	msgs.EXPECT().
		GetMessageByID(ctx, uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	chats.EXPECT().
		GetChatMembers(ctx, uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}, {ID: 8}}, nil)
	reacts.EXPECT().
		GetReactionByID(ctx, uint64(1)).
		Return(&domains.Reaction{ID: 1, Emoji: "👍"}, nil)
	reacts.EXPECT().
		AddMessageReaction(ctx, dto).
		Return(nil)
	broadcaster.EXPECT().
		BroadcastReactionAdded(ctx, uint64(42), uint64(10), uint64(7), uint64(1), "👍")

	assert.NoError(t, uc.AddReaction(ctx, dto))
}
```

- [ ] **Step 6: Прогнать тесты**

Run: `go test ./internal/usecases/reactions/... -v`
Expected: PASS.

- [ ] **Step 7: Создать `internal/usecases/reactions/trace_decorator.go` и `trace_decorator_test.go`**

Паттерн — `internal/usecases/messages/trace_decorator.go`. 5 методов (`ListReactions`, `AddReaction`, `RemoveReaction`, `AttachReactions`, `AttachReaction`), каждый — span + start/end + делегирование.

Тесты — по одному на метод, паттерн — `internal/usecases/messages/trace_decorator_test.go`.

- [ ] **Step 8: Прогнать все тесты пакета**

Run: `go test ./internal/usecases/reactions/... -v`
Expected: PASS.

- [ ] **Step 9: doc.md**

Создать `internal/usecases/reactions/doc.md`, обновить `internal/usecases/doc.md`.

- [ ] **Step 10: Commit**

```bash
git add internal/interfaces/controllers.go internal/interfaces/usecases.go \
        mocks/controllers/ws_broadcaster.go mocks/usecases/reactions_usecases.go \
        internal/usecases/reactions/ internal/usecases/doc.md
git commit -m "feat: usecase реакций с валидацией и AttachReactions"
```

---

## Task 6: WS-broadcaster для реакций

**Files:**
- Modify: `internal/controllers/http/handlers/api/ws/ws.go`
- Modify: `internal/controllers/http/handlers/api/ws/doc.md`

**Interfaces:**
- Consumes: `chatsUseCases.GetChatMembers`, существующий `sendToUser`.
- Produces: методы `BroadcastReactionAdded`, `BroadcastReactionRemoved` на `*ws.Handler`, реализующие расширенный интерфейс `interfaces.WSBroadcaster`.

- [ ] **Step 1: Добавить методы в `*ws.Handler`**

По образцу существующих `BroadcastMessageDeleted` / `BroadcastMessageEdited`. В конец файла:

```go
// BroadcastReactionAdded sends a reaction_added event to all chat members.
func (h *Handler) BroadcastReactionAdded(
	ctx context.Context,
	chatID, messageID, userID, reactionID uint64,
	emoji string,
) {
	chatMembers, err := h.chatsUseCases.GetChatMembers(ctx, chatID, userID)
	if err != nil {
		logging.LogErrorContext(
			ctx, h.logger,
			"Failed to get chat members for reaction added broadcast", err,
			"ChatID", chatID, "MessageID", messageID, "ReactionID", reactionID,
		)
		return
	}

	event := domains.WSEvent{
		Type: domains.WSEventReactionAdded,
		Payload: domains.ReactionAddedPayload{
			MessageID:  messageID,
			ChatID:     chatID,
			UserID:     userID,
			ReactionID: reactionID,
			Emoji:      emoji,
		},
	}
	for _, member := range chatMembers {
		h.sendToUser(ctx, member.ID, event)
	}
}

// BroadcastReactionRemoved sends a reaction_removed event to all chat members.
func (h *Handler) BroadcastReactionRemoved(
	ctx context.Context,
	chatID, messageID, userID, reactionID uint64,
) {
	chatMembers, err := h.chatsUseCases.GetChatMembers(ctx, chatID, userID)
	if err != nil {
		logging.LogErrorContext(
			ctx, h.logger,
			"Failed to get chat members for reaction removed broadcast", err,
			"ChatID", chatID, "MessageID", messageID, "ReactionID", reactionID,
		)
		return
	}

	event := domains.WSEvent{
		Type: domains.WSEventReactionRemoved,
		Payload: domains.ReactionRemovedPayload{
			MessageID:  messageID,
			ChatID:     chatID,
			UserID:     userID,
			ReactionID: reactionID,
		},
	}
	for _, member := range chatMembers {
		h.sendToUser(ctx, member.ID, event)
	}
}
```

- [ ] **Step 2: Проверить, что реализация покрывает интерфейс**

Run: `go build ./...`
Expected: успешно. Если типы разошлись — фикс.

- [ ] **Step 3: Расширить существующий `controller_test.go` (или новый файл) — тест на broadcast**

Паттерн — тесты для `BroadcastMessageEdited` (если есть). По одному тесту:
- `TestBroadcastReactionAdded_SendsToAllMembers` — мок `chatsUseCases.GetChatMembers` возвращает 2 юзеров, проверяем что `sendToUser` вызвана дважды (или использовать внутренний тестовый способ проверки WS-очереди — как принято в проекте).

Если существующие broadcast-тесты используют не unit-mocks, а имитацию отправки через тестовый WS-conn — использовать тот же подход.

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/controllers/http/handlers/api/ws/... -v`
Expected: PASS.

- [ ] **Step 5: Обновить `doc.md`**

- [ ] **Step 6: Commit**

```bash
git add internal/controllers/http/handlers/api/ws/
git commit -m "feat: WS broadcast для reaction_added/reaction_removed"
```

---

## Task 7: HTTP schemas и mapper для реакций

**Files:**
- Create: `internal/controllers/http/schemas/reactions.go`
- Create: `internal/controllers/http/mappers/reactions/reactions.go`
- Create: `internal/controllers/http/mappers/reactions/reactions_test.go`
- Modify: `internal/controllers/http/schemas/messages.go` (добавить `Reactions` в `Message`)
- Modify: `internal/controllers/http/mappers/messages/messages.go` (маппить `Reactions`)
- Modify: `internal/controllers/http/mappers/messages/messages_test.go` (кейс с реакциями)

**Interfaces:**
- Produces: `schemas.Reaction`, `schemas.MessageReaction`, `schemas.SetReactionInput`, `schemas.Message.Reactions`; `mappers/reactions.MapReaction`, `MapReactions`, `MapMessageReactions`.

- [ ] **Step 1: Создать `internal/controllers/http/schemas/reactions.go`**

```go
package schemas

// Reaction represents an emoji reaction available in the dictionary.
// swagger:model
type Reaction struct {
	// Unique identifier of the reaction.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`

	// Emoji character to display.
	// required: true
	// nullable: false
	// example: 👍
	Emoji string `json:"emoji"`
}

// MessageReaction represents an aggregated reaction on a message:
// the reaction itself and the list of user IDs who set it.
// swagger:model
type MessageReaction struct {
	// The reaction (id + emoji).
	// required: true
	// nullable: false
	Reaction Reaction `json:"reaction"`

	// IDs of users who set this reaction.
	// required: true
	// nullable: false
	UserIDs []uint64 `json:"userIds"`
}

// SetReactionInput is the request body for POST /messages/{id}/reactions.
// swagger:parameters SetMessageReaction
type SetReactionInput struct {
	// Reaction ID from the dictionary.
	// in: body
	// required: true
	Body struct {
		ReactionID uint64 `json:"reactionId"`
	}
}
```

- [ ] **Step 2: Расширить `schemas.Message`**

В `internal/controllers/http/schemas/messages.go`, добавить в структуру `Message`:

```go
// Reactions set on the message, aggregated by reaction id.
// required: false
// nullable: true
Reactions []MessageReaction `json:"reactions,omitempty"`
```

- [ ] **Step 3: Создать `internal/controllers/http/mappers/reactions/reactions.go`**

```go
package reactions

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapReaction(r domains.Reaction) schemas.Reaction {
	return schemas.Reaction{ID: r.ID, Emoji: r.Emoji}
}

func MapReactions(rs []domains.Reaction) []schemas.Reaction {
	result := make([]schemas.Reaction, len(rs))
	for i := range rs {
		result[i] = MapReaction(rs[i])
	}
	return result
}

func MapMessageReaction(s domains.MessageReactionSummary) schemas.MessageReaction {
	userIDs := make([]uint64, len(s.UserIDs))
	copy(userIDs, s.UserIDs)
	return schemas.MessageReaction{
		Reaction: MapReaction(s.Reaction),
		UserIDs:  userIDs,
	}
}

func MapMessageReactions(ss []domains.MessageReactionSummary) []schemas.MessageReaction {
	if len(ss) == 0 {
		return nil
	}
	result := make([]schemas.MessageReaction, len(ss))
	for i := range ss {
		result[i] = MapMessageReaction(ss[i])
	}
	return result
}
```

- [ ] **Step 4: Прошить в `mappers/messages/messages.go`**

В функцию `MapMessage`, перед `return mapped`, добавить:

```go
mapped.Reactions = reactionsmapper.MapMessageReactions(message.Reactions)
```

Импорт (алиас, чтобы избежать коллизии имён):

```go
reactionsmapper "github.com/DKhorkov/kfc/internal/controllers/http/mappers/reactions"
```

- [ ] **Step 5: Тесты `internal/controllers/http/mappers/reactions/reactions_test.go`**

Один тест на каждый маппер: `MapReaction`, `MapReactions` (порядок сохраняется), `MapMessageReaction`, `MapMessageReactions` (пустой вход → nil).

Эталон:

```go
func TestMapMessageReactions_EmptyInput_ReturnsNil(t *testing.T) {
	assert.Nil(t, reactionsmapper.MapMessageReactions(nil))
	assert.Nil(t, reactionsmapper.MapMessageReactions([]domains.MessageReactionSummary{}))
}

func TestMapMessageReaction_CopiesUserIDs(t *testing.T) {
	src := domains.MessageReactionSummary{
		Reaction: domains.Reaction{ID: 1, Emoji: "👍"},
		UserIDs:  []uint64{10, 20},
	}
	out := reactionsmapper.MapMessageReaction(src)
	assert.Equal(t, schemas.MessageReaction{
		Reaction: schemas.Reaction{ID: 1, Emoji: "👍"},
		UserIDs:  []uint64{10, 20},
	}, out)
}
```

- [ ] **Step 6: Обновить `mappers/messages/messages_test.go`**

Добавить кейс: `MapMessage` с непустым `Reactions` — результат содержит их в поле `Reactions`.

- [ ] **Step 7: Прогнать тесты**

Run: `go test ./internal/controllers/http/mappers/... -v`
Expected: PASS.

- [ ] **Step 8: doc.md**

Обновить `internal/controllers/http/schemas/doc.md` и `internal/controllers/http/mappers/doc.md`, добавить `internal/controllers/http/mappers/reactions/doc.md`.

- [ ] **Step 9: Commit**

```bash
git add internal/controllers/http/schemas/reactions.go \
        internal/controllers/http/schemas/messages.go \
        internal/controllers/http/mappers/reactions/ \
        internal/controllers/http/mappers/messages/messages.go \
        internal/controllers/http/mappers/messages/messages_test.go \
        internal/controllers/http/schemas/doc.md \
        internal/controllers/http/mappers/doc.md
git commit -m "feat: HTTP schemas и mapper для реакций"
```

---

## Task 8: HTTP handler GET /reactions

**Files:**
- Create: `internal/controllers/http/handlers/api/reactions/list/handler.go`
- Create: `internal/controllers/http/handlers/api/reactions/list/handler_test.go`
- Create: `internal/controllers/http/handlers/api/reactions/list/doc.md`
- Modify: `internal/controllers/http/handlers/api/setup.go` (константа URL + регистрация роута + добавить `reactionsUseCases` в аргументы `SetupHandlers`)

**Interfaces:**
- Consumes: `interfaces.ReactionsUseCases`.
- Produces: HTTP handler, отдающий `[]schemas.Reaction`.

- [ ] **Step 1: Handler**

```go
package list

import (
	"encoding/json"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	reactionsmapper "github.com/DKhorkov/kfc/internal/controllers/http/mappers/reactions"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

// swagger:route GET /api/reactions reactions ListReactions
//
// ListReactions
//
// Provides the dictionary of available emoji reactions for the reactions picker.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	200: []Reaction
//	401: Unauthorized
//	500: InternalServerError

// Handler returns the dictionary of available reactions.
func Handler(u interfaces.ReactionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reactions, err := u.ListReactions(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)
		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(reactionsmapper.MapReactions(reactions)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
```

- [ ] **Step 2: Handler test**

```go
func TestListReactionsHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	u := mockusecases.NewMockReactionsUseCases(ctrl)
	u.EXPECT().ListReactions(gomock.Any()).
		Return([]domains.Reaction{{ID: 1, Emoji: "👍"}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/reactions", nil)
	rec := httptest.NewRecorder()

	list.Handler(u).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[{"id":1,"emoji":"👍"}]`, rec.Body.String())
}

func TestListReactionsHandler_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	u := mockusecases.NewMockReactionsUseCases(ctrl)
	u.EXPECT().ListReactions(gomock.Any()).Return(nil, errors.New("boom"))

	req := httptest.NewRequest(http.MethodGet, "/api/reactions", nil)
	rec := httptest.NewRecorder()

	list.Handler(u).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
```

- [ ] **Step 3: Прогнать тесты**

Run: `go test ./internal/controllers/http/handlers/api/reactions/list/... -v`
Expected: PASS.

- [ ] **Step 4: Роутинг в `setup.go`**

Добавить константу:

```go
ReactionsURL          = "/reactions"
MessageReactionsURL   = MessagesURL + "/{%s}/reactions"
DeleteReactionURL     = MessageReactionsURL + "/{%s}"
```

Расширить сигнатуру `SetupHandlers`, добавив параметр `reactionsUseCases interfaces.ReactionsUseCases` (после `messagesUseCases`). Регистрация:

```go
getMux.Handle(ReactionsURL, reactions_list.Handler(reactionsUseCases))
```

Импорт (алиас — чтобы не путать с постом реакции):

```go
reactions_list "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/reactions/list"
```

Также обновить `RouteKey` для `{reactionId}` — использовать существующий `common.IDRouteKey` (уже есть) для messageID, и добавить в `common` новый ключ `ReactionIDRouteKey = "reactionId"`, либо использовать имя `reactionId` inline. Задокументировать выбор в `common/doc.md`.

Актуальный шаг: **добавить в `internal/controllers/http/handlers/common/` константу**:

```go
ReactionIDRouteKey = "reactionId"
```

- [ ] **Step 5: Проверить компиляцию**

Run: `go build ./...`
Expected: OK (в этот момент вызов `SetupHandlers` в `cmd/main.go` ещё не обновлён — Task 12 это исправит. Если Go ругается — временно разрулим через передачу `nil` в тесте `setup_test.go`, но это будет фикснуто по-настоящему в Task 12).

Если тесты `setup_test.go` компилируются с новой сигнатурой — обновить их (передать `nil` или мок).

- [ ] **Step 6: doc.md**

- [ ] **Step 7: Commit**

```bash
git add internal/controllers/http/handlers/api/reactions/list/ \
        internal/controllers/http/handlers/api/setup.go \
        internal/controllers/http/handlers/common/
git commit -m "feat: HTTP handler GET /reactions"
```

---

## Task 9: HTTP handler POST /messages/{id}/reactions

**Files:**
- Create: `internal/controllers/http/handlers/api/reactions/set/handler.go`
- Create: `internal/controllers/http/handlers/api/reactions/set/handler_test.go`
- Create: `internal/controllers/http/handlers/api/reactions/set/doc.md`
- Modify: `internal/controllers/http/handlers/api/setup.go` (роут)

- [ ] **Step 1: Handler**

```go
package set

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

// swagger:route POST /api/messages/{id}/reactions reactions SetMessageReaction
//
// SetMessageReaction
//
// Sets a reaction on a message for the current user.
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
//	409: Conflict
//	500: InternalServerError

// Handler sets reaction on a message.
func Handler(u interfaces.ReactionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		messageID, err := strconv.ParseUint(mux.Vars(r)[common.IDRouteKey], 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var dto domains.MessageReactionDTO
		if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		dto.MessageID = messageID
		dto.UserID = userID

		if dto.ReactionID == 0 {
			http.Error(w, "reactionId is required", http.StatusBadRequest)
			return
		}

		err = u.AddReaction(r.Context(), dto)
		switch {
		case errors.Is(err, customerrors.ErrReactionAlreadyExists):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case errors.Is(err, customerrors.ErrReactionNotFound),
			errors.Is(err, customerrors.ErrMessageNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, customerrors.ErrUserIsNotChatMember):
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

- [ ] **Step 2: Handler tests**

Кейсы: 204 happy path, 400 bad body / zero reactionId, 401 без userID, 403 not member, 404 message not found / reaction not found, 409 duplicate, 500 unexpected.

Один пример:

```go
func TestSetReactionHandler_Conflict_OnDuplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	u := mockusecases.NewMockReactionsUseCases(ctrl)

	u.EXPECT().
		AddReaction(gomock.Any(), gomock.Any()).
		Return(customerrors.ErrReactionAlreadyExists)

	body := `{"reactionId":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages/10/reactions", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{common.IDRouteKey: "10"})
	req = req.WithContext(contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, uint64(7)))

	rec := httptest.NewRecorder()
	set.Handler(u).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)
}
```

- [ ] **Step 3: Прогнать тесты**

Run: `go test ./internal/controllers/http/handlers/api/reactions/set/... -v`
Expected: PASS.

- [ ] **Step 4: Роут в `setup.go`**

В `postMux`:

```go
postMux.Handle(
	fmt.Sprintf(MessageReactionsURL, common.IDRouteKey),
	reactions_set.Handler(reactionsUseCases),
)
```

Импорт с алиасом.

- [ ] **Step 5: doc.md + commit**

```bash
git add internal/controllers/http/handlers/api/reactions/set/ \
        internal/controllers/http/handlers/api/setup.go
git commit -m "feat: HTTP handler POST /messages/{id}/reactions"
```

---

## Task 10: HTTP handler DELETE /messages/{id}/reactions/{reactionId}

**Files:**
- Create: `internal/controllers/http/handlers/api/reactions/unset/handler.go`
- Create: `internal/controllers/http/handlers/api/reactions/unset/handler_test.go`
- Create: `internal/controllers/http/handlers/api/reactions/unset/doc.md`
- Modify: `internal/controllers/http/handlers/api/setup.go` (роут)

- [ ] **Step 1: Handler**

```go
package unset

import (
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

// swagger:route DELETE /api/messages/{id}/reactions/{reactionId} reactions UnsetMessageReaction
//
// UnsetMessageReaction
//
// Removes a reaction from a message for the current user. Idempotent — 204 even if no such reaction.
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

// Handler removes reaction from a message.
func Handler(u interfaces.ReactionsUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		messageID, err := strconv.ParseUint(vars[common.IDRouteKey], 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		reactionID, err := strconv.ParseUint(vars[common.ReactionIDRouteKey], 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dto := domains.MessageReactionDTO{
			MessageID:  messageID,
			ReactionID: reactionID,
			UserID:     userID,
		}

		err = u.RemoveReaction(r.Context(), dto)
		switch {
		case errors.Is(err, customerrors.ErrMessageNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, customerrors.ErrUserIsNotChatMember):
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

- [ ] **Step 2: Handler tests**

Кейсы: 204 happy path (реакция удалена), 204 happy path (реакции не было — usecase вернул nil), 400 bad IDs, 401 без userID, 403 not member, 404 message not found, 500 unexpected.

- [ ] **Step 3: Прогнать тесты**

Run: `go test ./internal/controllers/http/handlers/api/reactions/unset/... -v`
Expected: PASS.

- [ ] **Step 4: Роут в `setup.go`**

В `deleteMux`:

```go
deleteMux.Handle(
	fmt.Sprintf(DeleteReactionURL, common.IDRouteKey, common.ReactionIDRouteKey),
	reactions_unset.Handler(reactionsUseCases),
)
```

- [ ] **Step 5: doc.md + commit**

```bash
git add internal/controllers/http/handlers/api/reactions/unset/ \
        internal/controllers/http/handlers/api/setup.go
git commit -m "feat: HTTP handler DELETE /messages/{id}/reactions/{reactionId}"
```

---

## Task 11: Интеграция AttachReactions в чтение сообщений

**Files:**
- Modify: `internal/usecases/messages/usecases.go`
- Modify: `internal/usecases/messages/usecases_test.go`
- Modify: `internal/usecases/messages/trace_decorator_test.go` (если сигнатуры конструктора изменились)
- Modify: `cmd/main.go` (передача `reactionsUseCases` в `messagesusecases.New`) — но реально это будет сделано на Task 12; здесь только правим usecases

**Interfaces:**
- Consumes: `interfaces.ReactionsUseCases`.
- Изменение конструктора: `messages.New(messagesService, chatsService, usersService, reactionsUseCases)`.

- [ ] **Step 1: Добавить зависимость в `messages.UseCases`**

```go
type UseCases struct {
	messagesService  interfaces.MessagesService
	usersService     interfaces.UsersService
	chatsService     interfaces.ChatsService
	reactionsUseCases interfaces.ReactionsUseCases
}

func New(
	messagesService interfaces.MessagesService,
	chatsService interfaces.ChatsService,
	usersService interfaces.UsersService,
	reactionsUseCases interfaces.ReactionsUseCases,
) *UseCases {
	return &UseCases{
		messagesService:  messagesService,
		chatsService:     chatsService,
		usersService:     usersService,
		reactionsUseCases: reactionsUseCases,
	}
}
```

- [ ] **Step 2: Обновить `GetChatMessages` и `GetMessageByID`**

```go
func (u *UseCases) GetChatMessages(
	ctx context.Context,
	userID uint64,
	chatID uint64,
	pagination *domains.Pagination,
) ([]domains.Message, error) {
	if _, err := u.usersService.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}
	chatMembers, err := u.chatsService.GetChatMembers(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(chatMembers, func(m domains.User) bool { return m.ID == userID }) {
		return nil, customerrors.ErrUserIsNotChatMember
	}

	msgs, err := u.messagesService.GetChatMessages(ctx, userID, chatID, pagination)
	if err != nil {
		return nil, err
	}

	return u.reactionsUseCases.AttachReactions(ctx, msgs)
}

func (u *UseCases) GetMessageByID(
	ctx context.Context,
	userID uint64,
	messageID uint64,
) (*domains.Message, error) {
	msg, err := u.messagesService.GetMessageByID(ctx, userID, messageID)
	if err != nil {
		return nil, err
	}
	return u.reactionsUseCases.AttachReaction(ctx, msg)
}
```

- [ ] **Step 3: Обновить существующие тесты**

Все тесты на `messages.New(...)` должны передавать 4-й аргумент — новый мок `mockusecases.NewMockReactionsUseCases`. Для тестов, где ветка до `AttachReactions` не доходит (ошибки валидации), мок можно ставить без `EXPECT()`.

Добавить новые тесты:
- `TestGetChatMessages_AttachesReactions` — успешный путь: сервис возвращает 2 сообщения, `reactionsUseCases.AttachReactions` — возвращает те же с проставленными `Reactions`.
- `TestGetMessageByID_AttachesReaction` — аналогично.

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/usecases/messages/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usecases/messages/
git commit -m "feat: подгрузка реакций в GetChatMessages/GetMessageByID"
```

---

## Task 12: DI в cmd/main.go и `SetupHandlers`

**Files:**
- Modify: `cmd/main.go`
- Modify: `internal/controllers/http/handlers/api/setup.go` (сигнатура + пробрасывание, если ещё не финализировали в Task 8)
- Modify: `internal/controllers/http/controller.go` (если оно вызывает `SetupHandlers`)
- Modify: `internal/config/*` — если конфиг трассинга нужен для реакций (`cfg.Tracing.Spans.Repositories.Reactions`, `Services.Reactions`, `UseCases.Reactions`) — надо добавить.

**Interfaces:**
- Финальная сборка графа зависимостей.

- [ ] **Step 1: Добавить конфиг спанов**

Найти, где определены `cfg.Tracing.Spans.Repositories.Messages` / `Services.Messages` / `UseCases.Messages`. Скопировать паттерн для `Reactions`. Файл — вероятно `internal/config/tracing.go` или подобное. Обновить YAML-конфиги, если они есть в `build/` (по паттерну существующих сервисов).

- [ ] **Step 2: Собрать `reactionsRepository` фабрику, `reactionsService`, `reactionsUseCases` в `cmd/main.go`**

```go
reactionsService := reactionsservice.NewTraceDecorator(
	traceProvider,
	cfg.Tracing.Spans.Services.Reactions,
	reactionsservice.New(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.ReactionsRepository {
			return reactionsrepository.NewTraceDecorator(
				traceProvider,
				cfg.Tracing.Spans.Repositories.Reactions,
				reactionsrepository.New(tx, logger),
			)
		},
	),
)

reactionsUseCases := reactionsusecases.NewTraceDecorator(
	traceProvider,
	cfg.Tracing.Spans.UseCases.Reactions,
	reactionsusecases.New(
		reactionsService,
		messagesService,
		chatsService,
		websocketHandler, // WSBroadcaster, объявляем после его создания
	),
)
```

**Проблема с порядком:** `websocketHandler` создаётся внутри `SetupHandlers` (см. `handlers/api/setup.go`). Для инъекции broadcaster'а в `reactionsUseCases`, нужно либо:
- **A.** Вынести создание `websocketHandler` до `SetupHandlers`, передавать его как параметр (сейчас там локально создаётся).
- **B.** Использовать паттерн "late binding" — создать пустую обёртку broadcaster'а, у которой поле-делегат ставится после `SetupHandlers`.

Смотреть, как решён этот circular-конструктор для существующих usecases сейчас (grep по `websocketHandler`). Если такого паттерна ещё нет — реализовать **вариант A**: создать `websocketHandler` в `cmd/main.go`, передать в `SetupHandlers`. Это требует правки `SetupHandlers`: добавить параметр `broadcaster interfaces.WSBroadcaster`.

Проверка: если `websocketHandler` уже создаётся только внутри `SetupHandlers` **и передаётся в другие usecases**, но `messagesUseCases` не имеет от него зависимости — значит цепочка не циклична и никаких proxy не нужно.

Реально: `reactionsUseCases` **зависит** от broadcaster. Если broadcaster создаётся внутри `SetupHandlers`, а `reactionsUseCases` создаётся снаружи и передаётся ВНУТРЬ, — цикл. Разрываем через **A**: broadcaster создаётся в `cmd/main.go` перед `reactionsUseCases`.

Дополнительная миграция: в `SetupHandlers` убрать создание `websocketHandler`, добавить параметр `websocketHandler *ws.Handler` (или `interfaces.WSBroadcaster + сам handler для регистрации роута /ws`). Понадобится и то, и другое — регистрация `/ws` требует handler'а с методом `Handle`, а `broadcaster` инъектится в usecase.

Обновить остальные вызовы броадкастеров в `handlers/api/messages/delete` и `update` — они получают `broadcaster` как параметр — сигнатуры не меняются.

- [ ] **Step 3: Обновить `messagesusecases.New` вызов**

Добавить 4-й параметр `reactionsUseCases`:

```go
messagesusecases.New(messagesService, chatsService, usersService, reactionsUseCases)
```

- [ ] **Step 4: Пробросить `reactionsUseCases` в `SetupHandlers`**

Расширенная сигнатура (после Task 8 это уже сделано):

```go
SetupHandlers(
	apiMux,
	cookiesConfig, natsConfig,
	usersUseCases, authUseCases, chatsUseCases,
	messagesUseCases, reactionsUseCases,
	settingsUseCases, webPushSubscriptionsUseCases, fileStorageUseCases,
	logger,
	upgrader, natsPublisher, vapidPublicKey,
	websocketHandler, // если решили вариант A
)
```

Обновить `setup_test.go`.

- [ ] **Step 5: Проверить сборку**

Run: `go build ./...`
Expected: OK.

Run: `go test ./... -short`
Expected: все существующие unit-тесты проходят.

- [ ] **Step 6: Прогнать всё**

Run: `go test ./...`
Expected: PASS.
Run: `task lint`
Expected: чисто.

- [ ] **Step 7: Обновить `docs/architecture.md` и `docs/modules.md`**

Добавить упоминание пакета `reactions` в таблицу модулей и в раздел про сообщения (реакции — часть модели сообщений).

- [ ] **Step 8: Обновить `cmd/doc.md`**

Кратко: добавили сборку графа реакций.

- [ ] **Step 9: Commit**

```bash
git add cmd/ internal/controllers/http/ internal/config/ docs/ \
        internal/usecases/messages/
git commit -m "feat: DI и wiring реакций в cmd/main.go и SetupHandlers"
```

---

## Task 13: E2E smoke — прогнать локально

**Files:** none (проверка)

- [ ] **Step 1: Поднять локальный стек**

Run: `task local`
Expected: PostgreSQL / Redis / NATS запускаются, приложение стартует, миграции применяются автоматически (или вручную `task migrate-up`).

- [ ] **Step 2: Проверить `GET /api/reactions`**

Через Swagger UI на `http://localhost:8080/docs` или curl:

```bash
curl -b "access_token=<токен>" http://localhost:8080/api/reactions
```

Expected: JSON-массив из 8 реакций (`👍❤️🔥💯😂😮😢😡`) в правильном порядке.

- [ ] **Step 3: Поставить и снять реакцию**

```bash
curl -X POST -b "access_token=<...>" -H "Content-Type: application/json" \
  -d '{"reactionId":1}' http://localhost:8080/api/messages/<id>/reactions
# 204

curl -X POST -b "access_token=<...>" -H "Content-Type: application/json" \
  -d '{"reactionId":1}' http://localhost:8080/api/messages/<id>/reactions
# 409 (дубликат)

curl -X DELETE -b "access_token=<...>" \
  http://localhost:8080/api/messages/<id>/reactions/1
# 204

curl -X DELETE -b "access_token=<...>" \
  http://localhost:8080/api/messages/<id>/reactions/1
# 204 (идемпотентно, нет ошибки)
```

- [ ] **Step 4: Проверить, что реакция подтягивается при чтении**

```bash
curl -b "access_token=<...>" http://localhost:8080/api/chats/<chatId>/messages
```

Expected: сообщения содержат поле `reactions` при наличии.

- [ ] **Step 5: Проверить WS-события**

Через второй WS-клиент (в браузере или wscat): открыть WS, поставить реакцию с первой сессии → второй клиент получает `{"type":"reaction_added", ...}`. Аналогично удаление → `reaction_removed`. Повторный DELETE — событие НЕ приходит.

- [ ] **Step 6: Если всё ок — финальный коммит-ремарка**

Ничего не коммитим (задача только проверка). Если нашли баги — фиксим отдельными коммитами.

---

## Self-Review

Проверил план против спеки:

1. **Spec §3 Schema** → Task 1 (миграция + seed) ✓
2. **Spec §4 Domain** → Task 2 (Reaction, MessageReactionSummary, MessageReactionDTO, поле `Reactions` на Message) ✓
3. **Spec §5.1 Repository** → Task 3 (все 5 методов + trace decorator) ✓
4. **Spec §5.2 Service** → Task 4 ✓
5. **Spec §5.3 Usecase + AttachReactions/AttachReaction** → Task 5 ✓
6. **Spec §5.4 Controllers** → Tasks 7 (schemas/mappers), 8 (GET), 9 (POST), 10 (DELETE) ✓
7. **Spec §5.5 Interfaces** → распределены по Tasks 3/4/5/6 (`ReactionsRepository`, `ReactionsService`, `ReactionsUseCases`, расширение `WSBroadcaster`) ✓
8. **Spec §6 WS events** → Task 2 (типы) + Task 6 (broadcaster) ✓
9. **Spec §7 Edge cases** — валидация member/message-not-found/duplicate → Task 5 usecase + Task 9 handler; идемпотентность REMOVE → Task 5 usecase + Task 10 handler ✓
10. **Spec §8 Загрузка справочника (без кэша)** → нативно в Task 4 (простой SELECT в `ListReactions`) ✓
11. **Spec §9 Тесты** — покрыты в каждой Task 3-10 ✓
12. **Spec §10 Out of scope** — не пишем API управления справочником, rate limiting и т.п. ✓
13. **Spec §11 doc.md** — обновления в каждой задаче, финализация в Task 12 ✓

Проверил типы:
- `ReactionsRepository.RemoveMessageReaction` возвращает `(bool, error)` — совпадает между интерфейсом (Task 3), сервисом (Task 4), usecase (Task 5).
- `AttachReactions([]domains.Message) ([]domains.Message, error)` — совпадает между Task 5 и Task 11.
- `MessageReactionDTO{MessageID, ReactionID, UserID}` — используется везде одинаково.
- `WSBroadcaster.BroadcastReactionAdded(ctx, chatID, messageID, userID, reactionID, emoji)` — сигнатура совпадает между интерфейсом (Task 5 Step 1), реализацией (Task 6), usecase-вызовом (Task 5 Step 4), тестом (Task 5 Step 5).

Плейсхолдеров вида "TBD" не нашёл.
