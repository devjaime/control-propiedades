import type { Metadata } from "next";
import "./styles.css";

export const metadata: Metadata = {
  title: "Control Propiedades",
  description: "Administración segura de propiedades y documentos",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="es">
      <body>{children}</body>
    </html>
  );
}
