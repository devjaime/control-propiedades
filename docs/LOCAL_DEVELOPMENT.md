# Desarrollo local

## Infraestructura

`docker-compose.yml` levanta PostgreSQL y MinIO. Ambos usan volúmenes persistentes. El proceso `minio-init` crea un bucket privado e idempotente.

```bash
cp .env.example .env
make infra-up
docker compose ps
```

## Migraciones

Las migraciones usan SQL con anotaciones de Goose:

```bash
set -a
source .env
set +a
cd apps/api
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 \
  -dir ../../migrations postgres "$DATABASE_URL" up
```

Para revertir únicamente la última migración local:

```bash
cd apps/api
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 \
  -dir ../../migrations postgres "$DATABASE_URL" down
```

## Procesos

```bash
make api-dev
make web-dev
```

La API falla al iniciar si una variable tipada es inválida. `/health/live` verifica el proceso y `/health/ready` verifica PostgreSQL.
