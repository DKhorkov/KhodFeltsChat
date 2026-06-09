# Scroll-to-Bottom Button with Unread Counter

## Summary

Кнопка «вернуться к низу чата» со счётчиком непрочитанных. Появляется, когда последнее сообщение чата вышло из видимой области. По клику — плавный скролл к низу. При входящих сообщениях, когда пользователь отсткроллил вверх, автоматический скролл больше не выполняется — вместо этого инкрементируется счётчик на кнопке.

Затрагивает web-клиент (KhodFeltsChat) и десктоп GUI (KhodFeltsChatGUI). Бэкенд не меняется.

---

## 1. Поведение

### 1.1 Состояния

- `unreadCount: number` — счётчик новых сообщений с момента, когда пользователь отскроллил вверх. По умолчанию `0`.
- `isAtBottom: boolean` — флаг видимости последнего сообщения чата. Управляется `IntersectionObserver` на последнем `.message-bubble`.
- Видимость кнопки: `isAtBottom === false`.
- Видимость бейджа: `unreadCount > 0` (вместе с кнопкой).

### 1.2 Сценарии

| Событие | Реакция |
|---------|---------|
| Открытие/переключение чата | Мгновенный скролл вниз (как сейчас). `unreadCount = 0`. |
| Отправка своего сообщения | Мгновенный скролл вниз (как сейчас). `unreadCount = 0`. |
| Входящее сообщение, `isAtBottom === true` | Мгновенный скролл вниз (follow mode). `unreadCount` остаётся `0`. |
| Входящее сообщение, `isAtBottom === false` | Скролл НЕ делаем. `unreadCount++`. |
| Пользователь вручную доскроллил до низа (IntersectionObserver: `isAtBottom → true`) | Кнопка скрывается. `unreadCount = 0`. |
| Клик по кнопке | Smooth scroll к низу. `unreadCount = 0` (произойдёт через переход `isAtBottom → true`). |
| Закрытие чата (`closeChat`) | Состояния сбрасываются: `unreadCount = 0`, `isAtBottom = true`. |

### 1.3 Формат счётчика

- `0` — бейдж скрыт.
- `1..99` — показываем число как есть.
- `> 99` — показываем `99+`.

---

## 2. Технические детали

### 2.1 IntersectionObserver на последнем сообщении

**Зачем не порог в пикселях:** требование — «появляется, когда даже последнее сообщение не полностью видно». Этот критерий нативно выражается через `IntersectionObserver` на последнем `.message-bubble`. Адаптивен к любым размерам viewport (важно для мобильной web-версии).

**Настройки observer:**
```js
new IntersectionObserver(callback, {
  root: messagesListContainer,
  threshold: 0.1, // считаем "видимым" хотя бы 10% последнего сообщения
})
```

`threshold: 0.1` — чтобы не дёргать состояние при пиксельной точности.

**Переподписка:**
- При смене последнего сообщения (новое сообщение пришло / отправлено / удалено / при загрузке новой страницы старых сообщений последнее НЕ меняется, но при первой загрузке чата нужна первая подписка) — `observer.disconnect()`, затем `observer.observe(newLastBubble)`.
- Реализация:
  - **Web:** после `renderMessages()`, `appendMessage()`, обработки `message_deleted` — пересчитать последний bubble и переподписаться.
  - **GUI:** `watch(messages, ...)` с `flush: 'post'`, чтобы наблюдатель срабатывал после рендера DOM.

**Колбэк:**
```js
(entries) => {
  const last = entries[entries.length - 1]
  isAtBottom = last.isIntersecting
  if (isAtBottom) unreadCount = 0
}
```

### 2.2 Smooth scroll по клику

```js
messagesListEl.scrollTo({ top: messagesListEl.scrollHeight, behavior: 'smooth' })
```

После анимации `IntersectionObserver` сам зафиксирует `isAtBottom = true` и сбросит `unreadCount`. Никаких ручных сбросов в обработчике клика не делаем — единый источник правды это observer.

### 2.3 Follow mode при входящем сообщении

В обработчике входящего WS-сообщения (web: `handleNewMessage`, GUI: `handleNewMessage`):

