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
let contextMenuMessageId = null; // ID сообщения для контекстного меню
let contextMenuInputWasFocused = false; // был ли input в фокусе при открытии меню

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

    // Открытие чата из push-уведомления (по query-параметру):
    const params = new URLSearchParams(window.location.search);
    const pushChatId = params.get('chatId');
    if (pushChatId && !isNaN(Number(pushChatId))) {
        await openChatById(Number(pushChatId));
        history.replaceState(null, '', window.location.pathname);
    }

    // Обработка postMessage от Service Worker (переход в чат по push-уведомлению):
    navigator.serviceWorker?.addEventListener('message', async (event) => {
        if (event.data?.type === 'open-chat' && event.data.chatId) {
            await openChatById(Number(event.data.chatId));
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
            handleNewMessage(envelope.payload);
        } else if (envelope.type === 'message_deleted') {
            handleMessageDeleted(envelope.payload);
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

function handleNewMessage(message) {
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

function handleMessageDeleted(payload) {
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

// ═══════════════════════════════════════
// Список чатов
// ═══════════════════════════════════════
async function loadChats() {
    try {
        const resp = await fetchWithAuth('/api/chats');
        if (!resp.ok) return;

        const chats = await resp.json();
        renderChatList(chats);
    } catch (err) {
        console.error(err);
    }
}

let loadChatsTimer = null;

function debouncedLoadChats() {
    if (loadChatsTimer) clearTimeout(loadChatsTimer);
    loadChatsTimer = setTimeout(loadChats, CHAT_LIST_DEBOUNCE_MS);
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

        const avatar = document.createElement('div');
        avatar.className = 'chat-item__avatar';
        avatar.dataset.chatId = chat.id;
        avatar.dataset.action = 'avatar';
        avatar.textContent = title.charAt(0).toUpperCase();

        const info = document.createElement('div');
        info.className = 'chat-item__info';

        const titleEl = document.createElement('div');
        titleEl.className = 'chat-item__title' + (!chat.isRead ? ' chat-item__title--bold' : '');
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

        if (!chat.isRead) {
            const dot = document.createElement('div');
            dot.className = 'chat-item__unread-dot';
            item.appendChild(dot);
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

    // Обновляем active в списке:
    debouncedLoadChats();

    // Подписка на подгрузку старых сообщений при скролле вверх:
    msgList.onscroll = async () => {
        if (msgList.scrollTop <= 10 && !isLoadingMore && hasMoreMessages) {
            const prevHeight = msgList.scrollHeight;
            await loadMoreMessages();
            msgList.scrollTop = msgList.scrollHeight - prevHeight;
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
    if (!text || !selectedChatId || !ws || ws.readyState !== WebSocket.OPEN) return;

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
    document.getElementById('member-avatar').textContent = user.username.charAt(0).toUpperCase();
    document.getElementById('member-username').textContent = user.username;
    document.getElementById('member-email').textContent = user.email;

    const confirmedEl = document.getElementById('member-email-confirmed');
    confirmedEl.textContent = user.emailConfirmed ? 'Да' : 'Нет';
    confirmedEl.className = 'profile-modal__value ' +
        (user.emailConfirmed ? 'profile-modal__value--success' : 'profile-modal__value--warning');

    document.getElementById('member-created-at').textContent = formatDate(user.createdAt);
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

    document.getElementById('group-chat-avatar').textContent = title.charAt(0).toUpperCase();
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

        const avatar = document.createElement('div');
        avatar.className = 'user-item__avatar';
        avatar.textContent = member.username.charAt(0).toUpperCase();

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

            const avatar = document.createElement('div');
            avatar.className = 'user-item__avatar';
            avatar.textContent = user.username.charAt(0).toUpperCase();

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

            const avatar = document.createElement('div');
            avatar.className = 'user-item__avatar';
            avatar.textContent = user.username.charAt(0).toUpperCase();

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
        if (e.target.closest('#message-input, #btn-send, #context-menu')) return;

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

    if (window.visualViewport) {
        window.visualViewport.addEventListener('resize', () => {
            // Запоминаем позицию скролла сообщений до изменения размера:
            const msgList = document.getElementById('messages-list');
            const wasAtBottom = msgList.scrollHeight - msgList.scrollTop - msgList.clientHeight < 30;
            const prevScrollBottom = msgList.scrollHeight - msgList.scrollTop;

            updateVh();
            window.scrollTo(0, 0);

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

    // Mobile: длинное нажатие
    let longPressTimer = null;
    let longPressTarget = null;

    msgList.addEventListener('touchstart', (e) => {
        const bubble = e.target.closest('.message-bubble');
        if (!bubble) return;

        longPressTarget = bubble;
        longPressTimer = setTimeout(() => {
            const touch = e.touches[0];
            showContextMenu(bubble, touch.clientX, touch.clientY);
            longPressTarget = null;
        }, LONG_PRESS_DELAY_MS);
    }, {passive: true});

    msgList.addEventListener('touchmove', () => {
        clearTimeout(longPressTimer);
        longPressTarget = null;
    }, {passive: true});

    msgList.addEventListener('touchend', () => {
        clearTimeout(longPressTimer);
        longPressTarget = null;
    }, {passive: true});

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
    });

    document.getElementById('ctx-delete-for-me').addEventListener('click', () => {
        menu.style.display = 'none';
        deleteMessage(contextMenuMessageId, false);
        restoreInputFocus();
    });

    document.getElementById('ctx-delete-for-all').addEventListener('click', () => {
        menu.style.display = 'none';
        deleteMessage(contextMenuMessageId, true);
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
// Ответ на сообщение
// ═══════════════════════════════════════
function setReply(message) {
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
