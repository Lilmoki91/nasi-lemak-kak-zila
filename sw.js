const CACHE_NAME = 'kak-zila-v3';
const ASSETS = [
    '/',
    '/index.html',
    '/manifest.json',
    '/nasi-lemak-icon-192.png',
    '/nasi-lemak-icon-512.png'
];

// Install
self.addEventListener('install', (event) => {
    console.log('✅ SW: Install');
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            console.log('✅ SW: Caching assets');
            return cache.addAll(ASSETS).catch((err) => {
                console.log('❌ SW: Cache failed', err);
            });
        })
    );
    self.skipWaiting();
});

// Activate
self.addEventListener('activate', (event) => {
    console.log('✅ SW: Activate');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.filter((key) => key !== CACHE_NAME)
                    .map((key) => {
                        console.log('🗑️ SW: Deleting old cache', key);
                        return caches.delete(key);
                    })
            );
        })
    );
    self.clients.claim();
});

// Fetch
self.addEventListener('fetch', (event) => {
    event.respondWith(
        caches.match(event.request).then((cached) => {
            if (cached) {
                return cached;
            }
            return fetch(event.request).then((response) => {
                if (!response || response.status !== 200 || response.type !== 'basic') {
                    return response;
                }
                const clone = response.clone();
                caches.open(CACHE_NAME).then((cache) => {
                    cache.put(event.request, clone);
                });
                return response;
            }).catch(() => {
                if (event.request.mode === 'navigate') {
                    return caches.match('/index.html');
                }
            });
        })
    );
});
