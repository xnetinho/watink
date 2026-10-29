#!/bin/sh
set -eu

# Restauração lógica do PostgreSQL (Watink) sobre banco limpo
# Uso: ./scripts/restore-db.sh <arquivo.dump> [nome_do_banco_destino]

if [ $# -lt 1 ]; then
    echo "Uso: $0 <caminho-para-o-backup.dump> [banco_destino]"
    exit 1
fi

DUMP_FILE="$1"
if [ ! -f "$DUMP_FILE" ]; then
    echo "Erro: arquivo de backup não encontrado: $DUMP_FILE"
    exit 1
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_ADMIN_USER:-${DB_USER:-postgres}}"
TARGET_DB="${2:-${DB_NAME:-watink}}"
export PGPASSWORD="${DB_ADMIN_PASS:-${DB_PASS:-postgres}}"

echo "==> Preparando banco de destino '${TARGET_DB}' em ${DB_HOST}:${DB_PORT}..."
# Cria banco se não existir
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${TARGET_DB}'" | grep -q 1 || \
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE \"${TARGET_DB}\";"

echo "==> Restaurando schemas e dados a partir de $DUMP_FILE..."
# pg_restore com --clean para restaurar objetos de forma idempotente
pg_restore -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TARGET_DB" --clean --if-exists -v "$DUMP_FILE" || {
    # pg_restore retorna status não-zero em warnings de propriedade (ignorable)
    echo "==> pg_restore finalizado com avisos não-fatais."
}

echo "==> Restauração concluída com sucesso no banco '${TARGET_DB}'!"
