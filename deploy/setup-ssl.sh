#!/usr/bin/env bash
set -e

DOMAIN="${1:-}"
EMAIL="${2:-}"

if [ -z "$DOMAIN" ]; then
    echo "Usage: ./setup-ssl.sh <domain> <email>"
    echo "Example: ./setup-ssl.sh propiedades.midominio.com admin@midominio.com"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LETSENCRYPT_DIR="$PROJECT_DIR/letsencrypt"

echo "========================================="
echo "  SSL Setup - Let's Encrypt"
echo "========================================="
echo "  Domain: $DOMAIN"
echo "  Email:  $EMAIL"
echo ""

mkdir -p "$LETSENCRYPT_DIR"

echo "[1/3] Creando contenedor temporal para certbot..."
docker run --rm \
    -v "$LETSENCRYPT_DIR:/etc/letsencrypt" \
    -p 80:80 \
    --entrypoint sh \
    certbot/certbot:latest \
    -c "tail -f /dev/null" &
CERTBOT_PID=$!
sleep 3

echo "[2/3] Deteniendo nginx temporalmente..."
docker compose -f "$PROJECT_DIR/docker-compose.prod.yml" stop nginx
sleep 2

echo "[3/3] Obteniendo certificado Let's Encrypt..."
docker run --rm \
    -v "$LETSENCRYPT_DIR:/etc/letsencrypt" \
    -p 80:80 \
    --entrypoint certbot \
    certbot/certbot:latest \
    certonly --standalone \
    --domains "$DOMAIN" \
    --email "$EMAIL" \
    --agree-tos \
    --no-eff-email \
    --keep-until-expiring \
    --standalone-supported-challenges http-01 || {
    echo "[WARN] SSL no pudo ser obtenido. Continuando sin SSL."
    kill $CERTBOT_PID 2>/dev/null || true
    docker compose -f "$PROJECT_DIR/docker-compose.prod.yml" start nginx
    exit 1
}

kill $CERTBOT_PID 2>/dev/null || true

echo "[OK] Certificados obtenidos. Iniciando nginx..."
docker compose -f "$PROJECT_DIR/docker-compose.prod.yml" start nginx

echo ""
echo "========================================="
echo "  ✓ SSL configurado!"
echo "========================================="
echo "  Certificados en: $LETSENCRYPT_DIR/live/$DOMAIN/"
echo ""
echo "  El renew automático está configurado en"
echo "  docker-compose.prod.yml (servicio certbot)."
