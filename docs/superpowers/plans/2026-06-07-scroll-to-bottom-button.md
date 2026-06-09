# Scroll-to-Bottom Button with Unread Counter — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить полупрозрачную кнопку «к последнему сообщению» в области чата (web + desktop GUI). Кнопка появляется, когда последнее сообщение вне видимой области; при входящем сообщении в этом режиме счётчик инкрементируется вместо автоскролла. Существующий мгновенный скролл сохраняется при открытии чата и отправке своего сообщения; при клике по кнопке — smooth scroll.

**Architecture:** На обоих фронтендах одинаковый паттерн: `IntersectionObserver` отслеживает последний `.message-bubble`. Состояние `isAtBottom` гейтит follow-mode при входящих сообщениях. Бэкенд не меняется.

**Tech Stack:** Vanilla JS + HTML/CSS (web), Vue 3 Options API с `setup()` + CSS (Wails GUI).

**Spec:** `docs/superpowers/specs/2026-06-07-scroll-to-bottom-button-design.md`

**Repositories:**
- `KhodFeltsChat` (этот репозиторий) — web-клиент
- `KhodFeltsChatGUI` (`/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI`) — Wails desktop GUI

---

## File Map

### Web (KhodFeltsChat)

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/controllers/http/handlers/web/templates/chat.html` | Добавить разметку кнопки + бейджа |
| Modify | `internal/controllers/http/handlers/web/static/css/chat.css` | Стили кнопки + бейджа, `position: relative` на composer |
| Modify | `internal/controllers/http/handlers/web/static/js/chat.js` | Состояние, IntersectionObserver, follow mode, click handler |
| Modify | `internal/controllers/http/handlers/web/templates/doc.md` | Упомянуть новый элемент |
| Modify | `internal/controllers/http/handlers/web/static/css/doc.md` | Упомянуть новые классы |
| Modify | `internal/controllers/http/handlers/web/static/js/doc.md` | Упомянуть scroll-down логику |

### Desktop GUI (KhodFeltsChatGUI)

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `frontend/src/components/ChatView/ChatView.vue` | Разметка кнопки |
| Modify | `frontend/src/components/ChatView/ChatView.css` | Стили кнопки + бейджа |
| Modify | `frontend/src/components/ChatView/ChatView.js` | Refs, observer, follow mode, click handler |
| Modify | `frontend/src/components/ChatView/doc.md` | Описать новое состояние и функции |

---

## Phase 1: Web Client (KhodFeltsChat)

### Task 1: Разметка кнопки в `chat.html`

**Files:**
- Modify: `internal/controllers/http/handlers/web/templates/chat.html`

- [ ] **Step 1: Открыть файл и найти `<div class="conversation__composer">`** (строка ~47)

- [ ] **Step 2: Добавить кнопку первым дочерним элементом composer**

Заменить:
```html
<div class="conversation__composer">
    <div class="conversation__reply-bar" id="reply-bar" style="display: none;">
```

На:
```html
<div class="conversation__composer">
    <button class="conversation__scroll-down" id="btn-scroll-down" hidden type="button" aria-label="К последнему сообщению">
        <svg class="conversation__scroll-down-icon" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="6 9 12 15 18 9"/>
        </svg>
        <span class="conversation__scroll-down-badge" id="scroll-down-badge" hidden>0</span>
    </button>
    <div class="conversation__reply-bar" id="reply-bar" style="display: none;">
```

- [ ] **Step 3: Не коммитим — кнопка не видна без CSS (атрибут `hidden`). Переходим к Task 2.**

---

### Task 2: Стили кнопки в `chat.css`

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/css/chat.css`

- [ ] **Step 1: Найти селектор `.conversation__composer`** (строка ~306 в файле; ориентир — комментарий `Ввод сообщения`)

- [ ] **Step 2: Добавить `position: relative` в `.conversation__composer`**

Найти блок (примерно так):
```css
.conversation__composer {
    padding: var(--space-lg);
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 0;
    background: var(--bg-surface);
}
```

Добавить в него строку `position: relative;`:
```css
.conversation__composer {
    padding: var(--space-lg);
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 0;
    background: var(--bg-surface);
    position: relative;
}
```

- [ ] **Step 3: Добавить новые блоки стилей в конец файла**

В самый конец `chat.css` добавить:

```css

/* ═══════════════════════════════════════
   Кнопка «к последнему сообщению»
   ═══════════════════════════════════════ */
.conversation__scroll-down {
    position: absolute;
    right: 16px;
    bottom: 100%;
    margin-bottom: 12px;
    width: 44px;
    height: 44px;
    border: none;
    border-radius: 50%;
    background: var(--accent);
    color: var(--text-on-accent);
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
    background: var(--danger);
    color: #fff;
    font-size: 11px;
    font-weight: 600;
    line-height: 20px;
    text-align: center;
    pointer-events: none;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

@media (max-width: 600px) {
    .conversation__scroll-down {
        width: 40px;
        height: 40px;
        right: 12px;
    }
}
```

