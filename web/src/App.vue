<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { store, setTheme, openEntry, currentVehicle } from './lib/store.js'
import { auth } from './lib/api.js'
import { vname } from './lib/format.js'
import Icon from './components/Icon.vue'
import Brand from './components/Brand.vue'
import EntryDrawer from './components/EntryDrawer.vue'
import AddVehicle from './components/AddVehicle.vue'

const route = useRoute()
const router = useRouter()
const nav = [
  { to: '/garage', label: 'My Garage', icon: 'garage', mobile: true },
  { to: '/overview', label: 'Overview', icon: 'overview', mobile: true },
  { to: '/log', label: 'Log', icon: 'log', mobile: true },
  { to: '/costs', label: 'Costs', icon: 'costs' },
  { to: '/tires', label: 'Tires', icon: 'tire' },
  { to: '/documents', label: 'Documents', icon: 'doc' },
  { to: '/reminders', label: 'Reminders', icon: 'bell', mobile: true },
]
const actions = [
  { kind: 'fillup', label: 'Add Fillup', icon: 'plus' },
  { kind: 'service', label: 'Add Service', icon: 'wrench' },
  { kind: 'expense', label: 'Add Expense', icon: 'receipt' },
]
const initials = computed(() => (store.user?.name || store.user?.email || '?').split(/\s+/).map((s) => s[0]).slice(0, 2).join('').toUpperCase())
const dueCount = computed(() => store.reminders.filter((r) => r.level > 0 && r.status !== 'snoozed').length)
const subtitle = computed(() => {
  const v = currentVehicle()
  if (['/overview', '/log', '/tires', '/documents'].includes(route.path) && v) return `${route.meta.title} · ${vname(v)}`
  return route.meta.title
})
function add(kind) {
  if (!store.vehicles.length) { store.wizard = true; return }
  openEntry(kind)
}
function logout() { auth.clear(); store.user = null; router.push('/login') }
</script>

<template>
  <RouterView v-if="route.meta.bare" />
  <div v-else class="shell">
    <nav class="side" aria-label="Primary">
      <RouterLink to="/settings" style="display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 6px 0 30px; color: var(--muted)">
        <span style="width: 72px; height: 72px; border-radius: 50%; border: 2px solid var(--accent); display: flex; align-items: center; justify-content: center; font-size: 24px; font-weight: 500; color: var(--text)">{{ initials }}</span>
        <span style="font-size: 14px">{{ store.user?.name || 'Account' }}</span>
      </RouterLink>
      <RouterLink v-for="n in nav" :key="n.to" :to="n.to" class="navbtn">
        <Icon :name="n.icon" />{{ n.label }}
        <span v-if="n.to === '/reminders' && dueCount" class="badge" style="margin-left: auto; background: var(--accent); color: var(--onAccent); padding: 2px 8px">{{ dueCount }}</span>
      </RouterLink>
      <div style="height: 1px; background: var(--line); margin: 14px 8px"></div>
      <button v-for="a in actions" :key="a.kind" class="navbtn" @click="add(a.kind)"><Icon :name="a.icon" />{{ a.label }}</button>
      <div style="flex-grow: 1"></div>
      <button class="navbtn" @click="setTheme(store.theme === 'dark' ? 'light' : 'dark')"><Icon :name="store.theme === 'dark' ? 'sun' : 'moon'" />{{ store.theme === 'dark' ? 'Light mode' : 'Dark mode' }}</button>
      <button class="navbtn" @click="logout"><Icon name="logout" />Sign out</button>
    </nav>

    <main class="main">
      <header class="between apphead" style="margin-bottom: 26px">
        <div class="row" style="gap: 18px">
          <RouterLink to="/garage" aria-label="Home" class="brandlink"><Brand :width="190" /></RouterLink>
          <span class="crumb" style="font-size: 13px; color: var(--faint)">/ {{ subtitle }}</span>
        </div>
        <div class="toplinks row" style="gap: 6px">
          <a class="navbtn" href="https://github.com/raccylabs/hammond" target="_blank" rel="noopener">Help</a>
          <RouterLink to="/settings" class="navbtn">Settings</RouterLink>
        </div>
        <div class="row bnav-head" style="gap: 4px">
          <button class="btn ghost icon mobile-only" :aria-label="store.theme === 'dark' ? 'Light mode' : 'Dark mode'" @click="setTheme(store.theme === 'dark' ? 'light' : 'dark')"><Icon :name="store.theme === 'dark' ? 'sun' : 'moon'" :size="20" /></button>
          <RouterLink to="/settings" class="btn ghost icon mobile-only" aria-label="Settings"><Icon name="settings" :size="20" /></RouterLink>
        </div>
      </header>
      <RouterView v-slot="{ Component }">
        <component :is="Component" :key="route.path" />
      </RouterView>
    </main>

    <nav class="bnav" aria-label="Primary">
      <RouterLink v-for="n in nav.filter((x) => x.mobile)" :key="n.to" :to="n.to" class="navbtn"><Icon :name="n.icon" />{{ n.label.replace('My ', '') }}</RouterLink>
      <button class="btn pri" aria-label="Add fillup" style="width: 52px; height: 52px; border-radius: 16px; padding: 0" @click="add('fillup')"><Icon name="add" :size="24" /></button>
    </nav>

    <EntryDrawer v-if="store.entry" />
    <AddVehicle v-if="store.wizard" />
  </div>
  <Teleport to="body">
    <div v-if="store.toast" :key="store.toast.msg" class="toast" :class="{ err: store.toast.err }" role="status">
      <Icon :name="store.toast.err ? 'x' : 'check'" :size="18" />{{ store.toast.msg }}
    </div>
  </Teleport>
</template>

<style>
.mobile-only { display: none !important; }
@media (max-width: 820px) {
  .mobile-only { display: inline-flex !important; }
  .crumb { display: none; }
  .apphead { margin-bottom: 18px !important; }
  .brandlink svg { width: 150px; height: auto; }
}
</style>
