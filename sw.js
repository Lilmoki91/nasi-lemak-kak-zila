// ==============================================
// 🍽️ ZILA FOOD - SERVICE WORKER v6
// ZERO CACHE 
// TAK SIMPAN APA-APA. BERSIH 100%.
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
    // Skip semua — guna network terus. Tak simpan cache.
    event.respondWith(fetch(event.request));
});

self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        self.skipWaiting();
    }
});
