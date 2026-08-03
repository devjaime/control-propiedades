import Link from "next/link";
import type { Document } from "@/lib/types";

const money = new Intl.NumberFormat("es-CL", { style: "currency", currency: "CLP", maximumFractionDigits: 0 });
const month = new Intl.DateTimeFormat("es-CL", { month: "long", year: "numeric", timeZone: "UTC" });

type BrokeragePeriod = {
  period: string;
  documents: Document[];
  netAmount: number;
  feeAmount: number;
};

export function BrokerageHistory({ documents }: { documents: Document[] }) {
  const brokerageDocuments = documents.filter((document) => document.tags.includes("corredora"));
  if (!brokerageDocuments.length) return null;

  const guarantees = brokerageDocuments.filter((document) => document.tags.includes("garantía"));
  const periods = groupPeriods(brokerageDocuments.filter((document) => !document.tags.includes("garantía")));

  return <section className="content-section brokerage-section" id="corredora">
    <div className="section-heading"><div><p className="eyebrow">Administración anterior</p><h2>Historial de corredora</h2></div><span className="status-badge">{brokerageDocuments.length} documentos verificados</span></div>
    <p className="section-intro">Cada periodo conserva la liquidación efectivamente depositada y la factura descontada por la corredora. La suma permite reconstruir la renta bruta sin confundir gastos de administración con depósitos directos.</p>
    {guarantees.map((document) => <article className="brokerage-guarantee" key={document.id}>
      <div><span>Garantía contractual</span><strong>{formatAmount(document.amount_minor)}</strong><small>{formatDate(document.document_date)} · registrada separadamente del arriendo</small></div>
      <DocumentActions document={document} />
    </article>)}
    <div className="brokerage-periods">{periods.map((item) => {
      const grossAmount = item.netAmount + item.feeAmount;
      const complete = item.netAmount > 0 && item.feeAmount > 0;
      return <article className="brokerage-period" key={item.period}>
        <header><div><span>{item.period}</span><h3>{formatPeriod(item.period)}</h3></div><strong className={complete ? "brokerage-balanced" : "brokerage-incomplete"}>{complete ? "Documentación cuadrada" : "Revisión pendiente"}</strong></header>
        <dl>
          <div><dt>Renta bruta documentada</dt><dd>{complete ? money.format(grossAmount) : "Incompleta"}</dd></div>
          <div><dt>Depositado por corredora</dt><dd>{money.format(item.netAmount)}</dd></div>
          <div><dt>Comisión o administración</dt><dd>{money.format(item.feeAmount)}</dd></div>
        </dl>
        <div className="brokerage-files">{item.documents.map((document) => <div key={document.id}>
          <div><strong>{document.document_type === "invoice" ? "Factura descontada" : "Liquidación bancaria"}</strong><span>{formatDate(document.document_date)} · {formatAmount(document.amount_minor)}</span></div>
          <DocumentActions document={document} />
        </div>)}</div>
        <p>Respaldo histórico. No genera un pago adicional ni duplica conciliaciones existentes.</p>
      </article>;
    })}</div>
  </section>;
}

function groupPeriods(documents: Document[]): BrokeragePeriod[] {
  const grouped = new Map<string, Document[]>();
  for (const document of documents) {
    if (!document.period) continue;
    const current = grouped.get(document.period) ?? [];
    current.push(document);
    grouped.set(document.period, current);
  }
  return Array.from(grouped, ([period, periodDocuments]) => ({
    period,
    documents: periodDocuments.toSorted((left, right) => left.document_type.localeCompare(right.document_type)),
    netAmount: sumAmounts(periodDocuments, "bank_receipt"),
    feeAmount: sumAmounts(periodDocuments, "invoice"),
  })).toSorted((left, right) => left.period.localeCompare(right.period));
}

function sumAmounts(documents: Document[], type: string): number {
  let total = 0;
  for (const document of documents) if (document.document_type === type) total += document.amount_minor ?? 0;
  return total;
}

function DocumentActions({ document }: { document: Document }) {
  return <div className="brokerage-actions"><Link className="text-link" href={`/documentos/${document.id}`}>Ver ficha</Link><a className="text-link" href={`/api/v1/documents/${document.id}/download?inline=1`} target="_blank" rel="noreferrer">Abrir PDF</a></div>;
}

function formatPeriod(period: string): string {
  return month.format(new Date(`${period}-01T12:00:00Z`));
}

function formatDate(value: string | null): string {
  if (!value) return "Sin fecha";
  return new Intl.DateTimeFormat("es-CL", { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${value}T12:00:00Z`));
}

function formatAmount(value: number | null): string {
  return value === null ? "Sin monto" : money.format(value);
}
