const MESSAGES_PAGE_SIZE = 50;
const SEARCH_DEBOUNCE_MS = 300;
const CHAT_LIST_DEBOUNCE_MS = 300;
const CHAT_LIST_POLL_INTERVAL_MS = 5000;
const WS_RECONNECT_DELAY_MS = 3000;
const EMOJI_CLOSE_DELAY_MS = 500;
const LONG_PRESS_DELAY_MS = 500;
const TOAST_DURATION_MS = 3000;
const IS_MOBILE = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent) || window.innerWidth <= 600;
const SWIPE_CLOSE_THRESHOLD_RATIO = 0.5; // 50% ширины экрана для закрытия чата свайпом
const SWIPE_LOCK_ANGLE_PX = 15;          // минимальное смещение до определения направления свайпа

let currentUser = null;
let selectedChatId = null;
let selectedChat = null;
let messages = [];
let ws = null;
let isLoadingMore = false;
let hasMoreMessages = true;
let returnToGroupChat = null;
let chatsList = [];
let replyToMessage = null;  // сообщение, на которое отвечаем
let editingMessage = null;  // сообщение, которое редактируем
let contextMenuMessageId = null; // ID сообщения для контекстного меню
let contextMenuInputWasFocused = false; // был ли input в фокусе при открытии меню

// Состояние кнопки "к последнему сообщению":
// unreadMessageIds — набор ID сообщений, прилетевших пока пользователь был отскроллен вверх.
// Хранится как Set (а не число), чтобы при удалении сообщения «у всех» корректно вычесть его из badge.
let unreadMessageIds = new Set();
let isAtBottom = true;
let lastMessageObserver = null;

// ═══════════════════════════════════════
// Инициализация
// ═══════════════════════════════════════
document.addEventListener('DOMContentLoaded', async () => {
    currentUser = await loadCurrentUser();
    if (!currentUser) {
        window.location.href = '/web/login';
        return;
    }

    await loadChats();
    connectWebSocket();
    await initWebPushNotifications();

    setupSendMessage();
    setupEmojiPicker();
    setupCloseChat();
    setupSwipeToCloseChat();
    setupCreateChatModal();
    setupSearchUsersModal();
    setupMemberProfileModal();
    setupGroupChatModal();
    setupEscapeHandler();
    setupChatListDelegation();
    setupMobileKeyboardDismiss();
    setupMobileViewportResize();
    setupContextMenu();
    setupReplyCancel();
    setupScrollDownButton();

    // Открытие чата из push-уведомления (по query-параметру):
    const params = new URLSearchParams(window.location.search);
    const pushChatId = params.get('chatId');
    if (pushChatId && !isNaN(Number(pushChatId))) {
        await openChatById(Number(pushChatId));
        history.replaceState(null, '', window.location.pathname);
    }

    // Обработка postMessage от Service Worker (переход в чат по push-уведомлению):
    navigator.serviceWorker?.addEventListener('message', async (event) => {
        try {
            if (event.data?.type === 'open-chat' && event.data.chatId) {
                await openChatById(Number(event.data.chatId));
            }
        } catch (err) {
            console.error('Failed to open chat from push notification:', err);
        }
    });

    // Периодическое обновление списка чатов:
    const chatsPollingInterval = setInterval(loadChats, CHAT_LIST_POLL_INTERVAL_MS);
});

async function loadCurrentUser() {
    try {
        const resp = await fetchWithAuth('/api/users/me');
        if (!resp.ok) return null;
        return await resp.json();
    } catch (err) {
        console.error(err);

        return null;
    }
}

// ═══════════════════════════════════════
// Глобальный обработчик Escape
// ═══════════════════════════════════════
function setupEscapeHandler() {
    document.addEventListener('keydown', async (e) => {
        if (e.key !== 'Escape') return;

        // Закрываем модалки по приоритету (сверху вниз):
        const modals = [
            'modal-member-profile',
            'modal-group-chat',
            'modal-search-users',
            'modal-create-chat',
        ];

        for (const id of modals) {
            const modal = document.getElementById(id);
            if (modal && modal.style.display !== 'none') {
                modal.style.display = 'none';

                // Если закрыли профиль участника и нужно вернуться к модалке группового чата:
                if (id === 'modal-member-profile' && returnToGroupChat) {
                    showGroupChatInfo(returnToGroupChat);
                    returnToGroupChat = null;
                }

                return;
            }
        }

        // Сначала сбрасываем редактирование/ответ:
        if (editingMessage) {
            cancelEdit();
            return;
        }

        if (replyToMessage) {
            cancelReply();
            return;
        }

        // Если модалок нет — закрываем панель чата:
        if (selectedChatId) {
            closeChat();
            debouncedLoadChats();
        }
    });
}

// ═══════════════════════════════════════
// WebSocket
// ═══════════════════════════════════════
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(protocol + '//' + window.location.host + '/api/ws');

    ws.onmessage = (event) => {
        const envelope = JSON.parse(event.data);

        if (envelope.type === 'new_message') {
            handleNewMessage(envelope.payload).catch(err => console.error('handleNewMessage error:', err));
        } else if (envelope.type === 'message_deleted') {
            handleMessageDeleted(envelope.payload).catch(err => console.error('handleMessageDeleted error:', err));
        } else if (envelope.type === 'message_edited') {
            handleMessageEdited(envelope.payload).catch(err => console.error('handleMessageEdited error:', err));
        }
    };

    ws.onclose = () => {
        // Переподключение через 3 секунды:
        setTimeout(connectWebSocket, WS_RECONNECT_DELAY_MS);
    };

    ws.onerror = () => {
        // На iOS WebSocket может оборваться без onclose:
        if (ws.readyState !== WebSocket.CLOSED && ws.readyState !== WebSocket.CLOSING) {
            ws.close();
        }
    };
}

