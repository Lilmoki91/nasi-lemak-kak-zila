// ==============================================
// 🍽️ ZILA FOOD - SERVICE WORKER v4
// GABUNGAN: Nasi Lemak Kak Zila + Azura AI
// STABIL UNTUK iPhone iOS & ANDROID
// AUTO PADAM CACHE LAMA + PRECACHE + CACHE FIRST
// ==============================================

const CACHE_NAME = 'zila-food-v4'; // 🔥 Tukar version bila deploy perubahan besar

// Senarai fail wajib di-cache (precache)
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
    console.log('🍽️ Zila Food SW v4: Install & Pre-cache');
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then((cache) => {
                console.log('📦 Pre-caching essential files...');
                return cache.addAll(FILES_TO_CACHE);
            })
            .then(() => {
                console.log('✅ Pre-cache siap!');
                return self.skipWaiting(); // Aktifkan segera
            })
            .catch((err) => {
                console.warn('⚠️ Some files failed to pre-cache:', err);
            })
    );
});

// ==============================================
// 🚀 ACTIVATE - BUANG CACHE VERSI LAMA SAHAJA
// ==============================================
self.addEventListener('activate', (event) => {
    console.log('⚡ Zila Food SW v4: Activate - Cleaning old caches');
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.map((key) => {
                    // 🔥 HANYA buang cache yang BUKAN versi semasa
                    if (key !== CACHE_NAME) {
                        console.log('🗑️ Old cache deleted:', key);
                        return caches.delete(key);
                    }
                })
            );
        }).then(() => {
            console.log('✅ Zila Food SW v4 sedia!');
            // 🔥 Ambil alih semua tabs
            return self.clients.claim();
        })
    );
});

// ==============================================
// 🔄 FETCH - Cache First, Network Fallback
// PENTING: Stabil untuk iPhone & Android
// Skip cache untuk API calls
// ==============================================
self.addEventListener('fetch', (event) => {
    const url = event.request.url;

    // 🔥 JANGAN cache API calls (untuk future use)
    if (url.includes('googleapis.com') ||
        url.includes('workers.dev') ||
        url.includes('gateway.pinata.cloud') ||
        url.includes('api.')) {
        return; // Biar browser handle biasa
    }

    // Skip untuk permintaan bukan GET
    if (event.request.method !== 'GET') return;

    event.respondWith(
        caches.match(event.request).then((cachedResponse) => {
            // ✅ Guna cache dulu (stabil untuk iPhone & offline)
            if (cachedResponse) {
                // Update cache di background (background sync)
                fetch(event.request).then((networkResponse) => {
                    if (networkResponse && networkResponse.status === 200) {
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
                // Cache response untuk guna kemudian
                if (networkResponse && networkResponse.status === 200) {
                    const responseClone = networkResponse.clone();
                    caches.open(CACHE_NAME).then((cache) => {
                        cache.put(event.request, responseClone);
                    });
                }
                return networkResponse;
            }).catch(() => {
                // Network & cache kedua-dua gagal
                // Return index.html untuk permintaan navigate (SPA fallback)
                if (event.request.mode === 'navigate') {
                    return caches.match('./index.html');
                }
                // Return response kosong untuk aset lain
                return new Response('Offline - Sila sambung internet', { status: 503 });
            });
        })
    );
});

// ==============================================
// ⚡ MESSAGE - Skip waiting untuk update manual
// ==============================================
self.addEventListener('message', (event) => {
    if (event.data === 'SKIP_WAITING') {
        console.log('⚡ Zila Food SW: Skip waiting');
        self.skipWaiting();
    }
});
