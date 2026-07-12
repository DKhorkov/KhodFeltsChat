# reactions — репозиторий реакций

## Назначение

Работа со справочником реакций (`reactions`) и M2M-таблицей `messages_reactions`
(юзер ↔ сообщение ↔ реакция). Multi-реакции на юзера: один и тот же юзер может
поставить несколько разных реакций на одно сообщение, но одну и ту же — не
может (UNIQUE `(message_id, user_id, reaction_id)`).

## Методы

| Метод | SQL | Семантика |
|---|---|---|
| `ListReactions(ctx)` | `SELECT id, emoji FROM reactions ORDER BY sort_order ASC` | Полный справочник для UI-пикера |
| `GetReactionByID(ctx, id)` | `SELECT id, emoji FROM reactions WHERE id = $1` | Валидация ID до записи. Возвращает `ErrReactionNotFound` при `sql.ErrNoRows` |
| `AddMessageReaction(ctx, dto)` | `INSERT ... ON CONFLICT DO NOTHING RETURNING id` | Дубликат по UNIQUE → `ErrReactionAlreadyExists` (через `sql.ErrNoRows` на `RETURNING`) |
| `RemoveMessageReaction(ctx, dto)` | `DELETE ... WHERE (message_id, user_id, reaction_id)` | Возвращает `(deleted bool, err error)` — `deleted` из `RowsAffected` |
| `ListReactionsForMessages(ctx, ids)` | `SELECT ... JOIN reactions ORDER BY message_id, sort_order, created_at` | Один SQL, группировка в Go в `map[messageID][]MessageReactionSummary` |

## Семантика ошибок

- `customerrors.ErrReactionNotFound` — реакция с таким `id` отсутствует в справочнике.
- `customerrors.ErrReactionAlreadyExists` — попытка поставить уже стоящую реакцию.

## Trace decorator

`NewTraceDecorator(provider, cfg, base)` оборачивает каждый метод в
OpenTelemetry-span с start/end-events (единый паттерн проекта).

## Зависимости

- `github.com/DKhorkov/libs/db/postgresql` — `pg.Transaction`.
- `github.com/Masterminds/squirrel` — построитель SQL.
- `github.com/DKhorkov/libs/logging` — логирование ошибок закрытия rows.
