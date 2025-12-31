const CACHE_NAME = 'coded-pwa-cache-v2';

const urlsToCache = [
  '/',
  '/index.html',
  '/login.html',
  '/signup.html',
  '/live-requests.html',
  '/asset/logo-192.png',
  '/asset/logo-512.png'
];

self.addEventListener('install', event => {
  self.skipWaiting();
  event.waitUntil(
    caches.open(CACHE_NAME).then(cache => cache.addAll(urlsToCache))
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(
        keys.filter(key => key !== CACHE_NAME).map(key => caches.delete(key))
      )
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', event => {
  event.respondWith(
    caches.match(event.request).then(response => {
      return response || fetch(event.request).catch(() =>
        caches.match('/index.html')
      );
    })
  );
});

self.addEventListener('push', event => {
  const data = event.data?.json() || {};
  const title = data.title || 'Coded';
  const options = {
    body: data.body || 'You have a new update',
    icon: '/asset/logo-192.png',
    badge: '/asset/logo-192.png'
  };

  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

self.addEventListener('notificationclick', event => {
  event.notification.close();
  event.waitUntil(
    clients.openWindow('/live-requests.html')
  );
});
