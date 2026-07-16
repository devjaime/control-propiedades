# Prompt técnico — Control Propiedades

## Instrucción para Codex

Actúa como **arquitecto de software, desarrollador full-stack senior y especialista en seguridad**. Debes construir una aplicación web multiusuario denominada provisionalmente **Control Propiedades**.

El sistema será inicialmente de uso personal, pero debe quedar preparado para evolucionar a un SaaS multiempresa de administración inmobiliaria con agentes de IA.

Debes trabajar sobre el repositorio actual, respetar sus convenciones y evitar reescrituras innecesarias.

Antes de modificar código:

1. Inspecciona el repositorio.
2. Detecta lenguajes, frameworks, versiones y estructura.
3. Ejecuta las pruebas existentes.
4. Identifica riesgos.
5. Propón una iteración pequeña.
6. Implementa cambios completos, no pseudocódigo.
7. Agrega migraciones, pruebas y documentación.
8. No agregues microservicios.
9. No almacenes secretos en el repositorio.
10. No realices acciones destructivas sin explicar el impacto.

---

# 1. Stack objetivo

## Backend

Preferencia:

- Go.
- API HTTP REST.
- Arquitectura de monolito modular.
- Framework HTTP: Echo, Fiber o Chi, priorizando el ya existente en el repositorio.
- PostgreSQL.
- `pgx` para conexión.
- `sqlc` o repositorios explícitos para acceso tipado.
- Migraciones versionadas con Goose, Atlas o herramienta equivalente.
- Validación de entrada.
- OpenAPI.
- Jobs internos y tabla outbox.
- Tests unitarios y de integración.

No usar ORM pesado salvo que el repositorio ya dependa de uno y cambiarlo no sea razonable.

## Frontend

Preferencia:

- TypeScript.
- Next.js o React con framework equivalente.
- Aplicación responsive.
- PWA instalable.
- Diseño mobile-first para carga documental y operación diaria.
- Formularios tipados.
- Cliente generado o contratos compartidos con OpenAPI.
- Manejo de sesión seguro.
- Componentes accesibles.

## Base de datos

- PostgreSQL local existente.
- UUID como identificadores externos.
- Migraciones idempotentes y versionadas.
- Restricciones, claves foráneas e índices.
- Fechas almacenadas en UTC.
- Zona horaria configurable por organización.
- Dinero almacenado como entero en unidad mínima o decimal exacto.
- Soporte para CLP, UF y otras monedas.
- No usar `float` para montos.

## Almacenamiento documental

Implementar una interfaz desacoplada:

```go
type ObjectStorage interface {
    Put(ctx context.Context, input PutObjectInput) (StoredObject, error)
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}
```

Adaptadores:

- Local filesystem para desarrollo simple.
- MinIO para entorno local compatible con S3.
- S3 compatible o Google Cloud Storage para producción.

El registro en PostgreSQL guarda metadatos, hash, tamaño, MIME y clave del objeto. No guardar binarios grandes en PostgreSQL.

## Desarrollo local

Usar Docker Compose con:

- PostgreSQL.
- MinIO.
- Backend.
- Frontend.
- Servicio de correo de desarrollo, por ejemplo Mailpit.
- Opcionalmente un worker separado solo si el repositorio ya lo justifica.

La aplicación debe poder ejecutarse también usando PostgreSQL local externo mediante variables de entorno.

## Producción

Arquitectura compatible con:

- Backend en contenedor.
- Frontend desplegable en Vercel, Cloud Run o contenedor.
- PostgreSQL administrado.
- Object storage administrado.
- Secret manager.
- HTTPS obligatorio.
- Backups.
- Observabilidad.

No acoplar la lógica a un proveedor específico.

---

# 2. Arquitectura

Usar un monolito modular con límites claros.

Estructura sugerida:

