# reactions — mapper

## Назначение

Конвертирует доменные типы реакций в HTTP-schema для сериализации в JSON. Без бизнес-логики.

## Функции

| Функция | Что делает |
|---|---|
| `MapReaction(r domains.Reaction) schemas.Reaction` | Один элемент справочника. |
| `MapReactions(rs []domains.Reaction) []schemas.Reaction` | Пачка (для `GET /reactions`), сохраняет порядок. |
| `MapMessageReaction(s domains.MessageReactionSummary) schemas.MessageReaction` | Агрегат на сообщении. `UserIDs` копируется (защита от мутаций). |
| `MapMessageReactions(ss []domains.MessageReactionSummary) []schemas.MessageReaction` | Пачка агрегатов. На пустой вход возвращает `nil` — для `omitempty` в JSON. |

## Использование

- `handlers/api/reactions/list` — `MapReactions` для body `GET /reactions`.
- `mappers/messages` вызывает `MapMessageReactions` для заполнения поля `Message.Reactions` в HTTP-схеме.

## Зависимости

- `internal/domains` — `Reaction`, `MessageReactionSummary`.
- `internal/controllers/http/schemas` — `Reaction`, `MessageReaction`.
