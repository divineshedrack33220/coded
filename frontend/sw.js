const CACHE_VERSION = 'v3';
const STATIC_CACHE = `coded-static-${CACHE_VERSION}`;
const RUNTIME_CACHE = `coded-runtime-${CACHE_VERSION}`;

const APP_SHELL = [
  '/',
  '/index.html',
  '/login.html',
  '/signup.html',
  '/live-requests.html',
  '/offline.html',
  '/manifest.json',
  '/asset/logo.png',
  '/asset/logo.png',
  '/logo.png'  // file in root
];

// INSTALL
self.addEventListener('install', event => {
  self.skipWaiting();
  event.waitUntil(
    caches.open(STATIC_CACHE).then(cache => {
      return Promise.all(
        APP_SHELL.map(url =>
          fetch(url)
            .then(res => {
              if (!res.ok) throw new Error(`${url} not found`);
              return cache.put(url, res);
            })
        )
      );
    })
  );
});

// ACTIVATE
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(
        keys.filter(key => !key.includes(CACHE_VERSION))
            .map(key => caches.delete(key))
      )
    )
  );
  self.clients.claim();
});

// FETCH
self.addEventListener('fetch', event => {
  const { request } = event;

  // HTML navigation → network first
  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request)
        .then(res => {
          const clone = res.clone();
          caches.open(RUNTIME_CACHE).then(c => c.put(request, clone));
          return res;
        })
        .catch(() => caches.match('/offline.html'))
    );
    return;
  }

  // Other assets → cache first
  event.respondWith(
    caches.match(request).then(cached => {
      if (cached) return cached;

      return fetch(request)
        .then(res => {
          caches.open(RUNTIME_CACHE).then(c => c.put(request, res.clone()));
          return res;
        })
        .catch(() => null);
    })
  );
});

// PUSH NOTIFICATIONS
self.addEventListener('push', event => {
  const data = event.data?.json() || {};
  event.waitUntil(
    self.registration.showNotification(data.title || 'Coded', {
      body: data.body || 'You have a new update',
      icon: 'asset/logo.png',
      badge: 'asset/logo.png',
      data: { url: '/live-requests.html' }
    })
  );
});

// NOTIFICATION CLICK
self.addEventListener('notificationclick', event => {
  event.notification.close();
  event.waitUntil(clients.openWindow(event.notification.data.url));
});
