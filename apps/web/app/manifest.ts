import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Control Propiedades",
    short_name: "Propiedades",
    description: "Administración segura de propiedades, arriendos y documentos.",
    start_url: "/panel",
    scope: "/",
    display: "standalone",
    background_color: "#f5f3ec",
    theme_color: "#246b4b",
    lang: "es-CL",
    icons: [
      {
        src: "/icons/control-propiedades-192.png",
        sizes: "192x192",
        type: "image/png",
        purpose: "any",
      },
      {
        src: "/icons/control-propiedades-512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "any",
      },
      {
        src: "/icons/control-propiedades-512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "maskable",
      },
    ],
  };
}
