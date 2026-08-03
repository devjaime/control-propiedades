#!/usr/bin/env node

import { createHash } from "node:crypto";
import { readFile, stat } from "node:fs/promises";
import { basename, extname, resolve } from "node:path";
import { createInterface } from "node:readline";

const apiBaseURL = requiredEnv("CONTROL_PROPIEDADES_API_URL").replace(/\/$/, "");
const apiToken = requiredEnv("CONTROL_PROPIEDADES_API_TOKEN");

async function api(path, init = {}) {
  const response = await fetch(`${apiBaseURL}/api/v1/agent${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${apiToken}`,
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body?.error?.message ?? `Control Propiedades respondió HTTP ${response.status}`);
  }
  return body;
}

function result(value) {
  return {
    content: [{ type: "text", text: JSON.stringify(value, null, 2) }],
    structuredContent: value,
  };
}

const mimeTypes = {
  ".pdf": "application/pdf",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".png": "image/png",
  ".webp": "image/webp",
};

async function uploadLocalFile(propertyID, filePath, { displayName, documentType, tags, incidentID = "" }) {
  const absolutePath = resolve(filePath);
  const fileInfo = await stat(absolutePath);
  if (!fileInfo.isFile()) throw new Error("La ruta indicada no corresponde a un archivo.");
  const mimeType = mimeTypes[extname(absolutePath).toLowerCase()];
  if (!mimeType) throw new Error("Formato no permitido. Usa PDF, JPG, PNG o WEBP.");
  const contents = await readFile(absolutePath);
  const sha256 = createHash("sha256").update(contents).digest("hex");
  const prepared = await api(`/properties/${propertyID}/document-uploads`, {
    method: "POST",
    body: JSON.stringify({
      original_name: basename(absolutePath),
      display_name: displayName,
      document_type: documentType,
      mime_type: mimeType,
      size_bytes: fileInfo.size,
      sha256,
      tags,
      incident_id: incidentID,
    }),
  });
  const upload = prepared.upload;
  let completion = {};
  if (upload.mode === "single") {
    const response = await fetch(upload.upload_url, { method: "PUT", headers: upload.headers ?? {}, body: contents });
    if (!response.ok) throw new Error(`El almacenamiento rechazó el archivo (HTTP ${response.status}).`);
  } else {
    const chunkSize = 5 * 1024 * 1024;
    const parts = [];
    for (const part of upload.parts ?? []) {
      const start = (part.part_number - 1) * chunkSize;
      const response = await fetch(part.upload_url, { method: "PUT", body: contents.subarray(start, Math.min(start + chunkSize, contents.length)) });
      if (!response.ok) throw new Error(`Falló la parte ${part.part_number} (HTTP ${response.status}).`);
      const etag = response.headers.get("etag");
      if (!etag) throw new Error(`El almacenamiento no devolvió ETag para la parte ${part.part_number}.`);
      parts.push({ part_number: part.part_number, etag });
    }
    completion = { parts };
  }
  const completed = await api(`/document-uploads/${upload.id}/complete`, { method: "POST", body: JSON.stringify(completion) });
  return { document: completed.document, absolute_path: absolutePath, sha256 };
}

