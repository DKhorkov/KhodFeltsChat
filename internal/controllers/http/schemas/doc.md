# schemas — HTTP схемы

## Назначение

Swagger-аннотированные request/response структуры. Чистые данные, без логики.

## Файлы

| Файл | Содержимое |
|------|-----------|
| `auth.go` | RegisterInput, LoginInput, RefreshTokenInput, ForgetPasswordInput, ChangePasswordInput, SendVerifyEmailInput, SendForgetPasswordInput, VerifyEmailInput |
| `users.go` | User, GetUsersInput, UpdateUserInput, GetUserByIDInput |
| `chats.go` | Chat, CreateChatInput, MemberInput, GetUserChatsInput |
| `messages.go` | Message (включает `ReplyToMessage *ReplyMessage` и `Reactions []MessageReaction`), ReplyMessage (ID, Sender, Text, CreatedAt), Sender, GetChatMessagesInput |
| `reactions.go` | Reaction (ID, Emoji) — элемент справочника; MessageReaction (Reaction + UserIDs) — агрегат на сообщении; SetReactionInput — body для POST /messages/{id}/reactions |
| `pagination.go` | Pagination (Limit, Offset) |
| `responses.go` | Swagger envelope types: OK, BadRequest, NotFound, InternalServerError, Conflict, Unauthorized, Forbidden, SeeOther, NoContent, SwitchingProtocols |
| `settings.go` | Settings (включает `EmailConsents` и `WebPushConsents`) |
| `web_push_subscriptions.go` | CreateWebPushSubscriptionRequest (плоская структура: `endpoint`, `encryptionKey`, `auth`), CreateWebPushSubscriptionResponse, VAPIDKeyResponse |
