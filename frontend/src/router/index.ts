// Composables
import { createRouter, createWebHistory } from 'vue-router'
import Login from '@/views/Login.vue'
import Data from '@/store/modules/data'
import Ws from '@/store/ws'
import { getBaseUrl } from '@/plugins/base-url'

const routes = [
  {
    path: '/login',
    name: 'pages.login',
    component: Login,
  },
  {
    path: '/',
    component: () => import('@/layouts/AuthenticatedHost.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '/',
        name: 'pages.home',
        component: () => import('@/views/Home.vue'),
      },
      {
        path: '/inbounds',
        name: 'pages.inbounds',
        component: () => import('@/views/Inbounds.vue'),
      },
      {
        path: '/clients',
        name: 'pages.clients',
        component: () => import('@/views/Clients.vue'),
      },
      {
        path: '/outbounds',
        name: 'pages.outbounds',
        component: () => import('@/views/Outbounds.vue'),
      },
      {
        path: '/services',
        name: 'pages.services',
        component: () => import('@/views/Services.vue'),
      },
      {
        path: '/endpoints',
        name: 'pages.endpoints',
        component: () => import('@/views/Endpoints.vue'),
      },
      {
        path: '/rules',
        name: 'pages.rules',
        component: () => import('@/views/Rules.vue'),
      },
      {
        path: '/tls',
        name: 'pages.tls',
        component: () => import('@/views/Tls.vue'),
      },
      {
        path: '/basics',
        redirect: '/settings?tab=basics',
      },
      {
        path: '/dns',
        name: 'pages.dns',
        component: () => import('@/views/Dns.vue'),
      },
      {
        path: '/admins',
        name: 'pages.admins',
        component: () => import('@/views/Admins.vue'),
      },
      {
        path: '/telegram',
        name: 'pages.telegram',
        component: () => import('@/views/TelegramSettings.vue'),
      },
      {
        path: '/audit',
        name: 'pages.audit',
        component: () => import('@/views/Audit.vue'),
      },
      {
        path: '/migrate-xui',
        name: 'pages.migrateXui',
        component: () => import('@/views/MigrateXui.vue'),
      },
      {
        path: '/paid-subscriptions',
        name: 'pages.paidSub',
        component: () => import('@/views/paidsub/PaidSubscriptions.vue'),
      },
      {
        path: '/settings',
        name: 'pages.settings',
        component: () => import('@/views/Settings.vue'),
      },
      {
        path: '/donations',
        name: 'pages.donations',
        component: () => import('@/views/Donations.vue'),
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(getBaseUrl()),
  routes,
})

// After a panel upgrade, the browser tab still holds the previous
// index.html with hashed chunk names. The first dynamic import (e.g.
// navigating to /clients) requests a chunk that no longer exists in the
// embedded FS and the request fails. Without this guard the page just
// stops navigating and shows the broken Clients tab. Reloading once
// fetches the new index.html with the new hashes. We use sessionStorage
// to make sure we never enter an infinite reload loop.
const reloadKey = 'sui:preload-error-reload'
const reloadOnce = () => {
  try {
    if (sessionStorage.getItem(reloadKey) === '1') return
    sessionStorage.setItem(reloadKey, '1')
  } catch {
    // sessionStorage may be disabled (private mode); fall through.
  }
  window.location.reload()
}
const isPreloadError = (err: any): boolean => {
  if (!err) return false
  const msg: string = err?.message ?? String(err)
  return /Failed to fetch dynamically imported module/i.test(msg) ||
    /Importing a module script failed/i.test(msg) ||
    /Failed to load module script/i.test(msg) ||
    err?.name === 'ChunkLoadError'
}
window.addEventListener('vite:preloadError', () => reloadOnce())
router.onError((err) => {
  if (isPreloadError(err)) reloadOnce()
})
router.afterEach(() => {
  try {
    sessionStorage.removeItem(reloadKey)
  } catch {
    // ignore
  }
})

let intervalId: any

// The session cookie is HttpOnly (set by api/session.go) so it cannot be
// observed from the client; auth is enforced server-side. Every API call
// returns `Invalid login` when the cookie is missing or expired, and
// httputil._handleMsg redirects the user to /login in that case. The router
// guard below only handles the UX detail of pulling fresh data on first
// navigation to a protected page and stopping the polling timer when we
// land on /login.
router.beforeEach((to) => {
  if (to.path !== '/login') {
    loadDataInterval()
    Ws().connect()
  } else if (intervalId) {
    clearInterval(intervalId)
    intervalId = undefined
    Ws().disconnect()
  }
})

const loadDataInterval = () => {
  if (intervalId) return
  Data().loadData()
  intervalId = setInterval(() => {
    Data().loadData()
  }, 10000)
}

export default router