const uuidPattern = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$";
const uuidRegex = new RegExp(uuidPattern);
const objectSchema = (properties = {}, required = []) => ({ type: "object", properties, required, additionalProperties: false });
const tools = [
  { name: "listar_propiedades", description: "Lista las propiedades accesibles para la organización.", inputSchema: objectSchema() },
  { name: "resumen_propiedad", description: "Obtiene la ficha, cobros recientes y tickets de una propiedad. No incluye archivos ni datos bancarios.", inputSchema: objectSchema({ property_id: { type: "string", pattern: uuidPattern } }, ["property_id"]) },
  { name: "listar_accionables", description: "Lista tareas pendientes de incidentes, incluyendo vencimiento, prioridad y si son visibles al arrendatario.", inputSchema: objectSchema({ property_id: { type: "string", pattern: uuidPattern } }, ["property_id"]) },
  { name: "agregar_actualizacion_incidente", description: "Agrega una actualización al expediente. Requiere confirm=true.", inputSchema: objectSchema({ incident_id: { type: "string", pattern: uuidPattern }, update_type: { type: "string", enum: ["note", "tenant_contact", "builder_claim", "insurance_claim", "inspection", "status_change"] }, content: { type: "string", minLength: 2, maxLength: 10000 }, occurred_at: { type: "string", format: "date-time" }, visible_to_tenant: { type: "boolean", default: false }, confirm: { type: "boolean", const: true } }, ["incident_id", "update_type", "content", "confirm"]) },
  { name: "actualizar_tarea_incidente", description: "Cambia el estado o visibilidad de una tarea. Requiere la versión obtenida al listar accionables y confirm=true.", inputSchema: objectSchema({ task_id: { type: "string", pattern: uuidPattern }, status: { type: "string", enum: ["pending", "completed", "cancelled"] }, version: { type: "integer", minimum: 1 }, visible_to_tenant: { type: "boolean" }, confirm: { type: "boolean", const: true } }, ["task_id", "status", "version", "confirm"]) },
  { name: "subir_comprobante_pago", description: "Sube un comprobante bancario local al almacenamiento privado y registra el abono contra un cobro. Requiere confirm=true. No confirma ni concilia el pago; lo deja pendiente de revisión.", inputSchema: objectSchema({ property_id: { type: "string", pattern: uuidPattern }, charge_id: { type: "string", pattern: uuidPattern }, file_path: { type: "string", minLength: 1 }, amount_minor: { type: "integer", minimum: 1 }, payment_date: { type: "string", pattern: "^\\d{4}-\\d{2}-\\d{2}$" }, payment_method: { type: "string", enum: ["bank_transfer", "deposit", "cash", "other"], default: "bank_transfer" }, bank_reference: { type: "string", maxLength: 200 }, observations: { type: "string", maxLength: 2000 }, confirm: { type: "boolean", const: true } }, ["property_id", "charge_id", "file_path", "amount_minor", "payment_date", "confirm"]) },
  { name: "confirmar_pago", description: "Concilia un pago pendiente usando su versión actual. Si completa una renta, emite automáticamente el comprobante verificable. Requiere confirm=true.", inputSchema: objectSchema({ payment_id: { type: "string", pattern: uuidPattern }, version: { type: "integer", minimum: 1 }, confirm: { type: "boolean", const: true } }, ["payment_id", "version", "confirm"]) },
  { name: "subir_evidencia_incidente", description: "Sube una fotografía o PDF local al almacenamiento privado y lo vincula a un incidente existente. Conserva hash y archivo original; requiere confirm=true.", inputSchema: objectSchema({ property_id: { type: "string", pattern: uuidPattern }, incident_id: { type: "string", pattern: uuidPattern }, file_path: { type: "string", minLength: 1 }, display_name: { type: "string", minLength: 1, maxLength: 240 }, tags: { type: "array", items: { type: "string" }, maxItems: 20 }, confirm: { type: "boolean", const: true } }, ["property_id", "incident_id", "file_path", "display_name", "confirm"]) },
];

function requireUUID(value, name) { if (typeof value !== "string" || !uuidRegex.test(value)) throw new Error(`${name} no es un UUID válido.`); }
function requireConfirmation(args) { if (args.confirm !== true) throw new Error("La escritura requiere confirm=true después de la autorización explícita del usuario."); }

