"use client";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { APIError, api, formatBytes, uploadDocumentDirect } from "@/lib/api";
import type { LeaseContractDraft, LeaseContractWorkflow, TenantApplication } from "@/lib/types";

const MAX_UPLOAD_BYTES = 50 * 1024 * 1024;
const today = new Date().toISOString().slice(0, 10);

function oneYearFromToday() {
  const date = new Date();
  date.setFullYear(date.getFullYear() + 1);
  return date.toISOString().slice(0, 10);
}

const applicationStatus: Record<TenantApplication["status"], string> = {
  draft: "Antecedentes iniciales",
  under_review: "En evaluación",
  approved: "Aprobada",
  rejected: "Rechazada",
  contracted: "Contrato formalizado",
  archived: "Archivada",
};

const draftStatus: Record<LeaseContractDraft["status"], string> = {
  draft: "Borrador pendiente",
  reviewed: "Revisado con observaciones",
  approved_for_signature: "Aprobado para firma",
  signed_notarized: "Firmado/notariado",
  superseded: "Reemplazado",
  void: "Anulado",
};

export function LeaseContractBuilder({ propertyID }: { propertyID: string }) {
  const [workflow, setWorkflow] = useState<LeaseContractWorkflow | null>(null);
  const [selectedApplicationID, setSelectedApplicationID] = useState("");
  const [working, setWorking] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [additionalGuarantee, setAdditionalGuarantee] = useState("none");

  const load = useCallback(async () => {
    const result = await api<LeaseContractWorkflow>(`/properties/${propertyID}/lease-contract-workflow`);
    setWorkflow(result);
    setSelectedApplicationID((current) => current || result.applications[0]?.id || "");
  }, [propertyID]);

  useEffect(() => {
    let active = true;
    api<LeaseContractWorkflow>(`/properties/${propertyID}/lease-contract-workflow`)
      .then((result) => {
        if (!active) return;
        setWorkflow(result);
        setSelectedApplicationID(result.applications[0]?.id || "");
      })
      .catch((caught) => { if (active) setError(caught instanceof Error ? caught.message : "No fue posible cargar los contratos."); });
    return () => { active = false; };
  }, [propertyID]);

  const selectedApplication = useMemo(
    () => workflow?.applications.find((application) => application.id === selectedApplicationID) ?? null,
    [selectedApplicationID, workflow],
  );
  const selectedDrafts = useMemo(
    () => workflow?.drafts.filter((draft) => draft.application_id === selectedApplicationID) ?? [],
    [selectedApplicationID, workflow],
  );

  async function run(action: () => Promise<void>, message: string) {
    setWorking(true); setError(""); setSuccess("");
    try {
      await action();
      await load();
      setSuccess(message);
    } catch (caught) {
      setError(caught instanceof APIError || caught instanceof Error ? caught.message : "No fue posible completar la operación.");
    } finally {
      setWorking(false);
    }
  }

  async function createApplication(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    let createdID = "";
    await run(async () => {
      const income = String(data.get("monthly_income_minor") ?? "").trim();
      const result = await api<{ application: TenantApplication }>(`/properties/${propertyID}/tenant-applications`, {
        method: "POST",
        body: JSON.stringify({
          full_name: data.get("full_name"), identifier: data.get("identifier"), nationality: data.get("nationality"),
          marital_status: data.get("marital_status"), profession: data.get("profession"), current_address: data.get("current_address"),
          email: data.get("email"), phone: data.get("phone"), employer: data.get("employer"),
          monthly_income_minor: income ? Number(income) : null,
          authorized_occupants: data.get("authorized_occupants"), notes: data.get("notes"),
        }),
      });
      createdID = result.application.id;
      form.reset();
    }, "Postulación creada. Ahora puedes adjuntar los antecedentes entregados por el postulante.");
    if (createdID) setSelectedApplicationID(createdID);
  }

  async function uploadBackground(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedApplication) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    const file = data.get("file");
    if (!(file instanceof File) || file.size === 0) { setError("Selecciona el antecedente que deseas cargar."); return; }
    if (file.size > MAX_UPLOAD_BYTES) { setError(`El archivo pesa ${formatBytes(file.size)}. El máximo es 50 MB.`); return; }
    const kind = String(data.get("document_kind") ?? "other");
    if (kind === "commercial_report" && data.get("provenance_confirmed") !== "on") {
      setError("Confirma que el informe comercial fue entregado por el postulante o se obtuvo con su autorización.");
      return;
    }
    await run(async () => {
      const uploaded = await uploadDocumentDirect<{ document: { id: string } }>(propertyID, file, {
        displayName: String(data.get("display_name") ?? "").trim() || file.name,
        documentType: "certificate",
        tags: ["antecedente-arrendatario", kind === "commercial_report" ? "informe-comercial" : kind, "confidencial"],
      });
      await api(`/tenant-applications/${selectedApplication.id}/documents`, {
        method: "POST",
        body: JSON.stringify({ document_id: uploaded.document.id, document_kind: kind, provenance_confirmed: data.get("provenance_confirmed") === "on" }),
      });
      form.reset();
    }, "Antecedente cargado y asociado de manera confidencial.");
  }

  async function createDraft(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedApplication) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    await run(async () => {
      await api(`/tenant-applications/${selectedApplication.id}/contract-drafts`, {
        method: "POST",
        body: JSON.stringify({
          landlord_name: data.get("landlord_name"), landlord_identifier: data.get("landlord_identifier"),
          landlord_nationality: data.get("landlord_nationality"), landlord_marital_status: data.get("landlord_marital_status"),
          landlord_profession: data.get("landlord_profession"), landlord_address: data.get("landlord_address"), landlord_email: data.get("landlord_email"),
          starts_on: data.get("starts_on"), ends_on: data.get("ends_on"), rent_amount_minor: Number(data.get("rent_amount_minor")),
          payment_day: Number(data.get("payment_day")), deposit_months: Number(data.get("deposit_months")),
          adjustment_frequency_months: Number(data.get("adjustment_frequency_months")), notice_days: Number(data.get("notice_days")),
          additional_guarantee: data.get("additional_guarantee"), additional_guarantee_text: data.get("additional_guarantee_text"),
          utilities_responsibility: data.get("utilities_responsibility"), special_terms: data.get("special_terms"),
        }),
      });
    }, "Borrador PDF generado. Revísalo antes de aprobarlo para firma.");
  }

  async function reviewDraft(event: FormEvent<HTMLFormElement>, draftID: string) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    await run(async () => {
      await api(`/lease-contract-drafts/${draftID}/review`, {
        method: "POST",
        body: JSON.stringify({ reviewer_name: data.get("reviewer_name"), approved: data.get("approved") === "on" }),
      });
    }, "Revisión registrada en la bitácora del contrato.");
  }

  async function uploadSigned(event: FormEvent<HTMLFormElement>, draftID: string) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const file = data.get("file");
    if (!(file instanceof File) || file.size === 0) { setError("Selecciona el contrato firmado en PDF."); return; }
    if (file.type !== "application/pdf") { setError("El contrato firmado debe cargarse en PDF."); return; }
    await run(async () => {
      const uploaded = await uploadDocumentDirect<{ document: { id: string } }>(propertyID, file, {
        displayName: String(data.get("display_name") ?? "").trim() || "Contrato de arriendo firmado y notariado",
        documentType: "lease",
        tags: ["contrato", "firmado", "notariado", "vigente"],
      });
      await api(`/lease-contract-drafts/${draftID}/signed-document`, {
        method: "POST",
        body: JSON.stringify({ document_id: uploaded.document.id, verification_confirmed: data.get("verification_confirmed") === "on" }),
      });
      form.reset();
    }, "Contrato firmado/notariado asociado. El borrador original se conservó sin modificaciones.");
  }

  return <section className="content-section contract-builder" id="contratos">
    <div className="section-heading"><div><p className="eyebrow">Contratación asistida</p><h2>Postulación y contrato de arriendo</h2><p>Genera un borrador versionado, reúne antecedentes y conserva por separado el contrato notariado.</p></div><span className="status-badge">Borrador jurídico asistido</span></div>
    <div className="legal-guardrail"><strong>Alcance de esta versión</strong><p>El informe comercial sólo se carga como archivo confidencial: la plataforma no consulta DICOM ni decide automáticamente. El PDF generado no queda vigente hasta su revisión, firma y formalización.</p></div>
    {error ? <p className="form-error" role="alert">{error}</p> : null}
    {success ? <p className="form-success" role="status">{success}</p> : null}

    <div className="contract-step-grid">
      <details className="contract-step" open={!workflow?.applications.length}>
        <summary><span>1</span><div><strong>Crear postulación</strong><small>Identidad y antecedentes básicos</small></div></summary>
        <form className="rental-form contract-form" onSubmit={createApplication}>
          <label>Nombre completo<input name="full_name" required /></label><label>RUT<input name="identifier" placeholder="12.345.678-5" required /></label>
          <label>Nacionalidad<input name="nationality" defaultValue="Chilena" required /></label><label>Estado civil<input name="marital_status" /></label>
          <label>Profesión u oficio<input name="profession" /></label><label>Domicilio actual<input name="current_address" /></label>
          <label>Correo<input name="email" type="email" /></label><label>Teléfono<input name="phone" /></label>
          <label>Empleador<input name="employer" /></label><label>Ingreso mensual acreditado<input name="monthly_income_minor" type="number" min="0" step="1" /></label>
          <label className="full-field">Ocupantes autorizados<textarea name="authorized_occupants" placeholder="Nombre y RUT de cada ocupante" /></label>
          <label className="full-field">Notas internas<textarea name="notes" placeholder="Antecedentes pendientes, observaciones de evaluación…" /></label>
          <button className="primary-button full-field" disabled={working}>Crear postulación</button>
        </form>
      </details>

      <div className="contract-step contract-selection"><div className="contract-step-title"><span>2</span><div><strong>Seleccionar postulante</strong><small>{workflow?.applications.length ?? 0} postulaciones</small></div></div>
        {workflow?.applications.length ? <select value={selectedApplicationID} onChange={(event) => setSelectedApplicationID(event.target.value)}>{workflow.applications.map((application) => <option value={application.id} key={application.id}>{application.full_name} · {application.identifier}</option>)}</select> : <p className="empty-copy">Crea la primera postulación para continuar.</p>}
        {selectedApplication ? <div className="candidate-summary"><span className="status-badge">{applicationStatus[selectedApplication.status]}</span><strong>{selectedApplication.full_name}</strong><small>{selectedApplication.identifier} · {selectedApplication.email || "sin correo"}</small><p>{selectedApplication.documents.some((document) => document.document_kind === "commercial_report") ? "✓ Informe comercial adjunto" : "○ Informe comercial pendiente (no bloquea el borrador)"}</p></div> : null}
      </div>
    </div>

    {selectedApplication ? <>
      <details className="contract-step">
        <summary><span>3</span><div><strong>Cargar antecedentes</strong><small>Archivos confidenciales entregados o autorizados</small></div></summary>
        <form className="upload-form contract-upload" onSubmit={uploadBackground}>
          <label>Archivo<input name="file" type="file" accept="application/pdf,image/jpeg,image/png,image/webp" required /></label>
          <label>Tipo<select name="document_kind" defaultValue="commercial_report"><option value="commercial_report">Informe comercial / DICOM</option><option value="identity">Identidad</option><option value="income_proof">Ingresos</option><option value="employment">Antecedente laboral</option><option value="guarantor">Codeudor</option><option value="insurance">Seguro</option><option value="other">Otro</option></select></label>
          <label>Nombre descriptivo<input name="display_name" placeholder="Informe comercial agosto 2026" /></label>
          <label className="check-field full-field"><input name="provenance_confirmed" type="checkbox" /> Confirmo que el documento fue entregado por el postulante o se obtuvo con su autorización.</label>
          <button className="secondary-button full-field" disabled={working}>Subir y asociar antecedente</button>
        </form>
        {selectedApplication.documents.length ? <div className="background-files">{selectedApplication.documents.map((document) => <div key={document.document_id}><span>{document.document_kind === "commercial_report" ? "Informe comercial" : document.document_kind}</span><strong>{document.display_name}</strong><a className="text-link" href={`/api/v1/documents/${document.document_id}/download`}>Descargar</a></div>)}</div> : <p className="empty-copy">Todavía no hay antecedentes cargados.</p>}
      </details>

      <details className="contract-step" open={!selectedDrafts.length}>
        <summary><span>4</span><div><strong>Generar borrador contractual</strong><small>Condiciones económicas, garantías y responsabilidades</small></div></summary>
        <form className="rental-form contract-form" onSubmit={createDraft}>
          <fieldset className="full-field"><legend>Identificación del arrendador</legend><div className="rental-form">
            <label>Nombre completo<input name="landlord_name" required /></label><label>RUT<input name="landlord_identifier" placeholder="12.345.678-5" required /></label>
            <label>Nacionalidad<input name="landlord_nationality" defaultValue="Chilena" /></label><label>Estado civil<input name="landlord_marital_status" /></label>
            <label>Profesión u oficio<input name="landlord_profession" /></label><label>Correo<input name="landlord_email" type="email" /></label>
            <label className="full-field">Domicilio<input name="landlord_address" /></label>
          </div></fieldset>
          <label>Inicio<input name="starts_on" type="date" defaultValue={today} required /></label><label>Término<input name="ends_on" type="date" defaultValue={oneYearFromToday()} required /></label>
          <label>Renta mensual<input name="rent_amount_minor" type="number" min="1" step="1" required /></label><label>Día de pago<input name="payment_day" type="number" min="1" max="28" defaultValue="5" required /></label>
          <label>Meses de garantía<select name="deposit_months" defaultValue="2"><option value="1">1 mes</option><option value="2">2 meses</option><option value="3">3 meses (revisión especial)</option></select></label><label>Reajuste IPC cada<input name="adjustment_frequency_months" type="number" min="1" max="36" defaultValue="12" required /></label>
          <label>Aviso contractual (días)<input name="notice_days" type="number" min="30" max="365" defaultValue="60" required /></label><label>Garantía adicional<select name="additional_guarantee" value={additionalGuarantee} onChange={(event) => setAdditionalGuarantee(event.target.value)}><option value="none">Sin garantía adicional</option><option value="co_debtor">Codeudor solidario</option><option value="insurance">Seguro</option></select></label>
          {additionalGuarantee !== "none" ? <label className="full-field">Detalle de garantía adicional<textarea name="additional_guarantee_text" required placeholder={additionalGuarantee === "insurance" ? "Compañía, producto, vigencia y póliza si existe" : "Nombre, RUT y alcance de la obligación del codeudor"} /></label> : <input name="additional_guarantee_text" type="hidden" value="" />}
          <label className="full-field">Servicios y gastos<textarea name="utilities_responsibility" placeholder="Si se deja vacío se aplicará la cláusula estándar de agua, luz, gas, aseo y gastos comunes." /></label>
          <label className="full-field">Condiciones particulares<textarea name="special_terms" placeholder="Serán destacadas como sujetas a revisión jurídica." /></label>
          <div className="contract-warning full-field"><strong>Cheque en garantía</strong><p>Esta versión no lo genera automáticamente. Por su naturaleza de orden de pago a la vista, sólo debe incorporarse mediante cláusula preparada para el caso concreto por un abogado.</p></div>
          <button className="primary-button full-field" disabled={working}>Generar PDF borrador</button>
        </form>
      </details>

      <div className="contract-step"><div className="contract-step-title"><span>5</span><div><strong>Revisar, firmar y subir</strong><small>El borrador nunca se sobrescribe</small></div></div>
        {!selectedDrafts.length ? <p className="empty-copy">Genera el primer borrador para iniciar la revisión.</p> : <div className="contract-draft-list">{selectedDrafts.map((draft) => <article key={draft.id} className="contract-draft-card">
          <div><span className="status-badge">{draftStatus[draft.status]}</span><h3>Versión {draft.draft_version}</h3><small>{draft.created_at} · {draft.template_version}</small><code title={draft.pdf_sha256}>{draft.pdf_sha256.slice(0, 16)}…</code></div>
          <div className="contract-draft-actions"><a className="secondary-button" href={`/api/v1/documents/${draft.pdf_document_id}/download`}>Revisar PDF</a><a className="secondary-button" href={`/api/v1/documents/${draft.docx_document_id}/download`}>Descargar Word para notaría</a>{draft.signed_document_id ? <a className="primary-button" href={`/api/v1/documents/${draft.signed_document_id}/download`}>Descargar firmado</a> : null}</div>
          {draft.status !== "signed_notarized" ? <form className="compact-form contract-review" onSubmit={(event) => reviewDraft(event, draft.id)}><label>Revisor jurídico<input name="reviewer_name" defaultValue={draft.legal_reviewed_by} placeholder="Nombre y calidad profesional" required /></label><label className="check-field"><input name="approved" type="checkbox" /> Aprobado para imprimir y firmar</label><button className="secondary-button" disabled={working}>Registrar revisión</button></form> : null}
          {draft.status === "approved_for_signature" ? <form className="compact-form signed-upload" onSubmit={(event) => uploadSigned(event, draft.id)}><label>Contrato firmado/notariado<input name="file" type="file" accept="application/pdf" required /></label><label>Nombre<input name="display_name" defaultValue={`Contrato firmado - ${selectedApplication.full_name}`} /></label><label className="check-field full-field"><input name="verification_confirmed" type="checkbox" required /> Confirmo que revisé identidad, firmas, certificación notarial, todas las páginas y anexos.</label><button className="primary-button" disabled={working}>Subir versión formalizada</button></form> : null}
        </article>)}</div>}
      </div>
    </> : null}
  </section>;
}
