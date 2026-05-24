# Пакет messages/delete

## Назначение

Удаление сообщения (у себя или у всех).

## Маршрут

`DELETE /api/messages/{id}` → `Handler(u, broadcaster)`

## Body

`{"forAll": bool}`

## Логика

1. Получает сообщение через `GetMessageByID` для определения `chatID`
2. Вызывает `DeleteMessage` с DTO
3. При `forAll=true` — `broadcaster.BroadcastMessageDeleted` рассылает WS-событие всем участникам чата
4. При `forAll=false` — `broadcaster.SendMessageDeletedToUser` отправляет WS-событие только текущему пользователю

## Ответы

- `204` — сообщение удалено
- `400` — некорректный запрос
- `401` — не авторизован
- `403` — не автор сообщения (для `forAll=true`)
- `404` — сообщение не найдено
