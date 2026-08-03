"use client";

import { FormEvent, useState } from "react";
import { APIError, api } from "@/lib/api";
import type { TenantPortalIncident } from "@/lib/types";

type PortalData = {
  property: { name: string; commune: string };
  rent_summary: { last_paid_period: string; current_period: string; amount_minor: number; received_minor: number; balance_minor: number; currency_code: string; status: string };
  payment_documents: Array<{
    id: string;
    kind: "bank_receipt" | "rent_receipt" | "utility";
    display_name: string;
    original_name: string;
    period: string;
    document_date: string;
    amount_minor: number;
    currency_code: string;
    mime_type: string;
    folio?: string;
    verification_url?: string;
  }>;
  incidents: TenantPortalIncident[];
  generated_at: string;
};

function periodLabel(period: string) {
  if (!period) return "—";
  return new Intl.DateTimeFormat("es-CL", { month: "long", year: "numeric", timeZone: "UTC" }).format(new Date(`${period}-01T12:00:00Z`));
}

function money(value: number) { return new Intl.NumberFormat("es-CL", { style: "currency", currency: "CLP", maximumFractionDigits: 0 }).format(value); }

const statusLabel: Record<string, string> = { open: "Abierto", in_progress: "En curso", resolved: "Resuelto", pending: "Pendiente", completed: "Completada", cancelled: "Cancelada" };
const documentKindLabel: Record<PortalData["payment_documents"][number]["kind"], string> = {
  bank_receipt: "Respaldo bancario",
  rent_receipt: "Voucher de arriendo",
  utility: "Pago de servicio",
};

