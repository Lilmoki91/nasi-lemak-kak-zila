// ==============================================
// 🍽️ NASI LEMAK KAK ZILA - SERVICE WORKER
// ==============================================

const CACHE_NAME = 'kak-zila-cache-v2';
const FILES_TO_CACHE = [
  './',
  './index.html',
  './manifest.json',
  './nasi-lemak-icon-192.png',
  './nasi-lemak-icon-512.png'
];

// ==============================================
// 📦 INSTALL
// ==============================================
self.addEventListener('install', event => {
  console.log('🍽️ Installing Kak Zila SW...');
  
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => {
        console.log('📦 Caching files...');
        return cache.addAll(FILES_TO_CACHE);
      })
      .then(() => {
        console.log('✅ Cache siap!');
        return self.skipWaiting();
      })
  );
});

// ==============================================
// 🚀 ACTIVATE
// ==============================================
self.addEventListener('activate', event => {
  console.log('⚡ Activating Kak Zila SW...');
  
  event.waitUntil(
    caches.keys().then(keys => {
      return Promise.all(
        keys.map(key => {
          if (key !== CACHE_NAME) {
            console.log(`🗑️ Deleting: ${key}`);
            return caches.delete(key);
          }
        })
      );
    }).then(() => {
      console.log('✅ Kak Zila SW sedia!');
      return self.clients.claim();
    })
  );
});

// ==============================================
// 🔄 FETCH
// ==============================================
self.addEventListener('fetch', event => {
  event.respondWith(
    caches.match(event.request)
      .then(cachedResponse => {
        if (cachedResponse) {
          return cachedResponse;
        }
        return fetch(event.request).then(response => {
          if (response.status === 200) {
            const responseClone = response.clone();
            caches.open(CACHE_NAME).then(cache => {
              cache.put(event.request, responseClone);
            });
          }
          return response;
        });
      })
  );
});

// ==============================================
// ⚡ SKIP WAITING MESSAGE
// ==============================================
self.addEventListener('message', event => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
