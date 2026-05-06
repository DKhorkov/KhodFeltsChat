document.addEventListener('DOMContentLoaded', async () => {
    const authContainer = document.getElementById('navbar-auth');
    if (!authContainer) return;

    try {
        const resp = await fetchWithAuth('/api/users/me');
        if (!resp.ok) return;

        const user = await resp.json();

        authContainer.innerHTML = '';

        const profile = document.createElement('a');
        profile.href = '/web/profile';
        profile.className = 'navbar__profile';

        const avatar = document.createElement('div');
        avatar.className = 'navbar__profile-avatar';
        avatar.textContent = user.username.charAt(0).toUpperCase();

        const name = document.createElement('span');
        name.className = 'navbar__profile-name';
        name.textContent = user.username;

        profile.appendChild(avatar);
        profile.appendChild(name);
        authContainer.appendChild(profile);
    } catch {
        // Не авторизован — оставляем кнопку "Войти"
    }
});
