# Пакет reactions/set

## Назначение

Ставит emoji-реакцию юзера на сообщение. Дубликат отклоняется с 409.

## Маршрут

`POST /api/messages/{id}/reactions` → `Handler(u, broadcaster)`

## Body

`{"reactionId": uint64}`

## Логика

1. Достаёт `userID` из JWT-контекста и `messageID` из URL.
2. Валидирует, что `reactionId != 0`.
3. Вызывает `u.AddReaction(ctx, dto)` — usecase проверяет member чата и наличие реакции в справочнике; возвращает `(*domains.Reaction, err)` (id + emoji).
4. При успехе — `broadcaster.BroadcastReactionAdded(messageID, userID, reactionID, reaction.Emoji)` фан-аутит WS всем участникам чата (chatID broadcaster резолвит сам через `messagesUseCases.GetMessageByID`).

## Ответы

- `204` — реакция поставлена, WS-событие `reaction_added` разослано.
- `400` — некорректный `id`/`reactionId` в URL/body.
- `401` — не авторизован.
- `403` — юзер не в чате (`ErrUserIsNotChatMember`).
- `404` — сообщение или реакция не найдены (`ErrMessageNotFound`, `ErrReactionNotFound`).
- `409` — реакция уже стоит (`ErrReactionAlreadyExists`), WS не публикуется.
- `500` — внутренняя ошибка.
