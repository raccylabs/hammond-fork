import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import './styles.css'
import App from './App.vue'
import { store, loadSession } from './lib/store.js'
import { auth, get, onUnauthorized } from './lib/api.js'

const routes = [
  { path: '/', redirect: '/garage' },
  { path: '/login', component: () => import('./views/Login.vue'), meta: { bare: true, anon: true } },
  { path: '/setup', component: () => import('./views/Setup.vue'), meta: { bare: true, anon: true } },
  { path: '/garage', component: () => import('./views/Garage.vue'), meta: { title: 'My Garage' } },
  { path: '/overview', component: () => import('./views/Overview.vue'), meta: { title: 'Overview' } },
  { path: '/log', component: () => import('./views/Log.vue'), meta: { title: 'Log' } },
  { path: '/costs', component: () => import('./views/Costs.vue'), meta: { title: 'Cost of ownership' } },
  { path: '/tires', component: () => import('./views/Tires.vue'), meta: { title: 'Tires' } },
  { path: '/documents', component: () => import('./views/Documents.vue'), meta: { title: 'Documents' } },
  { path: '/reminders', component: () => import('./views/Reminders.vue'), meta: { title: 'Reminders' } },
  { path: '/settings', component: () => import('./views/Settings.vue'), meta: { title: 'Settings' } },
  { path: '/vehicles/:id/edit', component: () => import('./views/VehicleEdit.vue'), meta: { title: 'Edit vehicle' } },
  { path: '/report/:id', component: () => import('./views/Report.vue'), meta: { bare: true, title: 'Service history' } },
  { path: '/:pathMatch(.*)*', redirect: '/garage' },
]

const router = createRouter({ history: createWebHistory(), routes, scrollBehavior: () => ({ top: 0 }) })

let checkedStatus = false
let initialized = true
router.beforeEach(async (to) => {
  if (!checkedStatus) {
    try { initialized = (await get('/system/status')).initialized } catch { initialized = true }
    checkedStatus = true
  }
  if (!initialized && to.path !== '/setup') return '/setup'
  if (initialized && to.path === '/setup') return '/login'
  if (to.meta.anon) return true
  if (!store.user) {
    const ok = await loadSession()
    if (!ok) return { path: '/login', query: { next: to.fullPath } }
  }
  return true
})
router.afterEach((to) => { document.title = to.meta.title ? `${to.meta.title} · Hammond` : 'Hammond' })

onUnauthorized.handler = () => {
  store.user = null
  if (router.currentRoute.value.path !== '/login') router.push('/login')
}
export const markInitialized = () => { initialized = true }

createApp(App).use(router).mount('#app')
void auth
