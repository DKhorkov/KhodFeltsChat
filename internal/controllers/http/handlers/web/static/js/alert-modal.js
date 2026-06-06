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

/**
 * Модалка подтверждения удаления с двумя кнопками.
 *
 * @param {string} message — текст подтверждения
 * @param {Function} onConfirm — колбэк при подтверждении
 * @param {Object} [options]
 * @param {string} [options.title='Подтверждение']
 * @param {string} [options.confirmText='Удалить']
 * @param {string} [options.cancelText='Отмена']
 * @param {'danger'|'primary'} [options.confirmType='danger']
 */
function showConfirmDelete(message, onConfirm, options = {}) {
    const {
        title = 'Подтверждение',
        confirmText = 'Удалить',
        cancelText = 'Отмена',
        confirmType = 'danger',
    } = options;

    const existing = document.getElementById('alert-modal-overlay');
    if (existing) {
        existing.remove();
    }

    const overlay = document.createElement('div');
    overlay.id = 'alert-modal-overlay';
    overlay.className = 'modal-overlay modal-overlay--alert';

    const modal = document.createElement('div');
    modal.className = 'alert-modal';

    const icon = document.createElement('div');
    icon.className = 'alert-modal__icon';
    icon.textContent = '❓';

    const titleEl = document.createElement('div');
    titleEl.className = 'alert-modal__title';
    titleEl.textContent = title;

    const msg = document.createElement('div');
    msg.className = 'alert-modal__message';
    msg.textContent = message;

    const actions = document.createElement('div');
    actions.className = 'alert-modal__actions';

    const cancelBtn = document.createElement('button');
    cancelBtn.className = 'alert-modal__btn alert-modal__btn--secondary';
    cancelBtn.textContent = cancelText;

    const confirmBtn = document.createElement('button');
    confirmBtn.className = 'alert-modal__btn alert-modal__btn--' + confirmType;
    confirmBtn.textContent = confirmText;

    actions.appendChild(cancelBtn);
    actions.appendChild(confirmBtn);

    modal.appendChild(icon);
    modal.appendChild(titleEl);
    modal.appendChild(msg);
    modal.appendChild(actions);

    overlay.appendChild(modal);
    document.body.appendChild(overlay);

    const closeModal = () => overlay.remove();

    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) closeModal();
    });

    cancelBtn.addEventListener('click', closeModal);

    confirmBtn.addEventListener('click', () => {
        closeModal();

        try {
            const result = onConfirm();
            if (result && typeof result.catch === 'function') {
                result.catch((err) => console.error('Confirm callback failed:', err));
            }
        } catch (err) {
            console.error('Confirm callback failed:', err);
        }
    });

    const onKeydown = (e) => {
        if (e.key === 'Escape') {
            e.stopImmediatePropagation();
            closeModal();
            document.removeEventListener('keydown', onKeydown, true);
        }
    };

    document.addEventListener('keydown', onKeydown, true);

    cancelBtn.focus();
}
