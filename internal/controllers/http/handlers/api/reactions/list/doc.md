# Пакет reactions/list

## Назначение

Отдаёт справочник доступных emoji-реакций для UI-пикера.

## Маршрут

`GET /api/reactions` → `Handler(u)`

## Логика

1. Вызывает `u.ListReactions(ctx)` — usecase проксирует из репо (`reactions` таблица).
2. Мапит `[]domains.Reaction` → `[]schemas.Reaction` через `mappers/reactions.MapReactions`.
3. Отдаёт `application/json`.

## Ответы

- `200` — массив `[{id, emoji, sortOrder}]`, отсортированный по `sort_order` из справочника.
- `500` — внутренняя ошибка (сбой репо или json.Encoder).

## Замечания

- Роут **публичный** — исключён из auth middleware в `controller.go`. Cookie с access-token не требуется.
- Справочник маленький (~10 строк) и меняется миграциями. Кэша нет — SQL достаточно быстрый.
- `sortOrder` нужен фронту для стабильного порядка в UI-пикере и на бейджах реакции сообщения (WS-события `reaction_added` могут прийти в неправильном порядке).
