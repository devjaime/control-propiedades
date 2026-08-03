#!/usr/bin/env bash
set -euo pipefail

: "${SOURCE_DATABASE_URL:?Define SOURCE_DATABASE_URL con la base local o de origen}"
: "${TARGET_DATABASE_URL:?Define TARGET_DATABASE_URL con la conexión directa de Neon}"

archive_dir="$(mktemp -d)"
archive_file="${archive_dir}/control-propiedades.dump"
trap 'rm -rf "${archive_dir}"' EXIT

pg_dump --format=custom --no-owner --no-acl --file="${archive_file}" "${SOURCE_DATABASE_URL}"
pg_restore --exit-on-error --single-transaction --no-owner --no-acl --dbname="${TARGET_DATABASE_URL}" "${archive_file}"

if command -v goose >/dev/null 2>&1; then
  goose -dir migrations postgres "${TARGET_DATABASE_URL}" up
else
  go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir migrations postgres "${TARGET_DATABASE_URL}" up
fi

psql "${TARGET_DATABASE_URL}" -v ON_ERROR_STOP=1 -Atc \
  "select 'properties='||count(*) from properties; select 'documents='||count(*) from documents; select 'receipts='||count(*) from rent_receipts; select 'incidents='||count(*) from incidents; select 'direct_uploads_ready='||(to_regclass('public.document_uploads') is not null);"
