import type { Document, Property } from "@/lib/types";

const REVIEW_STATES = [
  { key: "verified", label: "Verificados", className: "chart-verified" },
  { key: "pending", label: "Por revisar", className: "chart-pending" },
  { key: "rejected", label: "Rechazados", className: "chart-rejected" },
] as const;

export function ComparisonCharts({
  properties,
  documents,
}: {
  properties: Property[];
  documents: Document[];
}) {
  const documentCountByProperty = new Map<string, number>();
  const reviewCount = { verified: 0, pending: 0, rejected: 0 };

  for (const document of documents) {
    documentCountByProperty.set(
      document.property_id,
      (documentCountByProperty.get(document.property_id) ?? 0) + 1,
    );

    if (document.status === "verified") reviewCount.verified += 1;
    else if (document.status === "rejected") reviewCount.rejected += 1;
    else reviewCount.pending += 1;
  }

  const propertyRows = properties
    .map((property) => ({
      id: property.id,
      label: property.name,
      value: documentCountByProperty.get(property.id) ?? 0,
    }))
    .sort((a, b) => b.value - a.value || a.label.localeCompare(b.label, "es"));
  const maximumDocuments = Math.max(1, ...propertyRows.map((row) => row.value));
  const maximumReviewCount = Math.max(1, ...Object.values(reviewCount));

  if (!properties.length) return null;

  return (
    <section className="content-section" aria-labelledby="comparison-title">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Comparación</p>
          <h2 id="comparison-title">Estado del portafolio</h2>
        </div>
      </div>
      <p className="section-intro">
        Contrasta el respaldo documental de cada propiedad y detecta rápidamente lo que requiere revisión.
      </p>
      <div className="comparison-chart-grid">
        <article className="chart-card">
          <header>
            <div>
              <h3>Documentos por propiedad</h3>
              <p>Volumen visible en el expediente</p>
            </div>
            <strong>{documents.length}</strong>
          </header>
          <div className="bar-chart" role="img" aria-label="Comparación de documentos por propiedad">
            {propertyRows.map((row) => (
              <div className="bar-row" key={row.id}>
                <span title={row.label}>{row.label}</span>
                <div className="bar-track" aria-hidden="true">
                  <span style={{ width: `${(row.value / maximumDocuments) * 100}%` }} />
                </div>
                <strong>{row.value}</strong>
              </div>
            ))}
          </div>
        </article>

        <article className="chart-card">
          <header>
            <div>
              <h3>Documentos por estado</h3>
              <p>Comparación del avance de revisión</p>
            </div>
            <strong>{reviewCount.verified}</strong>
          </header>
          <div className="bar-chart" role="img" aria-label="Comparación de documentos por estado de revisión">
            {REVIEW_STATES.map((state) => {
              const value = reviewCount[state.key];
              return (
                <div className="bar-row" key={state.key}>
                  <span>{state.label}</span>
                  <div className="bar-track" aria-hidden="true">
                    <span
                      className={state.className}
                      style={{ width: `${(value / maximumReviewCount) * 100}%` }}
                    />
                  </div>
                  <strong>{value}</strong>
                </div>
              );
            })}
          </div>
          <p className="chart-note">
            “Por revisar” agrupa documentos recién cargados y pendientes de validación.
          </p>
        </article>
      </div>
    </section>
  );
}
