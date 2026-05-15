self.addEventListener('push', (event) => {
    const data = event.data ? event.data.json() : {};

    const title = data.title || 'Новое сообщение';
    const options = {
        body: data.body || '',
        icon: '/web/static/assets/icon.png',
        badge: '/web/static/assets/icon.png',
        timestamp: data.timestamp || Date.now(),
        vibrate: [200, 100, 200],
        data: {
            chatId: data.chatId,
        },
    };

    event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
    event.notification.close();

    const chatId = event.notification.data && event.notification.data.chatId;

    event.waitUntil(
        clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
            for (const client of windowClients) {
                if (client.url.includes('/web/chat') && 'focus' in client) {
                    return client.focus();
                }
            }

            const url = chatId ? '/web/chat?chatId=' + chatId : '/web/chat';

            return clients.openWindow(url);
        })
    );
});
