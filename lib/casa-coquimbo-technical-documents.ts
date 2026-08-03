export const CASA_COQUIMBO_POSTVENTA_URL =
  "https://www.pvi.cl/propietarios/postventagpr/propiedades";

export type CasaCoquimboTechnicalDocument = {
  title: string;
  category: string;
  specialty: string;
  sheet?: string;
  date?: string;
  fileName: string;
  url: string;
  tags: string[];
};

const baseUrl = "/uploads/casa-coquimbo/documentos-tecnicos";

export const CASA_COQUIMBO_TECHNICAL_DOCUMENTS: CasaCoquimboTechnicalDocument[] = [
  {
    title: "Especificaciones técnicas DOCA 63",
    category: "Especificaciones técnicas",
    specialty: "General",
    date: "2025-05-13",
    fileName: "EETT-DOCA-63-2025-05-13.pdf",
    url: `${baseUrl}/general/EETT-DOCA-63-2025-05-13.pdf`,
    tags: ["DOCA 63", "EETT", "especificaciones", "vivienda 63"],
  },
  {
    title: "Manual de acceso web GPR 2025",
    category: "Postventa",
    specialty: "Manual de acceso",
    date: "2025",
    fileName: "Manual-Acceso-Web-GPR-2025.pdf",
    url: `${baseUrl}/postventa/Manual-Acceso-Web-GPR-2025.pdf`,
    tags: ["GPR", "postventa", "manual", "portal propietarios"],
  },
  ...[
    ["L1", "Arquitectura", "Arquitectura"],
    ["L2", "Arquitectura", "Arquitectura"],
    ["L3", "Agua potable fría", "AP-FRIA"],
    ["L4", "Agua potable fría", "AP-FRIA"],
    ["L5", "Agua potable caliente", "AP-CALIENTE"],
    ["L6", "Agua potable caliente", "AP-CALIENTE"],
    ["L7", "Alcantarillado", "ALCANTARILLADO"],
    ["L8", "Alcantarillado", "ALCANTARILLADO"],
    ["L9", "Electricidad - alumbrado", "ALUMBRADO"],
    ["L10", "Electricidad - alumbrado", "ALUMBRADO"],
    ["L11", "Electricidad - enchufes", "ENCHUFES"],
    ["L12", "Electricidad - enchufes", "ENCHUFES"],
    ["L13", "Telecomunicaciones", "TELECO"],
    ["L14", "Telecomunicaciones", "TELECO"],
    ["L15", "Gas", "GAS"],
  ].map(([sheet, specialty, slug]) => ({
    title: `Plano ${sheet} de 15 · ${specialty}`,
    category: "Planos técnicos",
    specialty,
    sheet,
    fileName: `DOCA-63-${sheet}-${slug}.pdf`,
    url: `${baseUrl}/planos/DOCA-63-${sheet}-${slug}.pdf`,
    tags: ["DOCA 63", "plano", sheet, specialty.toLowerCase()],
  })),
];
