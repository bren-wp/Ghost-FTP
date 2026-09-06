const CACHE_NAME = 'ghostftp-static-v0.1.1';
const STATIC_EXTENSIONS = /\.(?:css|js|svg|png|jpg|jpeg|webp|ico|woff2?)$/i;

self.addEventListener('install', (event) => {
    event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', (event) => {
    event.waitUntil((async () => {
        const keys = await caches.keys();
        await Promise.all(keys
            .filter((key) => (
                (key.startsWith('ghostftp-static-') && key !== CACHE_NAME)
                || key.startsWith('GhostFTP-static-')
            ))
            .map((key) => caches.delete(key)));
        await self.clients.claim();
    })());
});

self.addEventListener('fetch', (event) => {
    const request = event.request;
    if (request.method !== 'GET') return;

    const url = new URL(request.url);
    if (url.origin !== self.location.origin) return;

    // Sensitive/navigation/API/download responses are never stored offline.
    const sensitive = request.mode === 'navigate'
        || /\/(?:api|login|logout|register|account|users|settings|setup|diagnostics|download|preview)(?:\/|$)/.test(url.pathname);
    if (sensitive) {
        event.respondWith(fetch(request));
        return;
    }

    if (STATIC_EXTENSIONS.test(url.pathname) || url.pathname.endsWith('/manifest.webmanifest')) {
        event.respondWith((async () => {
            const cache = await caches.open(CACHE_NAME);
            const cached = await cache.match(request);
            if (cached) {
                event.waitUntil(fetch(request).then((response) => {
                    if (response.ok) cache.put(request, response.clone());
                }).catch(() => undefined));
                return cached;
            }
            const response = await fetch(request);
            if (response.ok) await cache.put(request, response.clone());
            return response;
        })());
        return;
    }

    event.respondWith(fetch(request));
});
