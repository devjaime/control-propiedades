"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { PwaInstallButton } from "./pwa-install-button";

export function AppHeader({ organization }: { organization: string }) {
  const router = useRouter();
  async function logout() {
    await api<void>("/auth/logout", { method: "POST", body: JSON.stringify({}) });
    router.replace("/ingresar");
  }
  return (
    <header className="app-header">
      <div className="header-navigation">
        <Link href="/panel" className="app-brand">Control Propiedades</Link>
        <nav aria-label="Navegación principal">
          <Link href="/panel">Portafolio</Link>
          <Link href="/panel#documentos">Documentos</Link>
          <Link href="/integraciones/agente">Agente MCP</Link>
          <Link href="/onboarding">Guía inicial</Link>
        </nav>
      </div>
      <div className="header-actions">
        <PwaInstallButton />
        <span>{organization}</span>
        <button className="text-button" onClick={logout} type="button">Salir</button>
      </div>
    </header>
  );
}
