#!/bin/bash

# === Настройки ===
BACKUP_DIR="/backups"
LOG_FILE="$BACKUP_DIR/backup.log"
DUMP_FILE="${1:-latest.sql.gz}"
BACKUP_FILE="$BACKUP_DIR/$DUMP_FILE"

# === Проверка переменных ===
if [ -z "$POSTGRES_USER" ] || [ -z "$POSTGRES_DB" ] || [ -z "$POSTGRES_PORT" ] || [ -z "$POSTGRES_PASSWORD" ] || [ -z "$POSTGRES_HOST" ]; then
    echo "❌ Не все переменные окружения заданы: POSTGRES_USER, POSTGRES_DB, POSTGRES_PORT, POSTGRES_PASSWORD, POSTGRES_HOST" | tee -a "$LOG_FILE"
    exit 1
fi

# === Проверка существования файла ===
if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ Файл бэкапа не найден: $BACKUP_FILE" | tee -a "$LOG_FILE"
    exit 1
fi

export PGPASSWORD="$POSTGRES_PASSWORD"

echo "🔄 Начинаем восстановление базы '$POSTGRES_DB' из: $BACKUP_FILE" | tee -a "$LOG_FILE"

echo "🔌 Завершаем активные подключения к базе '$POSTGRES_DB'..." | tee -a "$LOG_FILE"
psql -U "$POSTGRES_USER" -d postgres -c "
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = '$POSTGRES_DB'
  AND pid <> pg_backend_pid();
" > /dev/null 2>&1

echo "🗑️ Удаляем старую базу данных..." | tee -a "$LOG_FILE"
psql -U "$POSTGRES_USER" -d postgres -c "DROP DATABASE IF EXISTS $POSTGRES_DB;" > /dev/null 2>&1

echo "🆕 Создаём новую базу данных..." | tee -a "$LOG_FILE"
psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE $POSTGRES_DB OWNER $POSTGRES_USER;" > /dev/null 2>&1

echo "📥 Распаковываем и восстанавливаем данные..." | tee -a "$LOG_FILE"
gunzip -c "$BACKUP_FILE" | psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" > /dev/null 2>&1

if [ $? -ne 0 ]; then
 echo "❌ Ошибка при восстановлении данных из $BACKUP_FILE!" | tee -a "$LOG_FILE"
 exit 1
fi

echo "✅ Данные восстановлены. Проверяем целостность..." | tee -a "$LOG_FILE"

# Количество таблиц в схеме public
TABLE_COUNT=$(psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT count(*) FROM pg_tables WHERE schemaname = 'public';" 2>/dev/null | xargs)

if [ -z "$TABLE_COUNT" ] || ! [[ "$TABLE_COUNT" =~ ^[0-9]+$ ]]; then
 echo "❌ Не удалось определить количество таблиц. Возможна ошибка подключения или повреждение БД." | tee -a "$LOG_FILE"
 exit 1
fi

echo "📊 В схеме public обнаружено таблиц: $TABLE_COUNT" | tee -a "$LOG_FILE"

if [ -z "$TABLE_COUNT" ] || [[ $TABLE_COUNT -eq 0 ]]; then
 echo "❌ Восстановленная база пуста — возможно, дамп был повреждён или пуст." | tee -a "$LOG_FILE"
 exit 1
fi

# Дополнительная проверка: простой запрос
HEALTH_CHECK=$(psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT 1;" 2>/dev/null | xargs)

if [ -z "$HEALTH_CHECK" ] || [[ "$HEALTH_CHECK" != "1" ]]; then
 echo "❌ База не прошла health-check: SELECT 1 вернул пустой результат." | tee -a "$LOG_FILE"
 exit 1
fi

# === Финал ===
echo "✅ Восстановление успешно завершено и проверено:" | tee -a "$LOG_FILE"
echo "   - Файл: $BACKUP_FILE" | tee -a "$LOG_FILE"
echo "   - База: $POSTGRES_DB" | tee -a "$LOG_FILE"
echo "   - Таблиц в public: $TABLE_COUNT" | tee -a "$LOG_FILE"
echo "   - Health-check: пройден" | tee -a "$LOG_FILE"
