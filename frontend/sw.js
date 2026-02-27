// sw.js - Instagram-like PWA Service Worker
const APP_NAME = 'InstaPing';
const VERSION = '1.0.0';
const CACHE_NAME = `instaping-${VERSION}`;
const CACHE_TTL = 10 * 60 * 1000; // 10 minutes in milliseconds
let lastCacheClear = Date.now();

// Essential files to cache (no HTML to prevent caching issues)
const CORE_ASSETS = [
  '/',
  '/index.html',
  '/asset/logo.jpeg',
  '/asset/logo.png',
  '/manifest.json'
];

// Clear ALL caches (including current one)
const clearAllCaches = async () => {
  console.log('[SW] Clearing all caches...');
  const keys = await caches.keys();
  return Promise.all(keys.map(key => caches.delete(key)));
};

// Install - Cache core assets
self.addEventListener('install', event => {
  console.log(`[SW] ${APP_NAME} v${VERSION} installing...`);
  lastCacheClear = Date.now();
  
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => {
        console.log('[SW] Caching core assets');
        return cache.addAll(CORE_ASSETS);
      })
      .then(() => {
        console.log('[SW] Installation complete');
        return self.skipWaiting();
      })
  );
});

// Activate - Clear old caches and claim clients
self.addEventListener('activate', event => {
  console.log(`[SW] ${APP_NAME} v${VERSION} activating...`);
  
  event.waitUntil(
    Promise.all([
      clearAllCaches(),
      self.clients.claim()
    ]).then(() => {
      console.log('[SW] Activation complete');
      lastCacheClear = Date.now();
      
      // Start periodic cache clearing
      startCacheClearTimer();
      
      // Notify all clients about update
      self.clients.matchAll().then(clients => {
        clients.forEach(client => {
          client.postMessage({
            type: 'SW_ACTIVATED',
            version: VERSION,
            cacheCleared: true,
            timestamp: Date.now()
          });
        });
      });
    })
  );
});

// Cache clearing timer function
const startCacheClearTimer = () => {
  // Clear cache immediately if TTL already expired (during service worker update)
  const now = Date.now();
  if (now - lastCacheClear >= CACHE_TTL) {
    clearAllCaches().then(() => {
      lastCacheClear = now;
      console.log(`[SW] Initial cache clear at ${new Date().toISOString()}`);
      
      // Notify clients
      notifyClientsCacheCleared();
    });
  }
  
  // Set up interval for periodic cache clearing
  setInterval(() => {
    clearAllCaches().then(() => {
      lastCacheClear = Date.now();
      console.log(`[SW] Periodic cache clear at ${new Date().toISOString()}`);
      
      // Notify clients
      notifyClientsCacheCleared();
    });
  }, CACHE_TTL);
};

// Notify all clients that cache was cleared
const notifyClientsCacheCleared = () => {
  self.clients.matchAll().then(clients => {
    clients.forEach(client => {
      client.postMessage({
        type: 'CACHE_CLEARED',
        timestamp: Date.now(),
        nextClear: Date.now() + CACHE_TTL
      });
    });
  });
};

// Network-first strategy for API calls, cache-first for assets
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);
  
  // Check if cache should be cleared based on TTL
  const now = Date.now();
  if (now - lastCacheClear >= CACHE_TTL) {
    clearAllCaches().then(() => {
      lastCacheClear = now;
      console.log(`[SW] Cache cleared on fetch at ${new Date().toISOString()}`);
      notifyClientsCacheCleared();
    });
  }
  
  // API calls - Network only, no caching
  if (url.pathname.startsWith('/api/') || 
      url.pathname === '/ws' ||
      url.search.includes('timestamp=') ||
      url.search.includes('_t=')) {
    console.log(`[SW] Network-only: ${url.pathname}`);
    event.respondWith(
      fetch(event.request, { 
        cache: 'no-store',
        credentials: 'include'
      })
    );
    return;
  }
  
  // Static assets - Cache first, then network
  if (url.pathname.startsWith('/asset/') ||
      url.pathname.includes('.css') ||
      url.pathname.includes('.js') ||
      url.pathname.includes('.jpeg') ||
      url.pathname.includes('.png')) {
    event.respondWith(
      caches.match(event.request)
        .then(cachedResponse => {
          // Return cached version immediately
          if (cachedResponse) {
            console.log(`[SW] Cache hit: ${url.pathname}`);
            return cachedResponse;
          }
          
          // Fetch from network and cache
          console.log(`[SW] Cache miss, fetching: ${url.pathname}`);
          return fetch(event.request)
            .then(response => {
              // Don't cache if not successful
              if (!response.ok) return response;
              
              // Clone response to cache and return
              const responseToCache = response.clone();
              caches.open(CACHE_NAME)
                .then(cache => {
                  cache.put(event.request, responseToCache);
                });
              
              return response;
            })
            .catch(error => {
              console.error('[SW] Fetch failed:', error);
              // Return a fallback for images
              if (url.pathname.includes('.jpeg') || url.pathname.includes('.png')) {
                return new Response(
                  '<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 24 24"><path fill="#666" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5 0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29 1.94-3.5 3.22-6 3.22z"/></svg>',
                  { headers: { 'Content-Type': 'image/svg+xml' } }
                );
              }
              throw error;
            });
        })
    );
    return;
  }
  
  // HTML pages - Network first, then cache as fallback
  if (url.pathname.endsWith('.html') || 
      url.pathname === '/' ||
      !url.pathname.includes('.')) {
    event.respondWith(
      fetch(event.request, { 
        cache: 'no-store',
        credentials: 'include'
      })
      .then(response => {
        // Don't cache HTML pages
        return response;
      })
      .catch(error => {
        console.log(`[SW] Network failed for ${url.pathname}, trying cache`);
        return caches.match(event.request)
          .then(cachedResponse => {
            if (cachedResponse) {
              console.log(`[SW] Serving ${url.pathname} from cache`);
              return cachedResponse;
            }
            
            // If nothing in cache, show offline page
            return caches.match('/offline.html')
              .then(offlineResponse => offlineResponse || 
                new Response('Network error. Please check your connection.', {
                  status: 503,
                  headers: { 'Content-Type': 'text/plain' }
                })
              );
          });
      })
    );
    return;
  }
  
  // Default - Network only
  event.respondWith(fetch(event.request, { cache: 'no-store' }));
});

