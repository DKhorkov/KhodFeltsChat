# workers — NATS Consumer Workers

## Назначение

Обработчики NATS сообщений для асинхронных операций (email уведомления).

## Структура

### handlers/builders/verify_email/
`Builder.MessageHandler(ctx)` возвращает `nats.MsgHandler`:
1. Десериализация `VerifyEmailNotificationDTO{UserID}`
2. Вызов `notificationsUseCases.SendVerifyEmailMessage(ctx, userID)`

### handlers/builders/forget_password/
Аналогичный паттерн для `ForgetPasswordNotificationDTO`.

### handlers/builders/push_notification/
`Builder.MessageHandler(ctx)` возвращает `nats.MsgHandler`:
1. Десериализация `PushNotificationDTO{UserID, MessageID}`
2. Получение текста сообщения через `messagesUseCases.GetMessageByID`
3. Получение всех push-подписок пользователя через `pushSubscriptionsUseCases.GetPushSubscriptionsByUserID`
4. Отправка push-уведомления на каждую подписку через `pushSubscriptionsUseCases.SendPushNotification`

### handlers/builders/tracing_decorator/
`Decorator` оборачивает любой `MessageHandlerBuilder`:
- Создаёт OpenTelemetry span вокруг вызова базового handler

## Поток данных

```
NATS Message (email)       → Builder.MessageHandler → Notifications UseCases → EmailsRepository → SMTP
NATS Message (push)        → Builder.MessageHandler → MessagesUseCases + PushSubscriptionsUseCases → Web Push API
```

## Зависимости

- `interfaces.NotificationsUseCases`
- `interfaces.MessagesUseCases`
- `interfaces.PushSubscriptionsUseCases`
- `interfaces.MessageHandlerBuilder` (для декоратора)
- OpenTelemetry SDK
