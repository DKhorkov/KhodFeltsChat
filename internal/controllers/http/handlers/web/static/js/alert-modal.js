/**
 * AlertModal — кастомная модалка для отображения ошибок и информационных сообщений.
 * Повторяет дизайн AlertModal из KhodFeltsChatGUI.
 *
 * Использование:
 *   showError('Текст ошибки');
 *   showInfo('Информационное сообщение');
 */

function showAlert(message, type) {
    // Удаляем предыдущую модалку, если есть
    const existing = document.getElementById('alert-modal-overlay');
    if (existing) {
        existing.remove();
    }

    const isError = type === 'error';

    const overlay = document.createElement('div');
    overlay.id = 'alert-modal-overlay';
    overlay.className = 'modal-overlay modal-overlay--alert';

    const modal = document.createElement('div');
    modal.className = 'alert-modal';

    const icon = document.createElement('div');
    icon.className = 'alert-modal__icon';
    icon.textContent = isError ? '❌' : 'ℹ️';

    const title = document.createElement('div');
    title.className = 'alert-modal__title';
    title.textContent = isError ? 'Ошибка' : 'Информация';

    const msg = document.createElement('div');
    msg.className = 'alert-modal__message';
    msg.textContent = message;

    const btn = document.createElement('button');
    btn.className = 'alert-modal__btn alert-modal__btn--' + type;
    btn.textContent = 'OK';

    modal.appendChild(icon);
    modal.appendChild(title);
    modal.appendChild(msg);
    modal.appendChild(btn);

    overlay.appendChild(modal);
    document.body.appendChild(overlay);

    const closeModal = () => overlay.remove();

    // Закрытие по клику на оверлей
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) {
            closeModal();
        }
    });

    // Закрытие по кнопке OK
    modal.querySelector('.alert-modal__btn').addEventListener('click', closeModal);

    // Закрытие по Escape (перехватываем на фазе capture, чтобы сработать раньше всех)
    const onKeydown = (e) => {
        if (e.key === 'Escape') {
            e.stopImmediatePropagation();
            closeModal();
            document.removeEventListener('keydown', onKeydown, true);
        }
    };

    document.addEventListener('keydown', onKeydown, true);

    // Фокус на кнопку OK для доступности
    modal.querySelector('.alert-modal__btn').focus();
}

function showError(message) {
    showAlert(message, 'error');
}

function showInfo(message) {
    showAlert(message, 'info');
}
