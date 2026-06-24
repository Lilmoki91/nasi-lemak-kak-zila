// ==============================================
// 🍽️ ZILA FOOD - SERVICE WORKER
// AUTO KILL CACHE LAMA + AUTO UPDATE
// ==============================================

const CACHE_NAME = 'zila-food-v1';

self.addEventListener('install', (event) => {
    console.log('🍽️ Zila Food SW: Install');
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    console.log('🍽️ Zila Food SW: Activate - Killing old caches');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.map((key) => {
                    console.log('🗑️ Deleted:', key);
                    return caches.delete(key);
                })
            );
        }).then(() => {
            console.log('✅ All cache deleted');
            return self.clients.claim();
        })
    );
});

self.addEventListener('fetch', (event) => {
    event.respondWith(
        fetch(event.request)
            .then((response) => {
                const clone = response.clone();
                caches.open(CACHE_NAME).then((cache) => {
                    cache.put(event.request, clone);
                });
                return response;
            })
            .catch(() => {
                return caches.match(event.request);
            })
    );
});

self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        console.log('⚡ Zila Food SW: Skip waiting');
        self.skipWaiting();
    }
});
