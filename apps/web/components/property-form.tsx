"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { api } from "@/lib/api";
import type { Property } from "@/lib/types";

export function PropertyForm() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setLoading(true); setError("");
    const data = new FormData(event.currentTarget);
    try {
      const result = await api<{ property: Property }>("/properties", { method: "POST", body: JSON.stringify(Object.fromEntries(data.entries())) });
      router.replace(`/propiedades/${result.property.id}`);
    } catch (caught) { setError(caught instanceof Error ? caught.message : "No fue posible registrar la propiedad."); }
    finally { setLoading(false); }
  }
  return <main className="form-page"><Link className="back-link" href="/panel">← Volver al panel</Link><section className="form-card wide"><p className="eyebrow">Nueva propiedad</p><h1 className="form-title">Registra los datos esenciales</h1><p className="form-intro">Estos datos crearán el expediente donde organizarás todos sus documentos.</p><form className="stack-form two-columns" onSubmit={submit}>
    <label className="full-field">Nombre interno<input name="name" required placeholder="Ej. Casa Coquimbo" /></label>
    <label>Tipo<select name="property_type" required><option value="house">Casa</option><option value="apartment">Departamento</option><option value="land">Terreno</option><option value="commercial">Comercial</option><option value="other">Otro</option></select></label>
    <label>Uso<select name="usage" required><option value="primary_residence">Vivienda principal</option><option value="rental">Arrendada</option><option value="second_home">Segunda vivienda</option><option value="vacant">Vacante</option><option value="renovation">En remodelación</option><option value="for_sale">En venta</option><option value="acquisition">En adquisición</option></select></label>
    <label className="full-field">Dirección<input name="address_line" required /></label><label>Comuna<input name="commune" required /></label><label>Región<input name="region" required defaultValue="Coquimbo" /></label>
    {error ? <p className="form-error full-field" role="alert">{error}</p> : null}<button className="primary-button full-field" disabled={loading} type="submit">{loading ? "Guardando…" : "Registrar propiedad"}</button>
  </form></section></main>;
}
