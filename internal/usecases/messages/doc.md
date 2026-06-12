# Пакет usecases/messages

## Назначение

Бизнес-логика для работы с сообщениями: сохранение новых и получение истории чата.

## Ключевые операции

### SaveMessage
- Прямая передача запроса в `MessagesService` без дополнительной логики.
- Возвращает сохранённый объект `domains.Message`.

### GetChatMessages
- Проверяет существование пользователя через `UsersService`.
- Получает список участников чата через `ChatsService`.
- Убеждается, что запрашивающий пользователь является членом чата.
- Возвращает сообщения с поддержкой пагинации.

### GetMessageByID
- Прямая передача запроса в `MessagesService`.
- Используется NATS-воркером push-уведомлений для получения текста сообщения.

### GetUserUnreadCount
- Прямая передача запроса в `MessagesService.GetUserUnreadCount`.
- Возвращает общее число непрочитанных и неудалённых сообщений пользователя по всем чатам.
- Используется для проставления `unreadCount` в payload push-уведомлений (PWA-бейдж).

### DeleteMessage
- Если `ForAll`: проверяет через `GetMessageByID`, что сообщение существует и запрашивающий — автор. Иначе `ErrMessageNotFound` / `ErrNotMessageAuthor`.
- Делегирует удаление в `MessagesService.DeleteMessage`.

### UpdateMessage
- Проверяет через `GetMessageByID`, что сообщение существует и запрашивающий — автор. Иначе `ErrMessageNotFound` / `ErrNotMessageAuthor`.
- Делегирует редактирование в `MessagesService.UpdateMessage`.

## Зависимости

- `internal/interfaces` — `MessagesService`, `ChatsService`, `UsersService`.
- `internal/domains` — `Message`, `Pagination`, `DeleteMessageDTO`, `UpdateMessageDTO`.
- `internal/errors` — `ErrUserIsNotChatMember`, `ErrMessageNotFound`, `ErrNotMessageAuthor`.
