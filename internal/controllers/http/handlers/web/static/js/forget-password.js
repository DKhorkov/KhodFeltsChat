document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('forget-password-form').addEventListener('submit', async (e) => {
        e.preventDefault();

        const form = e.target;
        const token = form.token.value.trim();
        const newPassword = form.newPassword.value;
        const confirmPassword = form.confirmPassword.value;

        if (!token) {
            showError('Введите код из письма');
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
            const resp = await fetch('/api/users/password/forget/' + encodeURIComponent(token), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'same-origin',
                body: JSON.stringify({ newPassword }),
            });

            if (!resp.ok) {
                const text = await resp.text();
                showError(mapError(text));
                return;
            }

            window.location.href = '/web/login?message=password-reset';
        } catch (err) {
            showError('Ошибка сети: ' + err.message);
        }
    });
});
