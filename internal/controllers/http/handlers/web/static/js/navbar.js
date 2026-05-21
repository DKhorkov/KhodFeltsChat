// ═══════════════════════════════════════
// Утилиты модалок (блокировка скролла фона)
// ═══════════════════════════════════════
const MODAL_IDS = [
    'modal-member-profile',
    'modal-group-chat',
    'modal-search-users',
    'modal-create-chat',
    'modal-my-profile',
];

function openModal(modalEl) {
    modalEl.style.display = '';
    document.body.style.overflow = 'hidden';
}

function closeModal(modalEl) {
    modalEl.style.display = 'none';

    // Снимаем блокировку скролла, если не осталось открытых модалок:
    const anyOpen = MODAL_IDS.some(id => {
        const el = document.getElementById(id);
        return el && el !== modalEl && el.style.display !== 'none';
    });

    if (!anyOpen) {
        document.body.style.overflow = '';
    }
}

// ═══════════════════════════════════════
// Тема: управление через серверные настройки пользователя
// ═══════════════════════════════════════

const THEME_LIGHT = 0;
const THEME_DARK = 1;

// Бит-маска согласий на уведомления
const CONSENT_NEW_MESSAGE = 1;

function applyTheme(themeDark) {
    if (themeDark) {
        document.documentElement.setAttribute('data-bs-theme', 'dark');
    } else {
        document.documentElement.removeAttribute('data-bs-theme');
    }
    localStorage.setItem('theme', themeDark ? 'dark' : 'light');
    updateThemeSwitchUI();
}

function isDarkTheme() {
    return document.documentElement.getAttribute('data-bs-theme') === 'dark';
}

async function toggleTheme() {
    const newDark = !isDarkTheme();

    // Применяем сразу для отзывчивости UI
    applyTheme(newDark);

    // Сохраняем на сервер
    await saveSettings({ theme: newDark ? THEME_DARK : THEME_LIGHT });
}

function updateThemeSwitchUI() {
    const container = document.getElementById('theme-switch-toggle');
    if (!container) return;

    const track = container.querySelector('.toggle__track');
    const thumb = container.querySelector('.toggle__thumb');
    if (!track || !thumb) return;

    const dark = isDarkTheme();
    track.classList.toggle('toggle__track--on', dark);
    thumb.classList.toggle('toggle__thumb--on', dark);
}

function clearTheme() {
    document.documentElement.removeAttribute('data-bs-theme');
    localStorage.removeItem('theme');
    updateThemeSwitchUI();
}

// ═══════════════════════════════════════
// Уведомления: управление согласиями
// ═══════════════════════════════════════

function updateSwitchUI(switchEl, on) {
    if (!switchEl) return;

    const track = switchEl.querySelector('.toggle__track');
    const thumb = switchEl.querySelector('.toggle__thumb');
    if (!track || !thumb) return;

    track.classList.toggle('toggle__track--on', on);
    thumb.classList.toggle('toggle__thumb--on', on);
}

// Текущее состояние настроек, хранится для отправки полного объекта при обновлении
let currentSettings = null;

async function saveSettings(patch) {
    if (currentSettings) {
        Object.assign(currentSettings, patch);
    }

    const body = currentSettings || patch;

    try {
        await fetchWithAuth('/api/users/me/settings', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
    } catch (err) {
        console.error('Не удалось сохранить настройки:', err);
    }
}

function hasConsent(mask, bit) {
    return (mask & bit) !== 0;
}

function toggleConsent(mask, bit) {
    return hasConsent(mask, bit) ? mask & ~bit : mask | bit;
}

function urlBase64ToArrayBuffer(base64String) {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const rawData = atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
        outputArray[i] = rawData.charCodeAt(i);
    }

    return outputArray.buffer.slice(0);
}

async function subscribeToWebPush(registration) {
    const resp = await fetchWithAuth('/api/web-push/vapid-key');
    if (!resp.ok) throw new Error('Failed to fetch VAPID key');

    const { publicKey } = await resp.json();

    const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToArrayBuffer(publicKey),
    });

    await sendWebPushSubscriptionToServer(subscription);
}

