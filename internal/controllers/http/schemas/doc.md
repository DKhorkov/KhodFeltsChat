# schemas — HTTP схемы

## Назначение

Swagger-аннотированные request/response структуры. Чистые данные, без логики.

## Файлы

| Файл | Содержимое |
|------|-----------|
| `auth.go` | RegisterInput, LoginInput, RefreshTokenInput, ForgetPasswordInput, ChangePasswordInput, SendVerifyEmailInput, SendForgetPasswordInput, VerifyEmailInput |
| `users.go` | User, GetUsersInput, UpdateUserInput, GetUserByIDInput |
| `chats.go` | Chat, CreateChatInput, MemberInput, GetUserChatsInput |
| `messages.go` | Message, Sender, GetChatMessagesInput |
| `pagination.go` | Pagination (Limit, Offset) |
| `responses.go` | Swagger envelope types: OK, BadRequest, NotFound, InternalServerError, Conflict, Unauthorized, Forbidden, SeeOther, NoContent, SwitchingProtocols |
| `push_subscriptions.go` | PushSubscriptionKeys, CreatePushSubscriptionRequest, CreatePushSubscriptionResponse, VAPIDKeyResponse |