```js
const wasAtBottom = isAtBottom  // снимок ДО мутации списка
appendMessage(message)
await nextTick() // (для Vue; в web — после insertAdjacentHTML)
if (wasAtBottom) {
  scrollToBottomInstant()
} else {
  unreadCount += 1
}
```

Снимок до мутации важен: после добавления нового последнего bubble observer ещё не успел отработать.

### 2.4 Edge cases

- **Своё сообщение через WS:** в текущей архитектуре все сообщения (включая свои) приходят через WS. Своё сообщение должно прокрутить вниз всегда. Различаем по `message.sender.id === currentUser.id`:
  ```js
  const isOwn = message.sender.id === currentUser.id
  if (isOwn || wasAtBottom) {
    scrollToBottomInstant()
  } else {
    unreadCount += 1
  }
  ```
- **Сообщения из неактивного чата:** уже игнорируются текущей логикой (показывается toast). Счётчик не трогаем.
- **`message_deleted` / `message_edited`:** не меняют `unreadCount`. После удаления последнего сообщения — переподписываем observer на новый последний bubble.
- **Загрузка старых сообщений (`loadMoreMessages`)**: добавляет в начало списка, последний bubble не меняется — переподписка не нужна. Но observer должен быть жив (не disconnect).

---

## 3. Web-клиент (KhodFeltsChat)

### 3.1 HTML — `internal/controllers/http/handlers/web/templates/chat.html`

Кнопка размещается **внутри `.conversation__composer`** первым дочерним элементом и позиционируется `position: absolute; bottom: 100%; right: 16px`. Это привязывает её к верхнему краю composer (то есть прямо над полем ввода), без магических чисел и без необходимости знать высоту composer. Composer получает `position: relative`.

```html
<div class="conversation__composer">
    <button class="conversation__scroll-down" id="btn-scroll-down" hidden type="button" aria-label="К последнему сообщению">
        <svg class="conversation__scroll-down-icon" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="6 9 12 15 18 9"/>
        </svg>
        <span class="conversation__scroll-down-badge" id="scroll-down-badge" hidden>0</span>
    </button>
    <div class="conversation__reply-bar" ...>...</div>
    <!-- остальной composer без изменений -->
</div>
```

Атрибут `hidden` — нативно скрываем без CSS. Через JS управляем `.hidden = true/false`.

### 3.2 CSS — `internal/controllers/http/handlers/web/static/css/chat.css`

```css
.conversation__composer {
    position: relative; /* добавить, если ещё не задано */
}

.conversation__scroll-down {
    position: absolute;
    right: 16px;
    bottom: 100%;          /* над верхней границей composer */
    margin-bottom: 12px;   /* визуальный отступ от composer */
    width: 44px;
    height: 44px;
    border: none;
    border-radius: 50%;
    background: var(--accent);
    color: var(--text-on-accent, #fff);
    cursor: pointer;
    opacity: 0.65;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
    transition: opacity var(--transition-base), transform var(--transition-base);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 5;
}

.conversation__scroll-down:hover {
    opacity: 1;
}

.conversation__scroll-down:active {
    transform: scale(0.95);
}

.conversation__scroll-down-icon {
    pointer-events: none;
}

.conversation__scroll-down-badge {
    position: absolute;
    top: -4px;
    right: -4px;
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    border-radius: 10px;
    background: var(--danger, #e53935);
    color: #fff;
    font-size: 11px;
    font-weight: 600;
    line-height: 20px;
    text-align: center;
    pointer-events: none;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

/* Мобильная версия — кнопка чуть меньше и ближе к краю */
@media (max-width: 600px) {
    .conversation__scroll-down {
        width: 40px;
        height: 40px;
        right: 12px;
    }
}
```

`bottom: 100% + margin-bottom` — позиционирование от верхней границы composer, поэтому динамическая высота composer (включая/исключая reply-bar/edit-bar) не влияет на позицию кнопки относительно поля ввода.

### 3.3 JS — `internal/controllers/http/handlers/web/static/js/chat.js`

