// ==============================================
// 🍽️ ZILA FOOD - SERVICE WORKER v7
// CACHE STRATEGIC + AUTO CLEAN OLD
// ==============================================

const CACHE_NAME = 'zila-food-v7';

// 🔥 FAIL PENTING UNTUK DI-CACHE
const FILES_TO_CACHE = [
    './',
    './index.html',
    './manifest.json',
    './nasi-lemak-icon-192.png',
    './nasi-lemak-icon-512.png',
    'https://cdn.jsdelivr.net/npm/fontsource-poppins@5.0.0/index.css',
    'https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css',
    'https://i.postimg.cc/nhVCC9Pd/media.jpg',
    'https://i.postimg.cc/vZ2gCR74/media-(2).jpg',
    'https://i.postimg.cc/Pxf94qbN/media(3).png'
];

// ==============================================
// 📦 INSTALL - PRECACHE FAIL PENTING
// ==============================================
self.addEventListener('install', (event) => {
    console.log('🍽️ Zila Food SW v7: Install');
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            console.log('📦 Pre-caching essential files...');
            return cache.addAll(FILES_TO_CACHE).catch((err) => {
                console.warn('⚠️ Some files failed to pre-cache:', err);
            });
        })
    );
    self.skipWaiting();
});

// ==============================================
// 🚀 ACTIVATE - BUANG CACHE LAMA SAHAJA
// ==============================================
self.addEventListener('activate', (event) => {
    console.log('⚡ Zila Food SW v7: Activate');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.map((key) => {
                    // 🔥 HANYA BUANG CACHE YANG BUKAN VERSI SEMASA
                    if (key !== CACHE_NAME) {
                        console.log('🗑️ Old cache deleted:', key);
                        return caches.delete(key);
                    }
                })
            );
        }).then(() => {
            console.log('✅ Activation complete — cache v7 ready');
            return self.clients.claim();
        })
    );
});

// ==============================================
// 🔄 FETCH - CACHE FIRST, NETWORK FALLBACK
// ==============================================
self.addEventListener('fetch', (event) => {
    // Skip untuk API calls
    const url = event.request.url;
    if (url.includes('onrender.com') || url.includes('api/')) {
        return; // Biar network handle — jangan cache API
    }
    
    if (event.request.method !== 'GET') return;
    
    event.respondWith(
        caches.match(event.request).then((cachedResponse) => {
            // ✅ Guna cache dulu (loading laju)
            if (cachedResponse) {
                // Update cache di background
                fetch(event.request).then((networkResponse) => {
                    if (networkResponse && networkResponse.status === 200) {
                        caches.open(CACHE_NAME).then((cache) => {
                            cache.put(event.request, networkResponse.clone());
                        });
                    }
                }).catch(() => {});
                return cachedResponse;
            }
            
            // Tiada cache — cuba network
            return fetch(event.request).then((networkResponse) => {
                if (networkResponse && networkResponse.status === 200) {
                    const clone = networkResponse.clone();
                    caches.open(CACHE_NAME).then((cache) => {
                        cache.put(event.request, clone);
                    });
                }
                return networkResponse;
            }).catch(() => {
                // Offline fallback
                if (event.request.mode === 'navigate') {
                    return caches.match('./index.html');
                }
                return new Response('Offline', { status: 503 });
            });
        })
    );
});

// ==============================================
// ⚡ MESSAGE - SKIP WAITING
// ==============================================
self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        console.log('⚡ SW v7: Skip waiting');
        self.skipWaiting();
    }
});