async function handleNewMessage(message) {
    if (selectedChatId === message.chatId) {
        const isOwn = message.sender.id === currentUser.id;
        const wasAtBottom = isAtBottom;

        messages.push(message);
        appendMessageBubble(message);

        if (isOwn || wasAtBottom) {
            scrollToBottom();
        } else {
            unreadMessageIds.add(message.id);
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

async function handleMessageDeleted(payload) {
    if (selectedChatId === payload.chatId) {
        const idx = messages.findIndex(m => m.id === payload.messageId);
        if (idx >= 0) {
            messages.splice(idx, 1);
            const bubble = document.querySelector(`.message-bubble[data-message-id="${payload.messageId}"]`);
            if (bubble) bubble.remove();
            updateUnreadDivider();
            reobserveLastMessage();

            if (unreadMessageIds.delete(payload.messageId)) {
                updateScrollDownUI();
            }
        } else {
            console.warn('message_deleted: сообщение не найдено в текущем списке', payload.messageId);
        }
    }

    debouncedLoadChats();
}

async function handleMessageEdited(payload) {
    if (selectedChatId === payload.chatId) {
        try {
            const resp = await fetchWithAuth('/api/messages/' + payload.messageId);
            if (resp.ok) {
                const updated = await resp.json();

                const idx = messages.findIndex(m => m.id === payload.messageId);
                if (idx >= 0) {
                    messages[idx].text = updated.text;
                    messages[idx].updatedAt = updated.updatedAt;

                    const bubble = document.querySelector(
                        `.message-bubble[data-message-id="${payload.messageId}"]`
                    );
                    if (bubble) {
                        const textEl = bubble.querySelector('.message-bubble__text');
                        if (textEl) textEl.textContent = updated.text;
                    }
                }
            }
        } catch (err) {
            console.error('handleMessageEdited error:', err);
        }
    }

    debouncedLoadChats();
}

// ═══════════════════════════════════════
// Список чатов
// ═══════════════════════════════════════
async function loadChats() {
    try {
        const resp = await fetchWithAuth('/api/chats');
        if (!resp.ok) return;

        const chats = await resp.json();
        renderChatList(chats);
        updateAppBadge(chats);
    } catch (err) {
        console.error(err);
    }
}

// Единственная точка обновления PWA-бейджа на клиенте.
// SW дублирует то же действие в push-обработчике.
function updateAppBadge(chats) {
    if (!('setAppBadge' in navigator)) return;
    const total = chats.reduce((sum, c) => sum + (c.unreadCount || 0), 0);
    try {
        if (total > 0) {
            navigator.setAppBadge(total);
        } else {
            navigator.clearAppBadge();
        }
    } catch (err) {
        console.warn('setAppBadge failed:', err);
    }
}

let loadChatsTimer = null;

function debouncedLoadChats() {
    if (loadChatsTimer) clearTimeout(loadChatsTimer);
    loadChatsTimer = setTimeout(loadChats, CHAT_LIST_DEBOUNCE_MS);
}

function updateProfileAvatar(elementId, user) {
    const el = document.getElementById(elementId);
    el.innerHTML = '';

    if (user.avatarPath) {
        const img = document.createElement('img');
        img.src = user.avatarPath;
        img.alt = user.username;
        img.style.width = '100%';
        img.style.height = '100%';
        img.style.borderRadius = 'inherit';
        img.style.objectFit = 'cover';
        img.onerror = function () {
            el.innerHTML = '';
            el.textContent = user.username.charAt(0).toUpperCase();
        };
        el.appendChild(img);
    } else {
        el.textContent = user.username.charAt(0).toUpperCase();
    }
}

function createAvatarElement(className, user, titleOverride) {
    const displayName = titleOverride || user.username;

    if (user.avatarPath) {
        const img = document.createElement('img');
        img.className = className;
        img.src = user.avatarPath;
        img.alt = displayName;
        img.onerror = function () {
            const fallback = document.createElement('div');
            fallback.className = className;
            fallback.textContent = displayName.charAt(0).toUpperCase();
            img.replaceWith(fallback);
        };

        return img;
    }

    const div = document.createElement('div');
    div.className = className;
    div.textContent = displayName.charAt(0).toUpperCase();

    return div;
}

function renderChatList(chats) {
    chatsList = chats;

    const list = document.getElementById('chat-list');
    list.innerHTML = '';

    for (const chat of chats) {
        const item = document.createElement('div');
        item.className = 'chat-item' + (selectedChatId === chat.id ? ' chat-item--active' : '');
        item.dataset.chatId = chat.id;

        const title = getChatTitle(chat);

        const otherMember = getOtherMember(chat);
        const avatarUser = otherMember || { username: title, avatarPath: null };
        const avatar = createAvatarElement('chat-item__avatar', avatarUser, title);
        avatar.dataset.chatId = chat.id;
        avatar.dataset.action = 'avatar';

        const info = document.createElement('div');
        info.className = 'chat-item__info';

        const titleEl = document.createElement('div');
        titleEl.className = 'chat-item__title' + (chat.unreadCount > 0 ? ' chat-item__title--bold' : '');
        titleEl.textContent = title;
        info.appendChild(titleEl);

        if (chat.messages && chat.messages.length > 0) {
            const lastMsg = chat.messages[chat.messages.length - 1];
            const preview = document.createElement('div');
            preview.className = 'chat-item__last-message';
            const senderName = lastMsg.sender.id === currentUser.id ? 'Вы' : lastMsg.sender.username;
            preview.textContent = senderName + ': ' + lastMsg.text;
            info.appendChild(preview);
        }

        item.appendChild(avatar);
        item.appendChild(info);

        if (chat.unreadCount > 0) {
            const badge = document.createElement('div');
            badge.className = 'chat-item__unread-badge';
            badge.textContent = chat.unreadCount > 99 ? '99+' : String(chat.unreadCount);
            item.appendChild(badge);
        }

        list.appendChild(item);
    }
}

function setupChatListDelegation() {
    const list = document.getElementById('chat-list');
    list.addEventListener('click', (e) => {
        const avatarEl = e.target.closest('[data-action="avatar"]');
        if (avatarEl) {
            e.stopPropagation();
            const chatId = Number(avatarEl.dataset.chatId);
            const chat = chatsList.find(c => c.id === chatId);
            if (chat) openChatInfo(chat);
            return;
        }

        const itemEl = e.target.closest('.chat-item');
        if (itemEl) {
            const chatId = Number(itemEl.dataset.chatId);
            const chat = chatsList.find(c => c.id === chatId);
            if (chat) selectChat(chat).catch(console.error);
        }
    });
}

function getChatTitle(chat) {
    if (chat.title) return chat.title;

    if (chat.type === 'private' && chat.members) {
        const other = chat.members.find(m => m.id !== currentUser.id);
        if (other) return other.username;
    }

    return 'Чат #' + chat.id;
}

function getOtherMember(chat) {
    if (chat.type !== 'private' || !chat.members) return null;
    return chat.members.find(m => m.id !== currentUser.id) || null;
}

async function openChatById(chatId) {
    try {
        const resp = await fetchWithAuth('/api/chats');
        if (!resp.ok) return;

        const chats = await resp.json();
        const chat = chats.find(c => c.id === chatId);
        if (chat) await selectChat(chat);
    } catch (err) {
        console.error('Open chat by ID error:', err);
    }
}

// ═══════════════════════════════════════
// Выбор чата и загрузка сообщений
// ═══════════════════════════════════════
async function selectChat(chat) {
    selectedChatId = chat.id;
    selectedChat = chat;
    messages = [];
    hasMoreMessages = true;

    const msgList = document.getElementById('messages-list');
    msgList.onscroll = null; // Убираем старый обработчик до очистки контента
    msgList.innerHTML = '';

    document.getElementById('conversation').style.display = '';
    document.getElementById('conversation-placeholder').style.display = 'none';
    document.querySelector('.chat-layout').classList.add('chat-layout--chat-open');

    const titleEl = document.getElementById('conversation-title');
    titleEl.textContent = getChatTitle(chat);
    titleEl.onclick = () => openChatInfo(chat);

    await loadMessages(chat.id, 0);
    scrollToBottom();
    resetScrollDownState();
    reobserveLastMessage();

    // Обновляем active в списке:
    debouncedLoadChats();

    // Подписка на подгрузку старых сообщений при скролле вверх:
    msgList.onscroll = async () => {
        try {
            if (msgList.scrollTop <= 10 && !isLoadingMore && hasMoreMessages) {
                const prevHeight = msgList.scrollHeight;
                await loadMoreMessages();
                msgList.scrollTop = msgList.scrollHeight - prevHeight;
            }
        } catch (err) {
            console.error('Failed to load more messages:', err);
        }
    };
}

async function loadMessages(chatId, offset) {
    try {
        const resp = await fetchWithAuth(
            '/api/chats/' + chatId + '/messages?limit=' + MESSAGES_PAGE_SIZE + '&offset=' + offset
        );
        if (!resp.ok) return;

        const fetched = await resp.json();
        if (!fetched || fetched.length === 0) {
            hasMoreMessages = false;
            return;
        }

        hasMoreMessages = fetched.length >= MESSAGES_PAGE_SIZE;

        // API возвращает от новых к старым, разворачиваем:
        const reversed = fetched.reverse();

        if (offset === 0) {
            messages = reversed;
            renderMessages();
        } else {
            messages = [...reversed, ...messages];
            prependMessages(reversed);
        }
    } catch (err) {
        console.error(err);
    }
}

function serverMessageCount() {
    return messages.length;
}

async function loadMoreMessages() {
    if (isLoadingMore || !hasMoreMessages || !selectedChatId) return;
    isLoadingMore = true;

    try {
        await loadMessages(selectedChatId, serverMessageCount());
    } finally {
        isLoadingMore = false;
    }
}

// ═══════════════════════════════════════
// Рендеринг сообщений
// ═══════════════════════════════════════
function renderMessages() {
    const container = document.getElementById('messages-list');
    container.innerHTML = '';

    for (let i = 0; i < messages.length; i++) {
        if (isFirstUnread(messages[i], i)) {
            const divider = document.createElement('div');
            divider.className = 'conversation__unread-divider';
            divider.innerHTML = '<span>Новые сообщения</span>';
            container.appendChild(divider);
        }

        container.appendChild(createMessageBubble(messages[i]));
    }
}

function prependMessages(olderMessages) {
    const container = document.getElementById('messages-list');
    const fragment = document.createDocumentFragment();

    for (const msg of olderMessages) {
        fragment.appendChild(createMessageBubble(msg));
    }

    container.insertBefore(fragment, container.firstChild);
}

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

function createMessageBubble(message) {
    const isOwn = message.sender.id === currentUser.id;

    const bubble = document.createElement('div');
    bubble.className = 'message-bubble' + (isOwn ? ' message-bubble--own' : '');
    bubble.dataset.messageId = message.id;

    // Плашка ответа:
    if (message.replyToMessage) {
        const replyBlock = document.createElement('div');
        replyBlock.className = 'message-bubble__reply';

        const replySender = document.createElement('span');
        replySender.className = 'message-bubble__reply-sender';
        replySender.textContent = message.replyToMessage.sender.id === currentUser.id
            ? 'Вы' : message.replyToMessage.sender.username;

        const replyText = document.createElement('span');
        replyText.className = 'message-bubble__reply-text';
        replyText.textContent = message.replyToMessage.text;

        replyBlock.appendChild(replySender);
        replyBlock.appendChild(replyText);

        replyBlock.addEventListener('click', () => {
            scrollToMessage(message.replyToMessage.id);
        });

        bubble.appendChild(replyBlock);
    }

    const header = document.createElement('div');
    header.className = 'message-bubble__header';

    const sender = document.createElement('span');
    sender.className = 'message-bubble__sender';
    sender.textContent = isOwn ? 'Вы' : message.sender.username;

    const time = document.createElement('span');
    time.className = 'message-bubble__time';
    time.textContent = formatTime(message.createdAt);

    header.appendChild(sender);
    header.appendChild(time);

    const text = document.createElement('div');
    text.className = 'message-bubble__text';
    text.textContent = message.text;

    bubble.appendChild(header);
    bubble.appendChild(text);

    return bubble;
}

function isFirstUnread(message, index) {
    if (message.isRead || message.sender.id === currentUser.id) return false;
    if (index === 0) return true;

    const prev = messages[index - 1];
    return prev.isRead || prev.sender.id === currentUser.id;
}

function updateUnreadDivider() {
    const container = document.getElementById('messages-list');

    // Убираем старый разделитель:
    const old = container.querySelector('.conversation__unread-divider');
    if (old) old.remove();

    // Ищем первое непрочитанное сообщение:
    let firstUnreadIdx = -1;
    for (let i = 0; i < messages.length; i++) {
        if (isFirstUnread(messages[i], i)) {
            firstUnreadIdx = i;
            break;
        }
    }

    if (firstUnreadIdx < 0) return;

    // Вставляем divider перед bubble этого сообщения:
    const bubble = container.querySelector(
        `.message-bubble[data-message-id="${messages[firstUnreadIdx].id}"]`
    );
    if (bubble) {
        const divider = document.createElement('div');
        divider.className = 'conversation__unread-divider';
        divider.innerHTML = '<span>Новые сообщения</span>';
        container.insertBefore(divider, bubble);
    }
}

function scrollToBottom() {
    requestAnimationFrame(() => {
        const container = document.getElementById('messages-list');
        container.scrollTop = container.scrollHeight;
    });
}

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
        unreadMessageIds.clear();
    }
    updateScrollDownUI();
}

function updateScrollDownUI() {
    const btn = document.getElementById('btn-scroll-down');
    const badge = document.getElementById('scroll-down-badge');
    if (!btn || !badge) return;

    btn.hidden = isAtBottom;

    const count = unreadMessageIds.size;
    if (count > 0) {
        badge.hidden = false;
        badge.textContent = count > 99 ? '99+' : String(count);
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
    unreadMessageIds.clear();
    isAtBottom = true;
    if (lastMessageObserver) lastMessageObserver.disconnect();
    updateScrollDownUI();
}

function onScrollDownClick() {
    const container = document.getElementById('messages-list');
    container.scrollTo({ top: container.scrollHeight, behavior: 'smooth' });
}

function formatTime(dateStr) {
    return new Date(dateStr).toLocaleString('ru-RU');
}

function formatDate(dateStr) {
    return new Date(dateStr).toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
    });
}

