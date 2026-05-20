# Remove `chats_members.is_read` Implementation Plan

**Goal:** Удалить денормализованный `chats_members.is_read`, вычислять прочитанность чата через `messages_statuses`.

**Spec:** [specs/2026-05-20-remove-chats-members-is-read.md](../specs/2026-05-20-remove-chats-members-is-read.md)

---

## File Map

### Modified Files

| File | Change |
|------|--------|
| `internal/interfaces/repositories.go` | Удалить `ChangeChatIsReadStatus` из `ChatsRepository` |
| `internal/repositories/chats/repository.go` | `GetUserChats`: заменить `chats_members.is_read` на подзапрос по `messages_statuses`. `CreateChat`: убрать `is_read` из INSERT. Удалить `ChangeChatIsReadStatus` |
| `internal/repositories/chats/trace_decorator.go` | Удалить метод `ChangeChatIsReadStatus` |
| `internal/services/messages/service.go` | `SaveMessage`: убрать цикл `ChangeChatIsReadStatus`. `GetChatMessages`: убрать вызов `ChangeChatIsReadStatus` |
| `mocks/repositories/chats_repository.go` | Перегенерировать (mockgen) |

### New Files

| File | Purpose |
|------|---------|
| `migrations/20260520_remove_chats_members_is_read.sql` | DROP COLUMN `is_read` из `chats_members` |

### Tests — Modified

| File | Change |
|------|--------|
| `internal/repositories/chats/repository_test.go` | Удалить `TestChangeChatIsReadStatus_*`, обновить `TestCreateChat` (без is_read), обновить `TestGetUserChats` |
| `internal/repositories/chats/trace_decorator_test.go` | Удалить `TestTraceDecorator_ChangeChatIsReadStatus` |
| `internal/services/messages/service_test.go` | Убрать все `ChangeChatIsReadStatus` mock expectations |

### Docs — Updated

| File | Change |
|------|--------|
| `internal/repositories/chats/doc.md` | Обновить описание |
| `internal/services/messages/doc.md` | Обновить описание |
| `migrations/doc.md` | Добавить новую миграцию |

---

## Steps

- [x] **1. Миграция** — создать `migrations/20260520000000_remove_chats_members_is_read.sql`
- [x] **2. Интерфейс** — удалить `ChangeChatIsReadStatus` из `ChatsRepository` в `internal/interfaces/repositories.go`
- [x] **3. Репозиторий чатов** — обновить `GetUserChats` (подзапрос), `CreateChat` (без is_read), удалить `ChangeChatIsReadStatus` в `internal/repositories/chats/repository.go`
- [x] **4. Trace decorator** — удалить `ChangeChatIsReadStatus` в `internal/repositories/chats/trace_decorator.go`
- [x] **5. Сервис сообщений** — убрать вызовы `ChangeChatIsReadStatus` в `internal/services/messages/service.go`
- [x] **6. Моки** — `go generate` для `ChatsRepository`
- [x] **7. Тесты** — обновить repository_test, trace_decorator_test, service_test
- [x] **8. Документация** — обновить doc.md файлы и migrations/doc.md

## Verification

```bash
go build ./...
go test ./...
task lint
```
