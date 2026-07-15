# web/static/css

## Назначение

CSS-стили веб-интерфейса. Используется BEM-подобная нотация (`блок__элемент--модификатор`).
Все цвета, отступы, шрифты, тени и радиусы заданы через CSS-переменные в `:root` (global.css),
тёмная тема переопределяется через `[data-bs-theme="dark"]`.

---

## global.css — Переменные, сброс, общие компоненты

### CSS-переменные (`:root`)

| Группа | Переменные |
|--------|-----------|
| Размеры | `--navbar-height` |
| Радиусы | `--radius-sm`, `--radius-md`, `--radius-lg`, `--radius-full` |
| Тени | `--shadow-sm`, `--shadow-md`, `--shadow-lg` |
| Отступы | `--space-xs` … `--space-3xl` |
| Шрифты | `--font-xs` … `--font-xl` |
| Анимации | `--transition-fast`, `--transition-base`, `--transition-slow` |
| Фоны | `--bg-app`, `--bg-panel`, `--bg-surface`, `--bg-hover`, `--bg-active`, `--bg-input`, `--bg-message-other`, `--bg-danger`, `--bg-info`, `--bg-toast` |
| Границы | `--border`, `--border-input`, `--border-danger`, `--border-danger-hover` |
| Текст | `--text-primary`, `--text-secondary`, `--text-muted`, `--text-placeholder`, `--text-on-accent`, `--text-toast` |
| Акценты | `--accent`, `--accent-hover`, `--accent-secondary`, `--danger`, `--info`, `--info-text` |

Все переменные переопределяются в `[data-bs-theme="dark"]` для тёмной темы.

### Сброс и типографика

| Селектор | Описание |
|----------|----------|
| `*` | Сброс margin/padding, `box-sizing: border-box` |
| `body` | Системный шрифт, сглаживание, базовый размер `--font-md` |
| `::-webkit-scrollbar` | Кастомный скроллбар (6px, прозрачный трек) |

### Анимации

| Имя | Описание |
|-----|----------|
| `fadeIn` | Плавное появление (opacity 0 → 1) |

### Toggle-переключатель

| Класс | Описание |
|-------|----------|
| `.toggle` | Контейнер, `cursor: pointer` |
| `.toggle__track` | Трек переключателя (44×24px, скруглённый) |
| `.toggle__track--on` | Активное состояние (фон `--accent`) |
| `.toggle__thumb` | Ползунок (20×20px, абсолютное позиционирование) |
| `.toggle__thumb--on` | Сдвиг вправо (`translateX(20px)`) |

### Модалки

| Класс | Описание |
|-------|----------|
| `.modal-overlay` | Полноэкранный оверлей с затемнением, flex-центрирование, `z-index: 1000` |
| `.modal-content` | Контейнер модалки (480px, скруглённый, тень, скролл) |
| `.modal-content__close` | Кнопка закрытия (абсолютное позиционирование, верхний правый угол) |
| `.modal-content__title` | Заголовок модалки |
| `.modal-content__form-group` | Группа формы (label + input/select) |
| `.modal-content__actions` | Контейнер кнопок (flex, выравнивание вправо) |

### Модалка профиля

| Класс | Описание |
|-------|----------|
| `.profile-modal` | Контейнер профиля (420px, фиксированная высота 80vh) |
| `.profile-modal__header` | Шапка с аватаром и именем |
| `.profile-modal__avatar` | Аватар (56×56px, круглый, фон `--accent`) |
| `.profile-modal__title h2` | Имя пользователя |
| `.profile-modal__email` | Email (мелкий шрифт, приглушённый цвет) |
| `.profile-modal__details` | Блок деталей (фон `--bg-panel`) |
| `.profile-modal__row` | Строка «Ключ — Значение» |
| `.profile-modal__label` | Ключ (вторичный цвет) |
| `.profile-modal__value` | Значение |
| `.profile-modal__value--success` | Зелёный текст |
| `.profile-modal__value--warning` | Красный текст (`--danger`) |