// ═══════════════════════════════════════
// Отправка сообщений
// ═══════════════════════════════════════
function setupSendMessage() {
    const input = document.getElementById('message-input');
    const btn = document.getElementById('btn-send');

    input.addEventListener('input', () => {
        btn.disabled = !input.value.trim();
    });

    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            if (IS_MOBILE) {
                // На мобильном Enter = перенос строки (поведение по умолчанию)
                return;
            }
            if (!e.shiftKey) {
                e.preventDefault();
                sendMessage();
            }
        }
    });

    btn.addEventListener('click', sendMessage);
}

function sendMessage() {
    const input = document.getElementById('message-input');
    const text = input.value.trim();
    if (!text || !selectedChatId) return;

    // Режим редактирования:
    if (editingMessage) {
        updateMessage(editingMessage.id, text).catch(err => console.error('updateMessage error:', err));
        cancelEdit();
        return;
    }

    // Обычная отправка через WebSocket:
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    const payload = {chatId: selectedChatId, text};
    if (replyToMessage) {
        payload.replyToMessage = {id: replyToMessage.id};
    }

    ws.send(JSON.stringify(payload));

    // Помечаем все сообщения как прочитанные и убираем разделитель:
    markAllAsRead();

    // Сбрасываем ответ:
    cancelReply();

    input.value = '';
    document.getElementById('btn-send').disabled = true;

    // Возвращаем фокус на поле ввода, чтобы клавиатура не закрывалась на мобильных:
    input.focus();

    loadChats().catch(console.error);
}

