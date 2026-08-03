"use client";

import { useEffect, useState } from "react";
import { APIError, api } from "@/lib/api";
import type { TenantPortalLink } from "@/lib/types";

const accessKindLabels: Record<string, string> = {
  owner: "Propietario",
  tenant: "Arrendatario",
  legacy_key: "Clave anterior",
};

export function TenantPortalManager({ propertyID }: { propertyID: string }) {
  const [link, setLink] = useState<TenantPortalLink | null>(null);
  const [newKey, setNewKey] = useState("");
  const [error, setError] = useState("");
  const [working, setWorking] = useState(false);

  useEffect(() => {
    api<{ link: TenantPortalLink | null }>(`/properties/${propertyID}/tenant-portal`)
      .then((data) => setLink(data.link))
      .catch((caught) => setError(caught instanceof Error ? caught.message : "No fue posible consultar el enlace."));
  }, [propertyID]);

  async function rotate() {
    if (link && !window.confirm("La clave anterior dejará de funcionar. ¿Crear un nuevo acceso?")) return;
    setWorking(true); setError(""); setNewKey("");
    try {
      const data = await api<{ link: TenantPortalLink }>(`/properties/${propertyID}/tenant-portal`, { method: "POST", body: "{}" });
      setLink(data.link);
      setNewKey(data.link.access_key ?? "");
    } catch (caught) {
      setError(caught instanceof APIError ? caught.message : "No fue posible crear el acceso.");
    } finally { setWorking(false); }
  }

  async function copy(value: string) {
    await navigator.clipboard.writeText(value);
  }

  return <section className="content-section tenant-portal-admin" id="seguimiento-arrendatario">
    <div className="section-heading"><div><p className="eyebrow">Portal compartido</p><h2>Pagos y seguimiento del arrendatario</h2></div></div>
    <p className="section-intro">Un mismo enlace protegido permite al propietario y al arrendatario consultar comprobantes vigentes, vouchers de arriendo y avances marcados como públicos. Las notas internas y las rutas privadas de almacenamiento nunca se exponen.</p>
    {error ? <p className="form-error" role="alert">{error}</p> : null}
    {link ? <div className="share-access-card">
      <label>Enlace directo<div className="copy-row"><input readOnly value={link.url} /><button className="secondary-button" onClick={() => copy(link.url)} type="button">Copiar</button></div></label>
      {newKey ? <div className="one-time-key"><strong>Clave nueva: <code>{newKey}</code></strong><p>Guárdala ahora: por seguridad no volverá a mostrarse.</p><button className="secondary-button" onClick={() => copy(newKey)} type="button">Copiar clave</button></div> : <p className="muted-copy">La clave actual está protegida y no puede recuperarse. Puedes rotarla si se perdió.</p>}
      <button className="secondary-button" disabled={working} onClick={rotate} type="button">{working ? "Generando…" : "Rotar enlace y clave"}</button>
      <div className="portal-access-history">
        <h3>Accesos recientes</h3>
        <p className="muted-copy">Registro interno de consultas exitosas. No almacena el RUT ni el código utilizado.</p>
        {link.access_events.length ? <ol>{link.access_events.map((event) => <li key={event.id}><strong>{accessKindLabels[event.credential_kind] ?? event.credential_kind}</strong><time dateTime={event.accessed_at}>{new Date(event.accessed_at).toLocaleString("es-CL")}</time></li>)}</ol> : <p className="empty-state">Todavía no se registran consultas.</p>}
      </div>
    </div> : <button className="primary-button" disabled={working} onClick={rotate} type="button">{working ? "Generando…" : "Crear enlace protegido"}</button>}
  </section>;
}
