// ═══════════════════════════════════════
// Тема: управление через серверные настройки пользователя
// ═══════════════════════════════════════

const THEME_LIGHT = 0;
const THEME_DARK = 1;

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
    try {
        await fetchWithAuth('/api/users/me/settings', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ theme: newDark ? THEME_DARK : THEME_LIGHT }),
        });
    } catch (err) {
        console.error('Не удалось сохранить тему:', err);
    }
}

function updateThemeSwitchUI() {
    const track = document.querySelector('.theme-switch__track');
    const thumb = document.querySelector('.theme-switch__thumb');
    if (!track || !thumb) return;

    const dark = isDarkTheme();
    track.classList.toggle('theme-switch__track--on', dark);
    thumb.classList.toggle('theme-switch__thumb--on', dark);
}

function clearTheme() {
    document.documentElement.removeAttribute('data-bs-theme');
    localStorage.removeItem('theme');
    updateThemeSwitchUI();
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

        // Подтягиваем тему пользователя с сервера
        try {
            const settingsResp = await fetchWithAuth('/api/users/me/settings');
            if (settingsResp.ok) {
                const settings = await settingsResp.json();
                applyTheme(settings.theme === THEME_DARK);
            }
        } catch (e) {
            console.log('Не удалось загрузить настройки:', e);
        }
    } catch (err) {
        console.log(err);
    }

    // --- Модалка профиля ---

    const modal = document.getElementById('modal-my-profile');
    const modalContent = document.getElementById('modal-my-profile-content');
    if (!modal) return;

    function openMyProfileModal(user) {
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

        modal.style.display = '';
    }

    function closeMyProfileModal() {
        modal.style.display = 'none';

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
                console.log(err);
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