**Новые модульные переменные:**
```js
let unreadCount = 0
let isAtBottom = true
let lastMessageObserver = null
let scrollDownBtn = null
let scrollDownBadge = null
```

**Инициализация в `init()` или при `DOMContentLoaded`:**
```js
scrollDownBtn = document.getElementById('btn-scroll-down')
scrollDownBadge = document.getElementById('scroll-down-badge')
scrollDownBtn.addEventListener('click', onScrollDownClick)

lastMessageObserver = new IntersectionObserver(onLastMessageVisibilityChange, {
    root: messagesListEl,
    threshold: 0.1,
})
```

**Функции:**

```js
function onLastMessageVisibilityChange(entries) {
    const last = entries[entries.length - 1]
    isAtBottom = last.isIntersecting
    if (isAtBottom) {
        unreadCount = 0
    }
    updateScrollDownUI()
}

function updateScrollDownUI() {
    scrollDownBtn.hidden = isAtBottom
    if (unreadCount > 0) {
        scrollDownBadge.hidden = false
        scrollDownBadge.textContent = unreadCount > 99 ? '99+' : String(unreadCount)
    } else {
        scrollDownBadge.hidden = true
    }
}

function reobserveLastMessage() {
    if (!lastMessageObserver) return
    lastMessageObserver.disconnect()
    const bubbles = messagesListEl.querySelectorAll('.message-bubble')
    const last = bubbles[bubbles.length - 1]
    if (last) lastMessageObserver.observe(last)
}

function onScrollDownClick() {
    messagesListEl.scrollTo({ top: messagesListEl.scrollHeight, behavior: 'smooth' })
}

function resetScrollDownState() {
    unreadCount = 0
    isAtBottom = true
    updateScrollDownUI()
    if (lastMessageObserver) lastMessageObserver.disconnect()
}
```

**Точки интеграции:**

| Существующая функция | Изменение |
|----------------------|-----------|
| `renderMessages()` (после полной перерисовки списка) | в конце: `reobserveLastMessage()` |
| `appendMessage(message)` (если такая есть) или место, где добавляется новый bubble в DOM | в конце: `reobserveLastMessage()` |
| `handleNewMessage(message)` (WS-обработчик нового сообщения) | См. ниже |
| Обработчик удаления сообщения (`handleMessageDeleted` / DOM-удаление) | в конце: `reobserveLastMessage()` |
| `closeChat()` / переключение чата | `resetScrollDownState()` |
| `selectChat()` после загрузки сообщений | `reobserveLastMessage()` после первого `scrollToBottom()` |

**Изменение `handleNewMessage`:**

Текущая логика (упрощённо): добавить в DOM → `scrollToBottom()`.

Новая логика:
```js
function handleNewMessage(message) {
    // ...существующая проверка чата, дедупликация, toast для неактивного чата...

    if (currentChatId !== message.chatId) {
        // toast — без изменений
        return
    }

    const isOwn = message.sender.id === currentUser.id
    const wasAtBottom = isAtBottom

    appendMessageToDOM(message)
    reobserveLastMessage()

    if (isOwn || wasAtBottom) {
        scrollToBottom() // мгновенно, как сейчас
    } else {
        unreadCount += 1
        updateScrollDownUI()
    }
}
```

Имя `appendMessageToDOM` — placeholder, реальное название уточняется по коду на этапе плана.

### 3.4 doc.md обновления

- `internal/controllers/http/handlers/web/templates/doc.md` — добавить упоминание `#btn-scroll-down` в `chat.html`.
- `internal/controllers/http/handlers/web/static/css/doc.md` — добавить блок про `.conversation__scroll-down*`.
- `internal/controllers/http/handlers/web/static/js/doc.md` — добавить блок про IntersectionObserver-логику и поведение follow mode.

---

## 4. Десктоп GUI (KhodFeltsChatGUI)

### 4.1 Шаблон — `frontend/src/components/ChatView/ChatView.vue`

Кнопка размещается **внутри `<div class="conversation__composer">`** первым дочерним элементом — позиционирование `position: absolute; bottom: 100%` относительно composer. Это даёт стабильную точку привязки к верхнему краю composer независимо от наличия reply-bar/edit-bar.

