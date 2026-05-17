# Куки аутентификации

Приложение использует две куки для аутентификации: `accessToken` и `refreshToken`.
Конфигурация задаётся через переменные окружения в `.env`.

## Переменные окружения

### Access Token

| Переменная | Описание | По умолчанию |
|------------|----------|:------------:|
| `COOKIES_ACCESS_TOKEN_PATH` | Path куки | `/` |
| `COOKIES_ACCESS_TOKEN_DOMAIN` | Domain куки | пусто |
| `COOKIES_ACCESS_TOKEN_MAX_AGE` | Max-Age в секундах (0 — не задан) | `0` |
| `COOKIES_ACCESS_TOKEN_EXPIRES` | Время жизни в минутах | `15` |
| `COOKIES_ACCESS_TOKEN_SECURE` | Отправлять только по HTTPS | `false` |
| `COOKIES_ACCESS_TOKEN_HTTP_ONLY` | Недоступна из JavaScript | `false` |
| `COOKIES_ACCESS_TOKEN_SAME_SITE` | SameSite политика (1=Default, 2=Lax, 3=Strict, 4=None) | `1` |

### Refresh Token

| Переменная | Описание | По умолчанию |
|------------|----------|:------------:|
| `COOKIES_REFRESH_TOKEN_PATH` | Path куки | `/` |
| `COOKIES_REFRESH_TOKEN_DOMAIN` | Domain куки | пусто |
| `COOKIES_REFRESH_TOKEN_MAX_AGE` | Max-Age в секундах (0 — не задан) | `0` |
| `COOKIES_REFRESH_TOKEN_EXPIRES` | Время жизни в часах | `168` (7 дней) |
| `COOKIES_REFRESH_TOKEN_SECURE` | Отправлять только по HTTPS | `false` |
| `COOKIES_REFRESH_TOKEN_HTTP_ONLY` | Недоступна из JavaScript | `false` |
| `COOKIES_REFRESH_TOKEN_SAME_SITE` | SameSite политика (1=Default, 2=Lax, 3=Strict, 4=None) | `1` |

## Настройка для HTTPS (production)

При деплое с HTTPS (nginx + SSL) необходимо выставить в `.env`:

```env
COOKIES_ACCESS_TOKEN_SECURE=true
COOKIES_ACCESS_TOKEN_HTTP_ONLY=true
COOKIES_ACCESS_TOKEN_SAME_SITE=2

COOKIES_REFRESH_TOKEN_SECURE=true
COOKIES_REFRESH_TOKEN_HTTP_ONLY=true
COOKIES_REFRESH_TOKEN_SAME_SITE=2
```

| Флаг | Зачем |
|------|-------|
| `Secure=true` | Куки отправляются только по HTTPS. Без этого флага куки утекут при случайном HTTP-запросе |
| `HttpOnly=true` | Куки недоступны из `document.cookie` — защита от XSS |
| `SameSite=Lax` | Куки отправляются при навигации на сайт, но не при cross-origin запросах — защита от CSRF |

## Настройка для локальной разработки (HTTP)

Дефолтные значения (`Secure=false`, `HttpOnly=false`, `SameSite=Default`) рассчитаны
на локальную разработку по HTTP (`localhost:8080`). Менять ничего не нужно.

## Где используются

Куки устанавливаются в хэндлерах:
- `login` — при входе в аккаунт
- `register` — при регистрации
- `refresh_tokens` — при обновлении токенов
- `logout` — при выходе (очистка кук)