```text
/
├── apps/
│   ├── api/
│   └── web/
├── internal/
│   ├── auth/
│   ├── organizations/
│   ├── users/
│   ├── properties/
│   ├── ownership/
│   ├── tenants/
│   ├── leases/
│   ├── billing/
│   ├── payments/
│   ├── receipts/
│   ├── documents/
│   ├── obligations/
│   ├── mortgages/
│   ├── maintenance/
│   ├── vendors/
│   ├── notifications/
│   ├── agents/
│   ├── audit/
│   └── platform/
├── migrations/
├── docs/
├── deployments/
└── docker-compose.yml
```

Adapta la estructura si el repositorio ya posee una convención distinta.

Cada módulo debe separar:

- Dominio.
- Casos de uso.
- Persistencia.
- Transporte HTTP.
- Validaciones.
- Eventos.
- Pruebas.

No sobrediseñar con arquitectura hexagonal ceremonial. Mantener dependencias explícitas y simples.

---

# 3. Multi-tenancy

Preparar multi-tenancy desde el inicio.

Reglas:

- Todas las tablas de negocio incluyen `organization_id`.
- Toda consulta debe filtrar por organización.
- El `organization_id` se obtiene de la sesión, no del cuerpo enviado por el cliente.
- No confiar en identificadores proporcionados por el frontend.
- Incorporar pruebas que intenten acceso cruzado entre organizaciones.
- Usar middleware de contexto de organización.
- Evaluar Row Level Security en PostgreSQL como defensa adicional, sin depender exclusivamente de ella.
- Las claves únicas que correspondan deben incluir `organization_id`.

Ejemplo:

```sql
CREATE UNIQUE INDEX ux_properties_org_slug
ON properties (organization_id, slug);
```

---

# 4. Autenticación y autorización

## Requisitos

- Inicio de sesión con correo y contraseña.
- Contraseñas con Argon2id o bcrypt con parámetros seguros.
- Sesiones seguras mediante cookies `HttpOnly`, `Secure` y `SameSite`.
- Protección CSRF si corresponde.
- Rotación y revocación de sesiones.
- Recuperación de contraseña.
- Verificación de correo.
- MFA preparada.
- Rate limiting.
- Auditoría de accesos.
- Bloqueo temporal ante intentos repetidos.

## RBAC

Roles iniciales:

- `platform_super_admin`
- `organization_admin`
- `property_manager`
- `owner`
- `viewer`
- `tenant`
- `agent_service`

Permisos explícitos, por ejemplo:

```text
organization.read
organization.manage
users.read
users.manage
properties.read
properties.create
properties.update
properties.delete
documents.read
documents.upload
documents.verify
documents.delete
leases.read
leases.manage
charges.read
charges.manage
payments.read
payments.create
payments.confirm
receipts.read
receipts.issue
receipts.replace
obligations.manage
maintenance.manage
agents.execute
agents.approve
audit.read
```

No codificar permisos solamente en componentes visuales. Deben aplicarse en backend.

---

# 5. Modelo de datos inicial

Usar UUID y campos comunes:

```sql
id uuid primary key,
organization_id uuid not null,
created_at timestamptz not null,
updated_at timestamptz not null,
created_by uuid null,
updated_by uuid null,
version bigint not null default 1
```

Entidades:

- organizations
- organization_settings
- users
- memberships
- roles
- permissions
- role_permissions
- user_property_assignments
- properties
- property_owners
- property_valuations
- tenants
- leases
- lease_tenants
- lease_adjustments
- charges
- payments
- payment_allocations
- receipts
- documents
- document_links
- obligations
- mortgages
- mortgage_payments
- expenses
- work_orders
- work_order_documents
- vendors
- notifications
- approval_requests
- agent_identities
- agent_runs
- agent_actions
- audit_events
- outbox_events

## Document links

Un documento puede vincularse a distintas entidades.

Evitar una asociación polimórfica débil sin restricciones cuando sea posible.

Opciones aceptables:

- Tablas de enlace específicas.
- Tabla `document_links` con validación de tipos y reglas de servicio.
- Referencias explícitas para las entidades principales.

## Dinero

Propuesta:

```sql
amount_minor bigint not null,
currency_code varchar(3) not null
```

Para UF y unidades con decimales:

```sql
amount_value numeric(20, 8),
unit_code varchar(10)
```

Documentar claramente la convención.

---

# 6. Documentos

