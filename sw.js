// ==============================================
// 🍽️ ZILA FOOD - SERVICE WORKER
// STABIL UNTUK IPHONE iOS & ANDROID
// ==============================================

const CACHE_NAME = 'zila-food-v2';

// Senarai fail wajib di-cache
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
// 📦 INSTALL - Pre-cache semua fail penting
// ==============================================
self.addEventListener('install', (event) => {
    console.log('🍽️ Zila Food SW: Install & Pre-cache');
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
// 🚀 ACTIVATE - Buang cache versi lama sahaja
// ==============================================
self.addEventListener('activate', (event) => {
    console.log('🍽️ Zila Food SW: Activate');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.map((key) => {
                    // HANYA buang cache yang BUKAN versi semasa
                    if (key !== CACHE_NAME) {
                        console.log('🗑️ Old cache deleted:', key);
                        return caches.delete(key);
                    }
                })
            );
        }).then(() => {
            console.log('✅ Activation complete');
            return self.clients.claim();
        })
    );
});

// ==============================================
// 🔄 FETCH - Cache first, network fallback
// PENTING untuk iPhone: guna cache dulu
// ==============================================
self.addEventListener('fetch', (event) => {
    // Skip untuk permintaan bukan GET
    if (event.request.method !== 'GET') return;

    event.respondWith(
        caches.match(event.request).then((cachedResponse) => {
            // Guna cache dulu (stabil untuk iPhone)
            if (cachedResponse) {
                // Update cache di background
                fetch(event.request).then((networkResponse) => {
                    if (networkResponse.status === 200) {
                        caches.open(CACHE_NAME).then((cache) => {
                            cache.put(event.request, networkResponse.clone());
                        });
                    }
                }).catch(() => {
                    // Network gagal — tak perlu buat apa, cache dah ada
                });
                return cachedResponse;
            }

            // Tiada cache — cuba network
            return fetch(event.request).then((networkResponse) => {
                if (networkResponse.status === 200) {
                    const clone = networkResponse.clone();
                    caches.open(CACHE_NAME).then((cache) => {
                        cache.put(event.request, clone);
                    });
                }
                return networkResponse;
            }).catch(() => {
                // Network & cache kedua-dua gagal
                // Return index.html untuk permintaan navigate
                if (event.request.mode === 'navigate') {
                    return caches.match('./index.html');
                }
                // Return response kosong untuk aset lain
                return new Response('Offline', { status: 503 });
            });
        })
    );
});

// ==============================================
// ⚡ MESSAGE - Skip waiting untuk update
// ==============================================
self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        console.log('⚡ Zila Food SW: Skip waiting');
        self.skipWaiting();
    }
});
