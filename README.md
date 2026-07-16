# Control Propiedades

Aplicación web para administrar propiedades, documentos, arriendos, pagos y mantenimiento. El proyecto parte como una herramienta personal y conserva límites de organización para evolucionar a SaaS.

## Estado

Primera iteración de fundación:

- API Go con endpoints de salud.
- Web Next.js con App Router.
- PostgreSQL y MinIO en Docker Compose.
- Configuración por variables de entorno.
- Primera migración del dominio base.

Todavía no incluye autenticación, CRUD, carga documental ni agentes.

## Requisitos

- Go 1.26 o superior.
- Node.js 20.9 o superior.
- Docker con Compose.
- Goose para ejecutar migraciones (`go install github.com/pressly/goose/v3/cmd/goose@v3.27.2`).

## Inicio local

```bash
cp .env.example .env
make bootstrap
make infra-up
set -a && source .env && set +a
cd apps/api && go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ../../migrations postgres "$DATABASE_URL" up
```

En terminales separadas:

```bash
make api-dev
make web-dev
```

- Web: http://localhost:3000
- API: http://localhost:8080
- MinIO Console: http://localhost:9001

## Verificación

```bash
make test
make build
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

Consulta [la arquitectura](docs/ARCHITECTURE.md) y [el desarrollo local](docs/LOCAL_DEVELOPMENT.md) para más detalles.