function arrayBufferToBase64Url(buffer) {
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (let i = 0; i < bytes.length; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

async function sendWebPushSubscriptionToServer(subscription) {
    const resp = await fetchWithAuth('/api/web-push/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            endpoint: subscription.endpoint,
            encryptionKey: arrayBufferToBase64Url(subscription.getKey('p256dh')),
            auth: arrayBufferToBase64Url(subscription.getKey('auth')),
        }),
    });

    if (resp.ok) {
        const data = await resp.json();
        localStorage.setItem('webPushSubscriptionId', data.id);
    }
}

function isIOSSafari() {
    return /iP(hone|ad|od)/.test(navigator.userAgent) && !window.MSStream;
}

function isStandaloneMode() {
    return window.matchMedia('(display-mode: standalone)').matches || navigator.standalone === true;
}

function showWebPushBanner() {
    // Не показываем повторно и не показываем, если уже отклонили в этой сессии
    if (document.getElementById('web-push-banner')) return;
    if (sessionStorage.getItem('webPushBannerDismissed')) return;

    const banner = document.createElement('div');
    banner.id = 'web-push-banner';
    banner.className = 'web-push-banner';

    const text = document.createElement('span');
    text.className = 'web-push-banner__text';
    text.textContent = 'Разрешите уведомления, чтобы не пропустить новые сообщения';

    const allowBtn = document.createElement('button');
    allowBtn.className = 'web-push-banner__btn';
    allowBtn.textContent = 'Включить';

    const closeBtn = document.createElement('button');
    closeBtn.className = 'web-push-banner__close';
    closeBtn.setAttribute('aria-label', 'Закрыть');
    closeBtn.textContent = '\u00D7';

    banner.appendChild(text);
    banner.appendChild(allowBtn);
    banner.appendChild(closeBtn);
    document.body.prepend(banner);

    allowBtn.addEventListener('click', async () => {
        banner.remove();
        sessionStorage.setItem('webPushBannerDismissed', '1');
        const ok = await ensureBrowserWebPushSubscription();
        if (!ok) {
            showInfo('Не удалось включить уведомления');
        }
    });

    closeBtn.addEventListener('click', () => {
        banner.remove();
        sessionStorage.setItem('webPushBannerDismissed', '1');
    });
}

async function checkExistingWebPushSubscription() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) return false;

    try {
        const registration = await navigator.serviceWorker.register('/web/sw.js');
        const existing = await registration.pushManager.getSubscription();
        return !!existing;
    } catch (err) {
        return false;
    }
}

async function ensureBrowserWebPushSubscription() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) return false;

    // На iOS web-push работает только из Home Screen (PWA)
    if (isIOSSafari() && !isStandaloneMode()) {
        showInfo('Для получения уведомлений на iOS добавьте сайт на экран «Домой» через кнопку «Поделиться»');
        return false;
    }

    if (Notification.permission === 'denied') return false;

    try {
        const registration = await navigator.serviceWorker.register('/web/sw.js');

        const existing = await registration.pushManager.getSubscription();
        if (existing) return true;

        if (Notification.permission === 'default') {
            const permission = await Notification.requestPermission();
            if (permission !== 'granted') return false;
        }

        await subscribeToWebPush(registration);
        return true;
    } catch (err) {
        console.error('Не удалось подписаться на web-push:', err);
    }

    return false;
}

async function removeBrowserWebPushSubscription() {
    if (!('serviceWorker' in navigator)) return;

    const registration = await navigator.serviceWorker.getRegistration('/web/sw.js');
    if (!registration) return;

    const subscription = await registration.pushManager.getSubscription();
    if (!subscription) return;

    await subscription.unsubscribe();

    const subId = localStorage.getItem('webPushSubscriptionId');
    if (subId) {
        try {
            await fetchWithAuth('/api/web-push/subscribe/' + subId, { method: 'DELETE' });
        } catch (err) {
            console.error('Unsubscribe error:', err);
        }

        localStorage.removeItem('webPushSubscriptionId');
    }
}