- [ ] **Step 4: Запустить локально и убедиться что в DevTools для `<button id="btn-scroll-down">` стили подхватились (кнопка скрыта из-за `hidden`, но стили в Computed должны быть применимы)**

Запустить:
```bash
task local
```

В браузере открыть `http://localhost:8080`, авторизоваться, открыть любой чат. DevTools → найти `#btn-scroll-down` → в Computed убедиться, что `position: absolute`, `border-radius: 50%`.

- [ ] **Step 5: Не коммитим — переходим к Task 3, чтобы коммит был осмысленным (разметка+стили+логика в одном).**

---

### Task 3: Состояние, observer и UI-хелперы в `chat.js`

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`

- [ ] **Step 1: Добавить модульные переменные**

Найти блок объявлений переменных в начале файла (строки ~13-25, начинается с `let currentUser = null;`).

После строки `let contextMenuInputWasFocused = false; // был ли input в фокусе при открытии меню` (строка ~25) добавить:

```javascript

// Состояние кнопки "к последнему сообщению":
let unreadCount = 0;
let isAtBottom = true;
let lastMessageObserver = null;
```

- [ ] **Step 2: Добавить helper-функции после `scrollToBottom()`**

Найти функцию `scrollToBottom()` (строка ~625):

```javascript
function scrollToBottom() {
    requestAnimationFrame(() => {
        const container = document.getElementById('messages-list');
        container.scrollTop = container.scrollHeight;
    });
}
```

Сразу после неё (перед `function formatTime`) добавить:

```javascript

// ═══════════════════════════════════════
// Кнопка "к последнему сообщению"
// ═══════════════════════════════════════
function setupScrollDownButton() {
    const btn = document.getElementById('btn-scroll-down');
    if (btn) {
        btn.addEventListener('click', onScrollDownClick);
    }

    const container = document.getElementById('messages-list');
    lastMessageObserver = new IntersectionObserver(onLastMessageVisibilityChange, {
        root: container,
        threshold: 0.1,
    });
}

function onLastMessageVisibilityChange(entries) {
    const last = entries[entries.length - 1];
    isAtBottom = last.isIntersecting;
    if (isAtBottom) {
        unreadCount = 0;
    }
    updateScrollDownUI();
}

function updateScrollDownUI() {
    const btn = document.getElementById('btn-scroll-down');
    const badge = document.getElementById('scroll-down-badge');
    if (!btn || !badge) return;

    btn.hidden = isAtBottom;

    if (unreadCount > 0) {
        badge.hidden = false;
        badge.textContent = unreadCount > 99 ? '99+' : String(unreadCount);
    } else {
        badge.hidden = true;
    }
}

function reobserveLastMessage() {
    if (!lastMessageObserver) return;
    lastMessageObserver.disconnect();
    const container = document.getElementById('messages-list');
    const bubbles = container.querySelectorAll('.message-bubble');
    const last = bubbles[bubbles.length - 1];
    if (last) lastMessageObserver.observe(last);
}

function resetScrollDownState() {
    unreadCount = 0;
    isAtBottom = true;
    if (lastMessageObserver) lastMessageObserver.disconnect();
    updateScrollDownUI();
}

function onScrollDownClick() {
    const container = document.getElementById('messages-list');
    container.scrollTo({ top: container.scrollHeight, behavior: 'smooth' });
}
```

- [ ] **Step 3: Зарегистрировать setup в `DOMContentLoaded`**

Найти `DOMContentLoaded` handler (строка ~30). После строки `setupReplyCancel();` (строка ~54) добавить:

```javascript
    setupScrollDownButton();
```

Итоговый фрагмент должен выглядеть:
```javascript
    setupContextMenu();
    setupReplyCancel();
    setupScrollDownButton();
```

- [ ] **Step 4: Промежуточный прогон**

```bash
task local
```

В браузере открыть чат. Кнопка `#btn-scroll-down` всё ещё `hidden` (потому что `isAtBottom = true` по умолчанию, observer ни на что не подписан). Никаких ошибок в консоли. Переходим к Task 4, который подключит observer к жизненному циклу чата.

- [ ] **Step 5: Не коммитим — продолжаем.**

---

### Task 4: Интеграция в selectChat / renderMessages / handleMessageDeleted

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`

- [ ] **Step 1: Добавить вызов `reobserveLastMessage()` в `selectChat()` после первого скролла**

Найти `selectChat()` (строка ~405). В нём строки 423-424:
```javascript
    await loadMessages(chat.id, 0);
    scrollToBottom();
```

Заменить на:
```javascript
    await loadMessages(chat.id, 0);
    scrollToBottom();
    resetScrollDownState();
    reobserveLastMessage();
