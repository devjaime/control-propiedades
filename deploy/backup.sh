#!/usr/bin/env bash
set -e

BACKUP_DIR="${BACKUP_DIR:-/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
DATABASE_URL="${DATABASE_URL:-postgres://control_propiedades:control_propiedades@postgres:5432/control_propiedades}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="backup_${TIMESTAMP}.sql.gz"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Iniciando backup..."

mkdir -p "$BACKUP_DIR"

pg_dump "$DATABASE_URL" | gzip > "$BACKUP_DIR/$BACKUP_NAME"

BACKUP_SIZE=$(du -h "$BACKUP_DIR/$BACKUP_NAME" | cut -f1)

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Backup creado: $BACKUP_NAME ($BACKUP_SIZE)"

if [ -n "$MINIO_ENDPOINT" ] && [ -n "$MINIO_BUCKET" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Subiendo a MinIO..."
    mc alias set backup "$MINIO_ENDPOINT" "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" 2>/dev/null || true
    mc cp "$BACKUP_DIR/$BACKUP_NAME" "backup/$MINIO_BUCKET/backups/" 2>/dev/null || true
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Upload a MinIO completado"
fi

find "$BACKUP_DIR" -name "backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Backup finalizado. Retención: $RETENTION_DAYS días"
