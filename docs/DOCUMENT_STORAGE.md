# Almacenamiento y búsqueda documental

Los archivos se almacenan de forma privada en MinIO durante desarrollo. PostgreSQL conserva los metadatos necesarios para autorizar, clasificar y encontrar cada documento.

## Organización

La clave física sigue esta estructura:

```text
organizations/{organization_id}/properties/{property_id}/documents/{random_id}.{extension}
```

El nombre original nunca se usa como ruta física. La tabla `documents` conserva propiedad, tipo, nombre visible, nombre original, MIME detectado, tamaño, SHA-256, etiquetas, estado y usuario que realizó la carga.

También registra fecha documental, período, emisor, monto, moneda, confidencialidad, usuario verificador y versión para control de concurrencia.

## Seguridad

- El bucket es privado.
- Las consultas filtran obligatoriamente por `organization_id` de la sesión.
- La API detecta el MIME usando el contenido, no la extensión.
- Solo se aceptan PDF, JPG, PNG y WebP.
- El límite predeterminado es 50 MB y puede configurarse con `MAX_UPLOAD_BYTES`.
- El proxy web permite 52 MB para incluir el overhead del formulario multipart.
- Los duplicados dentro de una organización se detectan por SHA-256.
- Las descargas pasan por autenticación y se transmiten desde la API.
- Un documento verificado es inmutable. Solo puede archivarse; una corrección futura deberá crear un reemplazo.

## Búsqueda

`GET /api/v1/documents` permite combinar:

- `q`: nombre original, nombre visible o etiqueta.
- `property_id`: propiedad específica.
- `type`: tipo documental.
- `status`: cargado, pendiente de revisión, verificado o rechazado.

Los índices cubren organización, propiedad, tipo, hash, etiquetas y búsqueda parcial de nombres.

## Revisión humana

Estados habilitados:

```text
uploaded -> pending_review -> verified
                         \-> rejected -> pending_review
verified -> archived
```

La verificación y el rechazo requieren un administrador de organización. El rechazo exige un motivo. Cada revisión:

- Incrementa `version` para detectar ediciones concurrentes.
- Registra el estado anterior y el nuevo en `audit_events`.
- Conserva quién verificó y cuándo.
- No modifica automáticamente contratos, cobros, pagos u otras entidades.
