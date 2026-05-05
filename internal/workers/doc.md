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

### handlers/builders/tracing_decorator/
`Decorator` оборачивает любой `MessageHandlerBuilder`:
- Создаёт OpenTelemetry span вокруг вызова базового handler

## Поток данных

```
NATS Message → Builder.MessageHandler → Notifications UseCases → EmailsRepository → SMTP
```

## Зависимости

- `interfaces.NotificationsUseCases`
- `interfaces.MessageHandlerBuilder` (для декоратора)
- OpenTelemetry SDK
