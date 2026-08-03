"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Verification = {
  verified: boolean;
  verification_label: string;
  folio: string;
  status: string;
  period: string;
  property: string;
  commune: string;
  tenant: string;
  amount_minor: string;
  currency_code: string;
  issued_at: string;
  verification_code: string;
  pdf_sha256: string;
  template_version: string;
};

const money = new Intl.NumberFormat("es-CL", { style: "currency", currency: "CLP", maximumFractionDigits: 0 });

export function ReceiptVerification({ token }: { token: string }) {
  const [data, setData] = useState<Verification | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    api<Verification>(`/public/receipts/${token}`).then(setData).catch((caught) => setError(caught instanceof Error ? caught.message : "No fue posible verificar el comprobante."));
  }, [token]);
  return <main className="verification-page"><Link className="brand-link" href="/">Control Propiedades</Link>{error ? <div className="verification-card invalid"><p className="eyebrow">No verificado</p><h1>Comprobante no encontrado</h1><p>{error}</p></div> : data ? <div className="verification-card"><span className="verification-seal">✓</span><p className="eyebrow">{data.verification_label}</p><h1>Comprobante válido</h1><p className="verification-intro">Este registro coincide con un comprobante emitido por la plataforma. Su hash permite comprobar que el PDF no fue alterado.</p><dl><div><dt>Folio</dt><dd>{data.folio}</dd></div><div><dt>Período</dt><dd>{data.period}</dd></div><div><dt>Propiedad</dt><dd>{data.property}, {data.commune}</dd></div><div><dt>Arrendatario</dt><dd>{data.tenant}</dd></div><div><dt>Monto</dt><dd>{money.format(Number(data.amount_minor))}</dd></div><div><dt>Emisión</dt><dd>{data.issued_at}</dd></div><div><dt>Código</dt><dd>{data.verification_code}</dd></div><div><dt>Estado</dt><dd>{data.status}</dd></div><div><dt>Plantilla</dt><dd>{data.template_version}</dd></div><div><dt>SHA-256 del PDF</dt><dd className="hash-value">{data.pdf_sha256}</dd></div></dl><small>Constancia privada de pago. No es un documento tributario ni modifica por sí sola el contrato. La información personal está parcialmente ocultada.</small></div> : <div className="verification-card"><p>Verificando comprobante…</p></div>}</main>;
}
