# web/templates

## Назначение

Go HTML-шаблоны (`html/template`) для страниц веб-интерфейса. Рендерятся
серверными хендлерами из `web/` пакетов (chat, login, home и т.д.).

## Структура

```
templates/
├── navbar.html          — навбар + модалка профиля (общий компонент, {{template "navbar"}})
├── chat.html            — страница чатов (WebSocket)
├── home.html            — главная страница
├── login.html           — вход / регистрация
├── forget_password.html — сброс пароля (форма с кодом из письма)
├── verify_email.html    — подтверждение email (автоматическая проверка по токену)
└── error.html           — страница ошибки (код + сообщение)
```

## Общий `<head>` блок

Все шаблоны содержат одинаковый набор мета-тегов и ссылок:

```html
<link rel="icon" type="image/png" href="/web/static/assets/icon.png">
<link rel="apple-touch-icon" href="/web/static/assets/icon.png">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="default">
<link rel="manifest" href="/web/static/manifest.json">
```

- `apple-touch-icon` — иконка при добавлении на Home Screen (iOS).
- `apple-mobile-web-app-capable` — включает standalone-режим PWA на iOS.
- `apple-mobile-web-app-status-bar-style` — стиль статус-бара в PWA.
- `manifest.json` — PWA-манифест (имя, иконки, display mode).

Тема восстанавливается из `localStorage` inline-скриптом до загрузки CSS
(предотвращает мигание при тёмной теме):

```html
<script>if(localStorage.getItem('theme')==='dark')document.documentElement.setAttribute('data-bs-theme','dark');</script>
```

## navbar.html

Определяет Go-шаблон `{{define "navbar"}}`, подключаемый в другие страницы
через `{{template "navbar"}}`. Содержит:

- Навбар с логотипом и кнопкой входа / профилем пользователя.
- Модалку профиля (`modal-my-profile`):
  - Аватар, имя, email, дата регистрации.
  - Контекстное меню аватара (`avatar-context-menu`): «Открыть фото» (если есть аватар), «Изменить фото», «Удалить фото» (если есть аватар).
  - Секция «Редактировать профиль» (смена логина).
  - Секция «Сменить пароль».
  - Секция «Уведомления» (тоглы email и web-push).
  - Тогл тёмной темы.
  - Кнопка выхода.
- Оверлей увеличения аватара (`avatar-zoom-overlay`) — общий для модалки своего профиля и профилей других пользователей; вынесен в navbar.html, так как профиль может быть открыт с любой страницы.

## Шаблонные данные

## chat.html — Ключевые UI-элементы

| Элемент | ID | Описание |
|---------|-----|----------|
| Плашка ответа | `reply-bar` | Показывается при ответе на сообщение (отправитель + текст) |
| Плашка редактирования | `edit-bar` | Показывается при редактировании сообщения (label «Редактирование» + текст) |
| Кнопка «Ответить» | `ctx-reply` | Контекстное меню — ответ на сообщение |
| Кнопка «Редактировать» | `ctx-edit` | Контекстное меню — редактирование (только свои, `display: none` по умолчанию) |
| Кнопка «Копировать текст» | `ctx-copy` | Контекстное меню — копирование текста |
| Группа «Удалить» | `ctx-delete-group` | Контекстное меню — удаление (только свои, `display: none` по умолчанию) |
| Кнопка «К последнему сообщению» | `btn-scroll-down` | Круглая кнопка с inline SVG «↓» внутри composer. Появляется при отскролле от низа чата. Содержит `#scroll-down-badge` (счётчик новых сообщений). |

## Шаблонные данные

| Шаблон | Данные от сервера |
|--------|-------------------|
| `forget_password.html` | `.Email` — адрес, на который отправлен код |
| `error.html` | `.Code`, `.Message` — код и текст ошибки |
| `verify_email.html` | — (JS обрабатывает токен из URL) |
| Остальные | — (данные загружаются через API на клиенте) |
