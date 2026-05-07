const EMOJI_CATEGORIES = [
    {
        name: 'smileys',
        label: 'Смайлы',
        icon: '😀',
        emojis: [
            '😀', '😃', '😄', '😁', '😆', '😅', '🤣', '😂',
            '🙂', '😉', '😊', '😇', '🥰', '😍', '🤩', '😘',
            '😋', '😛', '😜', '🤪', '😝', '🤗', '🤔', '🤭',
            '😐', '😑', '😶', '😏', '😒', '🙄', '😬', '😮‍💨',
            '🤥', '😌', '😔', '😪', '🤤', '😴', '😷', '🤒',
            '🤕', '🥴', '😵', '🤯', '🥳', '🥺', '😢', '😭',
            '😤', '😠', '😡', '🤬', '😈', '👿', '💀', '☠️',
        ],
    },
    {
        name: 'gestures',
        label: 'Жесты',
        icon: '👋',
        emojis: [
            '👋', '🤚', '🖐️', '✋', '🖖', '👌', '🤌', '🤏',
            '✌️', '🤞', '🤟', '🤘', '🤙', '👈', '👉', '👆',
            '👇', '☝️', '👍', '👎', '✊', '👊', '🤛', '🤜',
            '👏', '🙌', '👐', '🤲', '🤝', '🙏', '💪', '🦾',
        ],
    },
    {
        name: 'hearts',
        label: 'Сердца',
        icon: '❤️',
        emojis: [
            '❤️', '🧡', '💛', '💚', '💙', '💜', '🖤', '🤍',
            '🤎', '💔', '❤️‍🔥', '💕', '💞', '💓', '💗', '💖',
            '💘', '💝', '💟', '♥️', '❣️', '💋', '🫶', '💑',
        ],
    },
    {
        name: 'objects',
        label: 'Объекты',
        icon: '🎉',
        emojis: [
            '🎉', '🎊', '🎈', '🎁', '🏆', '🥇', '🎯', '🔥',
            '⭐', '🌟', '✨', '💫', '🌈', '☀️', '🌙', '⚡',
            '💡', '📌', '📎', '✏️', '📝', '💬', '💭', '🗨️',
            '✅', '❌', '⚠️', '❓', '❗', '💯', '🆗', '🔔',
        ],
    },
];

/**
 * Создаёт emoji picker и вставляет его в указанный контейнер.
 * При выборе эмодзи вызывает onSelect(emoji).
 */
function createEmojiPicker(container, onSelect) {
    let activeCategory = EMOJI_CATEGORIES[0].name;

    function render() {
        container.innerHTML = '';

        const tabs = document.createElement('div');
        tabs.className = 'emoji-picker__tabs';

        for (const category of EMOJI_CATEGORIES) {
            const tab = document.createElement('button');
            tab.type = 'button';
            tab.className = 'emoji-picker__tab' +
                (activeCategory === category.name ? ' emoji-picker__tab--active' : '');
            tab.title = category.label;
            tab.textContent = category.icon;
            tab.addEventListener('click', () => {
                activeCategory = category.name;
                render();
            });
            tabs.appendChild(tab);
        }

        const grid = document.createElement('div');
        grid.className = 'emoji-picker__grid';

        const current = EMOJI_CATEGORIES.find(c => c.name === activeCategory);
        for (const emoji of current.emojis) {
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'emoji-picker__item';
            btn.textContent = emoji;
            btn.addEventListener('click', () => onSelect(emoji));
            grid.appendChild(btn);
        }

        container.appendChild(tabs);
        container.appendChild(grid);
    }

    render();
}
