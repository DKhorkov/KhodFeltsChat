document.addEventListener('DOMContentLoaded', () => {
    const params = new URLSearchParams(window.location.search);
    const message = params.get('message');
    if (message === 'password-reset') {
        showInfo('Пароль был успешно сброшен. Теперь вы можете авторизоваться.');
        history.replaceState(null, '', window.location.pathname);
    } else if (message === 'email-verified') {
        showInfo('Почта успешно подтверждена. Теперь вы можете войти.');
        history.replaceState(null, '', window.location.pathname);
    } else if (message === 'verify-error') {
        showError(params.get('error') || 'Не удалось подтвердить почту.');
        history.replaceState(null, '', window.location.pathname);
    }

    const tabs = document.querySelectorAll('.login-card__tab');
    const tabLogin = document.getElementById('tab-login');
    const tabRegister = document.getElementById('tab-register');

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
        });
    });

    // Логин
    document.getElementById('login-form').addEventListener('submit', async (e) => {
        e.preventDefault();

        const form = e.target;
        const login = form.login.value.trim();
        const password = form.password.value;

        if (!validateLogin(login)) {
            showError(
                'Некорректный email или логин. ' +
                'Логин должен быть не менее 5 символов в длину и содержать только латинские буквы и цифры'
            );
            return;
        }

        if (!validatePassword(password)) {
            showError(
                'Пароль должен быть на латинице, не менее 8 символов в длину и содержать ' +
                'как минимум одну букву в верхнем и нижнем регистре, цифру и спецсимвол'
            );
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
                showError(mapError(text));
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

        const form = e.target;
        const email = form.email.value.trim();
        const username = form.username.value.trim();
        const password = form.password.value;
        const confirmPassword = form.confirmPassword.value;

        if (!validateEmail(email)) {
            showError('Некорректный адрес электронной почты');
            return;
        }

        if (!validateUsername(username)) {
            showError(
                'Логин должен быть не менее 5 символов в длину и содержать только латинские буквы и цифры'
            );
            return;
        }

        if (!validatePassword(password)) {
            showError(
                'Пароль должен быть на латинице, не менее 8 символов в длину и содержать ' +
                'как минимум одну букву в верхнем и нижнем регистре, цифру и спецсимвол'
            );
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
                showError(mapError(text));
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
        const login = document.querySelector('#login-form input[name="login"]').value.trim();

        if (!validateEmail(login)) {
            showError('Некорректный адрес электронной почты');
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
                showError(mapError(text));
                return;
            }

            showInfo('Письмо для подтверждения почты отправлено на ' + login);
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });

    // Сброс пароля
    document.getElementById('btn-forget-password').addEventListener('click', async () => {
        const login = document.querySelector('#login-form input[name="login"]').value.trim();

        if (!validateEmail(login)) {
            showError('Некорректный адрес электронной почты');
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
                showError(mapError(text));
                return;
            }

            window.location.href = '/web/forget-password?email=' + encodeURIComponent(login);
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });
});
