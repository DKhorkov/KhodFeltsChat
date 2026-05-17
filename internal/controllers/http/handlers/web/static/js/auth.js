/**
 * Обёртка над fetch с автоматическим обновлением токенов при 401.
 * При получении 401 пробует PUT /api/sessions и повторяет запрос.
 */
async function fetchWithAuth(url, options = {}) {
    const opts = { ...options, credentials: 'same-origin' };

    let resp = await fetch(url, opts);

    if (resp.status === 401) {
        const refreshResp = await fetch('/api/sessions', {
            method: 'PUT',
            credentials: 'same-origin',
        });

        if (refreshResp.ok) {
            resp = await fetch(url, opts);
        }
    }

    return resp;
}
