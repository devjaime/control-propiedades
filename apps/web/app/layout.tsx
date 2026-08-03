import type { Metadata } from "next";
import "./styles.css";

export const metadata: Metadata = {
  title: "Control Propiedades | Arriendos, evidencia y agentes",
  description: "Organiza propiedades, arriendos, pagos, incidentes y evidencia verificable. Conecta tu agente mediante MCP sin perder la aprobación humana.",
  applicationName: "Control Propiedades",
  manifest: "/manifest.webmanifest",
  appleWebApp: {
    capable: true,
    statusBarStyle: "default",
    title: "Propiedades",
  },
  icons: {
    icon: "/icons/control-propiedades.svg",
    apple: "/icons/apple-touch-icon.png",
  },
};

export const viewport = {
  themeColor: "#246b4b",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="es">
      <body>{children}</body>
    </html>
  );
}
