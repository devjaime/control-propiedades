# Despliegue gratuito en Vercel, Neon y Supabase Storage

Esta guía despliega el frontend Next.js y la API Go como dos proyectos Vercel. Neon conserva PostgreSQL y Supabase Storage almacena los archivos privados.

## 1. Crear la base PostgreSQL en Neon

1. Crea una cuenta en <https://console.neon.tech> y un proyecto llamado `control-propiedades`.
2. Usa PostgreSQL 17 y elige la región más cercana disponible.
3. En **Connect**, copia dos conexiones:
   - **Direct connection**: se usa una sola vez para migraciones.
   - **Pooled connection**: contiene `-pooler` en el host y se usa como `DATABASE_URL` en Vercel.
4. No publiques estas conexiones ni las guardes en Git.

Para copiar la base local a un proyecto Neon vacío:

```bash
export SOURCE_DATABASE_URL='postgresql://control_propiedades:control_propiedades@localhost:5432/control_propiedades?sslmode=disable'
export TARGET_DATABASE_URL='CONEXION_DIRECTA_DE_NEON'
./scripts/cloud-migrate-database.sh
```

El script no elimina tablas: debe ejecutarse contra una base Neon nueva y vacía. Al terminar compara los conteos principales.

## 2. Crear almacenamiento privado en Supabase Storage

1. Crea un proyecto Supabase en el plan **Free**.
2. En **Storage**, crea un bucket privado llamado `control-propiedades`.
3. En **Storage → Configuration → S3 Connection**, habilita S3 y genera credenciales.
4. Copia el endpoint completo, incluida la ruta `/storage/v1/s3`, y la región del proyecto.
5. Guarda `Access Key ID` y `Secret Access Key` únicamente en variables del servidor. Estas claves omiten RLS y nunca deben publicarse en el frontend.

Para copiar los archivos existentes desde MinIO local:

```bash
cp .env.storage.example .env.storage.local
# Edita .env.storage.local localmente y completa las credenciales S3 rotadas.
./scripts/cloud-migrate-storage.sh
```

El bucket debe permanecer privado. La aplicación entrega permisos temporales de diez minutos para subir y cinco minutos para descargar.

## 3. Crear el proyecto API en Vercel

1. En Vercel selecciona **Add New → Project** e importa el repositorio `control-propiedades`.
2. Nombra el proyecto `control-propiedades-api`.
3. Configura **Root Directory** como `apps/api`.
4. No definas Build Command ni Output Directory; Vercel detectará `api/index.go`.
5. Copia las variables de la sección API de [.env.cloud.example](../.env.cloud.example) a **Settings → Environment Variables**.
6. Usa la conexión **pooled** de Neon para `DATABASE_URL` y limita `DATABASE_MAX_CONNS=2`.
7. Define inicialmente `WEB_ORIGIN` con el dominio que tendrá el frontend, por ejemplo `https://control-propiedades.vercel.app`.
8. Despliega y verifica `https://DOMINIO-API/health/live` y `https://DOMINIO-API/health/ready`.

No agregues ningún secreto con prefijo `NEXT_PUBLIC_`.

## 4. Crear el proyecto web en Vercel

1. Importa el mismo repositorio como un segundo proyecto.
2. Nómbralo `control-propiedades-web`.
3. Configura **Root Directory** como `apps/web` y Framework **Next.js**.
4. Agrega solamente `API_INTERNAL_URL=https://DOMINIO-API`.
5. Despliega.
6. Si el dominio final difiere del valor inicial, actualiza `WEB_ORIGIN` en el proyecto API y vuelve a desplegar la API.

La web reenvía `/api/*` internamente hacia la API. Esto mantiene la cookie de sesión HTTP-only en el mismo dominio visible para el navegador.

## 5. Prueba de aceptación antes de producción

Realiza estas pruebas en este orden:

1. Abrir `/health/ready` en la API y comprobar `{"status":"ok"}`.
2. Iniciar sesión desde el frontend.
3. Abrir la propiedad existente y comprobar documentos, incidente y arriendo.
4. Subir una fotografía de prueba al incidente.
5. Abrir y descargar esa fotografía desde su ficha.
6. Registrar un pago pequeño de prueba o usar datos de prueba separados.
7. Confirmar el pago y comprobar que se genera el PDF con folio y QR.
8. Abrir el QR en una sesión privada y comprobar la página pública de verificación.
9. Eliminar únicamente los datos creados para la prueba siguiendo el flujo normal de la aplicación.

## 6. Producción y recuperación

- Mantén el PostgreSQL y MinIO locales sin borrarlos hasta completar las pruebas.
- Conserva un `pg_dump` cifrado fuera del computador antes de cada migración relevante.
- Vercel Preview debe usar una base Neon y un bucket Supabase distintos; nunca apuntes un preview a datos reales.
- Supabase Free pausa proyectos con poca actividad y limita Storage a 1 GB; monitorea esos límites si la solución pasa a uso operativo continuo.