async function updateMessage(messageId, text) {
    const input = document.getElementById('message-input');

    try {
        const resp = await fetchWithAuth('/api/messages/' + messageId, {
            method: 'PUT',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({text}),
        });

        if (!resp.ok) {
            const errText = await resp.text();
            showError(mapError(errText));
            return;
        }
    } catch (err) {
        console.error('updateMessage error:', err);
    }

    input.value = '';
    document.getElementById('btn-send').disabled = true;
    input.focus();
}

function markAllAsRead() {
    for (const msg of messages) {
        msg.isRead = true;
    }

    const divider = document.querySelector('.conversation__unread-divider');
    if (divider) divider.remove();
}

// ═══════════════════════════════════════
// Emoji picker
// ═══════════════════════════════════════
function setupEmojiPicker() {
    const wrapper = document.querySelector('.conversation__emoji-wrapper');
    const toggleBtn = document.getElementById('btn-emoji-toggle');
    const pickerContainer = document.getElementById('emoji-picker-container');
    const textarea = document.getElementById('message-input');
    let pickerInitialized = false;
    let emojiCloseTimer = null;

    function showEmojiPicker() {
        clearTimeout(emojiCloseTimer);

        if (!pickerInitialized) {
            createEmojiPicker(pickerContainer, (emoji) => {
                const start = textarea.selectionStart;
                const end = textarea.selectionEnd;
                const text = textarea.value;
                textarea.value = text.slice(0, start) + emoji + text.slice(end);

                const cursorPos = start + emoji.length;
                textarea.selectionStart = cursorPos;
                textarea.selectionEnd = cursorPos;
                textarea.focus();

                document.getElementById('btn-send').disabled = !textarea.value.trim();
            });
            pickerInitialized = true;
        }

        pickerContainer.style.display = '';
        toggleBtn.classList.add('conversation__emoji-toggle--active');
    }

    function scheduleEmojiClose() {
        emojiCloseTimer = setTimeout(() => {
            pickerContainer.style.display = 'none';
            toggleBtn.classList.remove('conversation__emoji-toggle--active');
        }, EMOJI_CLOSE_DELAY_MS);
    }

    wrapper.addEventListener('mouseenter', showEmojiPicker);
    wrapper.addEventListener('mouseleave', scheduleEmojiClose);
}

// ═══════════════════════════════════════
// Закрытие чата
// ═══════════════════════════════════════
function setupCloseChat() {
    document.getElementById('btn-close-chat').addEventListener('click', () => {
        closeChat();
        debouncedLoadChats();
    });
}

// ═══════════════════════════════════════
// Свайп вправо → закрытие чата (мобильная версия)
// ═══════════════════════════════════════
function lockPageScroll() {
    document.documentElement.style.overflowX = 'hidden';
    document.body.style.overflowX = 'hidden';
}

function unlockPageScroll() {
    document.documentElement.style.overflowX = '';
    document.body.style.overflowX = '';
}

function setupSwipeToCloseChat() {
    const conversation = document.getElementById('conversation');

    let touchStartX = 0;
    let touchStartY = 0;
    let isSwiping = false;    // свайп активен
    let directionLocked = false; // направление определено

    conversation.addEventListener('touchstart', (e) => {
        if (!selectedChatId || window.innerWidth > 600) return;
        if (e.target.closest('button, a, input, textarea')) return;
        touchStartX = e.touches[0].clientX;
        touchStartY = e.touches[0].clientY;
        isSwiping = false;
        directionLocked = false;
        conversation.style.transition = 'none';
    }, {passive: true});

    conversation.addEventListener('touchmove', (e) => {
        if (!selectedChatId) return;

        const dx = e.touches[0].clientX - touchStartX;
        const dy = Math.abs(e.touches[0].clientY - touchStartY);

        // Определяем направление жеста один раз:
        if (!directionLocked && (Math.abs(dx) > SWIPE_LOCK_ANGLE_PX || dy > SWIPE_LOCK_ANGLE_PX)) {
            directionLocked = true;
            isSwiping = dx > 0 && Math.abs(dx) > dy; // горизонтальный свайп вправо
            if (isSwiping) lockPageScroll();
        }

        if (!isSwiping) return;

        // Сдвигаем conversation вправо (только положительное направление):
        const offset = Math.max(0, dx);
        conversation.style.transform = `translateX(${offset}px)`;
    }, {passive: true});

    conversation.addEventListener('touchend', (e) => {
        if (!selectedChatId || !isSwiping) {
            conversation.style.transition = '';
            conversation.style.transform = '';
            return;
        }

        const dx = e.changedTouches[0].clientX - touchStartX;
        const threshold = window.innerWidth * SWIPE_CLOSE_THRESHOLD_RATIO;

        conversation.style.transition = 'transform 0.3s ease';

        if (dx >= threshold) {
            // Доводим до конца экрана, затем закрываем:
            conversation.style.transform = `translateX(100%)`;
            conversation.addEventListener('transitionend', function handler() {
                conversation.removeEventListener('transitionend', handler);
                conversation.style.transition = '';
                conversation.style.transform = '';
                unlockPageScroll();
                closeChat();
                debouncedLoadChats();
            });
        } else {
            // Возвращаем на место:
            conversation.style.transform = 'translateX(0)';
            conversation.addEventListener('transitionend', function handler() {
                conversation.removeEventListener('transitionend', handler);
                conversation.style.transition = '';
                conversation.style.transform = '';
                unlockPageScroll();
            });
        }

        isSwiping = false;
    }, {passive: true});
}

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

