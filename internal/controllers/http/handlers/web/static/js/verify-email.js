document.addEventListener('DOMContentLoaded', async () => {
    const pathParts = window.location.pathname.split('/');
    const token = pathParts[pathParts.length - 1];

    if (!token) {
        window.location.href = '/web/login?message=verify-error&error=' +
            encodeURIComponent('Некорректная ссылка для подтверждения почты.');
        return;
    }

    try {
        const resp = await fetch('/api/users/email/verify/' + encodeURIComponent(token), {
            credentials: 'same-origin',
        });

        if (!resp.ok) {
            const text = await resp.text();
            window.location.href = '/web/login?message=verify-error&error=' +
                encodeURIComponent(mapError(text));
            return;
        }

        window.location.href = '/web/login?message=email-verified';
    } catch (err) {
        window.location.href = '/web/login?message=verify-error&error=' +
            encodeURIComponent('Ошибка сети: ' + err.message);
    }
});
