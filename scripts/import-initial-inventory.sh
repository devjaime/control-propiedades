#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SOURCE_DIR="${INVENTORY_SOURCE_DIR:-}"
PROPERTY_ID="${PROPERTY_ID:-}"
USER_ID="${USER_ID:-}"
INVENTORY_DATE="${INVENTORY_DATE:-$(date +%F)}"
INVENTORY_PERIOD="${INVENTORY_PERIOD:-${INVENTORY_DATE:0:7}}"
INVENTORY_ISSUER="${INVENTORY_ISSUER:-Propietario}"
API_URL="${API_URL:-http://localhost:8080}"
ORIGIN="${ORIGIN:-http://localhost:3000}"
REPORT_DIR="${ROOT_DIR}/storage/import-reports"
REPORT_FILE="${REPORT_DIR}/inventario-inicial-${INVENTORY_DATE}.json"
WORK_DIR="$(mktemp -d)"
SESSION_ID=""

cleanup() {
  if [[ -n "${SESSION_ID}" && -n "${DATABASE_URL:-}" ]]; then
    psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -v sid="${SESSION_ID}" \
      -c "DELETE FROM sessions WHERE id = :'sid'::uuid" >/dev/null 2>&1 || true
  fi
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

if [[ -f "${ROOT_DIR}/.env" ]]; then
  set -a
  source "${ROOT_DIR}/.env"
  set +a
fi

: "${DATABASE_URL:?DATABASE_URL no está configurada}"
: "${INVENTORY_SOURCE_DIR:?INVENTORY_SOURCE_DIR no está configurado}"
: "${PROPERTY_ID:?PROPERTY_ID no está configurado}"
: "${USER_ID:?USER_ID no está configurado}"
command -v psql >/dev/null
command -v curl >/dev/null
command -v jq >/dev/null
command -v shasum >/dev/null
[[ -d "${SOURCE_DIR}" ]]

mkdir -p "${REPORT_DIR}"

TOKEN="$(openssl rand -hex 32)"
TOKEN_HASH="$(printf '%s' "${TOKEN}" | shasum -a 256 | awk '{print $1}')"
SESSION_ID="$(psql "${DATABASE_URL}" -Atq -v ON_ERROR_STOP=1 \
  -v uid="${USER_ID}" -v token_hash="${TOKEN_HASH}" \
  -c "INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (:'uid'::uuid, :'token_hash', now() + interval '2 hours') RETURNING id")"

find "${SOURCE_DIR}" -maxdepth 1 -type f \( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' -o -iname '*.webp' \) -print0 \
  | while IFS= read -r -d '' file; do
      hash="$(shasum -a 256 "${file}" | awk '{print $1}')"
      printf '%s\t%s\0' "${hash}" "${file}"
    done > "${WORK_DIR}/all.nul"

python3 - "${WORK_DIR}/all.nul" "${WORK_DIR}/unique.nul" "${WORK_DIR}/duplicates.json" <<'PY'
import json, sys
source, unique, duplicates = sys.argv[1:]
parts = open(source, 'rb').read().split(b'\0')
seen = {}
dupes = []
out = bytearray()
for part in parts:
    if not part:
        continue
    digest, path = part.decode().split('\t', 1)
    if digest in seen:
        dupes.append({'sha256': digest, 'kept': seen[digest], 'duplicate': path})
        continue
    seen[digest] = path
    out.extend(f'{digest}\t{path}\0'.encode())
open(unique, 'wb').write(out)
open(duplicates, 'w', encoding='utf-8').write(json.dumps(dupes, ensure_ascii=False, indent=2))
PY

TOTAL_FOUND="$(python3 -c "print(len([x for x in open('${WORK_DIR}/all.nul','rb').read().split(b'\\0') if x]))")"
TOTAL_UNIQUE="$(python3 -c "print(len([x for x in open('${WORK_DIR}/unique.nul','rb').read().split(b'\\0') if x]))")"

printf '[]' > "${WORK_DIR}/uploaded.json"
printf '[]' > "${WORK_DIR}/failed.json"
INDEX=0

while IFS= read -r -d '' entry; do
  hash="${entry%%$'\t'*}"
  file="${entry#*$'\t'}"
  INDEX=$((INDEX + 1))
  padded="$(printf '%03d' "${INDEX}")"
  name="Inventario inicial ${INVENTORY_DATE} - fotografía ${padded}"

  response="${WORK_DIR}/response-${padded}.json"
  status="$(curl -sS -o "${response}" -w '%{http_code}' \
    -H "Origin: ${ORIGIN}" \
    -H "Cookie: cp_session=${TOKEN}" \
    -F "file=@${file}" \
    "${API_URL}/api/v1/properties/${PROPERTY_ID}/documents")"

  if [[ "${status}" != "200" && "${status}" != "201" ]]; then
    jq --arg file "${file}" --arg hash "${hash}" --arg status "${status}" \
      '. + [{file:$file,sha256:$hash,httpStatus:$status,stage:"upload"}]' \
      "${WORK_DIR}/failed.json" > "${WORK_DIR}/failed.next"
    mv "${WORK_DIR}/failed.next" "${WORK_DIR}/failed.json"
    continue
  fi

  document_id="$(jq -r '.document.id // .data.id // .id // empty' "${response}")"
  if [[ -z "${document_id}" ]]; then
    jq --arg file "${file}" --arg hash "${hash}" \
      '. + [{file:$file,sha256:$hash,stage:"parse-response"}]' \
      "${WORK_DIR}/failed.json" > "${WORK_DIR}/failed.next"
    mv "${WORK_DIR}/failed.next" "${WORK_DIR}/failed.json"
    continue
  fi

  note="${INVENTORY_NOTE:-Inventario fotográfico inicial de entrega de la propiedad. Este registro documenta el estado anterior a mejoras posteriores.}"
  payload="$(jq -nc \
    --arg name "${name}" \
    --arg type "inventory" \
    --arg date "${INVENTORY_DATE}" \
    --arg period "${INVENTORY_PERIOD}" \
    --arg issuer "${INVENTORY_ISSUER}" \
    --arg confidentiality "internal" \
    --arg status "verified" \
    --arg notes "${note}" \
    --arg hash "${hash}" \
    '{name:$name,type:$type,date:$date,period:$period,issuer:$issuer,confidentiality:$confidentiality,status:$status,notes:$notes,tags:["inventario-inicial","entrega-arrendatario",$date,"registro-fotografico"],sha256:$hash}')"

  patch_response="${WORK_DIR}/patch-${padded}.json"
  patch_status="$(curl -sS -o "${patch_response}" -w '%{http_code}' \
    -X PATCH \
    -H "Origin: ${ORIGIN}" \
    -H "Cookie: cp_session=${TOKEN}" \
    -H 'Content-Type: application/json' \
    --data "${payload}" \
    "${API_URL}/api/v1/documents/${document_id}")"

  if [[ "${patch_status}" != "200" ]]; then
    jq --arg file "${file}" --arg hash "${hash}" --arg id "${document_id}" --arg status "${patch_status}" \
      '. + [{file:$file,sha256:$hash,documentId:$id,httpStatus:$status,stage:"metadata"}]' \
      "${WORK_DIR}/failed.json" > "${WORK_DIR}/failed.next"
    mv "${WORK_DIR}/failed.next" "${WORK_DIR}/failed.json"
    continue
  fi

  jq --arg file "${file}" --arg hash "${hash}" --arg id "${document_id}" --arg name "${name}" \
    '. + [{file:$file,sha256:$hash,documentId:$id,name:$name}]' \
    "${WORK_DIR}/uploaded.json" > "${WORK_DIR}/uploaded.next"
  mv "${WORK_DIR}/uploaded.next" "${WORK_DIR}/uploaded.json"
done < "${WORK_DIR}/unique.nul"

UPLOADED="$(jq 'length' "${WORK_DIR}/uploaded.json")"
FAILED="$(jq 'length' "${WORK_DIR}/failed.json")"
DUPLICATES="$(jq 'length' "${WORK_DIR}/duplicates.json")"

jq -n \
  --arg importedAt "$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
  --arg propertyId "${PROPERTY_ID}" \
  --arg sourceDirectory "${SOURCE_DIR}" \
  --argjson totalFound "${TOTAL_FOUND}" \
  --argjson totalUnique "${TOTAL_UNIQUE}" \
  --argjson duplicates "${DUPLICATES}" \
  --argjson uploaded "${UPLOADED}" \
  --argjson failed "${FAILED}" \
  --slurpfile uploadedFiles "${WORK_DIR}/uploaded.json" \
  --slurpfile duplicateFiles "${WORK_DIR}/duplicates.json" \
  --slurpfile failures "${WORK_DIR}/failed.json" \
  '{importedAt:$importedAt,propertyId:$propertyId,inventoryDate:"2025-11-03",sourceDirectory:$sourceDirectory,totalFound:$totalFound,totalUnique:$totalUnique,exactDuplicates:$duplicates,uploaded:$uploaded,failed:$failed,uploadedFiles:$uploadedFiles[0],duplicateFiles:$duplicateFiles[0],failures:$failures[0]}' \
  > "${REPORT_FILE}"

if [[ "${FAILED}" -ne 0 || "${UPLOADED}" -ne "${TOTAL_UNIQUE}" ]]; then
  exit 2
fi
