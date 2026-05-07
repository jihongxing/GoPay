#!/bin/bash
set -e

if [ "${SERVER_ENV}" = "production" ]; then
  : "${ADMIN_API_KEY:?ADMIN_API_KEY must be set in production}"
  : "${PUBLIC_BASE_URL:?PUBLIC_BASE_URL must be set in production}"
fi

# 等待数据库就绪
echo "Waiting for database..."
until PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; do
  echo "Database is unavailable - sleeping"
  sleep 2
done

echo "Database is up - running migrations"

# 运行迁移（使用安全的 golang-migrate 工具）
if [ "$RUN_MIGRATIONS" = "true" ]; then
  echo "Running database migrations..."
  /app/migrate up
  if [ $? -ne 0 ]; then
    echo "Migration failed!"
    exit 1
  fi
  echo "Migrations completed successfully"
else
  echo "Skipping migrations (RUN_MIGRATIONS != true)"
fi

# 启动应用
echo "Starting application..."
exec "$@"
