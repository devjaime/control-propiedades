# MCP de Control Propiedades para Hermes Agent

Servidor MCP local por `stdio`. Consume únicamente los endpoints de agente de la API y utiliza un token revocable con alcance limitado.

Incluye consultas de propiedades y accionables, actualizaciones controladas de incidentes y la carga de comprobantes de pago desde archivos locales hacia el almacenamiento privado en la nube. Las escrituras requieren `confirm=true`; un comprobante recién cargado queda pendiente de conciliación administrativa.

## Preparación

Desde el portal web abre **Agente MCP**. Allí puedes crear una credencial con vigencia de seis meses, copiar las variables de entorno, revisar su último uso y revocarla inmediatamente. La credencial completa sólo se muestra una vez.

El portal también publica un instalador autocontenido para equipos que no tengan el repositorio clonado:

```bash
curl -fsSL https://control-propiedades-web.vercel.app/downloads/install-control-propiedades-mcp.sh | sh
```

El instalador guarda el servidor en `~/.local/share/control-propiedades`, crea el comando `~/.local/bin/control-propiedades-mcp` y espera la credencial en `~/.config/control-propiedades/agent.env`.

```bash
npm install
npm run build:mcp
chmod +x apps/mcp/bin/control-propiedades-mcp
```

El archivo ignorado `.env.hermes.local` debe contener `CONTROL_PROPIEDADES_API_URL` y `CONTROL_PROPIEDADES_API_TOKEN`.

Para la versión productiva:

```dotenv
CONTROL_PROPIEDADES_API_URL=https://control-propiedades-api.vercel.app
CONTROL_PROPIEDADES_API_TOKEN=cp_CREDENCIAL_GENERADA_EN_EL_PORTAL
```

## Hermes

```bash
hermes mcp add control-propiedades --command /ruta/absoluta/apps/mcp/bin/control-propiedades-mcp
hermes mcp test control-propiedades
hermes mcp configure control-propiedades
```

Para cargar un comprobante, Hermes debe recibir una ruta local absoluta a un archivo PDF, JPG, PNG o WEBP, además del `property_id`, el `charge_id`, monto y fecha. Debe mostrar esos datos al usuario y obtener confirmación antes de invocar `subir_comprobante_pago`.

La carga queda pendiente de conciliación. Con una segunda confirmación explícita, `confirmar_pago` concilia el pago y, cuando completa el saldo mensual, la API emite automáticamente el comprobante de arriendo verificable.

`subir_evidencia_incidente` preserva fotografías o PDF locales en el almacenamiento privado y los vincula a un incidente existente mediante su `incident_id`. La herramienta no publica automáticamente el archivo en el portal del arrendatario.

También puede agregarse manualmente en `~/.hermes/config.yaml`:

```yaml
mcp_servers:
  control-propiedades:
    command: "/ruta/absoluta/apps/mcp/bin/control-propiedades-mcp"
    enabled: true
    supports_parallel_tool_calls: false
```
