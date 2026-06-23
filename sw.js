const CACHE_NAME = 'kak-zila-v2';
const ASSETS = [
    '/',
    '/index.html',
    '/manifest.json',
    '/nasi-lemak-icon-192.png',
    '/nasi-lemak-icon-512.png'
];

// Install - simpan aset dalam cache
self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            return cache.addAll(ASSETS);
        })
    );
    self.skipWaiting();
});

// Activate - buang cache lama
self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.filter((key) => key !== CACHE_NAME)
                    .map((key) => caches.delete(key))
            );
        })
    );
    self.clients.claim();
});

// Fetch - guna cache dulu, lepas tu network
self.addEventListener('fetch', (event) => {
    event.respondWith(
        caches.match(event.request).then((cached) => {
            return cached || fetch(event.request).then((response) => {
                const clone = response.clone();
                caches.open(CACHE_NAME).then((cache) => {
                    cache.put(event.request, clone);
                });
                return response;
            });
        }).catch(() => {
            return caches.match('/index.html');
        })
    );
});