// ═══════════════════════════════════════
// Информация о чате (роутер)
// ═══════════════════════════════════════
function openChatInfo(chat) {
    if (chat.type === 'private') {
        const member = getOtherMember(chat);
        if (member) showMemberProfile(member);
    } else {
        showGroupChatInfo(chat);
    }
}

// ═══════════════════════════════════════
// Модалка профиля участника
// ═══════════════════════════════════════
function setupMemberProfileModal() {
    const overlay = document.getElementById('modal-member-profile');

    document.getElementById('btn-close-member-profile').addEventListener('click', () => {
        closeMemberProfile();
    });

    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) closeMemberProfile();
    });
}

function closeMemberProfile() {
    document.getElementById('modal-member-profile').style.display = 'none';

    // Если пришли из модалки группового чата — возвращаемся к ней:
    if (returnToGroupChat) {
        showGroupChatInfo(returnToGroupChat);
        returnToGroupChat = null;
    }
}

function showMemberProfile(user) {
    updateProfileAvatar('member-avatar', user);
    document.getElementById('member-username').textContent = user.username;
    document.getElementById('member-email').textContent = user.email;

    const confirmedEl = document.getElementById('member-email-confirmed');
    confirmedEl.textContent = user.emailConfirmed ? 'Да' : 'Нет';
    confirmedEl.className = 'profile-modal__value ' +
        (user.emailConfirmed ? 'profile-modal__value--success' : 'profile-modal__value--warning');

    document.getElementById('member-created-at').textContent = formatDate(user.createdAt);

    const memberAvatarEl = document.getElementById('member-avatar');
    if (user.avatarPath) {
        memberAvatarEl.style.cursor = 'pointer';
        memberAvatarEl.onclick = () => {
            if (typeof window.openAvatarZoom === 'function') {
                window.openAvatarZoom(user.avatarPath);
            }
        };
    } else {
        memberAvatarEl.style.cursor = 'default';
        memberAvatarEl.onclick = null;
    }

    document.getElementById('modal-member-profile').style.display = '';
}

// ═══════════════════════════════════════
// Модалка группового чата
// ═══════════════════════════════════════
function setupGroupChatModal() {
    const overlay = document.getElementById('modal-group-chat');

    document.getElementById('btn-close-group-chat').addEventListener('click', () => {
        overlay.style.display = 'none';
    });

    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) overlay.style.display = 'none';
    });
}

function showGroupChatInfo(chat) {
    const title = getChatTitle(chat);

    const groupAvatarEl = document.getElementById('group-chat-avatar');
    groupAvatarEl.innerHTML = '';
    groupAvatarEl.textContent = title.charAt(0).toUpperCase();
    document.getElementById('group-chat-title').textContent = title;

    const descEl = document.getElementById('group-chat-description');
    descEl.textContent = chat.description || '';
    descEl.style.display = chat.description ? '' : 'none';

    const members = chat.members || [];
    document.getElementById('group-chat-members-count').textContent =
        'Участники (' + members.length + ')';

    const list = document.getElementById('group-chat-members-list');
    list.innerHTML = '';

    for (const member of members) {
        const item = document.createElement('div');
        item.className = 'user-item user-item--clickable';
        item.addEventListener('click', () => {
            // Запоминаем чат, чтобы вернуться после закрытия профиля:
            returnToGroupChat = chat;
            document.getElementById('modal-group-chat').style.display = 'none';
            showMemberProfile(member);
        });

        const avatar = createAvatarElement('user-item__avatar', member);

        const info = document.createElement('div');
        info.className = 'user-item__info';

        const name = document.createElement('div');
        name.className = 'user-item__name';
        name.textContent = member.username;

        if (member.id === currentUser.id) {
            const badge = document.createElement('span');
            badge.className = 'group-chat-modal__member-badge';
            badge.textContent = '(вы)';
            name.appendChild(badge);
        }

        const email = document.createElement('div');
        email.className = 'user-item__email';
        email.textContent = member.email;

        info.appendChild(name);
        info.appendChild(email);
        item.appendChild(avatar);
        item.appendChild(info);
        list.appendChild(item);
    }

    document.getElementById('modal-group-chat').style.display = '';
}

// ═══════════════════════════════════════
// Модалка создания чата
// ═══════════════════════════════════════
function setupCreateChatModal() {
    const overlay = document.getElementById('modal-create-chat');
    const typeSelect = document.getElementById('create-chat-type');
    const searchInput = document.getElementById('create-chat-search');
    const selectedUserIds = new Set();

    document.getElementById('btn-create-chat').addEventListener('click', () => {
        overlay.style.display = '';
        resetCreateChatModal();
    });

    document.getElementById('btn-close-create-chat').addEventListener('click', () => {
        overlay.style.display = 'none';
    });

    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) overlay.style.display = 'none';
    });

    typeSelect.addEventListener('change', () => {
        const isGroup = typeSelect.value === 'group';
        document.getElementById('group-title-field').style.display = isGroup ? '' : 'none';
        document.getElementById('group-description-field').style.display = isGroup ? '' : 'none';
    });

    let searchTimeout;
    searchInput.addEventListener('input', () => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => searchUsersForCreate(searchInput.value, selectedUserIds), SEARCH_DEBOUNCE_MS);
    });

    document.getElementById('btn-do-create-chat').addEventListener('click', async () => {
        if (selectedUserIds.size === 0) {
            showInfo('Укажите хотя бы одного участника');
            return;
        }

        const body = {
            type: typeSelect.value,
            members: Array.from(selectedUserIds).map(id => ({id})),
        };

        if (typeSelect.value === 'group') {
            const title = document.getElementById('create-chat-title').value.trim();
            const description = document.getElementById('create-chat-description').value.trim();
            if (title) body.title = title;
            if (description) body.description = description;
        }

        try {
            const resp = await fetchWithAuth('/api/chats', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(body),
            });

            if (!resp.ok) {
                const text = await resp.text();
                showError(mapError(text));
                return;
            }

            overlay.style.display = 'none';
            await loadChats();
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });

    function resetCreateChatModal() {
        typeSelect.value = 'private';
        document.getElementById('group-title-field').style.display = 'none';
        document.getElementById('group-description-field').style.display = 'none';
        document.getElementById('create-chat-title').value = '';
        document.getElementById('create-chat-description').value = '';
        searchInput.value = '';
        document.getElementById('create-chat-users').style.display = 'none';
        document.getElementById('create-chat-users').innerHTML = '';
        selectedUserIds.clear();
    }
}