document.addEventListener('DOMContentLoaded', async () => {
    // Обновляем UI переключателя темы (для случая, когда тема применена из localStorage в inline-скрипте)
    updateThemeSwitchUI();

    const themeSwitchToggle = document.getElementById('theme-switch-toggle');
    if (themeSwitchToggle) {
        themeSwitchToggle.addEventListener('click', toggleTheme);
    }

    const authContainer = document.getElementById('navbar-auth');
    if (!authContainer) return;

    let currentUser = null;

    try {
        const resp = await fetchWithAuth('/api/users/me');
        if (!resp.ok) return;

        currentUser = await resp.json();

        authContainer.innerHTML = '';

        const profile = document.createElement('button');
        profile.type = 'button';
        profile.className = 'navbar__profile';

        const avatar = document.createElement('div');
        avatar.className = 'navbar__profile-avatar';
        avatar.textContent = currentUser.username.charAt(0).toUpperCase();

        const name = document.createElement('span');
        name.className = 'navbar__profile-name';
        name.textContent = currentUser.username;

        profile.appendChild(avatar);
        profile.appendChild(name);
        authContainer.appendChild(profile);

        profile.addEventListener('click', () => openMyProfileModal(currentUser));

        // Подтягиваем настройки пользователя с сервера
        try {
            const settingsResp = await fetchWithAuth('/api/users/me/settings');
            if (settingsResp.ok) {
                currentSettings = await settingsResp.json();
                applyTheme(currentSettings.theme === THEME_DARK);
                await initNotificationToggles(currentSettings);
            }
        } catch (e) {
            console.error('Не удалось загрузить настройки:', e);
        }
    } catch (err) {
        console.error(err);
    }

    // --- Модалка профиля ---

    const modal = document.getElementById('modal-my-profile');
    if (!modal) return;

    async function openMyProfileModal(user) {
        document.getElementById('my-profile-avatar').textContent =
            user.username.charAt(0).toUpperCase();
        document.getElementById('my-profile-username').textContent = user.username;
        document.getElementById('my-profile-email').textContent = user.email;

        const confirmedEl = document.getElementById('my-profile-email-confirmed');
        confirmedEl.textContent = user.emailConfirmed ? 'Да' : 'Нет';
        confirmedEl.className = 'profile-modal__value' +
            (user.emailConfirmed ? ' profile-modal__value--success' : ' profile-modal__value--warning');

        document.getElementById('my-profile-created-at').textContent =
            new Date(user.createdAt).toLocaleDateString('ru-RU', {
                day: 'numeric',
                month: 'long',
                year: 'numeric',
            });

        openModal(modal);

        // Перезапрашиваем настройки при открытии модалки (синхронизация с другими устройствами)
        try {
            const settingsResp = await fetchWithAuth('/api/users/me/settings');
            if (settingsResp.ok) {
                currentSettings = await settingsResp.json();
                applyTheme(currentSettings.theme === THEME_DARK);
                await initNotificationToggles(currentSettings);
            }
        } catch (e) {
            console.error('Не удалось обновить настройки:', e);
        }
    }

    function closeMyProfileModal() {
        closeModal(modal);

        // Сбрасываем формы и закрываем секции
        const editForm = document.getElementById('my-profile-edit-form');
        const passwordForm = document.getElementById('my-profile-password-form');
        if (editForm) {
            editForm.style.display = 'none';
            editForm.reset();
        }
        if (passwordForm) {
            passwordForm.style.display = 'none';
            passwordForm.reset();
        }

        const notificationsPanel = document.getElementById('my-profile-notifications-panel');
        if (notificationsPanel) {
            notificationsPanel.style.display = 'none';
        }

        document.querySelectorAll('#modal-my-profile .profile-modal__chevron').forEach(
            ch => ch.classList.remove('profile-modal__chevron--open')
        );
    }

    // Закрытие
    document.getElementById('btn-close-my-profile').addEventListener('click', closeMyProfileModal);

    modal.addEventListener('click', (e) => {
        if (e.target === modal) closeMyProfileModal();
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && modal.style.display !== 'none') {
            closeMyProfileModal();
            e.stopImmediatePropagation();
        }
    });

    // Тогглы секций
    setupMyProfileToggle('my-profile-toggle-edit', 'my-profile-edit-form');
    setupMyProfileToggle('my-profile-toggle-password', 'my-profile-password-form');
    setupMyProfileToggle('my-profile-toggle-notifications', 'my-profile-notifications-panel');

    function setupMyProfileToggle(toggleId, formId) {
        const toggle = document.getElementById(toggleId);
        const form = document.getElementById(formId);
        if (!toggle || !form) return;

        const chevron = toggle.querySelector('.profile-modal__chevron');

        toggle.addEventListener('click', () => {
            const isOpen = form.style.display !== 'none';
            form.style.display = isOpen ? 'none' : '';
            if (chevron) chevron.classList.toggle('profile-modal__chevron--open', !isOpen);
        });
    }

    // Редактирование профиля
    const editForm = document.getElementById('my-profile-edit-form');
    if (editForm) {
        editForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            const username = e.target.username.value.trim();
            if (!username) {
                showInfo('Введите новый логин');
                return;
            }

            if (typeof validateUsername === 'function' && !validateUsername(username)) {
                showError(
                    'Логин должен быть не менее 5 символов в длину и содержать только латинские буквы и цифры'
                );
                return;
            }

            try {
                const resp = await fetchWithAuth('/api/users/me', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username }),
                });

                if (!resp.ok) {
                    const text = await resp.text();
                    showError(typeof mapError === 'function' ? mapError(text) : text);
                    return;
                }

                currentUser = await resp.json();

                // Обновляем модалку
                document.getElementById('my-profile-avatar').textContent =
                    currentUser.username.charAt(0).toUpperCase();
                document.getElementById('my-profile-username').textContent = currentUser.username;

                // Обновляем навбар
                const navAvatar = document.querySelector('.navbar__profile-avatar');
                const navName = document.querySelector('.navbar__profile-name');
                if (navAvatar) navAvatar.textContent = currentUser.username.charAt(0).toUpperCase();
                if (navName) navName.textContent = currentUser.username;

                e.target.username.value = '';
                showInfo('Профиль успешно обновлён');
            } catch (err) {
                showError('Ошибка сети: ' + err.message);
            }
        });
    }

    // Смена пароля
    const passwordForm = document.getElementById('my-profile-password-form');
    if (passwordForm) {
        passwordForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            const form = e.target;
            const oldPassword = form.oldPassword.value;
            const newPassword = form.newPassword.value;
            const confirmPassword = form.confirmPassword.value;

            if (!oldPassword || !newPassword || !confirmPassword) {
                showInfo('Заполните все поля');
                return;
            }

            if (typeof validatePassword === 'function' && !validatePassword(newPassword)) {
                showError(
                    'Пароль должен быть на латинице, не менее 8 символов в длину и содержать ' +
                    'как минимум одну букву в верхнем и нижнем регистре, цифру и спецсимвол'
                );
                return;
            }

            if (newPassword !== confirmPassword) {
                showError('Пароли не совпадают');
                return;
            }

            try {
                const resp = await fetchWithAuth('/api/users/password/change', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ oldPassword, newPassword }),
                });

                if (!resp.ok) {
                    const text = await resp.text();
                    showError(typeof mapError === 'function' ? mapError(text) : text);
                    return;
                }

                form.oldPassword.value = '';
                form.newPassword.value = '';
                form.confirmPassword.value = '';
                showInfo('Пароль успешно изменён');
            } catch (err) {
                showError('Ошибка сети: ' + err.message);
            }
        });
    }

    // Уведомления — обработчики тоглов

    async function initNotificationToggles(settings) {
        const emailToggle = document.getElementById('toggle-email-new-message');
        const webPushToggle = document.getElementById('toggle-webpush-new-message');

        updateSwitchUI(emailToggle, hasConsent(settings.emailConsents, CONSENT_NEW_MESSAGE));
        updateSwitchUI(webPushToggle, hasConsent(settings.webPushConsents, CONSENT_NEW_MESSAGE));

        // Авто-синхронизация: если на сервере согласие есть, проверяем подписку в браузере
        if (hasConsent(settings.webPushConsents, CONSENT_NEW_MESSAGE)) {
            // Проверяем, есть ли уже подписка (без запроса разрешения)
            const hasSubscription = await checkExistingWebPushSubscription();

            if (!hasSubscription) {
                if (Notification.permission === 'denied') {
                    // Браузер запретил — сбрасываем серверный бит
                    currentSettings.webPushConsents = currentSettings.webPushConsents & ~CONSENT_NEW_MESSAGE;
                    updateSwitchUI(webPushToggle, false);
                    await saveSettings({ webPushConsents: currentSettings.webPushConsents });
                } else if (Notification.permission === 'default') {
                    if (isIOSSafari()) {
                        // iOS требует user gesture — показываем баннер
                        showWebPushBanner();
                    } else {
                        // Десктоп — запрашиваем разрешение напрямую
                        ensureBrowserWebPushSubscription()
                            .catch(err => console.error('Web push subscription failed:', err));
                    }
                } else if (Notification.permission === 'granted') {
                    // Разрешение есть, но подписки нет — пересоздаём
                    ensureBrowserWebPushSubscription()
                        .catch(err => console.error('Web push subscription failed:', err));
                }
            }
        }
    }

    const emailNewMsgToggle = document.getElementById('toggle-email-new-message');
    if (emailNewMsgToggle) {
        emailNewMsgToggle.addEventListener('click', async () => {
            if (!currentSettings) return;

            currentSettings.emailConsents = toggleConsent(currentSettings.emailConsents, CONSENT_NEW_MESSAGE);
            updateSwitchUI(emailNewMsgToggle, hasConsent(currentSettings.emailConsents, CONSENT_NEW_MESSAGE));
            await saveSettings({ emailConsents: currentSettings.emailConsents });
        });
    }

    const webPushNewMsgToggle = document.getElementById('toggle-webpush-new-message');
    if (webPushNewMsgToggle) {
        webPushNewMsgToggle.addEventListener('click', async () => {
            if (!currentSettings) return;

            const turningOn = !hasConsent(currentSettings.webPushConsents, CONSENT_NEW_MESSAGE);

            if (turningOn) {
                if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
                    showInfo('Web Push-уведомления не поддерживаются вашим браузером');
                    return;
                }

                if (Notification.permission === 'denied') {
                    showInfo('Уведомления заблокированы в настройках браузера');
                    return;
                }

                const ok = await ensureBrowserWebPushSubscription();
                if (!ok) {
                    showInfo('Не удалось подписаться на Web Push-уведомления');
                    return;
                }
            } else {
                await removeBrowserWebPushSubscription();
            }

            currentSettings.webPushConsents = toggleConsent(currentSettings.webPushConsents, CONSENT_NEW_MESSAGE);
            updateSwitchUI(webPushNewMsgToggle, hasConsent(currentSettings.webPushConsents, CONSENT_NEW_MESSAGE));
            await saveSettings({ webPushConsents: currentSettings.webPushConsents });
        });
    }

    // Синхронизация настроек при возврате на вкладку (например, после изменения темы в GUI)
    document.addEventListener('visibilitychange', async () => {
        if (document.visibilityState !== 'visible') return;

        try {
            const settingsResp = await fetchWithAuth('/api/users/me/settings');
            if (!settingsResp.ok) return;

            currentSettings = await settingsResp.json();
            applyTheme(currentSettings.theme === THEME_DARK);
            await initNotificationToggles(currentSettings);
        } catch (e) {
            console.error('Не удалось обновить настройки:', e);
        }
    });

    // Выход
    const logoutBtn = document.getElementById('btn-my-profile-logout');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', async () => {
            try {
                await fetch('/api/sessions', {
                    method: 'DELETE',
                    credentials: 'same-origin',
                });
            } catch (err) {
                console.error(err);
            }

            // Сбрасываем тему на светлую при выходе
            clearTheme();

            // Удаляем куки на клиенте на случай, если сервер не обнулил их корректно
            document.cookie = 'accessToken=; Max-Age=0; path=/';
            document.cookie = 'refreshToken=; Max-Age=0; path=/';

            window.location.href = '/web/login';
        });
    }
});
