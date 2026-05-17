# contentbuilders — Генерация email контента

## Назначение

Генерация HTML тела и темы для исходящих email-ов. Также сохраняет одноразовые токены в Redis.

## Модули

### verify_email/
`ContentBuilder{baseURL, cacheProvider}`:
- `Subject()` — тема письма
- `Body(ctx, user)`:
  1. Генерирует UUID salt
  2. Кодирует `salt:userID` в base64 как токен
  3. Сохраняет в Redis: ключ `verifyEmailToken:{userID}`, TTL = 15 мин
  4. Возвращает HTML с ссылкой `baseURL/token`

### forget_password/
Аналогичный подход:
- Токен не в ссылке, а в теле письма (raw)
- Ключ Redis: `forgetPasswordToken:{userID}`

### new_message/
`ContentBuilder`:
- `Subject()` — тема письма для уведомления о новом сообщении
- `Body(ctx, user)` — возвращает HTML-шаблон уведомления о новом сообщении

## Гарантии безопасности

- **Один токен на пользователя**: новый перезаписывает старый в Redis
- **Одноразовость**: токен удаляется из кэша при использовании (в CacheDecorator usecases)
- **TTL**: автоматическое истечение через 15 минут
