# reactions — репозиторий реакций

## Назначение

Работа со справочником реакций (`reactions`) и M2M-таблицей `messages_reactions`
(юзер ↔ сообщение ↔ реакция). Multi-реакции на юзера: один и тот же юзер может
поставить несколько разных реакций на одно сообщение, но одну и ту же — не
может (UNIQUE `(message_id, user_id, reaction_id)`).

## Методы

| Метод | SQL | Семантика |
|---|---|---|
| `ListReactions(ctx)` | `SELECT id, emoji, sort_order FROM reactions ORDER BY sort_order ASC` | Полный справочник для UI-пикера. Скан через `pg.GetEntityColumns(&domains.Reaction{})` |
| `GetReactionByID(ctx, id)` | `SELECT id, emoji, sort_order FROM reactions WHERE id = $1` | Возвращает сырой `sql.ErrNoRows`, если реакции нет. Маппинг в доменный `ErrReactionNotFound` — на уровне сервиса |
| `AddMessageReaction(ctx, dto)` | `INSERT ... ON CONFLICT DO NOTHING` + `ExecContext` | При конфликте `RowsAffected == 0` → `ErrReactionAlreadyExists` |
| `RemoveMessageReaction(ctx, dto)` | `DELETE ... WHERE (message_id, user_id, reaction_id)` | `RowsAffected == 0` → `ErrReactionNotSet` (идемпотентная семантика) |
| `ListReactionsForMessages(ctx, ids)` | `SELECT ... JOIN reactions ORDER BY sort_order ASC` | Один SQL, группировка в Go в `map[messageID][]MessageReactionSummary` за один проход через `positions[msgID][reactionID] → index`. Скан через `pg.GetEntityColumns(&MessageReactionRowPg{})`. Порядок сообщений и `userIDs` внутри сводки не гарантирован — фронт всё равно сортирует реакции по `sortOrder`, а userIDs использует только для `.length` и lookup |

## Семантика ошибок

- `customerrors.ErrReactionAlreadyExists` — попытка поставить уже стоящую реакцию (репо).
- `customerrors.ErrReactionNotSet` — при удалении: реакции у юзера не было (репо).
- `sql.ErrNoRows` из `GetReactionByID` — репо не оборачивает; сервис маппит в `ErrReactionNotFound`.

## Trace decorator

`NewTraceDecorator(provider, cfg, base)` оборачивает каждый метод в
OpenTelemetry-span с start/end-events (единый паттерн проекта).

## Зависимости

- `github.com/DKhorkov/libs/db/postgresql` — `pg.Transaction`.
- `github.com/Masterminds/squirrel` — построитель SQL.
- `github.com/DKhorkov/libs/logging` — логирование ошибок закрытия rows.
