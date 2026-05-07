/**
 * Validation — клиентская валидация полей форм.
 * Повторяет правила из ValidationConfig в KhodFeltsChatGUI.
 */

const EMAIL_REGEXP = /^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$/;

const PASSWORD_RULES = [
    { regexp: /.{8,}/,    message: 'не менее 8 символов' },
    { regexp: /[a-z]/,    message: 'строчную латинскую букву' },
    { regexp: /[A-Z]/,    message: 'заглавную латинскую букву' },
    { regexp: /[0-9]/,    message: 'цифру' },
    { regexp: /[^\d\w]/,  message: 'спецсимвол' },
];

const USERNAME_RULES = [
    { regexp: /^.{5,70}$/,      message: 'от 5 до 70 символов' },
    { regexp: /^[A-Za-z0-9]+$/, message: 'только латинские буквы и цифры' },
];

function validateEmail(value) {
    return EMAIL_REGEXP.test(value.toLowerCase());
}

function validateLogin(value) {
    return validateEmail(value) || validateUsername(value);
}

function validateUsername(value) {
    return USERNAME_RULES.every(rule => rule.regexp.test(value));
}

function validatePassword(value) {
    return PASSWORD_RULES.every(rule => rule.regexp.test(value));
}
