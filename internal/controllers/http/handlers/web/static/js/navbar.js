document.addEventListener('DOMContentLoaded', async () => {
    const authContainer = document.getElementById('navbar-auth');
    const user = await getCurrentUser();
    if (!user) return;

    authContainer.innerHTML = '';

    const profile = document.createElement('a');
    profile.href = '/web/chat';
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
});

async function getCurrentUser() {
    try {
        let resp = await fetch('/api/users/me', { credentials: 'same-origin' });

        if (resp.status === 401) {
            const refreshResp = await fetch('/api/sessions', {
                method: 'PUT',
                credentials: 'same-origin',
            });

            if (!refreshResp.ok) return null;

            resp = await fetch('/api/users/me', { credentials: 'same-origin' });
        }

        if (!resp.ok) return null;

        return await resp.json();
    } catch {
        return null;
    }
}