### Кнопки

| Класс | Описание |
|-------|----------|
| `.btn--primary` | Основная кнопка (фон `--accent`, белый текст) |
| `.btn--danger` | Красная кнопка (`--danger`) |

---

## chat.css — Страница чатов

### Layout

| Класс | Описание |
|-------|----------|
| `.chat-layout` | Flex-контейнер (высота = viewport − navbar) |
| `.chat-layout--chat-open` | Мобильный: `position: relative`, `overflow: hidden` — для swipe-анимации |

### Sidebar (список чатов)

| Класс | Описание |
|-------|----------|
| `.sidebar` | Левая панель (300px, фон `--bg-panel`, flex-column) |
| `.sidebar__header` | Шапка с заголовком «Чаты» и кнопками |
| `.sidebar__header h3` | Заголовок |
| `.sidebar__icon-btn` | Иконка-кнопка (36×36px, скруглённая) |
| `.sidebar__list` | Скроллируемый список чатов |

### Элемент чата

| Класс | Описание |
|-------|----------|
| `.chat-item` | Строка чата (flex, padding, hover-эффект) |
| `.chat-item--active` | Выбранный чат (фон `--bg-active`) |
| `.chat-item__avatar` | Аватар чата (44×44px, круглый, масштабирование при hover) |
| `.chat-item__info` | Контейнер текста (flex: 1, min-width: 0 для ellipsis) |
| `.chat-item__title` | Название чата (ellipsis при переполнении) |
| `.chat-item__title--bold` | Жирный шрифт (непрочитанный чат) |
| `.chat-item__last-message` | Превью последнего сообщения (мелкий шрифт `--font-sm`, цвет `--text-secondary`, ellipsis) |
| `.chat-item__unread-dot` | Точка непрочитанного (8×8px, фон `--accent`) |

### Переписка (conversation)

| Класс | Описание |
|-------|----------|
| `.conversation` | Правая панель (flex: 1, flex-column) |
| `.conversation__header` | Шапка с названием чата и кнопкой закрытия |
| `.conversation__header-title` | Название (кликабельное, hover → `--accent`) |
| `.conversation__close-btn` | Кнопка закрытия (×) |
| `.conversation__messages` | Контейнер сообщений (flex-column, скролл, gap) |
| `.conversation__unread-divider` | Разделитель «Новые сообщения» (красная линия с текстом) |
| `.conversation__placeholder` | Заглушка «Выберите чат» |

### Ввод сообщения (composer)

| Класс | Описание |
|-------|----------|
| `.conversation__composer` | Контейнер ввода (flex, border-top, `position: relative` — якорь для scroll-down кнопки) |
| `.conversation__composer-input` | Обёртка textarea |
| `.conversation__composer-input textarea` | Поле ввода (resize: none, focus → `--accent`) |
| `.conversation__composer button` | Кнопка отправки (фон `--accent`) |

### Кнопка «к последнему сообщению»

| Класс | Описание |
|-------|----------|
| `.conversation__scroll-down` | Полупрозрачная круглая кнопка (44×44px на всех размерах экрана — соответствует Apple HIG / WCAG 2.5.5 минимум для тач-целей), позиционирована абсолютно от верхнего края composer (`bottom: 100%` + `margin-bottom`). На mobile (≤600px) сдвинута на `right: 12px` (вместо 16px). При hover становится непрозрачной, при active — лёгкое сжатие |
| `.conversation__scroll-down[hidden]` | `display: none` — переопределяет `display: flex` базового селектора, чтобы атрибут `hidden` фактически скрывал кнопку (без этого правила author-CSS перебивал user-agent `[hidden] { display: none }`) |
| `.conversation__scroll-down-icon` | Inline SVG-стрелка вниз (22×22px, `currentColor`) |
| `.conversation__scroll-down-badge` | Бейдж счётчика непрочитанных (правый верхний угол, фон `--danger`, минимум 20×20px, поддерживает «99+») |

### Пузырь сообщения

