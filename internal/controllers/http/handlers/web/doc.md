# Пакет handlers/web

## Назначение

Веб-интерфейс приложения. Раздаёт HTML-страницы, статические файлы (CSS, JS, assets) и Service Worker.

## Маршрутизация

`SetupHandlers(webMux)` регистрирует маршруты на subrouter с префиксом `/web`:

| Путь | Обработчик | Описание |
|------|-----------|----------|
| `/login` | `login.Handler()` | Страница входа и регистрации |
| `/forget-password` | `forget_password.Handler()` | Страница сброса пароля |
| `/chat` | `chat.Handler()` | Страница чатов |
| `/verify-email/{token}` | `verify_email.Handler()` | Страница подтверждения email |
| `/sw.js` | `serviceworker.Handler()` | Service Worker |
| `/static/**` | `http.FileServer` | CSS, JS, assets |

## Подпакеты

- `chat/` — страница чатов
- `login/` — страница входа/регистрации
- `forget_password/` — страница сброса пароля
- `verify_email/` — страница подтверждения email
- `service_worker/` — Service Worker с правильными заголовками
- `common/` — shared-утилиты (рендеринг ошибок)
- `templates/` — HTML-шаблоны
- `static/` — статические файлы (CSS, JS, assets, manifest)
