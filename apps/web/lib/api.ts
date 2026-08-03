export class APIError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
  ) {
    super(message);
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const isForm = init.body instanceof FormData;
  const response = await fetch(`/api/v1${path}`, {
    ...init,
    credentials: "same-origin",
    headers: {
      ...(isForm ? {} : { "Content-Type": "application/json" }),
      ...init.headers,
    },
  });

  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new APIError(
      body?.error?.message ?? "Ocurrió un error inesperado.",
      response.status,
      body?.error?.code ?? "unknown_error",
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export type DirectDocumentUpload = {
  displayName?: string;
  documentType: string;
  tags?: string[];
  incidentID?: string;
};

export async function uploadDocumentDirect<T>(
  propertyID: string,
  file: File,
  metadata: DirectDocumentUpload,
): Promise<T> {
  const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
  const sha256 = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
  const prepared = await api<{
    upload: {
      id: string;
      mode: "single" | "multipart";
      upload_url?: string;
      headers?: Record<string, string>;
      parts?: Array<{ part_number: number; upload_url: string }>;
    };
  }>(`/properties/${propertyID}/document-uploads`, {
    method: "POST",
    body: JSON.stringify({
      original_name: file.name,
      display_name: metadata.displayName ?? "",
      document_type: metadata.documentType,
      mime_type: file.type,
      size_bytes: file.size,
      sha256,
      tags: metadata.tags ?? [],
      incident_id: metadata.incidentID ?? "",
    }),
  });

  let completion: { parts?: Array<{ part_number: number; etag: string }> } = {};
  if (prepared.upload.mode === "multipart") {
    const chunkSize = 5 * 1024 * 1024;
    const completedParts: Array<{ part_number: number; etag: string }> = [];
    for (const part of prepared.upload.parts ?? []) {
      const start = (part.part_number - 1) * chunkSize;
      const uploaded = await fetch(part.upload_url, {
        method: "PUT",
        body: file.slice(start, Math.min(start + chunkSize, file.size)),
      });
      const etag = uploaded.headers.get("ETag");
      if (!uploaded.ok || !etag) {
        throw new APIError("El almacenamiento rechazó una parte del archivo. Revisa la conexión e intenta nuevamente.", uploaded.status, "storage_upload_failed");
      }
      completedParts.push({ part_number: part.part_number, etag });
    }
    if (!completedParts.length) {
      throw new APIError("No fue posible preparar las partes del archivo.", 500, "storage_upload_failed");
    }
    completion = { parts: completedParts };
  } else {
    if (!prepared.upload.upload_url) {
      throw new APIError("No fue posible preparar la carga del archivo.", 500, "storage_upload_failed");
    }
    const uploaded = await fetch(prepared.upload.upload_url, {
      method: "PUT",
      headers: prepared.upload.headers,
      body: file,
    });
    if (!uploaded.ok) {
      throw new APIError("El almacenamiento rechazó el archivo. Revisa la conexión e intenta nuevamente.", uploaded.status, "storage_upload_failed");
    }
  }
  return api<T>(`/document-uploads/${prepared.upload.id}/complete`, { method: "POST", body: JSON.stringify(completion) });
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