| Класс | Описание |
|-------|----------|
| `.message-bubble` | Пузырь (max-width: 70%, скруглённый, фон `--bg-message-other`) |
| `.message-bubble--own` | Своё сообщение (справа, фон `--accent`) |
| `.message-bubble__header` | Имя отправителя + время |
| `.message-bubble__text` | Текст (pre-wrap, word-wrap) |

### Контекстное меню сообщения

| Класс | Описание |
|-------|----------|
| `.context-menu` | Всплывающее меню действий над сообщением (`position: fixed`, `z-index: 3000`). `user-select: none` + `-webkit-touch-callout: none` — чтобы long press на мобильных не вызывал нативное выделение/callout на пунктах меню (меню открывается прямо под пальцем) |
| `.context-menu__item` | Пункт меню (`display: block`, `width: 100%`, hover → `--bg-hover`) |
| `.context-menu__item--danger` | Опасный пункт (цвет `--danger`) |
| `.context-menu__reactions` | Полоса реакций (горизонтальный скролл) |
| `.context-menu__reaction` | Кнопка реакции |
| `.context-menu__reaction--active` | Активная реакция (обводка `--accent`) |

### Список пользователей (в модалках)

| Класс | Описание |
|-------|----------|
| `.modal-content__users-list` | Контейнер списка (max-height: 300px, скролл) |
| `.modal-content__no-results` | Заглушка «Нет результатов» |
| `.user-item` | Строка пользователя |
| `.user-item--clickable` | Кликабельная строка |
| `.user-item--selectable` | Выделяемая строка (hover → `--bg-active`) |
| `.user-item__checkbox` | Чекбокс выбора |
| `.user-item__avatar` | Аватар пользователя (40×40px, масштабирование при hover) |
| `.user-item__info` | Контейнер текста |
| `.user-item__name` | Имя пользователя |
| `.user-item__email` | Email |

### Модалка группового чата

| Класс | Описание |
|-------|----------|
| `.group-chat-modal__members-header` | Заголовок «Участники» |
| `.group-chat-modal__members-list` | Список участников (max-height: 300px) |
| `.group-chat-modal__members-list .user-item__avatar` | Аватар без hover-эффекта |
| `.group-chat-modal__member-badge` | Бейдж роли участника |

### Toast-уведомления

| Класс | Описание |
|-------|----------|
| `.toast-container` | Контейнер (fixed, bottom-right, z-index: 2000) |
| `.toast` | Уведомление (320px, анимация toastIn/toastOut) |
| `.toast__body` | Тело (flex: 1) |
| `.toast__sender` | Имя отправителя |
| `.toast__text` | Текст сообщения (ellipsis) |
| `.toast__close` | Кнопка закрытия |

### Анимации

| Имя | Описание |
|-----|----------|
| `toastIn` | Появление toast справа (opacity + translateX) |
| `toastOut` | Исчезновение toast вправо |

### Мобильная адаптация (`@media max-width: 600px`)

- `.chat-layout` → column, sidebar на всю ширину, высота через CSS-переменную `--vh` (устанавливается JS из `visualViewport.height`)
- `.conversation` → absolute позиционирование поверх sidebar (для swipe-to-close), высота через `--vh` (реагирует на клавиатуру)
- `.chat-item__last-message` → многострочное превью (до 2 строк, `-webkit-line-clamp: 2`)
- `.message-bubble` → max-width: 85%, `will-change: transform` (для свайпа влево), `user-select: none` + `-webkit-touch-callout: none` (подавляет нативное выделение и callout Copy/Look Up при long press)
- `.message-bubble--pressing` → анимация увеличения при long press (`scale(1) → scale(1.05)` за 350мс, `longPressScale` keyframes)
- textarea и input → `font-size: 16px` (предотвращает zoom на iOS)
- `.modal-overlay--keyboard-open` → `height` ограничена `--vh * 100` (видимая область над клавиатурой), `bottom: auto`. Модалка центрируется в оставшемся пространстве (класс добавляется/убирается JS через `visualViewport.resize`)
- `.modal-overlay--keyboard-open .modal-content` → `max-height` через `--vh` (не вылезает за видимую область)
- `.toast-container` → сверху, на всю ширину

