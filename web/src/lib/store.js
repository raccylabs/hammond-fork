import { reactive } from 'vue'
import { get, auth } from './api.js'

function load(key, fallback) { try { return localStorage.getItem(key) || fallback } catch { return fallback } }
function save(key, v) { try { localStorage.setItem(key, v) } catch {} }

export const store = reactive({
  user: null,
  vehicles: [],
  reminders: [],
  vid: load('hm.vid', ''),
  theme: document.documentElement.dataset.theme || 'dark',
  toast: null,
  entry: null, // { kind, vehicleId, ... } opens the add-entry drawer
  wizard: false,
  dataVersion: 0, // bump to make views reload vehicle data
})

export function setTheme(t) {
  store.theme = t
  document.documentElement.dataset.theme = t
  document.querySelector('meta[name=theme-color]')?.setAttribute('content', t === 'light' ? '#ECE9E6' : '#1A1718')
  save('hm.theme', t)
}

export function selectVehicle(id) {
  store.vid = id
  save('hm.vid', id)
}

export function currentVehicle() {
  return store.vehicles.find((v) => v.id === store.vid) || store.vehicles[0] || null
}

let toastTimer
export function toast(msg, err = false) {
  store.toast = { msg, err }
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (store.toast = null), err ? 5000 : 2800)
}

export async function loadSession() {
  if (!auth.token) return false
  try {
    store.user = await get('/me')
    await refreshVehicles()
    return true
  } catch {
    return false
  }
}

export async function refreshVehicles() {
  store.vehicles = (await get('/me/vehicles')) || []
  if (!store.vehicles.find((v) => v.id === store.vid) && store.vehicles[0]) selectVehicle(store.vehicles[0].id)
  refreshReminders()
}

export async function refreshReminders() {
  try { store.reminders = (await get('/me/reminders')) || [] } catch { store.reminders = [] }
}

export function dataChanged() {
  store.dataVersion++
  refreshReminders()
}

export function openEntry(kind, opts = {}) {
  store.entry = { kind, vehicleId: opts.vehicleId || currentVehicle()?.id, ...opts }
}

export const isAdmin = () => store.user?.role === 0 || store.user?.roleDetail?.key === 'ADMIN'