async function searchUsersForCreate(query, selectedUserIds) {
    const container = document.getElementById('create-chat-users');

    if (!query.trim()) {
        container.style.display = 'none';
        container.innerHTML = '';
        return;
    }

    try {
        const resp = await fetchWithAuth('/api/users?username=' + encodeURIComponent(query));
        if (!resp.ok) return;

        const users = await resp.json();
        container.innerHTML = '';

        if (!users || users.length === 0) {
            container.style.display = 'none';
            return;
        }

        const filteredUsers = users.filter(u => u.id !== currentUser.id);

        if (filteredUsers.length === 0) {
            container.style.display = 'none';
            return;
        }

        container.style.display = '';

        for (const user of filteredUsers) {
            const label = document.createElement('label');
            label.className = 'user-item user-item--selectable';

            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.className = 'user-item__checkbox';
            checkbox.checked = selectedUserIds.has(user.id);
            checkbox.addEventListener('change', () => {
                if (checkbox.checked) {
                    selectedUserIds.add(user.id);
                } else {
                    selectedUserIds.delete(user.id);
                }
            });

            const avatar = createAvatarElement('user-item__avatar', user);

            const info = document.createElement('div');
            info.className = 'user-item__info';

            const name = document.createElement('div');
            name.className = 'user-item__name';
            name.textContent = user.username;

            const email = document.createElement('div');
            email.className = 'user-item__email';
            email.textContent = user.email;

            info.appendChild(name);
            info.appendChild(email);
            label.appendChild(checkbox);
            label.appendChild(avatar);
            label.appendChild(info);
            container.appendChild(label);
        }
    } catch (err) {
        console.error(err);
    }
}

// ═══════════════════════════════════════
// Модалка поиска пользователей
// ═══════════════════════════════════════
function setupSearchUsersModal() {
    const overlay = document.getElementById('modal-search-users');
    const input = document.getElementById('search-users-input');

    document.getElementById('btn-search-users').addEventListener('click', () => {
        overlay.style.display = '';
        input.value = '';
        document.getElementById('search-users-list').style.display = 'none';
        document.getElementById('search-users-list').innerHTML = '';
        document.getElementById('search-users-empty').style.display = 'none';
    });

    document.getElementById('btn-close-search-users').addEventListener('click', () => {
        overlay.style.display = 'none';
    });

    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) overlay.style.display = 'none';
    });

    let searchTimeout;
    input.addEventListener('input', () => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => searchUsersGlobal(input.value), SEARCH_DEBOUNCE_MS);
    });
}

async function searchUsersGlobal(query) {
    const container = document.getElementById('search-users-list');
    const emptyEl = document.getElementById('search-users-empty');

    if (!query.trim()) {
        container.style.display = 'none';
        container.innerHTML = '';
        emptyEl.style.display = 'none';
        return;
    }

    try {
        const resp = await fetchWithAuth('/api/users?username=' + encodeURIComponent(query));
        if (!resp.ok) return;

        const users = await resp.json();
        container.innerHTML = '';

        if (!users || users.length === 0) {
            container.style.display = 'none';
            emptyEl.style.display = '';
            return;
        }

        emptyEl.style.display = 'none';
        container.style.display = '';

        for (const user of users) {
            const item = document.createElement('div');
            item.className = 'user-item user-item--clickable';
            item.addEventListener('click', () => {
                document.getElementById('modal-search-users').style.display = 'none';
                showMemberProfile(user);
            });

            const avatar = createAvatarElement('user-item__avatar', user);

            const info = document.createElement('div');
            info.className = 'user-item__info';

            const name = document.createElement('div');
            name.className = 'user-item__name';
            name.textContent = user.username;

            const email = document.createElement('div');
            email.className = 'user-item__email';
            email.textContent = user.email;

            info.appendChild(name);
            info.appendChild(email);
            item.appendChild(avatar);
            item.appendChild(info);
            container.appendChild(item);
        }
    } catch (err) {
        console.error(err);
    }
}

// ═══════════════════════════════════════
// Мобильная клавиатура
// ═══════════════════════════════════════
function setupMobileKeyboardDismiss() {
    if (!IS_MOBILE) return;

    // Отличаем тап от скролла: если палец сдвинулся — это скролл, не закрываем клавиатуру.
    let touchStartX = 0;
    let touchStartY = 0;
    const TAP_THRESHOLD = 10; // px — максимальное смещение, чтобы считать жест тапом

    document.addEventListener('touchstart', (e) => {
        touchStartX = e.touches[0].clientX;
        touchStartY = e.touches[0].clientY;
    }, {passive: true});

    document.addEventListener('touchend', (e) => {
        if (e.target.closest('#message-input, #btn-send, #context-menu, #btn-cancel-reply')) return;

        // Не закрываем клавиатуру, если контекстное меню открыто (long press по bubble):
        const menu = document.getElementById('context-menu');
        if (menu && menu.style.display !== 'none') return;

        // Проверяем, был ли это тап (не скролл):
        const dx = Math.abs(e.changedTouches[0].clientX - touchStartX);
        const dy = Math.abs(e.changedTouches[0].clientY - touchStartY);
        if (dx > TAP_THRESHOLD || dy > TAP_THRESHOLD) return;

        document.activeElement?.blur();
    }, {passive: true});
}

