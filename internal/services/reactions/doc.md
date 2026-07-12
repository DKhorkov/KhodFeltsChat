# reactions — сервис реакций

## Назначение

Тонкая обёртка над `ReactionsRepository` через `UnitOfWork`. Проксирует все
методы репо (`ListReactions`, `GetReactionByID`, `AddMessageReaction`,
`RemoveMessageReaction`, `ListReactionsForMessages`) в рамках транзакции.

## Особенности

- Никакой бизнес-логики: сервис — граница транзакции. Валидация (member чата,
  реакция из справочника, soft-deleted и т.п.) живёт в usecase.
- `GetReactionByID`: репо отдаёт сырой `sql.ErrNoRows`, сервис маппит его в
  доменный `customerrors.ErrReactionNotFound` через `errors.Is`. Остальные
  sentinel-ошибки (`ErrReactionAlreadyExists`, `ErrReactionNotSet`) приходят
  из репо как есть.

## Trace decorator

`NewTraceDecorator(provider, cfg, base)` — единый паттерн проекта.

## Зависимости

- `github.com/DKhorkov/kfc/internal/interfaces` — `UnitOfWork`, `ReactionsRepository`.
- `github.com/DKhorkov/libs/db/postgresql` — `pg.Transaction`.
