# services/settings — Сервис настроек

## Назначение

Управляет настройками пользователя через `SettingsRepository` в рамках UoW-транзакции.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `GetSettingsByUserID` | Получение настроек по ID пользователя. При ошибке оборачивает в `ErrSettingsNotFound` |
| `UpdateSettings` | Обновление настроек + возврат актуального состояния после обновления |

## Зависимости

- Factory function для `SettingsRepository`
- `UnitOfWork`