## Carga

Flujo:

1. Validar autorización.
2. Validar tamaño y MIME.
3. Generar nombre seguro.
4. Calcular SHA-256 durante la carga.
5. Verificar duplicado dentro de la organización.
6. Guardar objeto.
7. Crear registro en estado `uploaded`.
8. Publicar evento outbox.
9. Ejecutar extracción opcional.
10. Dejar en revisión.

## Seguridad

- Bloquear ejecución de archivos.
- No confiar en la extensión.
- Validar magic bytes.
- Sanitizar nombres.
- Límite configurable.
- Cifrado del proveedor.
- URLs firmadas de corta duración.
- Protección contra path traversal.
- Logs sin contenido sensible.
- Preparar antivirus o análisis de malware mediante interfaz extensible.

## Versionado

Los documentos críticos deben ser inmutables.

Para reemplazos:

- Crear nuevo documento.
- Relacionarlo con el anterior.
- Marcar estado.
- Conservar auditoría.

## Hash

Guardar:

- Algoritmo.
- Hash.
- Tamaño.
- Fecha de cálculo.

---

# 7. Procesamiento con IA

Crear una capa de proveedor:

```go
type DocumentAIProvider interface {
    AnalyzeDocument(ctx context.Context, input AnalyzeDocumentInput) (DocumentAnalysis, error)
}
```

Adaptadores iniciales:

- `gemini`
- `local_agent`
- `mock`

La respuesta debe ser estructurada:

```json
{
  "document_type": "bank_transfer_receipt",
  "confidence": 0.91,
  "issuer": "Banco ...",
  "document_date": "2026-07-05",
  "period": "2026-07",
  "amount": {
    "value": "500000",
    "currency": "CLP"
  },
  "references": [],
  "warnings": [],
  "raw_text": ""
}
```

Reglas:

- Validar JSON con esquema.
- No confiar ciegamente en el modelo.
- Guardar proveedor, modelo y versión.
- Guardar confianza.
- Separar datos propuestos de datos confirmados.
- No exponer contratos completos a modelos externos sin consentimiento y configuración.
- Permitir modelos locales para documentos sensibles.
- Aplicar timeouts, reintentos y circuit breaker.
- Evitar enviar secretos.
- Permitir desactivar IA por organización.

---

# 8. Integración con OpenClaw y Hermes Agents

No permitir acceso directo a PostgreSQL.

Exponer API de agentes con autenticación separada.

## Identidad técnica

Cada agente debe poseer:

- `agent_id`
- Organización.
- Nombre.
- Estado.
- Permisos.
- Clave o token rotatorio.
- Restricción de red opcional.
- Modelo predeterminado.
- Límites.
- Fecha de último uso.

## API inicial

```http
POST /api/v1/agent/documents/intake
GET  /api/v1/agent/properties/{propertyId}/summary
GET  /api/v1/agent/obligations/upcoming
POST /api/v1/agent/payments/propose-match
POST /api/v1/agent/maintenance/report
POST /api/v1/agent/approvals
GET  /api/v1/agent/runs/{runId}
```

## Reglas

- Idempotency key en operaciones POST.
- Scope por organización.
- Rate limiting.
- Auditoría completa.
- Respuestas estructuradas.
- Nunca confirmar pagos automáticamente.
- Nunca eliminar documentos.
- Nunca modificar contratos vigentes.
- Toda acción sensible crea `approval_request`.
- Registrar modelo, prompt version, tokens y costo cuando exista.

## Eventos

Usar tabla outbox inicialmente:

```text
document.uploaded
document.analysis.requested
document.analysis.completed
payment.created
payment.match.proposed
payment.confirmed
receipt.issued
obligation.due_soon
work_order.created
approval.requested
approval.resolved
```

OpenClaw y Hermes pueden consumir mediante:

- Polling autenticado.
- Webhooks firmados.
- Endpoint de eventos.
- Adaptador local.

No incorporar Pub/Sub hasta que exista una necesidad real.

---

# 9. Cobros, pagos y conciliación

## Invariantes

