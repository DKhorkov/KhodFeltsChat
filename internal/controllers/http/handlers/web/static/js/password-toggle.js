(function () {
    'use strict';

    const EYE_OPEN_SVG = '<svg class="password-field__icon password-field__icon--hidden" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>';
    const EYE_OFF_SVG = '<svg class="password-field__icon password-field__icon--visible" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17.94 17.94A10.94 10.94 0 0 1 12 20c-7 0-11-8-11-8a19.77 19.77 0 0 1 5.06-5.94M9.9 4.24A10.94 10.94 0 0 1 12 4c7 0 11 8 11 8a19.87 19.87 0 0 1-2.16 3.19"/><path d="M1 1l22 22"/><path d="M14.12 14.12a3 3 0 1 1-4.24-4.24"/></svg>';

    function wrapInput(input) {
        if (input.dataset.passwordToggle === 'processed') {
            return;
        }
        input.dataset.passwordToggle = 'processed';

        const wrapper = document.createElement('div');
        wrapper.className = 'password-field';
        input.classList.add('password-field__input');

        input.parentNode.insertBefore(wrapper, input);
        wrapper.appendChild(input);

        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'password-field__toggle';
        button.setAttribute('aria-label', 'Показать пароль');
        button.setAttribute('aria-pressed', 'false');
        button.setAttribute('tabindex', '-1');
        button.innerHTML = EYE_OPEN_SVG + EYE_OFF_SVG;

        button.addEventListener('click', function () {
            const isHidden = input.type === 'password';
            input.type = isHidden ? 'text' : 'password';
            button.setAttribute('aria-pressed', String(isHidden));
            button.setAttribute('aria-label', isHidden ? 'Скрыть пароль' : 'Показать пароль');
        });

        wrapper.appendChild(button);
    }

    function scan(root) {
        const inputs = (root || document).querySelectorAll('input[type="password"]');
        inputs.forEach(wrapInput);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function () {
            scan(document);
        });
    } else {
        scan(document);
    }

    // Templates like navbar inject password inputs after initial load — watch for them.
    const observer = new MutationObserver(function (mutations) {
        mutations.forEach(function (mutation) {
            mutation.addedNodes.forEach(function (node) {
                if (node.nodeType !== Node.ELEMENT_NODE) {
                    return;
                }
                if (node.matches && node.matches('input[type="password"]')) {
                    wrapInput(node);
                } else if (node.querySelectorAll) {
                    scan(node);
                }
            });
        });
    });

    observer.observe(document.body || document.documentElement, {
        childList: true,
        subtree: true,
    });
})();
