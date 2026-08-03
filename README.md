# Control Propiedades

Aplicación web para administrar propiedades, documentos, arriendos, pagos y mantenimiento. El proyecto parte como una herramienta personal y conserva límites de organización para evolucionar a SaaS.

## Estado

Funcionalidades disponibles:

- Registro de cuenta y organización.
- Inicio y cierre de sesión mediante cookie segura.
- Registro de propiedades con aislamiento por organización.
- Carga privada de PDF e imágenes en MinIO.
- Clasificación por propiedad, tipo y etiquetas.
- Búsqueda documental y descarga autorizada.
- Detección de archivos duplicados mediante SHA-256.
- Bandeja de revisión, metadatos ampliados y verificación humana inmutable.
- Contratos con arrendatario fijo durante cada vigencia y cierre auditado.
- Arriendo mensual dividido entre una y doce cuotas con días configurables.
- Conciliación humana de depósitos y comprobante PDF con QR público verificable.
- Alertas de pagos, vencimiento contractual, restitución, reajuste IPC y mantenciones.
- Calendario trimestral de alertas y sugerencias de revisión.

La integración con OpenClaw, HermesAgents y otros agentes todavía está planificada; los datos operativos ya se almacenan estructurados para permitir su análisis posterior.

Consulta [el estado detallado de implementación](docs/IMPLEMENTATION_STATUS.md), [el diseño documental](docs/DOCUMENT_STORAGE.md), [el flujo de arriendos y pagos](docs/RENTALS_AND_PAYMENTS.md) y [la contratación asistida](docs/LEASE_CONTRACT_WORKFLOW.md).

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
make migrate-up
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

## Despliegue en la nube

La aplicación puede desplegarse con Vercel para la web y la API Go, Neon para PostgreSQL y Supabase Storage para archivos privados. Consulta la [guía de despliegue paso a paso](docs/CLOUD_DEPLOYMENT.md).
