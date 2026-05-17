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

    const tabs = document.createElement('div');
    tabs.className = 'emoji-picker__tabs';

    const grid = document.createElement('div');
    grid.className = 'emoji-picker__grid';

    for (const category of EMOJI_CATEGORIES) {
        const tab = document.createElement('span');
        tab.className = 'emoji-picker__tab';
        tab.dataset.category = category.name;
        tab.title = category.label;
        tab.textContent = category.icon;
        tab.addEventListener('click', () => {
            activeCategory = category.name;
            updateTabHighlight();
            renderGrid();
        });
        tabs.appendChild(tab);
    }

    function updateTabHighlight() {
        for (const t of tabs.children) {
            t.classList.toggle('emoji-picker__tab--active', t.dataset.category === activeCategory);
        }
    }

    function renderGrid() {
        grid.innerHTML = '';
        const current = EMOJI_CATEGORIES.find(c => c.name === activeCategory);
        for (const emoji of current.emojis) {
            const item = document.createElement('span');
            item.className = 'emoji-picker__item';
            item.textContent = emoji;
            item.addEventListener('click', () => onSelect(emoji));
            grid.appendChild(item);
        }
    }

    container.appendChild(tabs);
    container.appendChild(grid);
    updateTabHighlight();
    renderGrid();
}
