const CACHE_NAME = "ceerat-web-ui-v2";
const APP_SHELL = [
  "/offline",
  "/static/app.css",
  "/static/app.js",
  "/static/icon.svg",
  "/manifest.json"
];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL)));
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)))
    )
  );
  self.clients.claim();
});

self.addEventListener("fetch", (event) => {
  const request = event.request;

  if (request.mode === "navigate") {
    event.respondWith(fetch(request).catch(() => caches.match("/offline")));
    return;
  }

  if (request.method === "GET" && new URL(request.url).origin === self.location.origin) {
    event.respondWith(caches.match(request).then((cached) => cached || fetch(request)));
  }
});