async function callTool(name, args = {}) {
  if (name === "listar_propiedades") return result(await api("/properties"));
  if (name === "resumen_propiedad" || name === "listar_accionables") {
    requireUUID(args.property_id, "property_id");
    const suffix = name === "resumen_propiedad" ? "overview" : "open-actions";
    return result(await api(`/properties/${args.property_id}/${suffix}`));
  }
  if (name === "agregar_actualizacion_incidente") {
    requireUUID(args.incident_id, "incident_id"); requireConfirmation(args);
    if (!(["note", "tenant_contact", "builder_claim", "insurance_claim", "inspection", "status_change"].includes(args.update_type)) || typeof args.content !== "string" || args.content.length < 2 || args.content.length > 10000) throw new Error("Revisa el tipo y contenido de la actualización.");
    const { incident_id, ...body } = args;
    return result(await api(`/incidents/${incident_id}/updates`, { method: "POST", body: JSON.stringify({ visible_to_tenant: false, ...body }) }));
  }
  if (name === "actualizar_tarea_incidente") {
    requireUUID(args.task_id, "task_id"); requireConfirmation(args);
    if (!(["pending", "completed", "cancelled"].includes(args.status)) || !Number.isInteger(args.version) || args.version < 1) throw new Error("Revisa estado y versión de la tarea.");
    const { task_id, ...body } = args;
    return result(await api(`/incident-tasks/${task_id}`, { method: "PATCH", body: JSON.stringify(body) }));
  }
  if (name === "subir_comprobante_pago") {
    requireUUID(args.property_id, "property_id"); requireUUID(args.charge_id, "charge_id"); requireConfirmation(args);
    if (typeof args.file_path !== "string" || !Number.isInteger(args.amount_minor) || args.amount_minor < 1 || !/^\d{4}-\d{2}-\d{2}$/.test(args.payment_date ?? "")) throw new Error("Revisa archivo, fecha y monto del comprobante.");
    const paymentMethod = args.payment_method ?? "bank_transfer";
    if (!(["bank_transfer", "deposit", "cash", "other"].includes(paymentMethod))) throw new Error("El medio de pago no es válido.");
    const uploaded = await uploadLocalFile(args.property_id, args.file_path, { displayName: `Comprobante de pago ${args.payment_date}`, documentType: "bank_receipt", tags: ["pago", "carga-hermes"] });
    const registered = await api(`/properties/${args.property_id}/payments/from-document`, { method: "POST", body: JSON.stringify({ charge_id: args.charge_id, amount_minor: args.amount_minor, payment_date: args.payment_date, payment_method: paymentMethod, bank_reference: args.bank_reference ?? "", observations: args.observations ?? "", document_id: uploaded.document.id, confirm: true }) });
    return result({ status: "uploaded_pending_reconciliation", document: uploaded.document, payment: registered.payment, sha256: uploaded.sha256, note: "El archivo quedó preservado y el pago permanece pendiente de conciliación administrativa." });
  }
  if (name === "confirmar_pago") {
    requireUUID(args.payment_id, "payment_id"); requireConfirmation(args);
    if (!Number.isInteger(args.version) || args.version < 1) throw new Error("La versión del pago no es válida.");
    return result(await api(`/payments/${args.payment_id}/confirm`, { method: "POST", body: JSON.stringify({ version: args.version }) }));
  }
  if (name === "subir_evidencia_incidente") {
    requireUUID(args.property_id, "property_id"); requireUUID(args.incident_id, "incident_id"); requireConfirmation(args);
    if (typeof args.file_path !== "string" || typeof args.display_name !== "string" || args.display_name.trim().length < 1 || args.display_name.length > 240) throw new Error("Revisa la ruta y el nombre de la evidencia.");
    const tags = Array.isArray(args.tags) ? args.tags.filter(tag => typeof tag === "string").slice(0, 20) : [];
    const uploaded = await uploadLocalFile(args.property_id, args.file_path, { displayName: args.display_name.trim(), documentType: extname(args.file_path).toLowerCase() === ".pdf" ? "inspection" : "photo", tags: ["incidente", "evidencia", "carga-hermes", ...tags], incidentID: args.incident_id });
    return result({ status: "uploaded_and_linked", document: uploaded.document, incident_id: args.incident_id, sha256: uploaded.sha256, note: "El original quedó preservado en almacenamiento privado y vinculado al expediente." });
  }
  throw new Error(`Herramienta desconocida: ${name}`);
}

function send(message) { process.stdout.write(`${JSON.stringify(message)}\n`); }
const input = createInterface({ input: process.stdin, crlfDelay: Infinity });
for await (const line of input) {
  if (!line.trim()) continue;
  let request;
  try { request = JSON.parse(line); } catch { continue; }
  if (request.id === undefined) continue;
  try {
    let response;
    if (request.method === "initialize") response = { protocolVersion: request.params?.protocolVersion ?? "2024-11-05", capabilities: { tools: {} }, serverInfo: { name: "control-propiedades", version: "0.2.0" }, instructions: "Consulta primero propiedades y accionables. Los montos están en CLP. Nunca interpretes un abono parcial como descuento o condonación. Toda escritura requiere confirmación explícita." };
    else if (request.method === "ping") response = {};
    else if (request.method === "tools/list") response = { tools };
    else if (request.method === "tools/call") response = await callTool(request.params?.name, request.params?.arguments ?? {});
    else throw Object.assign(new Error(`Método no disponible: ${request.method}`), { code: -32601 });
    send({ jsonrpc: "2.0", id: request.id, result: response });
  } catch (error) {
    send({ jsonrpc: "2.0", id: request.id, error: { code: error.code ?? -32603, message: error instanceof Error ? error.message : "Error interno" } });
  }
}

function requiredEnv(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`Falta la variable ${name}`);
  return value;
}
