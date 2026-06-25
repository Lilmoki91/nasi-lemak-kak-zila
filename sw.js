// ==============================================
// 🍽️ ZILA FOOD - SERVICE WORKER v5
// AUTO PADAM CACHE & COOKIES
// ==============================================

const CACHE_NAME = 'zila-food-v5';

const FILES_TO_CACHE = [
    './',
    './index.html',
    './manifest.json',
    './nasi-lemak-icon-192.png',
    './nasi-lemak-icon-512.png',
    'https://cdn.jsdelivr.net/npm/fontsource-poppins@5.0.0/index.css',
    'https://i.postimg.cc/nhVCC9Pd/media.jpg',
    'https://i.postimg.cc/vZ2gCR74/media-(2).jpg',
    'https://i.postimg.cc/Pxf94qbN/media(3).png'
];

// ==============================================
// 📦 INSTALL
// ==============================================
self.addEventListener('install', (event) => {
    console.log('🍽️ Zila Food SW v5: Install');
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then((cache) => {
                console.log('📦 Pre-caching...');
                return cache.addAll(FILES_TO_CACHE);
            })
            .then(() => {
                console.log('✅ Cache siap!');
                return self.skipWaiting();
            })
            .catch((err) => {
                console.warn('⚠️ Pre-cache partial fail:', err);
            })
    );
});

// ==============================================
// 🚀 ACTIVATE - 🔥 BUNUH SEMUA CACHE LAMA
// ==============================================
self.addEventListener('activate', (event) => {
    console.log('⚡ Zila Food SW v5: Activate - KILL ALL OLD CACHE');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.map((key) => {
                    // 🔥 BUNUH SEMUA — termasuk yang bukan versi semasa
                    console.log('🗑️ Deleted:', key);
                    return caches.delete(key);
                })
            );
        }).then(() => {
            console.log('✅ All cache deleted, fresh start!');
            return self.clients.claim();
        })
    );
});

// ==============================================
// 🔄 FETCH - Network first, cache fallback
// ==============================================
self.addEventListener('fetch', (event) => {
    const url = event.request.url;

    // Skip API calls
    if (url.includes('googleapis.com') ||
        url.includes('workers.dev') ||
        url.includes('gateway.pinata.cloud') ||
        url.includes('api.')) {
        return;
    }

    if (event.request.method !== 'GET') return;

    event.respondWith(
        caches.match(event.request).then((cachedResponse) => {
            if (cachedResponse) {
                return cachedResponse;
            }
            return fetch(event.request).then((response) => {
                if (response.status === 200) {
                    const clone = response.clone();
                    caches.open(CACHE_NAME).then((cache) => {
                        cache.put(event.request, clone);
                    });
                }
                return response;
            }).catch(() => {
                if (event.request.mode === 'navigate') {
                    return caches.match('./index.html');
                }
                return new Response('Offline', { status: 503 });
            });
        })
    );
});

// ==============================================
// ⚡ MESSAGE
// ==============================================
self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        console.log('⚡ Zila Food SW: Skip waiting');
        self.skipWaiting();
    }
});
