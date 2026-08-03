"use client";

import { useEffect, useId, useState } from "react";

type FilePreviewProps = {
  src: string;
  name: string;
  mimeType?: string | null;
  className?: string;
  label?: string;
};

function isImageFile(name: string, mimeType?: string | null) {
  return Boolean(
    mimeType?.startsWith("image/") ||
      /\.(avif|gif|heic|heif|jpe?g|png|svg|webp)$/i.test(name),
  );
}

function isPdfFile(name: string, mimeType?: string | null) {
  return mimeType === "application/pdf" || /\.pdf$/i.test(name);
}

export function FilePreview({
  src,
  name,
  mimeType,
  className = "",
  label = "Vista previa",
}: FilePreviewProps) {
  const [isOpen, setIsOpen] = useState(false);
  const titleId = useId();
  const image = isImageFile(name, mimeType);
  const pdf = isPdfFile(name, mimeType);

  useEffect(() => {
    if (!isOpen) return;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setIsOpen(false);
    };

    document.addEventListener("keydown", onKeyDown);
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  return (
    <>
      <button
        type="button"
        className={className}
        onClick={() => setIsOpen(true)}
        aria-haspopup="dialog"
      >
        {label}
      </button>

      {isOpen ? (
        <div
          className="fixed inset-0 z-[100] flex items-center justify-center bg-black/70 p-3 sm:p-6"
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          onMouseDown={(event) => {
            if (event.currentTarget === event.target) setIsOpen(false);
          }}
        >
          <section className="flex h-full max-h-[92vh] w-full max-w-6xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl">
            <header className="flex items-center justify-between gap-4 border-b px-4 py-3 sm:px-6">
              <h2 id={titleId} className="truncate text-base font-semibold text-slate-900">
                {name}
              </h2>
              <div className="flex shrink-0 items-center gap-2">
                <a
                  href={src}
                  target="_blank"
                  rel="noreferrer"
                  className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
                >
                  Abrir aparte
                </a>
                <button
                  type="button"
                  autoFocus
                  onClick={() => setIsOpen(false)}
                  className="rounded-lg bg-slate-900 px-3 py-2 text-sm font-medium text-white hover:bg-slate-700"
                  aria-label="Cerrar vista previa"
                >
                  Cerrar
                </button>
              </div>
            </header>

            <div className="min-h-0 flex-1 bg-slate-100">
              {image ? (
                // A presigned object-storage URL cannot be known by Next at build time.
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={src}
                  alt={name}
                  className="h-full w-full object-contain"
                />
              ) : pdf ? (
                <iframe
                  src={src}
                  title={`Vista previa de ${name}`}
                  className="h-full min-h-[70vh] w-full border-0"
                />
              ) : (
                <div className="flex h-full min-h-72 flex-col items-center justify-center gap-4 p-8 text-center">
                  <p className="max-w-md text-slate-600">
                    Este formato no admite vista previa en el navegador.
                  </p>
                  <a
                    href={src}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white"
                  >
                    Abrir documento
                  </a>
                </div>
              )}
            </div>
          </section>
        </div>
      ) : null}
    </>
  );
}
