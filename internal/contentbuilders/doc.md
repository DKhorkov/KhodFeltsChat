# contentbuilders — Генерация email контента

## Назначение

Генерация HTML тела и темы для исходящих email-ов. Также сохраняет одноразовые 6-значные коды в Redis.

## Модули

### verify_email/
`ContentBuilder{baseURL, cacheProvider}`:
- `Subject()` — тема письма
- `Body(ctx, user)`:
  1. Генерирует 6-значный код (`common.GenerateOTP`) в диапазоне `[100000, 999999]`
  2. Атомарно сохраняет `{verify_email_token}:{code}` = userID через `SetNX` + `Get` (проверка на коллизию, до `OTPGenerateAttempts` попыток), TTL = 15 мин
  3. Возвращает HTML со ссылкой `baseURL/{code}` и кодом в теле

### forget_password/
Аналогичный подход:
- Код показан в теле письма (в тег `<b>`)
- Ключ Redis: `{forget_password_token}:{code}` = userID

### new_message/
`ContentBuilder`:
- `Subject()` — тема письма для уведомления о новом сообщении
- `Body(ctx, user)` — возвращает HTML-шаблон уведомления о новом сообщении

## Гарантии безопасности

- **Атомарность записи**: `SetNX` + `Get` — если код уже занят другим пользователем, регенерируем; собственную запись не перезатираем
- **Одноразовость**: код удаляется из кэша при использовании (в `CacheDecorator` usecases)
- **TTL**: автоматическое истечение через 15 минут
- **Коллизии**: до `OTPGenerateAttempts` попыток, затем `ErrOTPCollision`