---

## navbar.css — Навбар и модалка профиля

### Навбар

| Класс | Описание |
|-------|----------|
| `.navbar` | Flex-контейнер (padding, border-bottom) |
| `.navbar__brand` | Логотип/название (цвет `--accent`, `margin-right: auto`) |
| `.navbar__auth` | Контейнер кнопок авторизации |
| `.navbar__btn` | Кнопка навбара |
| `.navbar__btn--login` | Кнопка входа (фон `--accent`) |
| `.navbar__profile` | Кнопка профиля (аватар + имя, pill-форма) |
| `.navbar__profile-avatar` | Аватар в навбаре (28×28px) |
| `.navbar__profile-name` | Имя (ellipsis, max-width: 120px) |

### Секции модалки профиля

| Класс | Описание |
|-------|----------|
| `.profile-modal__section` | Аккордеон-секция (фон `--bg-panel`) |
| `.profile-modal__toggle` | Заголовок секции (кликабельный) |
| `.profile-modal__chevron` | Стрелка раскрытия |
| `.profile-modal__chevron--open` | Повёрнутая стрелка (90°) |
| `.profile-modal__form` | Форма внутри секции |
| `.profile-modal__logout-actions` | Контейнер кнопки выхода |

### Уведомления (в профиле)

| Класс | Описание |
|-------|----------|
| `.profile-modal__notifications` | Контейнер секции уведомлений |
| `.notifications__group` | Группа переключателей |
| `.notifications__group-label` | Заголовок группы |
| `.notifications__toggle-row` | Строка с toggle и подписью |
| `.notifications__channel-label` | Подпись канала (email/push) |
| `.profile-modal__theme-row` | Строка переключателя темы |

### Баннер web-push

| Класс | Описание |
|-------|----------|
| `.web-push-banner` | Фиксированный баннер сверху (фон `--accent`, z-index: 1100) |
| `.web-push-banner__text` | Текст баннера |
| `.web-push-banner__btn` | Кнопка «Разрешить» |
| `.web-push-banner__close` | Кнопка закрытия баннера |

### Контекстное меню и увеличение аватара (navbar.css)

| Класс | Описание |
|-------|----------|
| `.profile-modal__avatar-wrapper` | Обёртка аватара (`position: relative`, якорь для меню) |
| `.avatar-context-menu` | Выпадающее меню под аватаром (фон `--bg-surface`, граница `--border`, `z-index: 1000`) |
| `.avatar-context-menu__item` | Пункт меню (полная ширина, hover `--bg-hover`) |
| `.avatar-context-menu__item--danger` | Опасный пункт (цвет `--danger`, hover с красным оттенком) |
| `.avatar-zoom-overlay` | Полноэкранный оверлей для увеличенного аватара (`z-index: 2000`, фон `rgba(0,0,0,0.85)`) |
| `.avatar-zoom-overlay__img` | Картинка аватара (`max: 90vw/90vh`) |
| `.avatar-zoom-overlay__close` | Кнопка «×» в правом верхнем углу |

### Анимации

| Имя | Описание |
|-----|----------|
| `slideDown` | Появление баннера сверху (translateY) |

### Мобильная адаптация (`@media max-width: 600px`)

- `.navbar` → уменьшенные padding
- `.modal-content`, `.profile-modal` → ширина `calc(100vw - 40px)`

---

## login.css — Страница входа/регистрации

| Класс | Описание |
|-------|----------|
| `.login-container` | Flex-центрирование, градиентный фон |
| `.login-card` | Карточка формы (440px, тень) |
| `.login-card__tabs` | Табы «Вход / Регистрация» |
| `.login-card__tab` | Таб (flex: 1) |
| `.login-card__tab--active` | Активный таб (подчёркнутый `--accent`) |
| `.login-card__tab-content` | Контент таба |
| `.login-form` | Форма (flex-column, gap) |
| `.login-form__input` | Поле ввода |
| `.login-form__submit` | Кнопка отправки |
| `.login-form__links` | Контейнер ссылок |
| `.login-form__link` | Ссылка |
| `.login-form__link--danger` | Красная ссылка |

