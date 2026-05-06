document.addEventListener('DOMContentLoaded', async () => {
    const iconEl = document.getElementById('verify-icon');
    const titleEl = document.getElementById('verify-title');
    const messageEl = document.getElementById('verify-message');

    // Токен берётся из URL: /web/verify-email/{token}
    const pathParts = window.location.pathname.split('/');
    const token = pathParts[pathParts.length - 1];

    if (!token) {
        iconEl.textContent = '❌';
        titleEl.textContent = 'Ошибка';
        messageEl.textContent = 'Некорректная ссылка для подтверждения почты.';
        return;
    }

    try {
        const resp = await fetch('/api/users/email/verify/' + encodeURIComponent(token), {
            credentials: 'same-origin',
        });

        if (!resp.ok) {
            const text = await resp.text();
            iconEl.textContent = '❌';
            titleEl.textContent = 'Ошибка';
            messageEl.textContent = mapError(text);
            return;
        }

        window.location.href = '/web/login?message=email-verified';
    } catch (err) {
        iconEl.textContent = '❌';
        titleEl.textContent = 'Ошибка';
        messageEl.textContent = 'Ошибка сети: ' + err.message;
    }
});
