"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { APIError, api, formatBytes } from "@/lib/api";
import type { Document } from "@/lib/types";
import { FilePreview } from "./file-preview";

export function DocumentReview({ id }: { id: string }) {
  const [document, setDocument] = useState<Document | null>(null);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api<{ document: Document }>(`/documents/${id}`)
      .then((result) => setDocument(result.document))
      .catch((caught) => setError(caught instanceof Error ? caught.message : "No fue posible cargar el documento."));
  }, [id]);

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!document) return;
    setSaving(true); setError(""); setMessage("");
    const data = new FormData(event.currentTarget);
    const rawAmount = lockedValue(document.status === "verified", document.amount_minor, data.get("amount_minor"));
    const locked = document.status === "verified";
    const payload = {
      display_name: locked ? document.display_name : String(data.get("display_name") ?? ""),
      document_type: locked ? document.document_type : String(data.get("document_type") ?? ""),
      document_date: locked ? document.document_date ?? "" : String(data.get("document_date") ?? ""),
      period: locked ? document.period ?? "" : String(data.get("period") ?? ""),
      issuer: locked ? document.issuer ?? "" : String(data.get("issuer") ?? ""),
      amount_minor: rawAmount === "" ? null : Number(rawAmount),
      currency_code: rawAmount === "" ? "" : locked ? document.currency_code ?? "CLP" : String(data.get("currency_code") ?? "CLP"),
      confidentiality: locked ? document.confidentiality : String(data.get("confidentiality") ?? "internal"),
      tags: locked ? document.tags : String(data.get("tags") ?? "").split(",").map((tag) => tag.trim()).filter(Boolean),
      status: String(data.get("status") ?? "pending_review"),
      rejection_reason: String(data.get("rejection_reason") ?? ""),
      version: document.version,
    };
    try {
      const result = await api<{ document: Document }>(`/documents/${id}`, { method: "PATCH", body: JSON.stringify(payload) });
      setDocument(result.document);
      setMessage(result.document.status === "verified" ? "Documento verificado y bloqueado para edición." : "Cambios guardados correctamente.");
    } catch (caught) {
      const text = caught instanceof Error ? caught.message : "No fue posible guardar los cambios.";
      setError(caught instanceof APIError && caught.status === 409 ? `${text} Recarga la página.` : text);
    } finally { setSaving(false); }
  }

  if (!document) return <main className="loading-screen">{error || "Cargando documento…"}</main>;
  const locked = document.status === "verified";
  return <main className="app-main review-page">
    <Link className="back-link" href={`/propiedades/${document.property_id}`}>← Volver a {document.property_name}</Link>
    <section className="review-heading"><div><p className="eyebrow">Revisión documental</p><h1>{document.display_name}</h1><p>{document.original_name} · {formatBytes(document.size_bytes)}</p></div><div className="review-actions"><FilePreview src={`/api/v1/documents/${document.id}`} name="Vista previa del archivo" />
<a className="secondary-button link-button" href={`/api/v1/documents/${document.id}/download?inline=1`} target="_blank" rel="noreferrer">Abrir documento</a><a className="secondary-button link-button" href={`/api/v1/documents/${document.id}/download`}>Descargar original</a></div></section>
    {document.mime_type === "application/pdf" || document.mime_type.startsWith("image/") ? <section className="document-preview"><iframe src={`/api/v1/documents/${document.id}/download?inline=1`} title={`Vista previa de ${document.display_name}`} /></section> : null}
    {locked ? <div className="verified-notice">✓ Verificado {document.verified_at ? new Date(document.verified_at).toLocaleDateString("es-CL") : ""}. El registro es inmutable; para corregirlo deberá reemplazarse en una iteración posterior.</div> : null}
    {message ? <p className="form-success" role="status">{message}</p> : null}
    {error ? <p className="form-error" role="alert">{error}</p> : null}
    <form className="review-grid" onSubmit={save}>
      <section className="review-card"><p className="eyebrow">Clasificación</p><div className="stack-form two-columns">
        <label className="full-field">Nombre visible<input name="display_name" defaultValue={document.display_name} disabled={locked} required /></label>
        <label>Tipo<select name="document_type" defaultValue={document.document_type} disabled={locked}><option value="deed">Escritura</option><option value="domain_registration">Inscripción de dominio</option><option value="valuation">Tasación</option><option value="lease">Contrato</option><option value="annex">Anexo</option><option value="inventory">Inventario</option><option value="bank_receipt">Comprobante bancario</option><option value="rent_receipt">Comprobante de arriendo</option><option value="mortgage">Dividendo</option><option value="property_tax">Contribuciones</option><option value="insurance">Seguro</option><option value="invoice">Factura</option><option value="photo">Fotografía</option><option value="certificate">Certificado</option><option value="other">Otro</option></select></label>
        <label>Confidencialidad<select name="confidentiality" defaultValue={document.confidentiality} disabled={locked}><option value="internal">Interno</option><option value="confidential">Confidencial</option><option value="restricted">Restringido</option></select></label>
        <label className="full-field">Etiquetas<input name="tags" defaultValue={document.tags.join(", ")} disabled={locked} placeholder="legal, 2026, vigente" /></label>
      </div></section>
      <section className="review-card"><p className="eyebrow">Datos del documento</p><div className="stack-form two-columns">
        <label>Fecha documental<input name="document_date" type="date" defaultValue={document.document_date ?? ""} disabled={locked} /></label>
        <label>Período<input name="period" type="month" defaultValue={document.period ?? ""} disabled={locked} /></label>
        <label className="full-field">Emisor<input name="issuer" defaultValue={document.issuer ?? ""} disabled={locked} placeholder="Institución o persona emisora" /></label>
        <label>Monto<input name="amount_minor" type="number" min="0" step="1" defaultValue={document.amount_minor ?? ""} disabled={locked} /></label>
        <label>Moneda<select name="currency_code" defaultValue={document.currency_code ?? "CLP"} disabled={locked}><option value="CLP">CLP</option><option value="USD">USD</option><option value="EUR">EUR</option></select></label>
      </div></section>
      <section className="review-card full-review-card"><p className="eyebrow">Decisión humana</p><div className="decision-row">
        <label>Estado<select name="status" defaultValue={document.status} key={document.status}>{locked ? <><option value="verified">Verificado</option><option value="archived">Archivado</option></> : <><option value="pending_review">Pendiente de revisión</option>{document.status !== "rejected" ? <option value="verified">Verificado</option> : null}<option value="rejected">Rechazado</option><option value="archived">Archivado</option></>}</select></label>
        <label>Motivo del rechazo<input name="rejection_reason" defaultValue={document.rejection_reason ?? ""} disabled={locked} placeholder="Obligatorio al rechazar" /></label>
        <button className="primary-button" disabled={saving} type="submit">{saving ? "Guardando…" : locked ? "Archivar documento" : "Guardar revisión"}</button>
      </div></section>
    </form>
  </main>;
}

function lockedValue(locked: boolean, current: number | null, formValue: FormDataEntryValue | null): string {
  if (locked) return current === null ? "" : String(current);
  return String(formValue ?? "").trim();
}