```vue
<div class="conversation__composer">
  <button
      v-if="!isAtBottom"
      type="button"
      class="conversation__scroll-down"
      @click="onScrollDownClick"
      aria-label="К последнему сообщению"
  >
    <svg class="conversation__scroll-down-icon" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <polyline points="6 9 12 15 18 9"/>
    </svg>
    <span v-if="unreadCount > 0" class="conversation__scroll-down-badge">
      {{ unreadCount > 99 ? '99+' : unreadCount }}
    </span>
  </button>
  <!-- остальной composer без изменений: edit-bar / reply-bar / composer-row -->
</div>
```

### 4.2 Скрипт — `frontend/src/components/ChatView/ChatView.js`

**Новые refs:**
```js
const unreadCount = ref(0)
const isAtBottom = ref(true)
let lastMessageObserver = null
```

**Создание observer (один раз, в `onMounted` после получения `messagesListRef`):**

Поскольку `messagesListRef` может быть `null` при первом маунте (показывается `v-if="selectedChat"`), правильнее создать observer в `watch(messagesListRef, ...)`:

```js
watch(messagesListRef, (el, _, onCleanup) => {
    if (!el) {
        if (lastMessageObserver) {
            lastMessageObserver.disconnect()
            lastMessageObserver = null
        }
        return
    }

    lastMessageObserver = new IntersectionObserver((entries) => {
        const last = entries[entries.length - 1]
        isAtBottom.value = last.isIntersecting
        if (isAtBottom.value) unreadCount.value = 0
    }, { root: el, threshold: 0.1 })

    onCleanup(() => {
        if (lastMessageObserver) {
            lastMessageObserver.disconnect()
            lastMessageObserver = null
        }
    })

    // ...существующий scroll handler для пагинации остаётся...
})
```

**Переподписка на последний bubble через `watch(messages, ...)`:**

```js
watch(messages, async () => {
    await nextTick()
    if (!lastMessageObserver || !messagesListRef.value) return
    const bubbles = messagesListRef.value.querySelectorAll('.message-bubble')
    const last = bubbles[bubbles.length - 1]
    lastMessageObserver.disconnect()
    if (last) lastMessageObserver.observe(last)
}, { flush: 'post', deep: false })
```

`deep: false` — нам нужны только изменения длины/идентичности массива, не вложенных полей сообщений.

**Изменение `handleNewMessage`:**

```js
const handleNewMessage = async (message) => {
    try {
        if (!isWindowFocused && (webPushConsents.value & CONSENT_NEW_MESSAGE) !== 0) {
            ShowNotification(message.sender.username, message.text, message.chatId)
                .catch(err => console.error('Ошибка системного уведомления:', err))
        }

        if (selectedChat.value?.id === message.chatId) {
            const isOwn = message.sender.id === currentUser.value?.id
            const wasAtBottom = isAtBottom.value

            messages.value.push({...message, isRead: false})
            await nextTick()

            if (isOwn || wasAtBottom) {
                scrollToBottom() // мгновенно
            } else {
                unreadCount.value += 1
            }
        } else {
            emit('new-message-notification', {
                sender: message.sender.username,
                text: message.text,
                chatId: message.chatId,
            })
        }

        loadChats().catch(err => console.error("Фоновое обновление чатов не удалось:", err))
    } catch (err) {
        console.error("Ошибка обработки нового сообщения:", err)
    }
}
```

**Новая функция:**
```js
const onScrollDownClick = () => {
    if (messagesListRef.value) {
        messagesListRef.value.scrollTo({
            top: messagesListRef.value.scrollHeight,
            behavior: 'smooth',
        })
    }
}
```

**Сброс при закрытии чата:**

В `closeChat()` добавить:
```js
unreadCount.value = 0
isAtBottom.value = true
```

**Возвращаем из `setup()`:**

Добавить в return: `isAtBottom`, `unreadCount`, `onScrollDownClick`.

### 4.3 Стили — `frontend/src/components/ChatView/ChatView.css`

