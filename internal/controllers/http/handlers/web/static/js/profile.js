document.addEventListener('DOMContentLoaded', async () => {
    const user = await loadCurrentUser();
    if (!user) {
        window.location.href = '/web/login';
        return;
    }

    renderProfile(user);
    setupToggle('toggle-edit-profile', 'edit-profile-form');
    setupToggle('toggle-change-password', 'change-password-form');
    setupEditProfile(user);
    setupChangePassword();
    setupLogout();
});

async function loadCurrentUser() {
    try {
        const resp = await fetchWithAuth('/api/users/me');
        if (!resp.ok) return null;

        return await resp.json();
    } catch {
        return null;
    }
}

function renderProfile(user) {
    document.getElementById('profile-avatar').textContent =
        user.username.charAt(0).toUpperCase();
    document.getElementById('profile-username').textContent = user.username;
    document.getElementById('profile-email').textContent = user.email;

    const confirmedEl = document.getElementById('profile-email-confirmed');
    if (user.emailConfirmed) {
        confirmedEl.textContent = 'Да';
        confirmedEl.classList.add('profile-card__value--success');
    } else {
        confirmedEl.textContent = 'Нет';
        confirmedEl.classList.add('profile-card__value--warning');
    }

    document.getElementById('profile-created-at').textContent =
        new Date(user.createdAt).toLocaleDateString('ru-RU', {
            day: 'numeric',
            month: 'long',
            year: 'numeric',
        });
}

function setupToggle(toggleId, formId) {
    const toggle = document.getElementById(toggleId);
    const form = document.getElementById(formId);
    const chevron = toggle.querySelector('.profile-card__chevron');

    toggle.addEventListener('click', () => {
        const isOpen = form.style.display !== 'none';
        form.style.display = isOpen ? 'none' : '';
        chevron.classList.toggle('profile-card__chevron--open', !isOpen);
    });
}

function setupEditProfile(user) {
    document.getElementById('edit-profile-form').addEventListener('submit', async (e) => {
        e.preventDefault();

        const username = e.target.username.value.trim();
        if (!username) {
            showInfo('Введите новый логин');
            return;
        }

        if (!validateUsername(username)) {
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
                showError(mapError(text));
                return;
            }

            const updatedUser = await resp.json();
            renderProfile(updatedUser);
            e.target.username.value = '';
            showInfo('Профиль успешно обновлён');
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });
}

function setupChangePassword() {
    document.getElementById('change-password-form').addEventListener('submit', async (e) => {
        e.preventDefault();

        const form = e.target;
        const oldPassword = form.oldPassword.value;
        const newPassword = form.newPassword.value;
        const confirmPassword = form.confirmPassword.value;

        if (!oldPassword || !newPassword || !confirmPassword) {
            showInfo('Заполните все поля');
            return;
        }

        if (!validatePassword(newPassword)) {
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
                showError(mapError(text));
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

function setupLogout() {
    document.getElementById('btn-logout').addEventListener('click', async () => {
        try {
            await fetch('/api/sessions', {
                method: 'DELETE',
                credentials: 'same-origin',
            });
        } catch {
            // Игнорируем ошибки при logout
        }

        window.location.href = '/web/login';
    });
}
