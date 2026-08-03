"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { APIError, api, formatBytes, uploadDocumentDirect } from "@/lib/api";
import type { Document, Property, User } from "@/lib/types";
import { AppHeader } from "./app-header";
import { BrokerageHistory } from "./brokerage-history";
import { DocumentList } from "./document-list";
import { IncidentTracker } from "./incident-tracker";
import { LeaseContractBuilder } from "./lease-contract-builder";
import { RentalLedger } from "./rental-ledger";
import { TenantPortalManager } from "./tenant-portal-manager";

const MAX_UPLOAD_BYTES = 50 * 1024 * 1024;

export function PropertyDetail({ id }: { id: string }) {
  const [property, setProperty] = useState<Property | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [error, setError] = useState("");
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    Promise.all([
      api<{ user: User }>("/me"),
      api<{ property: Property }>(`/properties/${id}`),
      api<{ documents: Document[] }>(`/documents?property_id=${encodeURIComponent(id)}`),
    ]).then(([account, propertyData, documentData]) => { setUser(account.user); setProperty(propertyData.property); setDocuments(documentData.documents); })
      .catch((caught) => setError(caught instanceof Error ? caught.message : "No fue posible cargar la propiedad."));
  }, [id]);

  async function upload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError("");
    const form = event.currentTarget;
    const formData = new FormData(form);
    const file = formData.get("file");
    if (!(file instanceof File) || file.size === 0) {
      setError("Selecciona un archivo para subir.");
      return;
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      setError(`El archivo pesa ${formatBytes(file.size)}. El máximo permitido es 50 MB.`);
      return;
    }
    setUploading(true);
    try {
      const result = await uploadDocumentDirect<{ document: Document }>(id, file, {
        displayName: String(formData.get("display_name") ?? "").trim(),
        documentType: String(formData.get("document_type") ?? "other"),
        tags: String(formData.get("tags") ?? "").split(",").map((tag) => tag.trim()).filter(Boolean),
      });
      setDocuments((current) => [result.document, ...current]);
      form.reset();
    } catch (caught) {
      setError(caught instanceof APIError ? caught.message : "No fue posible subir el documento.");
    } finally { setUploading(false); }
  }

  if (!property || !user) return <main className="loading-screen">{error || "Cargando expediente…"}</main>;
  return <><AppHeader organization={user.organization} /><main className="app-main property-page"><Link className="back-link" href="/panel">← Todas las propiedades</Link><section className="property-hero"><div><p className="eyebrow">Expediente individual</p><h1>{property.name}</h1><p>{property.address_line} · {property.commune}, {property.region}</p></div><span className="status-badge">{property.status}</span></section>
    <nav className="property-menu" aria-label="Secciones de la propiedad"><a href="#arriendo">Arriendo y pagos</a><a href="#contratos">Contratación</a><a href="#seguimiento-arrendatario">Portal compartido</a><a href="#incidentes">Incidentes</a>{documents.some((document) => document.tags.includes("corredora")) ? <a href="#corredora">Corredora</a> : null}<a href="#documentos">Documentos</a></nav>
    {error ? <p className="form-error" role="alert">{error}</p> : null}
    <RentalLedger propertyID={id} />
    <LeaseContractBuilder propertyID={id} />
    <TenantPortalManager propertyID={id} />
    <IncidentTracker propertyID={id} onDocumentUploaded={(document) => setDocuments((current) => [document, ...current])} />
    <BrokerageHistory documents={documents} />
    <section className="upload-panel" id="documentos"><div><p className="eyebrow">Nuevo documento</p><h2>Agregar al expediente</h2><p>PDF o imagen, hasta 50 MB. Añade etiquetas para encontrarlo más adelante.</p></div><form className="upload-form" onSubmit={upload}>
      <label>Archivo<input name="file" type="file" accept="application/pdf,image/jpeg,image/png,image/webp" required /></label>
      <label>Nombre descriptivo<input name="display_name" placeholder="Ej. Escritura vigente" /></label>
      <label>Tipo<select name="document_type" required><option value="deed">Escritura</option><option value="domain_registration">Inscripción de dominio</option><option value="valuation">Tasación</option><option value="lease">Contrato de arriendo</option><option value="annex">Anexo</option><option value="bank_receipt">Comprobante bancario</option><option value="mortgage">Dividendo</option><option value="property_tax">Contribuciones</option><option value="insurance">Seguro</option><option value="invoice">Factura</option><option value="photo">Fotografía</option><option value="certificate">Certificado</option><option value="other">Otro</option></select></label>
      <label>Etiquetas<input name="tags" placeholder="legal, 2026, vigente" /></label><button className="primary-button" disabled={uploading} type="submit">{uploading ? "Subiendo…" : "Subir documento"}</button>
    </form></section>
    <section className="content-section"><div className="section-heading"><div><p className="eyebrow">Documentos</p><h2>{documents.length} en este expediente</h2></div></div><DocumentList documents={documents} /></section>
  </main></>;
}