- Un cobro posee monto original y saldo.
- El saldo no puede ser negativo.
- Un pago no puede aplicarse por más de su monto disponible.
- Una aplicación no puede exceder el saldo del cobro.
- Confirmar una aplicación debe ejecutarse en una transacción.
- Reversiones crean movimientos compensatorios.
- No editar registros financieros confirmados.
- Usar optimistic locking o versión.

## Estados

### Charge

- draft
- pending
- partially_paid
- paid
- overdue
- disputed
- cancelled

### Payment

- received
- pending_reconciliation
- reconciled
- rejected
- reversed
- duplicate

### Allocation

- proposed
- confirmed
- reversed

## Propuesta automática

El algoritmo puede comparar:

- Organización.
- Propiedad.
- Contrato.
- Arrendatario.
- Monto.
- Fecha.
- Período.
- Referencia.
- Tolerancia configurable.

Debe devolver puntaje y razones, pero no confirmar.

---

# 10. Comprobantes PDF y QR

## Folio

Formato configurable, por ejemplo:

```text
CP-{PROPERTY_CODE}-{YEAR}-{SEQUENCE}
```

Usar secuencia transaccional y única por organización.

## PDF

Debe generarse en backend.

Contenido:

- Emisor.
- Arrendatario parcialmente oculto.
- Propiedad limitada.
- Período.
- Conceptos.
- Fecha de pago.
- Monto.
- Medio de pago.
- Folio.
- Código de verificación.
- QR.
- Estado.
- Fecha de emisión.

## Inmutabilidad

Después de emitir:

- No actualizar contenido.
- Guardar snapshot JSON usado para generar.
- Guardar PDF.
- Guardar hash SHA-256.
- Guardar versión de plantilla.
- Para corregir, emitir reemplazo.

## QR

URL:

```text
https://{public-domain}/verify/receipt/{public_token}
```

El token debe:

- Ser aleatorio.
- No ser secuencial.
- No incluir IDs internos.
- Poder rotarse solo mediante reemplazo o anulación.

La página pública devuelve información mínima y evita indexación.

Agregar:

```http
X-Robots-Tag: noindex, nofollow
```

---

# 11. Obligaciones y jobs

Usar jobs idempotentes.

Casos:

- Generar cobros mensuales.
- Marcar cobros vencidos.
- Crear recordatorios.
- Detectar contratos próximos a vencer.
- Detectar reajustes.
- Detectar obligaciones próximas.
- Procesar documentos.
- Enviar correos.
- Crear resúmenes.

Implementación inicial:

- Tabla de jobs o scheduler interno.
- Advisory locks en PostgreSQL.
- Tabla outbox.
- Reintentos.
- Dead-letter lógico.
- Historial de ejecución.

No depender de cron local sin trazabilidad.

---

# 12. Notificaciones

Crear interfaz:

```go
type NotificationSender interface {
    Send(ctx context.Context, message NotificationMessage) error
}
```

Canales futuros:

- Correo.
- Push web.
- WhatsApp.
- Telegram.
- OpenClaw.
- Hermes.

MVP:

- Correo.
- Notificaciones internas.
- Web push opcional.

Todas las notificaciones deben:

- Usar plantillas versionadas.
- Registrar entrega.
- Evitar duplicados.
- Respetar configuración.
- No incluir datos sensibles innecesarios.

---

# 13. Frontend y experiencia móvil

## Requisitos

- Responsive.
- PWA.
- Manifest.
- Service worker.
- Instalación en teléfono.
- Cámara para capturar documentos.
- Upload con progreso.
- Reintento.
- Vista previa.
- Diseño accesible.
- Navegación protegida.
- Estados vacíos.
- Manejo de errores.

## Páginas iniciales

```text
/login
/dashboard
/properties
/properties/{id}
/properties/{id}/documents
/properties/{id}/leases
/properties/{id}/charges
/properties/{id}/payments
/properties/{id}/receipts
/properties/{id}/obligations
/properties/{id}/maintenance
/documents/inbox
/approvals
/users
/settings
/verify/receipt/{token}
```

## Carga desde teléfono

Debe permitir:

