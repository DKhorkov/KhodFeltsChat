# repositories/emails

## Назначение

Репозиторий для отправки email-уведомлений через SMTP. Не взаимодействует
с базой данных — доставляет письма напрямую через SMTP-сервер.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SendVerifyEmailMessage(ctx, user)` | Формирует и отправляет письмо с ссылкой для верификации email |
| `SendForgetPasswordMessage(ctx, user)` | Формирует и отправляет письмо для сброса пароля |

Оба метода:
1. Вызывают соответствующий `ContentBuilder` для генерации темы и HTML-тела письма.
2. Создают новый `gomail.Dialer` на каждый вызов (соединение не переиспользуется).
3. Отправляют письмо через `DialAndSend`.

## Зависимости

- `gopkg.in/gomail.v2` — формирование и отправка email по SMTP
- `github.com/DKhorkov/kfc/internal/config.SMTPConfig` — хост, порт, логин, пароль SMTP
- `github.com/DKhorkov/kfc/internal/interfaces.ContentBuilders` — генераторы содержимого писем
- `github.com/DKhorkov/kfc/internal/domains` — тип `User`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
