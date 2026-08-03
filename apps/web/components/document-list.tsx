import Link from "next/link";
import { formatBytes } from "@/lib/api";
import type { Document } from "@/lib/types";

const money = new Intl.NumberFormat("es-CL", { style: "currency", currency: "CLP", maximumFractionDigits: 0 });

export function DocumentList({ documents }: { documents: Document[] }) {
  if (!documents.length) {
    return <div className="empty-state compact"><p>No hay documentos que coincidan con la búsqueda.</p></div>;
  }
  return <div className="document-list">{documents.map((document) => (
    <article className="document-row" key={document.id}>
      <div className="file-icon">{document.mime_type === "application/pdf" ? "PDF" : "IMG"}</div>
      <div className="document-copy">
        <Link className="document-name" href={`/documentos/${document.id}`}>{document.display_name}</Link>
        <span>{document.property_name} · {typeLabel(document.document_type)} · {formatBytes(document.size_bytes)} · <span className={`document-status status-${document.status}`}>{statusLabel(document.status)}</span></span>
        <span>{[document.period, document.document_date, document.issuer, document.amount_minor === null ? "" : money.format(document.amount_minor), confidentialityLabel(document.confidentiality)].filter(Boolean).join(" · ")}</span>
        <div className="tag-list">{document.tags.map((tag) => <small key={tag}>#{tag}</small>)}</div>
      </div>
      <a className="text-link" href={`/api/v1/documents/${document.id}/download`}>Descargar</a>
    </article>
  ))}</div>;
}

function statusLabel(status: string): string {
  return ({ uploaded: "Cargado", pending_review: "Pendiente", verified: "Verificado", rejected: "Rechazado", duplicate: "Duplicado", archived: "Archivado" } as Record<string, string>)[status] ?? status;
}

function typeLabel(type: string): string {
  return ({ bank_receipt: "Comprobante bancario", invoice: "Factura", rent_receipt: "Vale de arriendo", utility: "Servicio", lease: "Contrato", deed: "Escritura" } as Record<string, string>)[type] ?? type;
}

function confidentialityLabel(value: Document["confidentiality"]): string {
  return ({ internal: "Interno", confidential: "Confidencial", restricted: "Restringido" })[value];
}
