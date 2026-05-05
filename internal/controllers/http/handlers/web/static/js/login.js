document.addEventListener('DOMContentLoaded', () => {
    const tabs = document.querySelectorAll('.login-card__tab');
    const tabLogin = document.getElementById('tab-login');
    const tabRegister = document.getElementById('tab-register');
    const messageEl = document.getElementById('message');

    // Переключение табов
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('login-card__tab--active'));
            tab.classList.add('login-card__tab--active');

            if (tab.dataset.tab === 'login') {
                tabLogin.style.display = '';
                tabRegister.style.display = 'none';
            } else {
                tabLogin.style.display = 'none';
                tabRegister.style.display = '';
            }
            clearMessage();
        });
    });

    function showError(text) {
        messageEl.innerHTML = '<div class="login-form__error">' + text + '</div>';
    }

    function showInfo(text) {
        messageEl.innerHTML = '<div class="login-form__info">' + text + '</div>';
    }

    function clearMessage() {
        messageEl.innerHTML = '';
    }

    // Логин
    document.getElementById('login-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        clearMessage();

        const form = e.target;
        const login = form.login.value.trim();
        const password = form.password.value;

        if (!login || !password) {
            showInfo('Пожалуйста, заполните все поля');
            return;
        }

        try {
            const resp = await fetch('/api/sessions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'same-origin',
                body: JSON.stringify({ login, password }),
            });

            if (!resp.ok) {
                const text = await resp.text();
                showError(text || 'Ошибка входа');
                return;
            }

            window.location.href = '/web/chat';
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });

    // Регистрация
    document.getElementById('register-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        clearMessage();

        const form = e.target;
        const email = form.email.value.trim();
        const username = form.username.value.trim();
        const password = form.password.value;
        const confirmPassword = form.confirmPassword.value;

        if (!email || !username || !password || !confirmPassword) {
            showInfo('Пожалуйста, заполните все поля');
            return;
        }

        if (password !== confirmPassword) {
            showError('Пароли не совпадают');
            return;
        }

        try {
            const resp = await fetch('/api/users', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'same-origin',
                body: JSON.stringify({ email, username, password }),
            });

            if (!resp.ok) {
                const text = await resp.text();
                showError(text || 'Ошибка регистрации');
                return;
            }

            // Переключаемся на таб логина
            tabs[0].click();
            document.querySelector('#login-form input[name="login"]').value = email;
            document.querySelector('#login-form input[name="password"]').value = password;
            showInfo('Регистрация прошла успешно. Теперь войдите');
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });

    // Повторная отправка письма для подтверждения почты
    document.getElementById('btn-verify-email').addEventListener('click', async () => {
        clearMessage();
        const login = document.querySelector('#login-form input[name="login"]').value.trim();

        if (!login) {
            showInfo('Введите email в поле авторизации');
            return;
        }

        try {
            const resp = await fetch('/api/users/email/verify', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'same-origin',
                body: JSON.stringify({ email: login }),
            });

            if (!resp.ok) {
                const text = await resp.text();
                showError(text || 'Ошибка отправки');
                return;
            }

            showInfo('Письмо для подтверждения почты отправлено на ' + login);
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });

    // Сброс пароля
    document.getElementById('btn-forget-password').addEventListener('click', async () => {
        clearMessage();
        const login = document.querySelector('#login-form input[name="login"]').value.trim();

        if (!login) {
            showInfo('Введите email в поле авторизации');
            return;
        }

        try {
            const resp = await fetch('/api/users/password/forget', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'same-origin',
                body: JSON.stringify({ email: login }),
            });

            if (!resp.ok) {
                const text = await resp.text();
                showError(text || 'Ошибка отправки');
                return;
            }

            showInfo('Письмо с кодом для сброса пароля отправлено на ' + login);
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });
});
