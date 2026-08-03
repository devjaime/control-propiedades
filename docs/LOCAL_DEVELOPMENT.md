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
make migrate-up
```

Para revertir únicamente la última migración local:

```bash
make migrate-down
```

## Procesos

```bash
make api-dev
make web-dev
```

La API falla al iniciar si una variable tipada es inválida. `/health/live` verifica el proceso y `/health/ready` verifica PostgreSQL. La web reenvía `/api/*` a la API para mantener las cookies en el mismo origen durante desarrollo.