```

`resetScrollDownState()` гарантирует чистое состояние (на случай переключения чатов), а `reobserveLastMessage()` подписывает observer на последний bubble нового чата.

- [ ] **Step 2: Добавить `reobserveLastMessage()` в `appendMessageBubble()`**

Найти `appendMessageBubble()` (строка ~518). Заменить тело функции:

Было:
```javascript
function appendMessageBubble(message) {
    const container = document.getElementById('messages-list');
    const index = messages.indexOf(message);

    if (index >= 0 && isFirstUnread(message, index)) {
        const divider = document.createElement('div');
        divider.className = 'conversation__unread-divider';
        divider.innerHTML = '<span>Новые сообщения</span>';
        container.appendChild(divider);
    }

    container.appendChild(createMessageBubble(message));
}
```

Стало:
```javascript
function appendMessageBubble(message) {
    const container = document.getElementById('messages-list');
    const index = messages.indexOf(message);

    if (index >= 0 && isFirstUnread(message, index)) {
        const divider = document.createElement('div');
        divider.className = 'conversation__unread-divider';
        divider.innerHTML = '<span>Новые сообщения</span>';
        container.appendChild(divider);
    }

    container.appendChild(createMessageBubble(message));
    reobserveLastMessage();
}
```

- [ ] **Step 3: Добавить `reobserveLastMessage()` в `handleMessageDeleted()`**

Найти `handleMessageDeleted()` (строка ~189). Внутри `if (selectedChatId === payload.chatId)`, после `updateUnreadDivider();` (строка ~196), добавить:

Было:
```javascript
async function handleMessageDeleted(payload) {
    if (selectedChatId === payload.chatId) {
        const idx = messages.findIndex(m => m.id === payload.messageId);
        if (idx >= 0) {
            messages.splice(idx, 1);
            const bubble = document.querySelector(`.message-bubble[data-message-id="${payload.messageId}"]`);
            if (bubble) bubble.remove();
            updateUnreadDivider();
        } else {
            console.warn('message_deleted: сообщение не найдено в текущем списке', payload.messageId);
        }
    }

    debouncedLoadChats();
}
```

Стало:
```javascript
async function handleMessageDeleted(payload) {
    if (selectedChatId === payload.chatId) {
        const idx = messages.findIndex(m => m.id === payload.messageId);
        if (idx >= 0) {
            messages.splice(idx, 1);
            const bubble = document.querySelector(`.message-bubble[data-message-id="${payload.messageId}"]`);
            if (bubble) bubble.remove();
            updateUnreadDivider();
            reobserveLastMessage();
        } else {
            console.warn('message_deleted: сообщение не найдено в текущем списке', payload.messageId);
        }
    }

    debouncedLoadChats();
}
```

`reobserveLastMessage()` нужен, если удалили именно последнее сообщение — observer переподпишется на новое последнее.

- [ ] **Step 4: Добавить сброс в `closeChat()`**

Найти `closeChat()` (строка ~886). Заменить тело:

Было:
```javascript
function closeChat() {
    selectedChatId = null;
    selectedChat = null;
    document.getElementById('conversation').style.display = 'none';
    document.getElementById('conversation-placeholder').style.display = '';
    document.querySelector('.chat-layout').classList.remove('chat-layout--chat-open');

    // Закрываем emoji picker:
    document.getElementById('emoji-picker-container').style.display = 'none';
    document.getElementById('btn-emoji-toggle').classList.remove('conversation__emoji-toggle--active');

    // Сбрасываем ответ:
    cancelReply();
}
```

Стало:
```javascript
function closeChat() {
    selectedChatId = null;
    selectedChat = null;
    document.getElementById('conversation').style.display = 'none';
    document.getElementById('conversation-placeholder').style.display = '';
    document.querySelector('.chat-layout').classList.remove('chat-layout--chat-open');

    // Закрываем emoji picker:
    document.getElementById('emoji-picker-container').style.display = 'none';
    document.getElementById('btn-emoji-toggle').classList.remove('conversation__emoji-toggle--active');

    // Сбрасываем ответ:
    cancelReply();

    // Сбрасываем состояние scroll-down:
    resetScrollDownState();
}
```

- [ ] **Step 5: Не коммитим — переходим к Task 5 (главное изменение поведения).**

---

### Task 5: Follow mode в `handleNewMessage()`

**Files:**
- Modify: `internal/controllers/http/handlers/web/static/js/chat.js`

- [ ] **Step 1: Заменить тело `handleNewMessage()`**

Найти `handleNewMessage()` (строка ~172):

Было:
```javascript
async function handleNewMessage(message) {
    if (selectedChatId === message.chatId) {
        messages.push(message);
        appendMessageBubble(message);
        scrollToBottom();
    }

    // Показываем toast, если сообщение не от текущего пользователя
    // и чат не открыт (или это другой чат):
    if (message.sender.id !== currentUser.id && selectedChatId !== message.chatId) {
        showToast(message.sender.username, message.text, message.chatId);
    }

    // Обновляем список чатов (непрочитанное):
    debouncedLoadChats();
}
```

Стало:
```javascript
async function handleNewMessage(message) {
    if (selectedChatId === message.chatId) {
        const isOwn = message.sender.id === currentUser.id;
        const wasAtBottom = isAtBottom;

        messages.push(message);
        appendMessageBubble(message); // внутри уже reobserveLastMessage()

        if (isOwn || wasAtBottom) {
            scrollToBottom();
        } else {
            unreadCount += 1;
            updateScrollDownUI();
        }
    }

    // Показываем toast, если сообщение не от текущего пользователя
    // и чат не открыт (или это другой чат):
    if (message.sender.id !== currentUser.id && selectedChatId !== message.chatId) {
        showToast(message.sender.username, message.text, message.chatId);
    }

    // Обновляем список чатов (непрочитанное):
    debouncedLoadChats();
}
```

Ключевое:
- Снимок `wasAtBottom` берётся ДО `appendMessageBubble`, т.к. после добавления DOM-элемента observer ещё не успеет среагировать.
- Своё сообщение скроллит вниз всегда (ожидаемый UX).
- Чужое в режиме follow (`wasAtBottom`) — скроллит.
- Чужое не в follow — счётчик.

- [ ] **Step 2: Запустить локально**

```bash
task local
```

- [ ] **Step 3: Ручная проверка сценариев из спеки §6.1**

Открыть `http://localhost:8080` в двух браузерах (или incognito + обычный) с двумя разными аккаунтами в общем чате.