---

## forget-password.css — Страница сброса пароля

| Класс | Описание |
|-------|----------|
| `.forget-password-container` | Flex-центрирование, градиентный фон |
| `.forget-password-card` | Карточка формы (440px) |
| `.forget-password-card__title` | Заголовок |
| `.forget-password-card__info` | Подсказка |
| `.forget-password-card__form` | Форма (flex-column) |
| `.forget-password-card__actions` | Контейнер кнопок |
| `.btn--secondary` | Вторичная кнопка (фон `--bg-hover`) |

---

## error.css — Страница ошибки

| Класс | Описание |
|-------|----------|
| `.error-container` | Flex-центрирование, градиентный фон |
| `.error-card` | Карточка ошибки (440px) |
| `.error-card__icon` | Иконка (48px) |
| `.error-card__title` | Заголовок (цвет `--danger`) |
| `.error-card__code` | Код ошибки (48px, крупный шрифт) |
| `.error-card__message` | Текст ошибки |
| `.error-card__btn` | Кнопка «На главную» |

---

## alert-modal.css — Модалка уведомлений

| Класс | Описание |
|-------|----------|
| `.modal-overlay--alert` | Оверлей с повышенным z-index (3000) |
| `.alert-modal` | Контейнер (400px, text-align: center) |
| `.alert-modal__icon` | Иконка (40px) |
| `.alert-modal__title` | Заголовок |
| `.alert-modal__message` | Текст сообщения |
| `.alert-modal__btn` | Кнопка OK |
| `.alert-modal__btn--error` | Красная кнопка |
| `.alert-modal__btn--info` | Синяя кнопка (`--accent`) |
| `.alert-modal__btn--danger` | Красная кнопка (`--danger`) — для подтверждения опасного действия |
| `.alert-modal__btn--primary` | Кнопка-акцент (`--accent`) — для нейтрального подтверждения |
| `.alert-modal__btn--secondary` | Серая кнопка (`--bg-hover`) — для «Отмена» |
| `.alert-modal__actions` | Контейнер двух кнопок (flex, gap, fill width) для модалки подтверждения |

---

## emoji-picker.css — Emoji picker

| Класс | Описание |
|-------|----------|
| `.emoji-picker` | Контейнер (absolute, над textarea, max 320px) |
| `.emoji-picker__grid` | Сетка emoji (8 колонок, max-height: 260px) |
| `.emoji-picker__item` | Элемент emoji (hover → scale 1.15) |
| `.conversation__emoji-wrapper` | Обёртка кнопки emoji (absolute, правый нижний угол textarea) |
| `.conversation__emoji-toggle` | Кнопка открытия picker |
| `.conversation__emoji-toggle--active` | Активное состояние (opacity: 1) |

### Мобильная адаптация (`@media max-width: 600px`)

- `.conversation__emoji-toggle` и `.emoji-picker` — скрыты (`display: none !important`). На мобильных устройствах используется встроенная клавиатура с эмодзи (iOS, Android).

---

## home.css — Главная страница

| Класс | Описание |
|-------|----------|
| `.hero` | Flex-центрирование, градиентный фон (на всю высоту) |
| `.hero__content` | Контент (max-width: 600px, центрирование) |
| `.hero__title` | Заголовок (48px, белый) |
| `.hero__subtitle` | Подзаголовок (полупрозрачный белый) |
| `.hero__actions` | Контейнер кнопок |
| `.hero__btn` | Кнопка |
| `.hero__btn--primary` | Основная кнопка (белый фон, цвет `--accent`) |
| `.hero__btn--secondary` | Вторичная кнопка (прозрачная, белая рамка) |

### Мобильная адаптация (`@media max-width: 600px`)

- Заголовок → 32px
- Кнопки → column layout
