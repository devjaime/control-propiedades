"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { APIError, api } from "@/lib/api";
import type { AgentToken, Document, Property, User } from "@/lib/types";
import { AppHeader } from "./app-header";

type SetupData = {
  user: User;
  properties: Property[];
  documents: Document[];
  tokens: AgentToken[];
};

export function Onboarding() {
  const router = useRouter();
  const [data, setData] = useState<SetupData | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([
      api<{ user: User }>("/me"),
      api<{ properties: Property[] }>("/properties"),
      api<{ documents: Document[] }>("/documents"),
      api<{ tokens: AgentToken[] }>("/agent-tokens"),
    ]).then(([account, portfolio, archive, agentAccess]) => setData({
      user: account.user,
      properties: portfolio.properties,
      documents: archive.documents,
      tokens: agentAccess.tokens,
    })).catch((caught) => {
      if (caught instanceof APIError && caught.status === 401) {
        router.replace("/ingresar");
        return;
      }
      setError(caught instanceof Error ? caught.message : "No fue posible preparar la guía.");
    });
  }, [router]);

  if (!data) return <main className="loading-screen">{error || "Preparando tu guía…"}</main>;

  const hasProperty = data.properties.length > 0;
  const hasDocuments = data.documents.length > 0;
  const hasAgent = data.tokens.some((token) => token.status === "active");
  const completed = [true, hasProperty, hasDocuments, hasAgent].filter(Boolean).length;
  const firstProperty = data.properties[0];

  const steps = [
    { number: "01", title: "Cuenta y organización", description: `${data.user.organization} ya está preparada para separar sus propiedades y usuarios.`, done: true, href: "/panel", action: "Ver portafolio" },
    { number: "02", title: "Registra cada propiedad", description: "Crea una ficha independiente por inmueble. Los pagos, incidentes y documentos nunca se mezclan entre propiedades.", done: hasProperty, href: "/propiedades/nueva", action: hasProperty ? "Agregar otra propiedad" : "Registrar primera propiedad" },
    { number: "03", title: "Completa el expediente", description: "Carga contrato, inventario y respaldos. Si está arrendada, configura el contrato y genera el portal protegido del arrendatario.", done: hasDocuments, href: firstProperty ? `/propiedades/${firstProperty.id}` : "/propiedades/nueva", action: firstProperty ? "Abrir expediente" : "Primero crea la propiedad" },
    { number: "04", title: "Conecta tu agente", description: "Genera una credencial revocable y conecta Hermes para consultar propiedades, subir evidencias y preparar acciones con aprobación humana.", done: hasAgent, href: "/integraciones/agente", action: hasAgent ? "Administrar conexión" : "Conectar agente MCP" },
  ];

  return <><AppHeader organization={data.user.organization} /><main className="app-main onboarding-page"><section className="onboarding-hero"><div><p className="eyebrow">Guía inicial</p><h1>Deja tu portafolio operativo</h1><p>Cuatro pasos para pasar desde una cuenta vacía a un expediente ordenado, verificable y conectado con tu agente.</p></div><div className="setup-progress" aria-label={`${completed} de 4 pasos completados`}><strong>{completed}/4</strong><span>completados</span><div><i style={{ width: `${completed * 25}%` }} /></div></div></section><ol className="onboarding-steps">{steps.map((step) => <li className={step.done ? "onboarding-step is-complete" : "onboarding-step"} key={step.number}><span className="step-number">{step.done ? "✓" : step.number}</span><div><h2>{step.title}</h2><p>{step.description}</p></div><Link className={step.done ? "secondary-button link-button" : "primary-button link-button"} href={step.href}>{step.action}</Link></li>)}</ol><section className="onboarding-finish"><div><p className="eyebrow">Puedes avanzar por etapas</p><h2>La plataforma crece con tu portafolio</h2><p>No necesitas completar todo hoy. Cada nueva propiedad conserva su propio historial y puede tener un portal compartido diferente.</p></div><Link className="secondary-button link-button" href="/panel">Ir al panel general</Link></section></main></>;
}