| Сценарий | Ожидание |
|----------|----------|
| Открыл чат с >50 сообщениями | Скролл внизу, кнопка скрыта |
| Прокрутил вверх на половину экрана | Кнопка появилась, бейджа нет |
| Прокрутил вниз обратно | Кнопка исчезла |
| Прокрутил вверх. Из второго браузера прислали 3 сообщения | Кнопка с бейджем «3», скролл не сдвинулся |
| Кликнул кнопку | Плавный скролл вниз, кнопка и бейдж скрылись |
| Прокрутил вверх. Прислал >99 сообщений из второго браузера (можно зажать enter; для скорости — отправлять короткие) | Бейдж показывает «99+» |
| Прокрутил вверх, отправил своё сообщение | Скролл вниз, бейдж сброшен, кнопка скрыта |
| Прокрутил вверх. Из второго браузера прислали в ДРУГОЙ чат | Бейдж/кнопка не дёргается. Toast отображается |
| Закрыл чат через X, открыл снова | Скролл внизу, кнопка скрыта, бейджа нет |
| DevTools: Toggle device toolbar → iPhone | Кнопка 40×40, отступ 12px |

- [ ] **Step 4: Если все сценарии прошли — переходим к Task 6 (документация).**

---

### Task 6: Обновить `doc.md` web-клиента

**Files:**
- Modify: `internal/controllers/http/handlers/web/templates/doc.md`
- Modify: `internal/controllers/http/handlers/web/static/css/doc.md`
- Modify: `internal/controllers/http/handlers/web/static/js/doc.md`

- [ ] **Step 1: Прочитать текущее содержимое каждого `doc.md`**

```bash
cat /Users/dskhorkov/GolandProjects/KhodFeltsChat/internal/controllers/http/handlers/web/templates/doc.md
cat /Users/dskhorkov/GolandProjects/KhodFeltsChat/internal/controllers/http/handlers/web/static/css/doc.md
cat /Users/dskhorkov/GolandProjects/KhodFeltsChat/internal/controllers/http/handlers/web/static/js/doc.md
```

- [ ] **Step 2: Шаблон — `templates/doc.md`**

Найти раздел или таблицу, описывающий элементы `chat.html`. Добавить строку про новый `#btn-scroll-down`:

> `#btn-scroll-down` — круглая кнопка с inline SVG «↓» внутри composer, появляется при отскролле от низа. Содержит `#scroll-down-badge` (счётчик непрочитанных в текущем чате).

Если в `templates/doc.md` нет описания элементов по ID — добавить короткий раздел «Кнопка возврата к последнему сообщению» с тем же текстом.

- [ ] **Step 3: CSS — `static/css/doc.md`**

Добавить пункт о новых классах. Один абзац:

> `.conversation__scroll-down` — полупрозрачная круглая кнопка «к последнему сообщению» (44px / 40px на mobile), позиционирована абсолютно от верхнего края composer (`bottom: 100% + margin-bottom`). При hover становится непрозрачной. `.conversation__scroll-down-icon` — SVG-стрелка. `.conversation__scroll-down-badge` — красный бейдж со счётчиком в правом верхнем углу.

- [ ] **Step 4: JS — `static/js/doc.md`**

