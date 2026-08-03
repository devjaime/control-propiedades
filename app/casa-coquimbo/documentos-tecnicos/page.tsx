import Link from "next/link";
import {
  CASA_COQUIMBO_POSTVENTA_URL,
  CASA_COQUIMBO_TECHNICAL_DOCUMENTS,
} from "@/lib/casa-coquimbo-technical-documents";

export const metadata = {
  title: "Documentos técnicos · Casa Coquimbo",
  description: "Planos, especificaciones técnicas y acceso de postventa de Casa Coquimbo.",
};

export default function CasaCoquimboTechnicalDocumentsPage() {
  const groups = Map.groupBy(
    CASA_COQUIMBO_TECHNICAL_DOCUMENTS,
    (document) => document.category,
  );

  return (
    <main className="mx-auto min-h-screen max-w-7xl px-6 py-10 lg:px-10">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-5">
        <div>
          <p className="mb-2 text-sm font-semibold uppercase tracking-[0.2em] text-emerald-700">
            Casa Coquimbo · Expediente técnico
          </p>
          <h1 className="text-4xl font-semibold tracking-tight text-slate-950 lg:text-5xl">
            Documentos técnicos
          </h1>
          <p className="mt-3 max-w-3xl text-base text-slate-600">
            {CASA_COQUIMBO_TECHNICAL_DOCUMENTS.length} documentos clasificados por especialidad,
            disponibles para consulta y descarga.
          </p>
        </div>
        <Link
          href={CASA_COQUIMBO_POSTVENTA_URL}
          target="_blank"
          rel="noreferrer"
          className="rounded-full bg-emerald-800 px-5 py-3 text-sm font-semibold text-white transition hover:bg-emerald-900"
        >
          Abrir portal de postventa
        </Link>
      </div>

      <section className="mb-8 rounded-3xl border border-emerald-200 bg-emerald-50 p-6">
        <h2 className="text-xl font-semibold text-emerald-950">Postventa GPR</h2>
        <p className="mt-2 text-sm leading-6 text-emerald-900">
          Acceso directo al portal del propietario. El manual de acceso web está archivado en la
          categoría Postventa de este expediente.
        </p>
      </section>

      <div className="space-y-10">
        {[...groups.entries()].map(([category, documents]) => (
          <section key={category}>
            <div className="mb-4 flex items-center justify-between gap-3">
              <h2 className="text-2xl font-semibold text-slate-950">{category}</h2>
              <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-600">
                {documents.length} documentos
              </span>
            </div>
            <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
              {documents.map((document) => (
                <article
                  key={document.url}
                  className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
                >
                  <div className="aspect-[4/3] bg-slate-100">
                    <iframe
                      src={`${document.url}#page=1&view=FitH`}
                      title={`Vista previa de ${document.title}`}
                      loading="lazy"
                      className="h-full w-full border-0"
                    />
                  </div>
                  <div className="p-5">
                    <p className="text-xs font-semibold uppercase tracking-wider text-emerald-700">
                      {document.specialty}
                    </p>
                    <h3 className="mt-2 text-lg font-semibold text-slate-950">{document.title}</h3>
                    <p className="mt-2 line-clamp-1 text-xs text-slate-500">{document.fileName}</p>
                    <div className="mt-4 flex gap-3">
                      <Link
                        href={document.url}
                        target="_blank"
                        className="rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white"
                      >
                        Ver PDF
                      </Link>
                      <a
                        href={document.url}
                        download={document.fileName}
                        className="rounded-full border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700"
                      >
                        Descargar
                      </a>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </section>
        ))}
      </div>
    </main>
  );
}