function setupMobileViewportResize() {
    if (!IS_MOBILE) return;

    function updateVh() {
        const vh = (window.visualViewport ? window.visualViewport.height : window.innerHeight) / 100;
        document.documentElement.style.setProperty('--vh', vh + 'px');
    }

    updateVh();

    // Порог уменьшения viewport для определения открытой клавиатуры (150px):
    const KEYBOARD_THRESHOLD_PX = 150;
    const fullViewportHeight = window.innerHeight;

    if (window.visualViewport) {
        window.visualViewport.addEventListener('resize', () => {
            // Запоминаем позицию скролла сообщений до изменения размера:
            const msgList = document.getElementById('messages-list');
            const wasAtBottom = msgList.scrollHeight - msgList.scrollTop - msgList.clientHeight < 30;
            const prevScrollBottom = msgList.scrollHeight - msgList.scrollTop;

            updateVh();
            window.scrollTo(0, 0);

            // Определяем, открыта ли клавиатура (viewport уменьшился значительно):
            const keyboardOpen = fullViewportHeight - window.visualViewport.height > KEYBOARD_THRESHOLD_PX;
            document.querySelectorAll('.modal-overlay').forEach(overlay => {
                overlay.classList.toggle('modal-overlay--keyboard-open', keyboardOpen);
            });

            // Восстанавливаем позицию скролла после перестроения layout:
            requestAnimationFrame(() => {
                if (wasAtBottom) {
                    msgList.scrollTop = msgList.scrollHeight;
                } else {
                    // Сохраняем ту же позицию относительно низа (контейнер сжался снизу):
                    msgList.scrollTop = msgList.scrollHeight - prevScrollBottom;
                }
            });
        });
    } else {
        window.addEventListener('resize', updateVh);
    }

    window.addEventListener('scroll', () => {
        if (window.scrollY !== 0) window.scrollTo(0, 0);
    });
}

// ═══════════════════════════════════════
// Контекстное меню сообщений
// ═══════════════════════════════════════
function setupContextMenu() {
    const menu = document.getElementById('context-menu');
    const msgList = document.getElementById('messages-list');

    // Desktop: правый клик
    msgList.addEventListener('contextmenu', (e) => {
        const bubble = e.target.closest('.message-bubble');
        if (!bubble) return;

        e.preventDefault();
        showContextMenu(bubble, e.clientX, e.clientY);
    });

    // Mobile: длинное нажатие с анимацией увеличения
    let longPressTimer = null;
    let longPressTarget = null;

    msgList.addEventListener('touchstart', (e) => {
        const bubble = e.target.closest('.message-bubble');
        if (!bubble) return;

        longPressTarget = bubble;

        // Запускаем анимацию увеличения на время long press:
        if (IS_MOBILE) {
            bubble.classList.add('message-bubble--pressing');
        }

        longPressTimer = setTimeout(() => {
            if (longPressTarget) {
                longPressTarget.classList.remove('message-bubble--pressing');
            }
            const touch = e.touches[0];
            showContextMenu(bubble, touch.clientX, touch.clientY);
            longPressTarget = null;
        }, LONG_PRESS_DELAY_MS);
    }, {passive: true});

    msgList.addEventListener('touchmove', () => {
        if (longPressTarget) {
            longPressTarget.classList.remove('message-bubble--pressing');
        }
        clearTimeout(longPressTimer);
        longPressTarget = null;
    }, {passive: true});

    msgList.addEventListener('touchend', () => {
        if (longPressTarget) {
            longPressTarget.classList.remove('message-bubble--pressing');
        }
        clearTimeout(longPressTimer);
        longPressTarget = null;
    }, {passive: true});

    // Mobile: свайп влево для ответа на сообщение
    if (IS_MOBILE) {
        setupSwipeToReply(msgList);
    }

    // Закрытие меню при клике в другом месте:
    document.addEventListener('click', (e) => {
        if (!e.target.closest('#context-menu')) {
            menu.style.display = 'none';
        }
    });

    document.addEventListener('touchstart', (e) => {
        if (!e.target.closest('#context-menu') && menu.style.display !== 'none') {
            menu.style.display = 'none';
        }
    }, {passive: true});

    // Действия:
    document.getElementById('ctx-reply').addEventListener('click', () => {
        menu.style.display = 'none';
        const msg = messages.find(m => m.id === contextMenuMessageId);
        if (msg) setReply(msg);
        restoreInputFocus();
    });

    document.getElementById('ctx-edit').addEventListener('click', () => {
        menu.style.display = 'none';
        const msg = messages.find(m => m.id === contextMenuMessageId);
        if (msg) setEdit(msg);
        restoreInputFocus();
    });

    document.getElementById('ctx-copy').addEventListener('click', () => {
        menu.style.display = 'none';
        const msg = messages.find(m => m.id === contextMenuMessageId);
        if (msg) navigator.clipboard.writeText(msg.text).catch(console.error);
        restoreInputFocus();
    });

    document.getElementById('ctx-delete-toggle').addEventListener('click', () => {
        document.getElementById('ctx-delete-sub').style.display = '';
        document.getElementById('ctx-delete-toggle').style.display = 'none';
        restoreInputFocus();
    });

    document.getElementById('ctx-delete-for-me').addEventListener('click', async () => {
        try {
            menu.style.display = 'none';
            await deleteMessage(contextMenuMessageId, false);
        } catch (err) {
            console.error('Failed to delete message:', err);
        }

        restoreInputFocus();
    });

    document.getElementById('ctx-delete-for-all').addEventListener('click', async () => {
        try {
            menu.style.display = 'none';
            await deleteMessage(contextMenuMessageId, true);
        } catch (err) {
            console.error('Failed to delete message:', err);
        }

        restoreInputFocus();
    });
}

function restoreInputFocus() {
    if (contextMenuInputWasFocused) {
        document.getElementById('message-input').focus();
    }
}

function showContextMenu(bubble, x, y) {
    const menu = document.getElementById('context-menu');
    const messageId = Number(bubble.dataset.messageId);
    contextMenuMessageId = messageId;
    contextMenuInputWasFocused = document.activeElement === document.getElementById('message-input');

    const msg = messages.find(m => m.id === messageId);
    const isOwn = msg && msg.sender.id === currentUser.id;
    document.getElementById('ctx-edit').style.display = isOwn ? '' : 'none';
    document.getElementById('ctx-delete-group').style.display = isOwn ? '' : 'none';
    document.getElementById('ctx-delete-toggle').style.display = isOwn ? '' : 'none';
    document.getElementById('ctx-delete-sub').style.display = 'none';

    menu.style.display = '';
    menu.style.left = x + 'px';
    menu.style.top = y + 'px';

    // Не выходим за границы экрана:
    const rect = menu.getBoundingClientRect();
    if (rect.right > window.innerWidth) {
        menu.style.left = (window.innerWidth - rect.width - 8) + 'px';
    }
    if (rect.bottom > window.innerHeight) {
        menu.style.top = (window.innerHeight - rect.height - 8) + 'px';
    }
}

// ═══════════════════════════════════════
// Свайп влево → ответ на сообщение (мобильная версия)
// ═══════════════════════════════════════
const SWIPE_REPLY_THRESHOLD_PX = 80;