// Handle messages from clients
self.addEventListener('message', event => {
  console.log('[SW] Message received:', event.data);
  
  if (event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
  
  if (event.data.type === 'CLEAR_CACHE_NOW') {
    clearAllCaches().then(() => {
      lastCacheClear = Date.now();
      event.ports?.[0]?.postMessage({ 
        success: true,
        timestamp: lastCacheClear 
      });
      notifyClientsCacheCleared();
    });
  }
  
  if (event.data.type === 'GET_CACHE_INFO') {
    event.ports?.[0]?.postMessage({ 
      version: VERSION,
      lastCacheClear: lastCacheClear,
      nextCacheClear: lastCacheClear + CACHE_TTL,
      ttlMinutes: CACHE_TTL / 60000
    });
  }
  
  if (event.data.type === 'FORCE_RELOAD') {
    self.clients.matchAll().then(clients => {
      clients.forEach(client => {
        client.postMessage({
          type: 'FORCE_RELOAD_PAGE',
          timestamp: Date.now()
        });
      });
    });
  }
});

// Background sync for offline actions
self.addEventListener('sync', event => {
  console.log('[SW] Background sync:', event.tag);
  
  if (event.tag === 'sync-messages') {
    event.waitUntil(syncMessages());
  }
  
  if (event.tag === 'sync-posts') {
    event.waitUntil(syncPosts());
  }
});

// Push notifications (Instagram-style)
self.addEventListener('push', event => {
  console.log('[SW] Push notification received:', event);
  
  const options = {
    body: event.data ? event.data.text() : 'New update on InstaPing!',
    icon: 'asset/logo.jpeg',
    badge: 'asset/badge.png',
    vibrate: [200, 100, 200],
    data: {
      dateOfArrival: Date.now(),
      primaryKey: 1
    },
    actions: [
      {
        action: 'open',
        title: 'Open app'
      },
      {
        action: 'close',
        title: 'Dismiss'
      }
    ]
  };
  
  event.waitUntil(
    self.registration.showNotification('InstaPing', options)
  );
});

self.addEventListener('notificationclick', event => {
  console.log('[SW] Notification click:', event.notification.tag);
  event.notification.close();
  
  if (event.action === 'open') {
    event.waitUntil(
      clients.matchAll({ type: 'window' })
        .then(clientList => {
          for (const client of clientList) {
            if (client.url === '/' && 'focus' in client) {
              return client.focus();
            }
          }
          if (clients.openWindow) {
            return clients.openWindow('/');
          }
        })
    );
  }
});

// Sync functions (simplified)
async function syncMessages() {
  // This would sync unsent messages
  console.log('[SW] Syncing messages...');
}

async function syncPosts() {
  // This would sync unsent posts
  console.log('[SW] Syncing posts...');
}

// Periodic sync (if browser supports it)
if ('periodicSync' in self.registration) {
  self.addEventListener('periodicsync', event => {
    if (event.tag === 'update-feed') {
      console.log('[SW] Periodic sync: Updating feed');
      // Update feed in background
    }
  });
}

// Start the cache clearing timer when service worker loads
startCacheClearTimer();