# web/static

## Назначение

Статические ресурсы веб-интерфейса: CSS, JavaScript, PWA-манифест,
Service Worker и иконки. Отдаются через `http.FileServer` с префиксом
`/web/static/`.

## Структура

```
static/
├── assets/
│   └── icon.png         — иконка приложения (192×512), используется в PWA и push-уведомлениях
├── css/
│   ├── global.css       — CSS-переменные, сброс стилей, общие компоненты (модалки, кнопки, тоглы, формы)
│   ├── navbar.css       — навбар, модалка профиля (секции, уведомления, тема), баннер web-push, адаптивность
│   ├── chat.css         — страница чатов (список, сообщения, ввод, emoji-picker)
│   ├── home.css         — главная страница
│   ├── login.css        — страница входа/регистрации
│   ├── forget-password.css — страница сброса пароля
│   ├── error.css        — страница ошибки
│   ├── alert-modal.css  — модалка уведомлений (showInfo/showError)
│   └── emoji-picker.css — выбор emoji
├── js/
│   ├── auth.js          — fetchWithAuth(): обёртка fetch с автообновлением JWT при 401
│   ├── navbar.js        — тема, профиль, настройки, web-push подписка, баннер уведомлений (iOS)
│   ├── chat.js          — WebSocket-чат, отправка/получение сообщений, UI чатов
│   ├── login.js         — логин/регистрация, валидация форм
│   ├── forget-password.js — сброс пароля
│   ├── verify-email.js  — подтверждение email по токену
│   ├── errors-mapper.js — маппинг серверных ошибок в пользовательские сообщения на русском
│   ├── validation.js    — клиентская валидация (email, пароль, username)
│   ├── alert-modal.js   — showInfo(), showError() — модалки уведомлений
│   └── emoji-picker.js  — emoji-picker для чата
├── sw.js                — Service Worker: push-уведомления, навигация по клику
└── manifest.json        — PWA-манифест (имя, иконки, display: standalone)
```

## Ключевые файлы

### sw.js — Service Worker

- Обрабатывает `push` event: показывает системное уведомление.
- На iOS `showNotification` вызывается **всегда** — iOS отбрасывает push
  без этого вызова и может отозвать разрешение.
- На десктопе подавляет уведомление, если чат в фокусе.
- `notificationclick`: открывает/фокусирует вкладку чата, передаёт `chatId`.

### manifest.json — PWA

- `display: standalone` — обязательно для web-push на iOS (16.4+).
- `id: "/"` — предотвращает потерю подписки при переустановке PWA (iOS 17+).
- Иконки с `purpose: "any"`.

### navbar.js

- **Тема**: загружается с сервера, применяется через `data-bs-theme` атрибут.
  Синхронизируется при возврате на вкладку (`visibilitychange`) и при
  открытии модалки профиля.
- **Web Push**: ключи подписки кодируются в base64url. `applicationServerKey`
  передаётся как `ArrayBuffer` (`.buffer.slice(0)`) — iOS не принимает `Uint8Array`.
- **Баннер (iOS)**: `Notification.requestPermission()` на iOS требует user gesture.
  Если серверный consent включён, но разрешение не запрошено — показывается
  баннер вверху страницы. `sessionStorage` — один раз за сессию PWA.
  На десктопе баннер не показывается, разрешение запрашивается напрямую.

### errors-mapper.js

Маппинг серверных строк ошибок (из `internal/errors/`) в пользовательские
сообщения на русском. Ключи отсортированы по убыванию длины для приоритета
более специфичных совпадений.

### global.css

CSS-переменные (цвета, шрифты, отступы, радиусы, тени) с поддержкой
тёмной темы через `[data-bs-theme="dark"]`. Общие стили компонентов:
модалки (`modal-overlay`, `modal-content`), кнопки, формы, тоглы.
Модалка профиля: фиксированная высота `80vh`, `overflow-y: auto`.
