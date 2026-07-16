# Arquitectura

## Decisión inicial

Control Propiedades usa un monolito modular. La API Go concentra reglas de negocio, autorización, auditoría y persistencia. La aplicación Next.js consume exclusivamente la API. PostgreSQL es la fuente oficial de estado; MinIO/S3 conserva archivos privados.

```text
Navegador/PWA -> API Go -> PostgreSQL
                         -> MinIO/S3
                         -> Outbox (futuro)
Agentes       -> API de agentes (futura)
```

OpenClaw y HermesAgents no tendrán acceso directo a PostgreSQL ni al almacenamiento. En una fase posterior usarán identidades técnicas, permisos explícitos y endpoints auditados.

## Estructura

```text
apps/api/       API HTTP y módulos Go
apps/web/       interfaz Next.js
migrations/     migraciones PostgreSQL
docs/           arquitectura y operación
```

## Multi-tenancy

- Las entidades de negocio incluyen `organization_id`.
- La organización se obtendrá de la sesión autenticada.
- Los índices únicos de negocio incluyen la organización cuando corresponda.
- Las pruebas de cada módulo deben cubrir intentos de acceso entre organizaciones.

Durante esta iteración el esquema prepara estos límites, pero aún no expone endpoints de negocio.