export function TenantPortalView({ token }: { token: string }) {
  const [accessKey, setAccessKey] = useState(() => typeof window === "undefined" ? "" : (sessionStorage.getItem(`tenant-portal:${token}`) ?? ""));
  const [data, setData] = useState<PortalData | null>(null);
  const [error, setError] = useState("");
  const [working, setWorking] = useState(false);
  const [downloadingID, setDownloadingID] = useState("");
  const [downloadError, setDownloadError] = useState("");

  async function open(event: FormEvent) {
    event.preventDefault(); setWorking(true); setError("");
    try {
      const response = await api<PortalData>(`/public/tenant-portals/${token}/view`, { method: "POST", body: JSON.stringify({ access_key: accessKey }) });
      sessionStorage.setItem(`tenant-portal:${token}`, accessKey);
      setData(response);
    } catch (caught) {
      setData(null);
      setError(caught instanceof APIError ? caught.message : "No fue posible abrir el seguimiento.");
    } finally { setWorking(false); }
  }

  async function downloadDocument(document: PortalData["payment_documents"][number]) {
    setDownloadingID(document.id);
    setDownloadError("");
    try {
      const response = await fetch(`/api/v1/public/tenant-portals/${token}/documents/${document.id}/download`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ access_key: accessKey }),
      });
      if (!response.ok) {
        const body = await response.json().catch(() => null);
        throw new Error(body?.error?.message ?? "No fue posible descargar el comprobante.");
      }
      const blobURL = URL.createObjectURL(await response.blob());
      const anchor = window.document.createElement("a");
      anchor.href = blobURL;
      anchor.download = document.original_name;
      anchor.click();
      URL.revokeObjectURL(blobURL);
    } catch (caught) {
      setDownloadError(caught instanceof Error ? caught.message : "No fue posible descargar el comprobante.");
    } finally {
      setDownloadingID("");
    }
  }

  if (!data) return <main className="public-portal-shell"><section className="portal-login-card"><p className="eyebrow">Control Propiedades</p><h1>Pagos y seguimiento</h1><p>Ingresa los últimos cuatro dígitos del número de RUT del propietario o del arrendatario, sin puntos, guion ni dígito verificador.</p><form className="stack-form" onSubmit={open}><label>Código de acceso<input autoComplete="one-time-code" autoFocus inputMode="numeric" maxLength={4} minLength={4} onChange={(event) => setAccessKey(event.target.value.replace(/\D/g, "").slice(0, 4))} pattern="[0-9]{4}" required value={accessKey} /></label>{error ? <p className="form-error" role="alert">{error}</p> : null}<button className="primary-button" disabled={working} type="submit">{working ? "Verificando…" : "Abrir portal"}</button></form><small>El código se valida de forma segura, no se incorpora al enlace y el acceso se bloquea temporalmente después de varios intentos fallidos.</small></section></main>;

  return <main className="public-portal-shell portal-dashboard"><header><div><p className="eyebrow">Portal compartido de la propiedad</p><h1>{data.property.name}</h1><p>{data.property.commune} · Actualizado {new Date(data.generated_at).toLocaleString("es-CL")}</p></div><button className="text-button" onClick={() => { sessionStorage.removeItem(`tenant-portal:${token}`); setData(null); setAccessKey(""); }} type="button">Cerrar acceso</button></header>
    <section className="portal-notice"><strong>Información operativa</strong><p>Este panel comunica el avance de reparaciones. No modifica el contrato ni constituye por sí solo un acuerdo económico.</p></section>
    <section className="portal-rent-summary" aria-labelledby="rent-summary-title"><div><p className="eyebrow">Estado de arriendo</p><h2 id="rent-summary-title">Último período pagado: {periodLabel(data.rent_summary.last_paid_period)}</h2></div>{data.rent_summary.current_period ? <div className="portal-rent-values"><span><small>Período siguiente</small><strong>{periodLabel(data.rent_summary.current_period)}</strong></span><span><small>Abonado</small><strong>{money(data.rent_summary.received_minor)}</strong></span><span><small>Saldo registrado</small><strong>{money(data.rent_summary.balance_minor)}</strong></span></div> : <p>Sin saldos pendientes registrados.</p>}<small>Información basada en pagos conciliados. No representa una rebaja, compensación o condonación salvo acuerdo escrito entre las partes.</small></section>
    <section className="portal-payment-archive" aria-labelledby="payment-archive-title"><div className="portal-section-heading"><div><p className="eyebrow">Archivo compartido</p><h2 id="payment-archive-title">Comprobantes, servicios y vouchers</h2></div><span>{data.payment_documents.length} archivos vigentes</span></div><p className="muted-copy">El archivo distingue los respaldos bancarios, los vouchers emitidos después de conciliar el arriendo y los pagos verificados de servicios de la propiedad.</p>{downloadError ? <p className="form-error" role="alert">{downloadError}</p> : null}{data.payment_documents.length ? <ul className="portal-document-list">{data.payment_documents.map((document) => <li key={document.id}><div className="portal-document-copy"><span className={`document-kind document-kind-${document.kind}`}>{documentKindLabel[document.kind]}</span><strong>{document.display_name || periodLabel(document.period)}</strong><small>{document.folio ? `${document.folio} · ` : ""}{money(document.amount_minor)} · {new Date(`${document.document_date}T12:00:00`).toLocaleDateString("es-CL")}</small></div><div className="portal-document-actions"><button className="secondary-button" disabled={downloadingID === document.id} onClick={() => void downloadDocument(document)} type="button">{downloadingID === document.id ? "Descargando…" : "Descargar"}</button>{document.verification_url ? <a className="text-link" href={document.verification_url} rel="noreferrer" target="_blank">Verificar</a> : null}</div></li>)}</ul> : <p className="empty-state">Todavía no hay comprobantes verificados disponibles.</p>}<small className="portal-privacy-note">Acceso privado y registrado. Los respaldos pueden contener datos personales; no reenvíes este enlace ni tu código fuera de las partes del contrato.</small></section>
    <section className="ticket-grid">{data.incidents.length ? data.incidents.map((incident) => <article className="ticket-card" key={incident.id}><div className="ticket-heading"><div><span className={`priority-dot priority-${incident.priority}`} aria-hidden="true" /><p>{incident.reference}</p><h2>{incident.title}</h2></div><span className="status-badge">{statusLabel[incident.status] ?? incident.status}</span></div>{incident.summary ? <p className="ticket-summary">{incident.summary}</p> : null}<h3>Acciones</h3>{incident.tasks.length ? <ul className="portal-task-list">{incident.tasks.map((task) => <li key={task.id}><div><strong>{task.title}</strong>{task.due_on ? <small>Fecha objetivo: {new Date(`${task.due_on}T12:00:00`).toLocaleDateString("es-CL")}</small> : null}</div><span>{statusLabel[task.status] ?? task.status}</span></li>)}</ul> : <p className="muted-copy">Sin acciones públicas pendientes.</p>}{incident.updates.length ? <><h3>Últimas novedades</h3><ol className="portal-update-list">{incident.updates.map((update) => <li key={update.id}><time>{new Date(update.occurred_at).toLocaleString("es-CL")}</time><p>{update.content}</p></li>)}</ol></> : null}</article>) : <p className="empty-state">No hay tickets visibles en curso.</p>}</section>
  </main>;
}