Добавить пункт. Один абзац:

> **Кнопка «к последнему сообщению»**: модульное состояние `unreadCount`, `isAtBottom`, `lastMessageObserver`. `IntersectionObserver` на последнем `.message-bubble` (порог 0.1) управляет `isAtBottom`. При входящем сообщении в открытом чате: если своё сообщение или пользователь был внизу — `scrollToBottom()`, иначе `unreadCount++`. Клик по кнопке делает smooth scroll к низу. Функции: `setupScrollDownButton()`, `onLastMessageVisibilityChange()`, `updateScrollDownUI()`, `reobserveLastMessage()`, `resetScrollDownState()`, `onScrollDownClick()`. Точки переподписки: `selectChat()` после загрузки, `appendMessageBubble()`, `handleMessageDeleted()`. Сброс в `closeChat()`.

- [ ] **Step 5: Перечитать все три обновлённых файла и убедиться, что добавленный текст вписался в существующий стиль (заголовки, форматирование).**

- [ ] **Step 6: Не коммитим — Task 7 = commit всей фазы.**

---

### Task 7: Финальный коммит web-фазы

**Files:** уже изменены.

- [ ] **Step 1: Проверить git status и diff**

```bash
git status
git diff --stat
```

Ожидаемый список изменённых файлов:
- `internal/controllers/http/handlers/web/templates/chat.html`
- `internal/controllers/http/handlers/web/templates/doc.md`
- `internal/controllers/http/handlers/web/static/css/chat.css`
- `internal/controllers/http/handlers/web/static/css/doc.md`
- `internal/controllers/http/handlers/web/static/js/chat.js`
- `internal/controllers/http/handlers/web/static/js/doc.md`

- [ ] **Step 2: НЕ КОММИТИМ автоматически. Уведомить пользователя: «Web-фаза готова. Изменения видны в IDE. Коммит делает пользователь сам (правило из памяти).»**

Если пользователь явно просит закоммитить — использовать формат:

```bash
git add internal/controllers/http/handlers/web/templates/chat.html \
        internal/controllers/http/handlers/web/templates/doc.md \
        internal/controllers/http/handlers/web/static/css/chat.css \
        internal/controllers/http/handlers/web/static/css/doc.md \
        internal/controllers/http/handlers/web/static/js/chat.js \
        internal/controllers/http/handlers/web/static/js/doc.md
git commit -m "$(cat <<'EOF'
add scroll-to-bottom button with unread counter (web)

Кнопка появляется при отскролле вверх. При входящем сообщении в этом
режиме счётчик инкрементируется без автоскролла. Открытие чата и
отправка своего сообщения по-прежнему скроллят вниз мгновенно.
EOF
)"
```

---

## Phase 2: Desktop GUI (KhodFeltsChatGUI)

> **Working directory для всех задач Phase 2:** `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI`

### Task 8: Разметка кнопки в `ChatView.vue`

**Files:**
- Modify: `frontend/src/components/ChatView/ChatView.vue`

- [ ] **Step 1: Найти `<div class="conversation__composer">`** (строка ~110)

- [ ] **Step 2: Добавить кнопку первым дочерним элементом composer**

Найти:
```vue
        <div class="conversation__composer">
          <div v-if="editingMessage" class="conversation__edit-bar">
```

Заменить на:
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
          <div v-if="editingMessage" class="conversation__edit-bar">
```

- [ ] **Step 3: Пока без коммита — переходим к Task 9 (стили и JS будут вместе).**

---

### Task 9: Стили кнопки в `ChatView.css`

**Files:**
- Modify: `frontend/src/components/ChatView/ChatView.css`

- [ ] **Step 1: Найти селектор `.conversation__composer`** (строка ~306)

Селектор уже имеет `position: relative` неявно? Проверим: текущий код:
```css
.conversation__composer {
    padding: var(--space-lg);
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 0;
    background: var(--bg-surface);
}
```

`position: relative` не задан — добавляем.

- [ ] **Step 2: Добавить `position: relative` в `.conversation__composer`**

Заменить блок выше на:
```css
.conversation__composer {
    padding: var(--space-lg);
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 0;
    background: var(--bg-surface);
    position: relative;
}
```

- [ ] **Step 3: Добавить новые блоки стилей в конец файла**

В конец `ChatView.css` добавить:

```css

/* ═══════════════════════════════════════
   Кнопка «к последнему сообщению»
   ═══════════════════════════════════════ */
.conversation__scroll-down {
    position: absolute;
    right: 16px;
    bottom: 100%;
    margin-bottom: 12px;
    width: 44px;
    height: 44px;
    border: none;
    border-radius: 50%;
    background: var(--accent);
    color: var(--text-on-accent);
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
    background: var(--danger);
    color: #fff;
    font-size: 11px;
    font-weight: 600;
    line-height: 20px;
    text-align: center;
    pointer-events: none;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}
