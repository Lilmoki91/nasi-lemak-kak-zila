// ==============================================
// 🍽️ ZILA FOOD - SERVICE WORKER v6
// ZERO CACHE - ZERO STORAGE - BERSIH 100%
// ==============================================

self.addEventListener('install', () => {
    console.log('🍽️ Zila Food SW v6: Install');
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    console.log('⚡ Zila Food SW v6: Activate - KILL ALL');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.map((key) => {
                    console.log('🗑️ Cache deleted:', key);
                    return caches.delete(key);
                })
            );
        }).then(() => {
            console.log('✅ All cache deleted');
            return self.clients.claim();
        })
    );
});

// ZERO cache — terus network, tak simpan apa
self.addEventListener('fetch', (event) => {
    event.respondWith(fetch(event.request));
});

self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        self.skipWaiting();
    }
});
