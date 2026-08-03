#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "========================================="
echo "  Control Propiedades - Deploy"
echo "========================================="

if [ ! -f "$PROJECT_DIR/.env" ]; then
    echo "[ERROR] No se encontró .env. Copia .env.production.example a .env y configura."
    exit 1
fi

cd "$PROJECT_DIR"

echo "[1/6] Creando directorios necesarios..."
mkdir -p letsencrypt data/certbot backups

echo "[2/6] Deteniendo servicios anteriores..."
docker compose -f docker-compose.prod.yml down --remove-orphans || true

echo "[3/6] Construyendo imágenes..."
docker compose -f docker-compose.prod.yml build --no-cache

echo "[4/6] Iniciando servicios..."
docker compose -f docker-compose.prod.yml up -d

echo "[5/6] Verificando salud..."
sleep 15

if curl -sf http://localhost/health > /dev/null 2>&1; then
    echo "  ✓ Salud verificada"
else
    echo "[ERROR] El servicio no respondió. Revisá los logs:"
    docker compose -f docker-compose.prod.yml logs
    exit 1
fi

echo "[6/6] Verificando certificados SSL..."
if [ -d "letsencrypt/live" ]; then
    echo "  ✓ SSL ya configurado"
else
    echo "  ℹ SSL no configurado aún. Ejecuta: ./deploy/setup-ssl.sh <dominio> <email>"
fi

IP=$(curl -s ifconfig.me 2>/dev/null || echo "YOUR_VPS_IP")

echo ""
echo "========================================="
echo "  ✓ Deploy exitoso!"
echo "========================================="
echo "  Web:        https://$IP"
echo "  API:        https://$IP/api"
echo "  MinIO:      https://$IP:9000"
echo ""
echo "  Ver logs:   docker compose -f docker-compose.prod.yml logs -f"
echo "  Stop:       docker compose -f docker-compose.prod.yml down"
echo ""
echo "  SSL:        ./deploy/setup-ssl.sh <dominio> <email>"
echo "  Restore:    gunzip < backups/backup_YYYYMMDD_HHMMSS.sql.gz | psql"
