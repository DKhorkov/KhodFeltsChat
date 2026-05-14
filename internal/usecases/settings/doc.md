# Пакет usecases/settings

## Назначение

Бизнес-логика для работы с пользовательскими настройками: получение и обновление.

## Ключевые операции

### GetSettingsByUserID
- Делегирует вызов в `SettingsService`.

### UpdateSettings
- Делегирует обновление настроек в `SettingsService`.
- Возвращает обновлённый объект настроек.

## Зависимости

- `internal/interfaces` — `SettingsService`.
- `internal/domains` — `Settings`.
