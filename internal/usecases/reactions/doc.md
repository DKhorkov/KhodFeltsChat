# reactions — usecase реакций

## Назначение

Оркестрация установки/снятия реакций и обогащение сообщений реакциями при чтении.

## Методы

| Метод | Описание |
|---|---|
| `ListReactions(ctx)` | Проксирует справочник для UI-пикера |
| `AddReaction(ctx, dto)` | Валидация + `AddMessageReaction` + WS `reaction.added` |
| `RemoveReaction(ctx, dto)` | Валидация + `RemoveMessageReaction` + WS `reaction.removed` (только при `deleted=true`) |
| `AttachReactions(ctx, msgs)` | Пачкой подгружает реакции для списка сообщений |
| `AttachReaction(ctx, msg)` | Подгружает реакции для одного сообщения |

## Валидация в Add/RemoveReaction

1. `messagesService.GetMessageByID(userID, messageID)` — существует ли сообщение и
   доступно ли юзеру (`messages_statuses.is_deleted = false`). При отсутствии — `ErrMessageNotFound`.
2. `chatsService.GetChatMembers(chatID, userID)` — юзер является участником чата.
   Если нет — `ErrUserIsNotChatMember`.
3. Для Add: `reactionsService.GetReactionByID(reactionID)` — реакция есть в справочнике.
   Иначе — `ErrReactionNotFound`.
4. Add: `reactionsService.AddMessageReaction(dto)` — может вернуть `ErrReactionAlreadyExists`.
5. Remove: `reactionsService.RemoveMessageReaction(dto)` — возвращает `(deleted bool, err error)`.

## WS-семантика

- `AddReaction`: событие публикуется только после успешного `AddMessageReaction`.
- `RemoveReaction`: событие публикуется только если `deleted == true` — иначе бы шёл спам
  по чату на повторных DELETE.

## Trace decorator

Стандартный паттерн проекта.

## Зависимости

- `interfaces.ReactionsService`, `MessagesService`, `ChatsService`, `WSBroadcaster`.
- `internal/errors` — sentinel-ошибки.
