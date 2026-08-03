const CACHE_NAME = "control-propiedades-static-v1";
const PRECACHE_URLS = [
  "/icons/control-propiedades.svg",
  "/icons/control-propiedades-192.png",
  "/icons/control-propiedades-512.png",
  "/icons/apple-touch-icon.png",
];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(PRECACHE_URLS)));
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  const url = new URL(request.url);

  // Los HTML autenticados, documentos y respuestas de API nunca se guardan en el dispositivo.
  if (
    request.method !== "GET" ||
    url.origin !== self.location.origin ||
    (!url.pathname.startsWith("/_next/static/") && !url.pathname.startsWith("/icons/"))
  ) {
    return;
  }

  event.respondWith(
    caches.match(request).then((cached) => {
      const networkResponse = fetch(request).then((response) => {
        if (response.ok) {
          const copy = response.clone();
          caches.open(CACHE_NAME).then((cache) => cache.put(request, copy));
        }
        return response;
      });
      return cached ?? networkResponse;
    }),
  );
});
