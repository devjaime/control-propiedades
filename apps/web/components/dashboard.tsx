"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { APIError, api } from "@/lib/api";
import type { AgentToken, Document, Property, User } from "@/lib/types";
import { AppHeader } from "./app-header";
import { ComparisonCharts } from "./comparison-charts";
import { DocumentList } from "./document-list";

const propertyTypeLabel: Record<string, string> = { house: "Casa", apartment: "Departamento", land: "Terreno", commercial: "Comercial", other: "Otro" };
const usageLabel: Record<string, string> = { rental: "Arrendada", primary_residence: "Vivienda principal", second_home: "Segunda vivienda", vacant: "Vacante", renovation: "En remodelación", for_sale: "En venta", acquisition: "En adquisición" };

export function Dashboard() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [properties, setProperties] = useState<Property[]>([]);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [tokens, setTokens] = useState<AgentToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([
      api<{ user: User }>("/me"),
      api<{ properties: Property[] }>("/properties"),
      api<{ documents: Document[] }>("/documents"),
      api<{ tokens: AgentToken[] }>("/agent-tokens"),
    ]).then(([account, propertyData, documentData, tokenData]) => {
      setUser(account.user);
      setProperties(propertyData.properties);
      setDocuments(documentData.documents);
      setTokens(tokenData.tokens);
    }).catch((caught) => {
      if (caught instanceof APIError && caught.status === 401) { router.replace("/ingresar"); return; }
      setError(caught instanceof Error ? caught.message : "No fue posible cargar el panel.");
    }).finally(() => setLoading(false));
  }, [router]);

  async function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError("");
    const data = new FormData(event.currentTarget);
    const params = new URLSearchParams();
    for (const key of ["q", "property_id", "type", "status"]) {
      const value = String(data.get(key) ?? "").trim();
      if (value) params.set(key, value);
    }
    try {
      const result = await api<{ documents: Document[] }>(`/documents?${params}`);
      setDocuments(result.documents);
    } catch (caught) { setError(caught instanceof Error ? caught.message : "No fue posible buscar."); }
  }

  if (loading) return <main className="loading-screen">Preparando tu portafolio…</main>;
  if (!user) return <main className="loading-screen">{error || "Redirigiendo…"}</main>;

  const activeAgent = tokens.some((token) => token.status === "active");
  const pendingDocuments = documents.filter((item) => item.status === "uploaded" || item.status === "pending_review").length;
  const setupCompleted = [properties.length > 0, documents.length > 0, activeAgent].filter(Boolean).length;
  const documentCountByProperty = new Map<string, number>();
  for (const document of documents) documentCountByProperty.set(document.property_id, (documentCountByProperty.get(document.property_id) ?? 0) + 1);

  return <><AppHeader organization={user.organization} /><main className="app-main dashboard-page"><section className="dashboard-heading"><div><p className="eyebrow">Portafolio general</p><h1>Hola, {user.name.split(" ")[0]}</h1><p>Una vista para decidir dónde actuar; cada propiedad mantiene su propio expediente, arriendo, incidentes y portal compartido.</p></div><Link className="primary-button link-button" href="/propiedades/nueva">+ Nueva propiedad</Link></section>{error ? <p className="form-error" role="alert">{error}</p> : null}<section className="portfolio-overview"><div className="portfolio-metrics"><article><span>Propiedades</span><strong>{properties.length}</strong><small>{properties.filter((property) => property.usage === "rental").length} destinadas a arriendo</small></article><article><span>Archivo</span><strong>{documents.length}</strong><small>{pendingDocuments} requieren revisión</small></article><article><span>Agente MCP</span><strong>{activeAgent ? "Activo" : "Pendiente"}</strong><small>{activeAgent ? "Conectado a la nube" : "Aún sin credencial activa"}</small></article></div>{setupCompleted < 3 ? <aside className="setup-callout"><span>{setupCompleted}/3 preparado</span><h2>Termina la configuración inicial</h2><p>La guía te lleva desde la primera propiedad hasta la conexión de Hermes.</p><Link className="text-link" href="/onboarding">Continuar onboarding →</Link></aside> : <aside className="setup-callout is-ready"><span>Sistema operativo</span><h2>Tu base está configurada</h2><p>Puedes agregar otra propiedad o revisar la conexión del agente.</p><Link className="text-link" href="/integraciones/agente">Administrar agente →</Link></aside>}</section><section className="content-section portfolio-section"><div className="section-heading"><div><p className="eyebrow">Propiedades</p><h2>{properties.length ? "Elige un expediente" : "Comienza tu portafolio"}</h2></div><span className="section-context">{properties.length} {properties.length === 1 ? "propiedad" : "propiedades"}</span></div>{properties.length ? <div className="property-grid">{properties.map((property) => <Link className="property-card" key={property.id} href={`/propiedades/${property.id}`}><div className="property-card-top"><span className="property-type">{propertyTypeLabel[property.property_type] ?? property.property_type}</span><span className="property-use">{usageLabel[property.usage] ?? property.usage}</span></div><h3>{property.name}</h3><p>{property.address_line}</p><small>{property.commune}, {property.region}</small><div className="property-card-footer"><span>{documentCountByProperty.get(property.id) ?? 0} documentos</span><strong>Abrir propiedad →</strong></div></Link>)}</div> : <div className="empty-state actionable-empty"><h3>Registra tu primera propiedad</h3><p>Luego podrás configurar su arriendo, subir documentos y compartir seguimiento.</p><Link className="primary-button link-button" href="/propiedades/nueva">Registrar propiedad</Link></div>}</section>{properties.length ? <ComparisonCharts properties={properties} documents={documents} /> : null}<section className="content-section archive-section" id="documentos"><div className="section-heading"><div><p className="eyebrow">Archivo transversal</p><h2>Documentos de todo el portafolio</h2></div><span className="section-context">Filtra por propiedad</span></div><p className="section-intro">Usa esta vista sólo cuando necesites buscar entre varias propiedades. Para trabajar en un inmueble específico, entra primero a su expediente.</p><form className="search-bar" onSubmit={search}><input name="q" placeholder="Nombre, archivo o etiqueta" aria-label="Buscar documentos" /><select name="property_id" aria-label="Filtrar por propiedad"><option value="">Todas las propiedades</option>{properties.map((property) => <option value={property.id} key={property.id}>{property.name}</option>)}</select><select name="type" aria-label="Filtrar por tipo"><option value="">Todos los tipos</option><option value="deed">Escritura</option><option value="lease">Contrato</option><option value="bank_receipt">Comprobante bancario</option><option value="rent_receipt">Voucher de arriendo</option><option value="photo">Fotografía</option><option value="other">Otro</option></select><select name="status" aria-label="Filtrar por estado"><option value="">Todos los estados</option><option value="uploaded">Cargados</option><option value="pending_review">Pendientes</option><option value="verified">Verificados</option><option value="rejected">Rechazados</option></select><button className="secondary-button" type="submit">Aplicar filtros</button></form><DocumentList documents={documents} /></section></main></>;
}