Те же CSS-правила, что и в web (см. 3.2): `.conversation__composer { position: relative }`, `.conversation__scroll-down { position: absolute; bottom: 100%; right: 16px; margin-bottom: 12px; ... }`, `.conversation__scroll-down-badge { ... }`.

Имена дизайн-токенов в GUI отличаются (используются `--accent`, `--space-md`, `--transition-base` — есть в `frontend/src/styles/global.css`). При реализации проверить наличие `--danger`; если его нет, использовать локальный fallback (`#e53935`).

### 4.4 doc.md — `frontend/src/components/ChatView/doc.md`

Добавить:
- В таблицу refs: `unreadCount`, `isAtBottom`.
- В таблицу функций: `onScrollDownClick()`, описание поведения follow mode в `handleNewMessage`.
- Новый раздел «UI: кнопка возврата к последнему сообщению» с кратким описанием.

---

## 5. Что НЕ делаем

- Не добавляем серверную логику unread-счётчиков. Бейдж считает локально с момента, когда пользователь отскроллил вверх в открытом чате. Это не то же самое, что глобальный непрочитанный счётчик чата (`chat.isRead`) — оставляем как есть.
- Не меняем существующие `scrollToBottom()` (мгновенный). Smooth — только для клика по новой кнопке.
- Не делаем «двухшаговый» скролл (мгновенный подскок + smooth). Если потребуется по факту тестирования — добавим отдельно.
- Не трогаем пагинацию (`loadMoreMessages` при скролле вверх).
- Не трогаем `scrollToMessage()` (переход по reply) — он уже использует `scrollIntoView({behavior: 'smooth'})` и хайлайт.
- Бэкенд не меняется.

---

## 6. Тестирование

### 6.1 Ручное (web + GUI)

| Сценарий | Ожидание |
|----------|----------|
| Открыл чат с сообщениями | Кнопка скрыта, скролл внизу |
| Прокрутил вверх на половину экрана | Кнопка появилась, бейджа нет |
| Прокрутил вниз обратно | Кнопка исчезла |
| Прокрутил вверх, пришло 3 сообщения | Кнопка с бейджем «3», скролл не сдвинулся |
| Кликнул кнопку | Плавный скролл вниз, кнопка и бейдж скрылись |
| Прокрутил вверх, пришло >99 сообщений | Бейдж показывает «99+» |
| Прокрутил вверх, отправил своё сообщение | Скролл вниз, бейдж сброшен |
| Прокрутил вверх, пришло сообщение в другой чат | Бейдж не меняется, кнопка не дёргается |
| Закрыл чат, открыл снова | Состояние сброшено, скролл внизу |
| Прокрутил вверх, прокрутил полностью вниз (доскролл) | Бейдж сбросился, кнопка скрылась |
| Mobile viewport (web): прокрутил выше последнего сообщения на одно сообщение | Кнопка появилась |

### 6.2 Unit-тесты

Для текущей фронтенд-кодобазы нет фреймворка unit-тестов JS (web — чистый JS, GUI — Vue без Vitest). Тесты не пишем. Если в будущем появится тестовый фреймворк, кандидаты:
- Логика `updateScrollDownUI(unreadCount)` — корректность формата «99+».
- Логика follow mode (mock observer state, проверить вызов `scrollToBottom` vs `unreadCount++`).

Go-кода фича не касается — backend-тесты не добавляются.

---

## 7. План работ (черновик; финальный план — отдельным документом)

1. **KhodFeltsChat web:**
   - Разметка кнопки в `chat.html`.
   - Стили в `chat.css`.
   - Логика в `chat.js`: observer, `handleNewMessage`, точки интеграции.
   - Обновить 3 `doc.md`.
   - Ручной прогон сценариев из 6.1.

2. **KhodFeltsChatGUI:**
   - Разметка кнопки в `ChatView.vue`.
   - Стили в `ChatView.css`.
   - Логика в `ChatView.js`: refs, observer, watches, `handleNewMessage`, `closeChat`.
   - Обновить `doc.md`.
   - Ручной прогон сценариев из 6.1.

Порядок — последовательный (web → GUI), чтобы можно было верифицировать паттерн на одном фронте перед переносом.
