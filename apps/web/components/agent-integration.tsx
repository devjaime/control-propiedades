"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { APIError, api } from "@/lib/api";
import type { AgentToken, User } from "@/lib/types";
import { AppHeader } from "./app-header";

const apiURL = "https://control-propiedades-api.vercel.app";
const installerURL = "https://control-propiedades-web.vercel.app/downloads/install-control-propiedades-mcp.sh";

export function AgentIntegration() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [tokens, setTokens] = useState<AgentToken[]>([]);
  const [newToken, setNewToken] = useState("");
  const [error, setError] = useState("");
  const [working, setWorking] = useState(false);

  useEffect(() => {
    Promise.all([api<{ user: User }>("/me"), api<{ tokens: AgentToken[] }>("/agent-tokens")])
      .then(([account, tokenData]) => { setUser(account.user); setTokens(tokenData.tokens); })
      .catch((caught) => {
        if (caught instanceof APIError && caught.status === 401) { router.replace("/ingresar"); return; }
        setError(caught instanceof Error ? caught.message : "No fue posible cargar la integración.");
      });
  }, [router]);

  async function createToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setWorking(true); setError(""); setNewToken("");
    const form = event.currentTarget;
    const name = String(new FormData(form).get("name") ?? "").trim();
    try {
      const result = await api<{ token: AgentToken }>("/agent-tokens", { method: "POST", body: JSON.stringify({ name }) });
      setTokens((current) => [result.token, ...current]);
      setNewToken(result.token.token ?? "");
      form.reset();
    } catch (caught) { setError(caught instanceof Error ? caught.message : "No fue posible crear la credencial."); }
    finally { setWorking(false); }
  }

  async function revoke(token: AgentToken) {
    if (!window.confirm(`¿Revocar el acceso ${token.name}? Hermes dejará de conectarse con esta credencial.`)) return;
    setWorking(true); setError("");
    try {
      await api<void>(`/agent-tokens/${token.id}/revoke`, { method: "POST", body: "{}" });
      setTokens((current) => current.map((item) => item.id === token.id ? { ...item, status: "revoked" } : item));
    } catch (caught) { setError(caught instanceof Error ? caught.message : "No fue posible revocar la credencial."); }
    finally { setWorking(false); }
  }

  async function copy(value: string) { await navigator.clipboard.writeText(value); }

  if (!user) return <main className="loading-screen">{error || "Cargando integración…"}</main>;
  const envBlock = newToken ? `CONTROL_PROPIEDADES_API_URL=${apiURL}\nCONTROL_PROPIEDADES_API_TOKEN=${newToken}` : "Genera una credencial para obtener estas variables.";
  const installCommand = `curl -fsSL ${installerURL} | sh\nhermes mcp add control-propiedades --command $HOME/.local/bin/control-propiedades-mcp\nhermes mcp test control-propiedades`;

  return <><AppHeader organization={user.organization} /><main className="app-main integration-page"><section className="integration-hero"><div><p className="eyebrow">Integraciones · MCP</p><h1>Tu agente, conectado al control real</h1><p>El servidor MCP se ejecuta en tu equipo junto a Hermes y opera sobre esta versión en la nube. Las consultas son de sólo lectura; toda escritura sensible exige confirmación explícita.</p></div><span className="integration-status"><i /> API en la nube disponible</span></section>{error ? <p className="form-error" role="alert">{error}</p> : null}<section className="integration-flow" aria-label="Flujo de conexión"><article><span>1</span><strong>Hermes local</strong><p>Recibe tu instrucción y solicita confirmación cuando corresponde.</p></article><article><span>2</span><strong>Servidor MCP</strong><p>Valida argumentos y limita las capacidades del agente.</p></article><article><span>3</span><strong>API en Vercel</strong><p>Autentica la credencial, registra actividad y aplica reglas.</p></article><article><span>4</span><strong>Neon + Supabase</strong><p>Conservan registros y archivos privados de cada propiedad.</p></article></section><div className="integration-layout"><section className="integration-card"><p className="eyebrow">Paso 1</p><h2>Crear credencial</h2><p>Vigencia de seis meses. Se mostrará completa una sola vez; después sólo verás su prefijo.</p><form className="stack-form" onSubmit={createToken}><label>Nombre del acceso<input name="name" defaultValue="Hermes Agent local" minLength={3} maxLength={120} required /></label><button className="primary-button" disabled={working} type="submit">{working ? "Generando…" : "Generar credencial"}</button></form>{newToken ? <div className="secret-reveal"><strong>Guárdala ahora</strong><code>{newToken}</code><button className="secondary-button" onClick={() => copy(newToken)} type="button">Copiar credencial</button></div> : null}</section><section className="integration-card"><p className="eyebrow">Paso 2</p><h2>Instalar y configurar Hermes</h2><p>Guarda primero las variables copiadas en <code>~/.config/control-propiedades/agent.env</code>. Luego ejecuta el instalador y registra el servidor local.</p><div className="code-block"><code>{envBlock}</code><button onClick={() => copy(envBlock)} type="button">Copiar</button></div><div className="code-block"><code>{installCommand}</code><button onClick={() => copy(installCommand)} type="button">Copiar</button></div><div className="integration-actions"><a className="text-link" download href="/downloads/control-propiedades-mcp.mjs">Descargar servidor MCP</a><a className="text-link" download href="/downloads/install-control-propiedades-mcp.sh">Descargar instalador</a><Link className="text-link" href="/onboarding">Volver a la guía</Link></div></section></div><section className="integration-card token-management"><div className="section-heading"><div><p className="eyebrow">Seguridad</p><h2>Credenciales de agente</h2></div></div>{tokens.length ? <div className="token-list">{tokens.map((token) => <article key={token.id}><div><strong>{token.name}</strong><span><code>{token.prefix}…</code> · {token.status === "active" ? "Activo" : "Revocado"}</span><small>{token.last_used_at ? `Último uso: ${new Date(token.last_used_at).toLocaleString("es-CL")}` : "Todavía no utilizado"}{token.expires_at ? ` · vence ${new Date(token.expires_at).toLocaleDateString("es-CL")}` : ""}</small></div>{token.status === "active" ? <button className="text-button danger-text" disabled={working} onClick={() => void revoke(token)} type="button">Revocar</button> : null}</article>)}</div> : <p className="empty-state">No hay credenciales creadas.</p>}</section><section className="integration-capabilities"><div><p className="eyebrow">Herramientas disponibles</p><h2>Qué puede hacer el agente</h2></div><ul><li><strong>Consultar</strong><span>Propiedades, cobros, incidentes y accionables.</span></li><li><strong>Preservar evidencia</strong><span>Subir PDF o imágenes al expediente privado.</span></li><li><strong>Registrar pagos</strong><span>Cargar comprobantes sin conciliarlos automáticamente.</span></li><li><strong>Ejecutar con aprobación</strong><span>Actualizar tareas o confirmar pagos sólo después de tu autorización.</span></li></ul></section></main></>;
}
