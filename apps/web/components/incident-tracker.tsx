"use client";

import { FormEvent, useEffect, useState } from "react";
import { APIError, api, formatBytes, uploadDocumentDirect } from "@/lib/api";
import type { Document, Incident, IncidentTask, IncidentUpdate } from "@/lib/types";

const MAX_UPLOAD_BYTES = 50 * 1024 * 1024;

const statusLabels = { open: "Abierto", in_progress: "En seguimiento", resolved: "Resuelto", closed: "Cerrado" };
const priorityLabels = { low: "Baja", medium: "Media", high: "Alta", critical: "Crítica" };
const updateLabels = {
  note: "Nota", tenant_contact: "Contacto arrendatario", builder_claim: "Postventa constructora",
  insurance_claim: "Seguro", inspection: "Inspección", status_change: "Cambio de estado",
};

function localDate(value: string) {
  return new Intl.DateTimeFormat("es-CL", { dateStyle: "medium" }).format(new Date(`${value}T12:00:00`));
}

export function IncidentTracker({ propertyID, onDocumentUploaded }: { propertyID: string; onDocumentUploaded?: (document: Document) => void }) {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ incidents: Incident[] }>(`/properties/${propertyID}/incidents`)
      .then((result) => setIncidents(result.incidents))
      .catch((caught) => setError(caught instanceof Error ? caught.message : "No fue posible cargar los incidentes."))
      .finally(() => setLoading(false));
  }, [propertyID]);

  async function createIncident(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(""); setSaving(true);
    const form = event.currentTarget;
    const data = new FormData(form);
    try {
      const result = await api<{ incident: Incident }>(`/properties/${propertyID}/incidents`, {
        method: "POST",
        body: JSON.stringify({
          reference: String(data.get("reference") ?? "").trim(),
          title: String(data.get("title") ?? "").trim(),
          summary: String(data.get("summary") ?? "").trim(),
          priority: data.get("priority"), event_date: data.get("event_date"),
          habitability: data.get("habitability"),
          categories: String(data.get("categories") ?? "").split(",").map((item) => item.trim()).filter(Boolean),
          tags: String(data.get("tags") ?? "").split(",").map((item) => item.trim()).filter(Boolean),
          tasks: [],
        }),
      });
      setIncidents((current) => [result.incident, ...current]);
      form.reset();
    } catch (caught) {
      setError(caught instanceof APIError ? caught.message : "No fue posible registrar el incidente.");
    } finally { setSaving(false); }
  }

  async function changeTask(incidentID: string, task: IncidentTask, visibility?: boolean) {
    setError("");
    try {
      const result = await api<{ task: IncidentTask }>(`/incident-tasks/${task.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status: visibility === undefined ? (task.status === "completed" ? "pending" : "completed") : task.status, version: task.version, ...(visibility === undefined ? {} : { visible_to_tenant: visibility }) }),
      });
      setIncidents((current) => current.map((incident) => incident.id !== incidentID ? incident : {
        ...incident, tasks: incident.tasks.map((item) => item.id === task.id ? result.task : item),
      }));
    } catch (caught) { setError(caught instanceof Error ? caught.message : "No fue posible actualizar la tarea."); }
  }

  async function addUpdate(event: FormEvent<HTMLFormElement>, incidentID: string) {
    event.preventDefault(); setError(""); setSaving(true);
    const form = event.currentTarget;
    const data = new FormData(form);
    try {
      const result = await api<{ update: IncidentUpdate }>(`/incidents/${incidentID}/updates`, {
        method: "POST",
        body: JSON.stringify({
          update_type: data.get("update_type"), content: String(data.get("content") ?? "").trim(),
          occurred_at: new Date().toISOString(), visible_to_tenant: data.get("visible_to_tenant") === "on",
        }),
      });
      setIncidents((current) => current.map((incident) => incident.id === incidentID ? {
        ...incident, updates: [result.update, ...incident.updates],
      } : incident));
      form.reset();
    } catch (caught) { setError(caught instanceof Error ? caught.message : "No fue posible guardar la actualización."); }
    finally { setSaving(false); }
  }

  async function uploadEvidence(event: FormEvent<HTMLFormElement>, incidentID: string) {
    event.preventDefault(); setError(""); setSaving(true);
    const form = event.currentTarget;
    const data = new FormData(form);
    const file = data.get("file");
    if (!(file instanceof File) || file.size === 0) {
      setError("Selecciona una fotografía o PDF como evidencia."); setSaving(false); return;
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      setError(`El archivo pesa ${formatBytes(file.size)}. El máximo permitido es 50 MB.`); setSaving(false); return;
    }
    try {
      const result = await uploadDocumentDirect<{ document: Document }>(propertyID, file, {
        displayName: String(data.get("display_name") ?? "").trim(),
        documentType: file.type.startsWith("image/") ? "photo" : "inspection",
        tags: ["evidencia", "incidente"],
        incidentID,
      });
      setIncidents((current) => current.map((incident) => incident.id === incidentID ? {
        ...incident,
        documents: [{ id: result.document.id, display_name: result.document.display_name, mime_type: result.document.mime_type }, ...incident.documents],
      } : incident));
      onDocumentUploaded?.(result.document);
      form.reset();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "No fue posible subir la evidencia.");
    } finally { setSaving(false); }
  }

  return <section className="content-section incident-section" id="incidentes">
    <div className="section-heading"><div><p className="eyebrow">Riesgos y postventa</p><h2>Incidentes</h2></div></div>
    <p className="section-intro">Registra daños, compromisos, comunicaciones y evidencia hasta su resolución definitiva.</p>
    {error ? <p className="form-error" role="alert">{error}</p> : null}
    {loading ? <p className="incident-loading">Cargando seguimiento…</p> : null}
    <div className="incident-list">
      {incidents.map((incident) => {
        const completed = incident.tasks.filter((task) => task.status === "completed").length;
        return <article className="incident-card" key={incident.id}>
          <header className="incident-card-header">
            <div><span className="incident-reference">{incident.reference}</span><h3>{incident.title}</h3><p>{localDate(incident.event_date)}</p></div>
            <div className="incident-badges"><span className={`incident-priority priority-${incident.priority}`}>{priorityLabels[incident.priority]}</span><span className="status-badge">{statusLabels[incident.status]}</span></div>
          </header>
          <p className="incident-summary">{incident.summary}</p>
          <div className="tag-list">{incident.categories.map((category) => <small key={category}>#{category}</small>)}</div>
          <div className="incident-columns">
            <section><h4>Acciones · {completed}/{incident.tasks.length}</h4><div className="incident-task-list">
              {incident.tasks.map((task) => <div className="incident-task-row" key={task.id}><button className={`incident-task ${task.status === "completed" ? "completed" : ""}`} onClick={() => changeTask(incident.id, task)} type="button"><span>{task.status === "completed" ? "✓" : "○"}</span><span>{task.title}{task.due_on ? <small>Vence {localDate(task.due_on)}</small> : null}</span></button><button className="visibility-toggle" onClick={() => changeTask(incident.id, task, !task.visible_to_tenant)} type="button">{task.visible_to_tenant ? "Visible al arrendatario" : "Sólo interno"}</button></div>)}
              {!incident.tasks.length ? <p className="muted-copy">Sin acciones pendientes.</p> : null}
            </div></section>
            <section><h4>Evidencia · {incident.documents.length}</h4><div className="incident-evidence">
              {incident.documents.map((document) => <a href={`/documentos/${document.id}`} key={document.id}>{document.mime_type.startsWith("image/") ? "IMG" : "PDF"}<span>{document.display_name}</span></a>)}
              {!incident.documents.length ? <p className="muted-copy">Sin archivos vinculados.</p> : null}
              <form className="incident-evidence-form" onSubmit={(event) => uploadEvidence(event, incident.id)}>
                <input name="file" type="file" accept="application/pdf,image/jpeg,image/png,image/webp" aria-label="Archivo de evidencia" required />
                <input name="display_name" placeholder="Descripción de la evidencia" aria-label="Descripción de la evidencia" />
                <button className="secondary-button" disabled={saving} type="submit">{saving ? "Subiendo…" : "Subir evidencia"}</button>
              </form>
            </div></section>
          </div>
          <section className="incident-timeline"><h4>Bitácora</h4>{incident.updates.map((update) => <div key={update.id}><strong>{updateLabels[update.update_type]} {update.visible_to_tenant ? <small>· visible al arrendatario</small> : null}</strong><p>{update.content}</p><small>{new Intl.DateTimeFormat("es-CL", { dateStyle: "medium", timeStyle: "short" }).format(new Date(update.occurred_at))} · {update.created_by}</small></div>)}</section>
          <form className="incident-update-form" onSubmit={(event) => addUpdate(event, incident.id)}><select name="update_type" aria-label="Tipo de actualización"><option value="note">Nota</option><option value="tenant_contact">Contacto arrendatario</option><option value="builder_claim">Postventa constructora</option><option value="insurance_claim">Seguro</option><option value="inspection">Inspección</option></select><input name="content" aria-label="Detalle de actualización" placeholder="Agregar novedad o compromiso…" required /><label className="inline-check"><input name="visible_to_tenant" type="checkbox" /> Mostrar al arrendatario</label><button className="secondary-button" disabled={saving} type="submit">Agregar</button></form>
        </article>;
      })}
    </div>
    <details className="incident-create"><summary>Registrar otro incidente</summary><form className="stack-form two-columns" onSubmit={createIncident}>
      <label>Referencia<input name="reference" placeholder="PROP-…" required /></label><label>Fecha del evento<input name="event_date" type="date" required /></label>
      <label className="full-field">Título<input name="title" required /></label><label className="full-field">Resumen<textarea name="summary" rows={4} required /></label>
      <label>Prioridad<select name="priority" defaultValue="medium"><option value="low">Baja</option><option value="medium">Media</option><option value="high">Alta</option><option value="critical">Crítica</option></select></label>
      <label>Habitabilidad<select name="habitability" defaultValue="unaffected"><option value="unaffected">Sin afectar</option><option value="partially_affected">Parcialmente afectada</option><option value="uninhabitable">Inhabitable</option></select></label>
      <label>Categorías<input name="categories" placeholder="Inundación, Postventa" /></label><label>Etiquetas<input name="tags" placeholder="lluvias, seguro" /></label>
      <button className="primary-button full-field" disabled={saving} type="submit">Registrar incidente</button>
    </form></details>
  </section>;
}
