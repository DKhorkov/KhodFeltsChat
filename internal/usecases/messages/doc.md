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

### DeleteMessage
- Если `ForAll`: проверяет через `GetMessageByID`, что сообщение существует и запрашивающий — автор. Иначе `ErrMessageNotFound` / `ErrNotMessageAuthor`.
- Делегирует удаление в `MessagesService.DeleteMessage`.

## Зависимости

- `internal/interfaces` — `MessagesService`, `ChatsService`, `UsersService`.
- `internal/domains` — `Message`, `Pagination`, `DeleteMessageDTO`.
- `internal/errors` — `ErrUserIsNotChatMember`, `ErrMessageNotFound`, `ErrNotMessageAuthor`.
