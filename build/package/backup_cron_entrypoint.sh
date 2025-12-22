#!/bin/bash
# backup_cron_entrypoint.sh — Устанавливает pg_dump 17 и запускает цикл бэкапов

set -e  # Останавливаем скрипт при любой ошибке

echo "🔄 Начинаем инициализацию контейнера..."

# Обновляем пакеты
apt-get update

# Устанавливаем зависимости
apt-get install -y ca-certificates wget gnupg bash gzip

# Добавляем репозиторий PostgreSQL 17
echo "📦 Добавляем репозиторий PostgreSQL 17..."
echo "deb https://apt.postgresql.org/pub/repos/apt noble-pgdg main" > /etc/apt/sources.list.d/pgdg.list

# Скачиваем и устанавливаем ключ GPG (без TTY)
echo "🔐 Устанавливаем GPG-ключ PostgreSQL..."
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | \
gpg --batch --yes --dearmor --output /etc/apt/trusted.gpg.d/postgresql.gpg

# Устанавливаем клиент
echo "🔧 Устанавливаем postgresql-client-17..."
apt-get update
apt-get install -y postgresql-client-17

# Проверяем установку
if ! command -v pg_dump &> /dev/null; then
 echo "❌ Ошибка: pg_dump не установлен!"
 exit 1
fi

pg_dump --version

# Делаем скрипт бэкапа исполняемым
if [ ! -f "/scripts/backup.sh" ]; then
 echo "❌ Ошибка: /scripts/backup.sh не найден!"
 exit 1
fi

chmod +x /scripts/backup.sh

# Проверяем переменную окружения
if [ -z "${BACKUP_CRON_INTERVAL}" ]; then
 echo "⚠️ Переменная BACKUP_CRON_INTERVAL не задана. Используем значение по умолчанию: 86400 (24 часа)"
 export BACKUP_CRON_INTERVAL=86400
else
 echo "✅ Интервал бэкапа: ${BACKUP_CRON_INTERVAL} секунд"
fi

# Запускаем цикл
echo "🟢 Установка завершена. Запуск планировщика бэкапов..."
while :; do
 echo "🔄 Запуск бэкапа..."
 /scripts/backup.sh
 echo "💤 Ждём ${BACKUP_CRON_INTERVAL} секунд до следующего бэкапа..."
 sleep "${BACKUP_CRON_INTERVAL}"
done