```

- [ ] **Step 4: Не коммитим — Task 10 добавит логику и кнопка станет рабочей.**

---

### Task 10: Состояние, observer и refs в `ChatView.js`

**Files:**
- Modify: `frontend/src/components/ChatView/ChatView.js`

- [ ] **Step 1: Добавить refs**

Найти блок объявлений refs (строки ~31-59, начинается с `const chats = ref([])`).

Сразу после `const editingMessage = ref(null)` (строка ~58, перед `const contextMenu = ref(...)` ) добавить:

```javascript
        const unreadCount = ref(0)
        const isAtBottom = ref(true)
        let lastMessageObserver = null
```

- [ ] **Step 2: Добавить функции — `onScrollDownClick` и сброс состояния**

Найти `scrollToBottom` (строка ~221):

```javascript
        const scrollToBottom = () => {
            if (messagesListRef.value) {
                messagesListRef.value.scrollTop = messagesListRef.value.scrollHeight
            }
        }
```

Сразу после неё добавить:

```javascript

        const onScrollDownClick = () => {
            if (messagesListRef.value) {
                messagesListRef.value.scrollTo({
                    top: messagesListRef.value.scrollHeight,
                    behavior: 'smooth',
                })
            }
        }

        const resetScrollDownState = () => {
            unreadCount.value = 0
            isAtBottom.value = true
            if (lastMessageObserver) {
                lastMessageObserver.disconnect()
            }
        }
```

- [ ] **Step 3: Создавать observer в `watch(messagesListRef, ...)`**

Найти существующий `watch(messagesListRef, ...)` (строка ~493):

```javascript
        watch(messagesListRef, (el, _, onCleanup) => {
            if (!el) return

            scrollHandler = async () => {
                if (el.scrollTop <= 10 && !isLoadingMore && hasMoreMessages && selectedChat.value) {
                    const prevHeight = el.scrollHeight
                    await loadMoreMessages()
                    await nextTick()
                    el.scrollTop = el.scrollHeight - prevHeight
                }
            }

            el.addEventListener('scroll', scrollHandler)
            onCleanup(() => el.removeEventListener('scroll', scrollHandler))
        })
```

Заменить на:

```javascript
        watch(messagesListRef, (el, _, onCleanup) => {
            if (!el) {
                if (lastMessageObserver) {
                    lastMessageObserver.disconnect()
                    lastMessageObserver = null
                }
                return
            }

            scrollHandler = async () => {
                if (el.scrollTop <= 10 && !isLoadingMore && hasMoreMessages && selectedChat.value) {
                    const prevHeight = el.scrollHeight
                    await loadMoreMessages()
                    await nextTick()
                    el.scrollTop = el.scrollHeight - prevHeight
                }
            }

            el.addEventListener('scroll', scrollHandler)

            lastMessageObserver = new IntersectionObserver((entries) => {
                const last = entries[entries.length - 1]
                isAtBottom.value = last.isIntersecting
                if (isAtBottom.value) {
                    unreadCount.value = 0
                }
            }, { root: el, threshold: 0.1 })

            onCleanup(() => {
                el.removeEventListener('scroll', scrollHandler)
                if (lastMessageObserver) {
                    lastMessageObserver.disconnect()
                    lastMessageObserver = null
                }
            })
        })
```

- [ ] **Step 4: Переподписка observer при изменении `messages`**

Сразу после `watch(messagesListRef, ...)` блока (см. step 3) добавить:

```javascript

        watch(messages, async () => {
            await nextTick()
            if (!lastMessageObserver || !messagesListRef.value) return
            const bubbles = messagesListRef.value.querySelectorAll('.message-bubble')
            const last = bubbles[bubbles.length - 1]
            lastMessageObserver.disconnect()
            if (last) lastMessageObserver.observe(last)
        }, { flush: 'post', deep: false })
```

`deep: false` — реагируем только на изменения длины/идентичности массива, не на изменения вложенных полей сообщений (это исключит лишние срабатывания при `messages[idx].text = ...` в `handleMessageEdited`).

- [ ] **Step 5: Экспортировать новые символы из `setup()`**

Найти `return { ... }` в конце `setup()` (строка ~509). В список свойств добавить `isAtBottom`, `unreadCount`, `onScrollDownClick`.

Например, в конец списка перед `}`:
```javascript
            replyToMessage,
            editingMessage,
            highlightedMessageId,
            contextMenu,
            cancelReply,
            cancelEdit,
            editContextMessage,
            scrollToMessage,
            openContextMenu,
            replyToContextMessage,
            copyContextMessage,
            deleteContextMessage,
            isAtBottom,
            unreadCount,
            onScrollDownClick,
        }