function setupSwipeToReply(msgList) {
    let swipeStartX = 0;
    let swipeStartY = 0;
    let swipeBubble = null;
    let swipeActive = false;
    let swipeDirectionLocked = false;

    msgList.addEventListener('touchstart', (e) => {
        const bubble = e.target.closest('.message-bubble');
        if (!bubble) return;

        swipeStartX = e.touches[0].clientX;
        swipeStartY = e.touches[0].clientY;
        swipeBubble = bubble;
        swipeActive = false;
        swipeDirectionLocked = false;
        bubble.style.transition = 'none';
    }, {passive: true});

    msgList.addEventListener('touchmove', (e) => {
        if (!swipeBubble) return;

        const dx = e.touches[0].clientX - swipeStartX;
        const dy = Math.abs(e.touches[0].clientY - swipeStartY);

        // Определяем направление один раз:
        if (!swipeDirectionLocked && (Math.abs(dx) > SWIPE_LOCK_ANGLE_PX || dy > SWIPE_LOCK_ANGLE_PX)) {
            swipeDirectionLocked = true;
            swipeActive = dx < 0 && Math.abs(dx) > dy; // горизонтальный свайп влево
        }

        if (!swipeActive) return;

        // Сдвигаем bubble влево (только отрицательное направление), ограничиваем максимум:
        const offset = Math.max(-SWIPE_REPLY_THRESHOLD_PX * 1.5, Math.min(0, dx));
        swipeBubble.style.transform = `translateX(${offset}px)`;
    }, {passive: true});

    msgList.addEventListener('touchend', (e) => {
        if (!swipeBubble || !swipeActive) {
            if (swipeBubble) {
                swipeBubble.style.transition = '';
                swipeBubble.style.transform = '';
            }
            swipeBubble = null;
            return;
        }

        const dx = e.changedTouches[0].clientX - swipeStartX;
        const bubble = swipeBubble;
        swipeBubble = null;

        bubble.style.transition = 'transform 0.2s ease';
        bubble.style.transform = '';

        bubble.addEventListener('transitionend', function handler() {
            bubble.removeEventListener('transitionend', handler);
            bubble.style.transition = '';
        });

        // Если свайп достаточный — активируем reply:
        if (Math.abs(dx) >= SWIPE_REPLY_THRESHOLD_PX) {
            const messageId = Number(bubble.dataset.messageId);
            const msg = messages.find(m => m.id === messageId);
            if (msg) {
                setReply(msg);
            }
        }

        swipeActive = false;
    }, {passive: true});
}

// ═══════════════════════════════════════
// Ответ на сообщение
// ═══════════════════════════════════════
function setReply(message) {
    // Если было редактирование — сбрасываем:
    if (editingMessage) cancelEdit();

    replyToMessage = message;
    const bar = document.getElementById('reply-bar');
    document.getElementById('reply-bar-sender').textContent =
        message.sender.id === currentUser.id ? 'Вы' : message.sender.username;
    document.getElementById('reply-bar-text').textContent = message.text;
    bar.style.display = '';
    document.getElementById('message-input').focus();
}

function cancelReply() {
    replyToMessage = null;
    document.getElementById('reply-bar').style.display = 'none';
    document.getElementById('message-input').focus();
}

function setEdit(message) {
    // Если был ответ — сбрасываем:
    cancelReply();

    editingMessage = message;
    const bar = document.getElementById('edit-bar');
    document.getElementById('edit-bar-text').textContent = message.text;
    bar.style.display = '';

    const input = document.getElementById('message-input');
    input.value = message.text;
    document.getElementById('btn-send').disabled = !input.value.trim();
    input.focus();
}

function cancelEdit() {
    editingMessage = null;
    document.getElementById('edit-bar').style.display = 'none';
    const input = document.getElementById('message-input');
    input.value = '';
    document.getElementById('btn-send').disabled = true;
    input.focus();
}

function scrollToMessage(messageId) {
    const bubble = document.querySelector(`.message-bubble[data-message-id="${messageId}"]`);
    if (bubble) {
        bubble.scrollIntoView({behavior: 'smooth', block: 'center'});
        bubble.classList.add('message-bubble--highlight');
        bubble.addEventListener('animationend', () => {
            bubble.classList.remove('message-bubble--highlight');
        }, {once: true});
    }
}

// ═══════════════════════════════════════
// Удаление сообщений
// ═══════════════════════════════════════
function setupReplyCancel() {
    document.getElementById('btn-cancel-reply').addEventListener('click', cancelReply);
    document.getElementById('btn-cancel-edit').addEventListener('click', cancelEdit);
}

async function deleteMessage(messageId, forAll) {
    try {
        const resp = await fetchWithAuth('/api/messages/' + messageId, {
            method: 'DELETE',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({forAll}),
        });

        if (!resp.ok) {
            const text = await resp.text();
            showError(mapError(text));
            return;
        }

        debouncedLoadChats();
    } catch (err) {
        showError('Ошибка сети: ' + err.message);
    }
}

// ═══════════════════════════════════════
// Toast-уведомления
// ═══════════════════════════════════════
function showToast(senderName, text, chatId) {
    let container = document.getElementById('toast-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        container.className = 'toast-container';
        document.body.appendChild(container);
    }

    const toast = document.createElement('div');
    toast.className = 'toast';

    const body = document.createElement('div');
    body.className = 'toast__body';

    const sender = document.createElement('div');
    sender.className = 'toast__sender';
    sender.textContent = senderName;

    const msg = document.createElement('div');
    msg.className = 'toast__text';
    msg.textContent = text;

    body.appendChild(sender);
    body.appendChild(msg);

    const closeBtn = document.createElement('button');
    closeBtn.className = 'toast__close';
    closeBtn.innerHTML = '&times;';
    closeBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        toast.remove();
    });

    toast.appendChild(body);
    toast.appendChild(closeBtn);

    toast.addEventListener('click', async () => {
        toast.remove();
        try {
            const resp = await fetchWithAuth('/api/chats');
            if (resp.ok) {
                const chats = await resp.json();
                const chat = chats.find(c => c.id === chatId);
                if (chat) await selectChat(chat);
            }
        } catch (err) {
            console.error(err);
        }
    });

    container.appendChild(toast);

    setTimeout(() => {
        if (toast.parentNode) toast.remove();
    }, TOAST_DURATION_MS);
}

// ═══════════════════════════════════════
// Web Push уведомления
// ═══════════════════════════════════════
async function initWebPushNotifications() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
        return;
    }

    try {
        const registration = await navigator.serviceWorker.register('/web/sw.js');

        const existingSubscription = await registration.pushManager.getSubscription();
        if (existingSubscription) {
            await sendWebPushSubscriptionToServer(existingSubscription);
            return;
        }

        if (Notification.permission === 'denied') {
            return;
        }

        if (Notification.permission === 'default') {
            const permission = await Notification.requestPermission();
            if (permission !== 'granted') {
                return;
            }
        }

        await subscribeToWebPush(registration);
    } catch (err) {
        console.error('Web push init error:', err);
    }
}