- Seleccionar cámara.
- Capturar imagen.
- Recortar o rotar.
- Comprimir de forma segura.
- Subir.
- Continuar si la red se interrumpe.
- Ver estado de procesamiento.

---

# 14. API

## Convenciones

- Versionada: `/api/v1`.
- JSON.
- Errores consistentes.
- IDs UUID.
- Paginación.
- Filtros.
- Orden.
- Búsqueda.
- Idempotency keys.
- Correlation IDs.
- OpenAPI.
- Fechas ISO 8601.
- ETags o versionado para concurrencia.

## Ejemplo de error

```json
{
  "error": {
    "code": "payment_allocation_exceeds_balance",
    "message": "La aplicación excede el saldo disponible.",
    "details": {},
    "correlation_id": "..."
  }
}
```

---

# 15. Auditoría

Registrar:

- Login.
- Logout.
- Cambio de contraseña.
- Creación y desactivación de usuarios.
- Cambio de permisos.
- Creación y edición de propiedad.
- Carga, descarga y eliminación de documentos.
- Confirmación y reversión de pagos.
- Emisión, reemplazo y anulación de comprobantes.
- Cambios contractuales.
- Aprobaciones.
- Acciones de agentes.

Los eventos de auditoría no deben poder modificarse mediante la API normal.

---

# 16. Observabilidad

Implementar:

- Logs estructurados JSON.
- Correlation ID.
- Métricas.
- Health checks.
- Readiness.
- Trazas cuando sea razonable.
- Errores centralizables.
- Métricas de jobs.
- Métricas de agentes.
- Métricas de carga documental.

No registrar:

- Contraseñas.
- Tokens.
- Contratos completos.
- RUT completo.
- Datos bancarios completos.
- Respuestas privadas de modelos sin sanitización.

---

# 17. Backups y recuperación

Definir:

- Backup de PostgreSQL.
- Versionado o backup de objetos.
- Política de retención.
- Prueba de restauración.
- Exportación por organización.
- Procedimiento de recuperación.
- Integridad entre documento y registro.
- Job de verificación de objetos huérfanos.

---

# 18. Pruebas

## Unitarias

- Estados.
- Validaciones.
- Reglas financieras.
- Permisos.
- Folios.
- Reemplazos.
- Hash.
- Propuestas de conciliación.

## Integración

- PostgreSQL real mediante contenedor.
- Migraciones.
- Object storage.
- Autenticación.
- Multi-tenancy.
- Emisión de comprobante.
- Jobs.
- Outbox.

## E2E

- Crear organización.
- Crear propiedad.
- Subir documento.
- Crear contrato.
- Generar cobro.
- Registrar pago.
- Confirmar conciliación.
- Emitir comprobante.
- Verificar QR.
- Registrar reparación.

## Seguridad

- Acceso cruzado entre organizaciones.
- Escalada de privilegios.
- Upload malicioso.
- Path traversal.
- IDOR.
- CSRF.
- Rate limiting.
- Token expirado.
- URL firmada vencida.

---

# 19. CI/CD

Pipeline mínimo:

1. Lint Go.
2. Lint TypeScript.
3. Tests unitarios.
4. Tests de integración.
5. Build backend.
6. Build frontend.
7. Validación de migraciones.
8. Escaneo de dependencias.
9. Construcción de imágenes.
10. Despliegue por ambiente.

Ambientes:

- local
- test
- staging
- production

---

# 20. Variables de entorno

Ejemplo:

```dotenv
APP_ENV=local
APP_BASE_URL=http://localhost:3000
API_BASE_URL=http://localhost:8080
DATABASE_URL=postgres://...
SESSION_SECRET=...
STORAGE_DRIVER=minio
STORAGE_ENDPOINT=http://localhost:9000
STORAGE_BUCKET=control-propiedades
STORAGE_ACCESS_KEY=...
STORAGE_SECRET_KEY=...
MAIL_DRIVER=smtp
MAIL_HOST=localhost
MAIL_PORT=1025
AI_PROVIDER=mock
GEMINI_API_KEY=
OPENCLAW_WEBHOOK_SECRET=
HERMES_AGENT_TOKEN=
```

