# workers — NATS Consumer Workers

## Назначение

Обработчики NATS сообщений для асинхронных операций (email уведомления).

## Структура

### handlers/builders/email_notification/
Унифицированный обработчик всех email-уведомлений.
`Builder.MessageHandler(ctx)` возвращает `nats.MsgHandler`:
1. Десериализация `EmailNotificationDTO{Type, UserID}`
2. Маршрутизация по `Type`:
   - `VerifyEmail` → `notificationsUseCases.SendVerifyEmailMessage`
   - `ForgetPassword` → `notificationsUseCases.SendForgetPasswordMessage`
   - `NewMessage` → `notificationsUseCases.SendNewMessageNotification`

### handlers/builders/web_push_notification/
`Builder.MessageHandler(ctx)` возвращает `nats.MsgHandler`:
1. Десериализация `WebPushNotificationDTO{Type, UserID, MessageID}`
2. Проверка согласия пользователя на уведомления через `settingsUseCases`
3. Получение текста сообщения через `messagesUseCases.GetMessageByID`
4. Получение всех push-подписок пользователя через `webPushSubscriptionsUseCases.GetWebPushSubscriptionsByUserID`
5. Отправка push-уведомления на каждую подписку через `webPushSubscriptionsUseCases.SendWebPushNotification`

### handlers/builders/tracing_decorator/
`Decorator` оборачивает любой `MessageHandlerBuilder`:
- Создаёт OpenTelemetry span вокруг вызова базового handler

## Поток данных

```
NATS Message (email)       → Builder.MessageHandler → (dispatch by Type) → Notifications UseCases → EmailsRepository → SMTP
NATS Message (push)        → Builder.MessageHandler → SettingsUseCases (consent check) → MessagesUseCases + WebPushSubscriptionsUseCases → Web Push API
```

## Зависимости

- `interfaces.NotificationsUseCases`
- `interfaces.MessagesUseCases`
- `interfaces.WebPushSubscriptionsUseCases`
- `interfaces.SettingsUseCases`
- `interfaces.MessageHandlerBuilder` (для декоратора)
- OpenTelemetry SDK
