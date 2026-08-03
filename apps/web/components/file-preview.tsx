"use client";

import { useEffect, useMemo, useState } from "react";

type FilePreviewProps = {
  src: string;
  name: string;
  mimeType?: string | null;
  className?: string;
  label?: string;
};

function getPreviewKind(src: string, mimeType?: string | null) {
  const normalizedMime = mimeType?.toLowerCase() ?? "";
  const cleanSrc = src.split("?")[0]?.toLowerCase() ?? "";

  if (normalizedMime.startsWith("image/") || /\.(avif|gif|jpe?g|png|webp)$/.test(cleanSrc)) {
    return "image";
  }

  if (normalizedMime === "application/pdf" || cleanSrc.endsWith(".pdf")) {
    return "pdf";
  }

  return "document";
}

export function FilePreview({
  src,
  name,
  mimeType,
  className,
  label = "Vista previa",
}: FilePreviewProps) {
  const [isOpen, setIsOpen] = useState(false);
  const kind = useMemo(() => getPreviewKind(src, mimeType), [src, mimeType]);

  useEffect(() => {
    if (!isOpen) return;

    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setIsOpen(false);
    };

    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  return (
    <>
      <button
        type="button"
        onClick={() => setIsOpen(true)}
        className={
          className ??
          "inline-flex min-h-10 items-center justify-center rounded-full border border-emerald-800 px-4 py-2 text-sm font-semibold text-emerald-950 transition hover:bg-emerald-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-800"
        }
      >
        {label}
      </button>

      {isOpen ? (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center bg-black/75 p-3 sm:p-6"
          role="dialog"
          aria-modal="true"
          aria-labelledby="file-preview-title"
          onMouseDown={(event) => {
            if (event.currentTarget === event.target) setIsOpen(false);
          }}
        >
          <div className="flex h-[min(92vh,960px)] w-full max-w-6xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl">
            <header className="flex items-center justify-between gap-4 border-b px-4 py-3 sm:px-6">
              <h2 id="file-preview-title" className="truncate text-base font-semibold text-slate-900">
                {name}
              </h2>
              <button
                type="button"
                onClick={() => setIsOpen(false)}
                className="rounded-full border px-3 py-1.5 text-sm font-semibold text-slate-700 hover:bg-slate-100"
                aria-label="Cerrar vista previa"
              >
                Cerrar
              </button>
            </header>

            <div className="min-h-0 flex-1 bg-slate-100">
              {kind === "image" ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={src} alt={name} className="h-full w-full object-contain" />
              ) : (
                <iframe
                  src={src}
                  title={`Vista previa de ${name}`}
                  className="h-full w-full border-0 bg-white"
                />
              )}
            </div>

            <footer className="flex justify-end border-t px-4 py-3 sm:px-6">
              <a
                href={src}
                target="_blank"
                rel="noreferrer"
                className="rounded-full bg-emerald-900 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-800"
              >
                Abrir en otra pestaña
              </a>
            </footer>
          </div>
        </div>
      ) : null}
    </>
  );
}
