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

    modal.innerHTML =
        '<div class="alert-modal__icon">' + (isError ? '❌' : 'ℹ️') + '</div>' +
        '<div class="alert-modal__title">' + (isError ? 'Ошибка' : 'Информация') + '</div>' +
        '<div class="alert-modal__message">' + message + '</div>' +
        '<button class="alert-modal__btn alert-modal__btn--' + type + '">OK</button>';

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

    // Закрытие по Escape
    const onKeydown = (e) => {
        if (e.key === 'Escape') {
            closeModal();
            document.removeEventListener('keydown', onKeydown);
        }
    };

    document.addEventListener('keydown', onKeydown);

    // Фокус на кнопку OK для доступности
    modal.querySelector('.alert-modal__btn').focus();
}

function showError(message) {
    showAlert(message, 'error');
}

function showInfo(message) {
    showAlert(message, 'info');
}
