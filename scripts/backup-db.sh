#!/bin/sh
set -eu

# Backup lógico completo do PostgreSQL (Watink)
# Preserva todos os schemas (public, whatsmeow) e dados com integridade referencial.

BACKUP_DIR="${BACKUP_DIR:-/tmp/watink-backups}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
OUTPUT_FILE="${1:-${BACKUP_DIR}/watink_backup_${TIMESTAMP}.dump}"

mkdir -p "$(dirname "$OUTPUT_FILE")"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_ADMIN_USER:-${DB_USER:-postgres}}"
DB_NAME="${DB_NAME:-watink}"
export PGPASSWORD="${DB_ADMIN_PASS:-${DB_PASS:-postgres}}"

echo "==> Iniciando backup do banco '${DB_NAME}' em ${DB_HOST}:${DB_PORT}..."
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -Fc -b -v -f "$OUTPUT_FILE" "$DB_NAME"

echo "==> Backup concluído com sucesso: $OUTPUT_FILE ($(du -h "$OUTPUT_FILE" | cut -f1))"
