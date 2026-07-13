# reactions — usecase реакций

## Назначение

Оркестрация установки/снятия реакций. Broadcast **делает HTTP-handler**, не
usecase — так у usecase-слоя нет обратной зависимости на `*ws.Handler` и нет
транспортных деталей в сигнатурах.

Обогащение сообщений реакциями при чтении делает `messagesUseCases` через
приватный метод `attachReactions`, обращаясь **напрямую к `ReactionsService`**.
Usecase-слои не зависят друг от друга — только через сервисы.

## Методы

| Метод | Описание |
|---|---|
| `ListReactions(ctx) ([]Reaction, error)` | Справочник для UI-пикера |
| `AddReaction(ctx, dto) (*Reaction, error)` | Валидация + `AddMessageReaction`; возвращает доменную реакцию (id + emoji) — handler использует её для broadcast |
| `RemoveReaction(ctx, dto) error` | Валидация + `RemoveMessageReaction`. При отсутствующей реакции — `ErrReactionNotSet` (handler даёт 200 без WS) |

## Валидация в Add/RemoveReaction

1. `messagesService.GetMessageByID(userID, messageID)` — существует ли сообщение и
   доступно ли юзеру (`messages_statuses.is_deleted = false`). При отсутствии — `ErrMessageNotFound`.
2. `chatsService.GetChatMembers(chatID, userID)` — юзер является участником чата.
   Если нет — `ErrUserIsNotChatMember`.
3. Для Add: `reactionsService.GetReactionByID(reactionID)` — реакция есть в справочнике.
   Иначе — `ErrReactionNotFound`.
4. Add: `reactionsService.AddMessageReaction(dto)` — может вернуть `ErrReactionAlreadyExists`.
5. Remove: `reactionsService.RemoveMessageReaction(dto)` — может вернуть `ErrReactionNotSet`.

## Trace decorator

Стандартный паттерн проекта.

## Зависимости

- `interfaces.ReactionsService`, `MessagesService`, `ChatsService`.
- `internal/errors` — sentinel-ошибки.