Crear `.env.example` sin secretos reales.

---

# 21. Documentación requerida

Mantener:

```text
README.md
docs/ARCHITECTURE.md
docs/DOMAIN_MODEL.md
docs/SECURITY.md
docs/LOCAL_DEVELOPMENT.md
docs/DEPLOYMENT.md
docs/DOCUMENT_STORAGE.md
docs/RECEIPT_SPEC.md
docs/AGENT_API.md
docs/AGENT_PERMISSIONS.md
docs/BACKUP_RESTORE.md
docs/ADR/
openapi.yaml
```

Crear ADR para:

- Monolito modular.
- Multi-tenancy.
- Almacenamiento documental.
- Autenticación.
- IA externa versus local.
- Outbox.
- PDF y QR.

---

# 22. Fases técnicas

## Fase 0 — Fundación

- Monorepo.
- Docker Compose.
- Configuración.
- PostgreSQL.
- Migraciones.
- Backend.
- Frontend.
- Health checks.
- CI.

## Fase 1 — Identidad y tenancy

- Organizaciones.
- Usuarios.
- Sesiones.
- Roles.
- Permisos.
- Auditoría.

## Fase 2 — Propiedades y documentos

- Propiedades.
- Propietarios.
- Storage.
- Upload.
- Hash.
- Metadatos.
- Bandeja documental.

## Fase 3 — Contratos y pagos

- Arrendatarios.
- Contratos.
- Cobros.
- Pagos.
- Conciliación.
- Estados.

## Fase 4 — Comprobantes

- Folios.
- PDF.
- QR.
- Validación pública.
- Reemplazos.
- Correo.

## Fase 5 — Obligaciones y mantenimiento

- Obligaciones.
- Jobs.
- Hipotecas.
- Contribuciones.
- Gastos.
- Órdenes de trabajo.

## Fase 6 — IA y agentes

- Provider abstraction.
- Gemini.
- Proveedor local.
- API de agentes.
- Approval requests.
- OpenClaw.
- Hermes.
- MiniMax 2.7.

## Fase 7 — SaaS

- Registro de organizaciones.
- Planes.
- Límites.
- Facturación.
- Portal de arrendatario.
- Corredor virtual.
- Soporte multiempresa.

---

# 23. Primera iteración técnica

Ejecuta esta primera iteración:

1. Inspecciona el repositorio.
2. Propón la estructura final respetando lo existente.
3. Crea `docker-compose.yml` con PostgreSQL y MinIO si no existen.
4. Crea configuración tipada.
5. Crea migraciones para:
   - organizations
   - users
   - memberships
   - roles
   - permissions
   - properties
   - property_owners
   - audit_events
6. Implementa autenticación básica segura.
7. Implementa RBAC.
8. Implementa CRUD de propiedades.
9. Implementa filtro obligatorio por organización.
10. Agrega pruebas de aislamiento entre organizaciones.
11. Documenta ejecución local.
12. No implementes todavía documentos, contratos, pagos ni agentes.

Criterios de aceptación:

- El proyecto inicia con un solo comando documentado.
- PostgreSQL persiste datos entre reinicios.
- Se puede crear una organización.
- Se puede crear un administrador.
- El administrador inicia sesión.
- Puede crear una propiedad.
- No puede acceder a datos de otra organización.
- Todas las acciones quedan auditadas.
- Las pruebas pasan.
- La API aparece documentada.
- La interfaz funciona en escritorio y teléfono.

---

# 24. Formato obligatorio de respuesta de Codex

## Inspección

- Stack detectado.
- Estructura.
- Riesgos.
- Decisiones.

## Plan de iteración

- Máximo 5 tareas.
- Archivos.
- Migraciones.
- Pruebas.

## Cambios realizados

- Lista concreta.
- Justificación.
- Compatibilidad.

## Comandos

```bash
# instalación
# migraciones
# pruebas
# ejecución
```

## Validación

- Resultado de tests.
- Resultado de build.
- Limitaciones.
- Pendientes.

## Siguiente iteración

Una única propuesta concreta, sin comenzar a implementarla.