```

- [ ] **Step 6: Запустить GUI и убедиться, что компонент компилируется**

```bash
wails dev
```

Ожидание: окно запустилось, чата открыли — кнопка скрыта (т.к. `isAtBottom = true` после загрузки). Открыли DevTools (Right-click → Inspect) → консоль без ошибок. Прокрутили вверх → кнопка появилась без бейджа. Прокрутили обратно вниз → исчезла.

- [ ] **Step 7: Не коммитим — Task 11 = главное изменение поведения.**

---

### Task 11: Follow mode в `handleNewMessage()` + сброс в `closeChat()`

**Files:**
- Modify: `frontend/src/components/ChatView/ChatView.js`

- [ ] **Step 1: Заменить `handleNewMessage()`**

Найти `handleNewMessage` (строка ~192):

Было:
```javascript
        const handleNewMessage = async (message) => {
            try {
                if (!isWindowFocused && (webPushConsents.value & CONSENT_NEW_MESSAGE) !== 0) {
                    ShowNotification(message.sender.username, message.text, message.chatId)
                        .catch(err => console.error('Ошибка системного уведомления:', err))
                }

                if (selectedChat.value?.id === message.chatId) {
                    messages.value.push({...message, isRead: false})
                    await nextTick()
                    scrollToBottom()
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

Стало:
```javascript
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
                        scrollToBottom()
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

- [ ] **Step 2: Добавить сброс в `closeChat()`**

Найти `closeChat()` (строка ~130):

Было:
```javascript
        const closeChat = () => {
            isEmojiPickerVisible.value = false
            selectedChat.value = null
            messages.value = []
            replyToMessage.value = null
            editingMessage.value = null
            contextMenu.value.visible = false
        }
```

Стало:
```javascript
        const closeChat = () => {
            isEmojiPickerVisible.value = false
            selectedChat.value = null
            messages.value = []
            replyToMessage.value = null
            editingMessage.value = null
            contextMenu.value.visible = false
            resetScrollDownState()
        }
```

- [ ] **Step 3: Перезапустить `wails dev` и прогнать сценарии**

В Wails окне (можно вторым окном открыть web-версию в браузере с другим пользователем, чтобы slать сообщения в общий чат):

| Сценарий | Ожидание |
|----------|----------|
| Открыл чат с >50 сообщениями | Скролл внизу, кнопка скрыта |
| Прокрутил вверх | Кнопка появилась |
| Прокрутил обратно вниз | Кнопка исчезла |
| Прокрутил вверх, пришли 3 сообщения от другого пользователя | Кнопка с бейджем «3» |
| Клик по кнопке | Smooth scroll вниз, кнопка/бейдж скрылись |
| Прокрутил вверх, прислано >99 сообщений | Бейдж «99+» |
| Прокрутил вверх, отправил своё | Скролл вниз, бейдж сброшен |
| Прокрутил вверх, удалили последнее сообщение из контекстного меню | Кнопка остаётся, observer переподписался (можно проверить: прокрутить вниз — кнопка исчезнет) |
| Прокрутил вверх, отредактировал текущее последнее сообщение | Состояние observer не меняется (флаг `deep: false`), кнопка остаётся видимой |
| Переключился в другой чат | Состояние сброшено |
| Закрыл чат (Esc), открыл снова | Состояние сброшено |

- [ ] **Step 4: Если всё работает — переходим к Task 12 (доки).**

---

### Task 12: Обновить `doc.md` GUI-компонента

**Files:**
- Modify: `frontend/src/components/ChatView/doc.md`

- [ ] **Step 1: Прочитать текущий doc.md**

```bash
cat /Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/components/ChatView/doc.md
```

- [ ] **Step 2: Добавить в таблицу refs**

Найти таблицу `## Ключевое состояние (refs)` и добавить в конец:

```markdown
| `unreadCount` | `number` | Счётчик новых сообщений с момента отскролла вверх (для бейджа на кнопке) |
| `isAtBottom` | `boolean` | Видно ли последнее сообщение чата (IntersectionObserver на последнем bubble) |
```

- [ ] **Step 3: Добавить в таблицу ключевых функций**

Найти `## Ключевые функции` и добавить:

```markdown
| `onScrollDownClick()` | Плавный скролл к последнему сообщению. Сброс счётчика происходит через IntersectionObserver, когда последнее сообщение становится видимым |
| `resetScrollDownState()` | Сброс `unreadCount = 0`, `isAtBottom = true` и `disconnect()` observer. Вызывается в `closeChat()` |
```

И обновить описание `handleNewMessage(msg)`:

Было (или похожее):
```markdown
| `handleNewMessage(msg)` | Обработчик Wails-события `new_message`: добавляет в открытый чат или показывает уведомление |
```

Стало:
```markdown
| `handleNewMessage(msg)` | Обработчик Wails-события `new_message`. В открытом чате: если своё сообщение или `isAtBottom` — мгновенный скролл вниз; иначе — `unreadCount++` без скролла. В неактивном чате — пуш-уведомление + toast через emit |
```

- [ ] **Step 4: Добавить новый раздел в конце документа**

```markdown
## UI: кнопка «к последнему сообщению»

- Появляется, когда последнее сообщение чата не видно полностью (`IntersectionObserver` на последнем `.message-bubble`, threshold 0.1).
- Полупрозрачная круглая кнопка 44px в правом нижнем углу области сообщений (позиционирована относительно `.conversation__composer` через `bottom: 100% + margin`).
- При входящем чужом сообщении в открытом чате и состоянии `isAtBottom === false` — автоскролл не выполняется, инкрементируется `unreadCount`. Бейдж: число до 99, дальше «99+».
- Клик: smooth scroll к низу через `scrollTo({ behavior: 'smooth' })`. Existing `scrollToBottom()` (мгновенный) сохраняется для открытия чата, отправки своего сообщения и follow-mode.
- Переподписка observer на новый последний bubble — через `watch(messages, ..., { flush: 'post', deep: false })`.
- CSS-классы: `.conversation__scroll-down`, `.conversation__scroll-down-icon`, `.conversation__scroll-down-badge`.
```

- [ ] **Step 5: Перечитать обновлённый doc.md и убедиться, что он целостный.**

```bash
cat /Users/dskhorkov/GolandProjects/KhodFeltsChatGUI/frontend/src/components/ChatView/doc.md
```

- [ ] **Step 6: Переходим к Task 13.**

---

### Task 13: Финальный коммит GUI-фазы

**Working directory:** `/Users/dskhorkov/GolandProjects/KhodFeltsChatGUI`

- [ ] **Step 1: Проверить git status и diff**

```bash
cd /Users/dskhorkov/GolandProjects/KhodFeltsChatGUI && git status && git diff --stat
```

Ожидаемый список изменённых файлов:
- `frontend/src/components/ChatView/ChatView.vue`
- `frontend/src/components/ChatView/ChatView.css`
- `frontend/src/components/ChatView/ChatView.js`
- `frontend/src/components/ChatView/doc.md`

- [ ] **Step 2: НЕ КОММИТИМ автоматически. Сообщить пользователю: «GUI-фаза готова. Коммит делает пользователь.»**

Если пользователь явно просит закоммитить:

```bash
cd /Users/dskhorkov/GolandProjects/KhodFeltsChatGUI && \
git add frontend/src/components/ChatView/ChatView.vue \
        frontend/src/components/ChatView/ChatView.css \
        frontend/src/components/ChatView/ChatView.js \
        frontend/src/components/ChatView/doc.md && \
git commit -m "$(cat <<'EOF'
add scroll-to-bottom button with unread counter

Кнопка появляется при отскролле вверх. При входящем сообщении в этом
режиме счётчик инкрементируется без автоскролла. Открытие чата и
отправка своего сообщения по-прежнему скроллят вниз мгновенно.
EOF
)"
```

---

## Self-review checklist

- [ ] Каждая задача имеет точные пути к файлам.
- [ ] Каждый шаг с кодом показывает полный код, не только описание.
- [ ] Каждая фаза заканчивается ручной верификацией по конкретным сценариям из спеки.
- [ ] Имена символов согласованы между задачами: `unreadCount`, `isAtBottom`, `lastMessageObserver`, `setupScrollDownButton`, `onLastMessageVisibilityChange`, `updateScrollDownUI`, `reobserveLastMessage`, `resetScrollDownState`, `onScrollDownClick`.
- [ ] Имена CSS-классов согласованы: `.conversation__scroll-down`, `.conversation__scroll-down-icon`, `.conversation__scroll-down-badge`.
- [ ] `--accent`, `--text-on-accent`, `--danger`, `--transition-base` есть в обоих `global.css` (проверено).
- [ ] Бэкенд не меняется — Go-тесты не добавляются.
- [ ] Покрытие спеки:
  - §1.1 состояния — Task 3 (web), Task 10 (GUI)
  - §1.2 сценарии — Task 5 (web), Task 11 (GUI)
  - §1.3 формат «99+» — `updateScrollDownUI` (Task 3) и шаблон (Task 8)
  - §2.1 IntersectionObserver — Task 3 (web), Task 10 (GUI)
  - §2.2 smooth scroll по клику — `onScrollDownClick` в Task 3 / Task 10
  - §2.3 follow mode — Task 5 / Task 11
  - §2.4 edge cases — `reobserveLastMessage` в Task 4 (delete) / `deep: false` watch в Task 10 (edit)
  - §3 web-разметка/стили/JS — Task 1-5
  - §4 GUI-разметка/стили/JS — Task 8-11
  - §6.1 ручное тестирование — встроено в Step "Ручная проверка" Task 5 и Task 11
- [ ] Коммиты — пользователь делает сам (правило из памяти `feedback_no_autocommit.md`); план только указывает, какие файлы стейджить, если пользователь попросит.
