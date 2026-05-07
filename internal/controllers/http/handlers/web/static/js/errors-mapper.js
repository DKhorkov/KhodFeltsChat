/**
 * ErrorsMapper — маппинг серверных ошибок в пользовательские сообщения на русском.
 * Повторяет логику internal/errors/mapper.go из KhodFeltsChatGUI.
 *
 * Маппинг отсортирован по длине ключа (убывание),
 * чтобы более специфичные ключи проверялись первыми.
 */

const ERROR_DEFAULT = 'Что-то пошло не так...';

const errorMapping = [
    // Auth
    { key: 'new password can not be equal to old password', message: 'Старый пароль не может быть использован в качестве нового пароля' },
    { key: 'access token does not belong to refresh token', message: 'Ошибка авторизации' },
    { key: 'invalid jwt token: invalid forget_password_token', message: 'Некорректный код для сброса пароля' },
    { key: 'email already confirmed', message: 'Эта почта уже подтверждена' },
    { key: 'email not confirmed', message: 'Почта не была подтверждена' },
    { key: 'email already exist', message: 'Этот почтовый адрес уже занят' },
    { key: 'passwords does not match', message: 'Пароли не совпадают' },
    { key: 'user already exists', message: 'Пользователь с такой почтой или логином уже существует' },
    { key: 'invalid jwt token', message: 'Ошибка авторизации' },
    { key: 'validation failed', message: 'Ошибка валидации данных' },
    { key: 'invalid password', message: 'Пароль должен быть на латинице, не менее 8 символов в длину и содержать как минимум одну букву в верхнем и нижнем регистре, цифру и спецсимвол' },
    { key: 'invalid login', message: 'Некорректный email или логин. Логин должен быть не менее 5 символов в длину и содержать только латинские буквы и цифры' },
    { key: 'invalid username', message: 'Логин должен быть не менее 5 символов в длину и содержать только латинские буквы и цифры' },
    { key: 'invalid email', message: 'Некорректный адрес электронной почты' },
    { key: 'registration failed', message: 'Ошибка регистрации' },
    { key: 'wrong password', message: 'Неверный логин или пароль' },
    { key: 'token expired', message: 'Токен устарел' },
    { key: 'login failed', message: 'Неверный логин или пароль' },
    { key: 'user not found', message: 'Такого пользователя не существует' },
    { key: 'limit exceeded', message: 'Превышен лимит. Попробуйте позже' },

    // Users
    { key: 'duplicate key value violates unique constraint "users_username_key"', message: 'Пользователь с таким логином уже существует' },

    // Chats
    { key: 'user is not a chat member', message: 'У вас нет доступа к этому чату' },
    { key: 'chat already exists', message: 'Такой чат уже существует' },
    { key: 'failed to get chat messages', message: 'Не удалось получить сообщения для чата' },
    { key: 'failed to get user chats', message: 'Не удалось получить чаты пользователя' },
    { key: 'failed to create chat', message: 'Не удалось создать чат' },
    { key: 'chat not found', message: 'Чат не найден' },
    { key: 'invalid chat', message: 'Неверный чат' },
];

function mapError(rawMessage) {
    if (!rawMessage) {
        return ERROR_DEFAULT;
    }

    const msg = rawMessage.toLowerCase();
    for (const entry of errorMapping) {
        if (msg.includes(entry.key)) {
            return entry.message;
        }
    }

    return ERROR_DEFAULT;
}
