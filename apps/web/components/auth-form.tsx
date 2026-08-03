"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { api } from "@/lib/api";

type AuthFormProps = { mode: "register" | "login" };

export function AuthForm({ mode }: AuthFormProps) {
  const router = useRouter();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);
    const data = new FormData(event.currentTarget);
    const payload = Object.fromEntries(data.entries());
    try {
      await api(`/auth/${mode === "register" ? "register" : "login"}`, {
        method: "POST",
        body: JSON.stringify(payload),
      });
      router.replace(registering ? "/onboarding" : "/panel");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "No fue posible continuar.");
    } finally {
      setLoading(false);
    }
  }

  const registering = mode === "register";
  return (
    <main className="auth-shell">
      <Link className="brand-link" href="/">Control Propiedades</Link>
      <section className="form-card">
        <p className="eyebrow">{registering ? "Crear cuenta" : "Bienvenido de vuelta"}</p>
        <h1 className="form-title">{registering ? "Comienza tu expediente digital" : "Ingresa a tu cuenta"}</h1>
        <p className="form-intro">
          {registering
            ? "Crea tu organización personal. Después podrás registrar propiedades y ordenar sus documentos."
            : "Continúa administrando tus propiedades y documentos."}
        </p>
        <form onSubmit={submit} className="stack-form">
          {registering ? (
            <>
              <label>Tu nombre<input name="name" autoComplete="name" required minLength={2} /></label>
              <label>Nombre de la organización<input name="organization_name" required minLength={2} placeholder="Ej. Patrimonio familiar" /></label>
            </>
          ) : null}
          <label>Correo electrónico<input name="email" type="email" autoComplete="email" required /></label>
          <label>Contraseña<input name="password" type="password" autoComplete={registering ? "new-password" : "current-password"} required minLength={10} /></label>
          {error ? <p className="form-error" role="alert">{error}</p> : null}
          <button className="primary-button" disabled={loading} type="submit">
            {loading ? "Procesando…" : registering ? "Crear cuenta" : "Ingresar"}
          </button>
        </form>
        <p className="form-switch">
          {registering ? "¿Ya tienes una cuenta?" : "¿Todavía no tienes cuenta?"}{" "}
          <Link href={registering ? "/ingresar" : "/registro"}>{registering ? "Ingresar" : "Registrarme"}</Link>
        </p>
      </section>
    </main>
  );
}
