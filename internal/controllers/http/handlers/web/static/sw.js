self.addEventListener('install', async () => {
    await self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    event.waitUntil(clients.claim());
});

self.addEventListener('push', (event) => {
    const data = event.data ? event.data.json() : {};

    const title = data.title || 'Новое сообщение';
    const options = {
        body: data.body || '',
        icon: '/web/static/assets/icon.png',
        timestamp: data.timestamp || Date.now(),
        data: {
            chatId: data.chatId,
        },
    };

    event.waitUntil((async () => {
        // Бейдж PWA: сервер всегда присылает абсолютное число непрочитанных,
        // ставим его до показа уведомления, чтобы iOS не успел убить SW.
        if ('setAppBadge' in self.navigator && typeof data.unreadCount === 'number') {
            try {
                if (data.unreadCount > 0) {
                    await self.navigator.setAppBadge(data.unreadCount);
                } else {
                    await self.navigator.clearAppBadge();
                }
            } catch (err) {
                console.warn('setAppBadge failed:', err);
            }
        }

        const windowClients = await clients.matchAll({ type: 'window', includeUncontrolled: true });
        // На десктопе не показываем уведомление, если чат в фокусе.
        // На iOS подавлять нельзя — iOS требует showNotification на каждый push,
        // иначе молча отбрасывает уведомление и может отозвать разрешение.
        const isIOS = /iP(hone|ad|od)/.test(self.navigator.userAgent);
        if (!isIOS) {
            const hasFocusedClient = windowClients.some(
                (client) => client.focused && client.url.includes('/web/chat')
            );
            if (hasFocusedClient) return;
        }

        await self.registration.showNotification(title, options);
    })());
});

self.addEventListener('notificationclick', (event) => {
    event.notification.close();

    const chatId = event.notification.data && event.notification.data.chatId;

    event.waitUntil(
        clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
            for (const client of windowClients) {
                if (client.url.includes('/web/chat') && 'focus' in client) {
                    if (chatId) {
                        client.postMessage({ type: 'open-chat', chatId: chatId });
                    }
                    return client.focus();
                }
            }

            const url = chatId ? '/web/chat?chatId=' + chatId : '/web/chat';

            return clients.openWindow(url);
        })
    );
});
