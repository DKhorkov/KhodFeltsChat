# Пакет reactions/unset

## Назначение

Снимает emoji-реакцию юзера с сообщения. Идемпотентно: 204 даже если реакции не было.

## Маршрут

`DELETE /api/messages/{id}/reactions/{reactionId}` → `Handler(u, broadcaster)`

Плейсхолдер `{reactionId}` собирается через экспортированную константу `ReactionIDRouteKey` (`"reactionId"`) — по паттерну `verify_email.TokenRouteKey`.

## Логика

1. Достаёт `userID` из JWT-контекста и `messageID`, `reactionId` из URL.
2. Вызывает `u.RemoveReaction(ctx, dto)` — usecase проверяет member чата и вызывает репо; возвращает `error`.
3. Если `ErrReactionNotSet` — отдаём 204 без broadcast (юзер повторно снимает — не спамим WS).
4. При успехе — `broadcaster.BroadcastReactionRemoved(messageID, userID, reactionID)` фан-аутит WS всем участникам чата (chatID broadcaster резолвит сам через `messagesUseCases.GetMessageByID`).

## Ответы

- `204` — реакция снята и разослан WS-событие, ИЛИ реакции не было (идемпотентно, без WS).
- `400` — некорректные `id`/`reactionId` в URL.
- `401` — не авторизован.
- `403` — юзер не в чате.
- `404` — сообщение не найдено.
- `500` — внутренняя ошибка.

## Замечания

`ErrReactionNotSet` — отдельный sentinel от `ErrReactionNotFound`: первое означает «на сообщении не было такой реакции», второе — «в справочнике нет реакции с таким id».
