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
  - Секция «Редактировать профиль» (смена логина).
  - Секция «Сменить пароль».
  - Секция «Уведомления» (тоглы email и web-push).
  - Тогл тёмной темы.
  - Кнопка выхода.

## Шаблонные данные

| Шаблон | Данные от сервера |
|--------|-------------------|
| `forget_password.html` | `.Email` — адрес, на который отправлен код |
| `error.html` | `.Code`, `.Message` — код и текст ошибки |
| `verify_email.html` | — (JS обрабатывает токен из URL) |
| Остальные | — (данные загружаются через API на клиенте) |
