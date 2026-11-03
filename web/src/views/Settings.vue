<script setup>
import { ref, reactive, onMounted } from 'vue'
import { store, setTheme, toast, isAdmin, loadSession, dataChanged } from '../lib/store.js'
import { get, post, upload } from '../lib/api.js'
import { vname } from '../lib/format.js'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'

const me = reactive({ currency: store.user?.currency || 'USD', distanceUnit: store.user?.distanceUnit ?? 0, dateFormat: store.user?.dateFormat || 'MM/dd/yyyy' })
async function saveMe() {
  try { await post('/me/settings', { ...me, currency: me.currency.toUpperCase() }); await loadSession(); toast('Preferences saved') } catch (x) { toast(x.message, true) }
}
const pw = reactive({ oldPassword: '', newPassword: '' })
async function changePw() {
  if (pw.newPassword.length < 8) { toast('Use at least 8 characters', true); return }
  try { await post('/changePassword', pw); pw.oldPassword = pw.newPassword = ''; toast('Password changed') } catch (x) { toast(x.message, true) }
}

const users = ref([])
const invite = reactive({ open: false, name: '', email: '', password: '', role: 1 })
async function loadUsers() { if (isAdmin()) users.value = (await get('/users')) || [] }
onMounted(loadUsers)
async function addUser() {
  try {
    await post('/register', { name: invite.name, email: invite.email.trim().toLowerCase(), password: invite.password, role: Number(invite.role), currency: me.currency, distanceUnit: me.distanceUnit })
    invite.open = false; toast('Account created. Share the password with them.'); loadUsers()
  } catch (x) { toast(x.message, true) }
}
async function toggleUser(u) {
  try { await post(`/users/${u.id}/${u.isDisabled ? 'enable' : 'disable'}`); loadUsers() } catch (x) { toast(x.message, true) }
}

const imp = reactive({ source: 'fuelly', vehicleId: store.vehicles[0]?.id || '', file: null, location: true, busy: false })
async function runImport() {
  if (!imp.file) { toast('Choose a CSV file first', true); return }
  const fd = new FormData()
  fd.append('file', imp.file)
  if (imp.source === 'drivvo') { fd.append('vehicleID', imp.vehicleId); fd.append('importLocation', String(imp.location)) }
  imp.busy = true
  try { await upload(`/import/${imp.source}`, fd); toast('Import finished'); dataChanged() } catch (x) { toast('Import failed: ' + x.message, true) } finally { imp.busy = false }
}
</script>

<template>
  <section aria-label="Settings" class="g12">
    <div class="card c6 stack" style="gap: 20px">
      <h2 class="ctitle">Appearance &amp; units</h2>
      <div class="between wrap"><div><div>Theme</div><div class="label">Track (dark) or Turbo (light)</div></div><div class="seg"><button :class="{ on: store.theme === 'dark' }" @click="setTheme('dark')">Dark</button><button :class="{ on: store.theme === 'light' }" @click="setTheme('light')">Light</button></div></div>
      <div class="between wrap"><div><div>Distance</div><div class="label">Used for odometer and efficiency</div></div><div class="seg"><button :class="{ on: me.distanceUnit === 0 }" @click="me.distanceUnit = 0">Miles</button><button :class="{ on: me.distanceUnit === 1 }" @click="me.distanceUnit = 1">Kilometers</button></div></div>
      <div class="between wrap"><div><div>Currency</div><div class="label">Three-letter code, e.g. USD, EUR, GBP</div></div><label class="field" style="width: 110px"><span class="sr-only">Currency</span><input v-model="me.currency" maxlength="3" style="text-transform: uppercase" /></label></div>
      <button class="btn pri" style="align-self: flex-start" @click="saveMe">Save preferences</button>
    </div>

    <div class="card c6 stack" style="gap: 16px; animation-delay: 0.05s">
      <h2 class="ctitle">Your account</h2>
      <div class="row"><div style="width: 44px; height: 44px; border-radius: 50%; border: 2px solid var(--accent); display: flex; align-items: center; justify-content: center">{{ (store.user?.name || '?')[0] }}</div><div class="grow"><div>{{ store.user?.name }}</div><div class="label">{{ store.user?.email }} · {{ isAdmin() ? 'Admin' : 'Member' }}</div></div></div>
      <div class="grid2"><label class="field">Current password<input v-model="pw.oldPassword" type="password" autocomplete="current-password" /></label><label class="field">New password<input v-model="pw.newPassword" type="password" autocomplete="new-password" /></label></div>
      <button class="btn ghost" style="align-self: flex-start" @click="changePw">Change password</button>
    </div>

    <div v-if="isAdmin()" class="card c6 stack" style="gap: 14px; animation-delay: 0.1s">
      <div class="between"><h2 class="ctitle">People</h2><button class="chip" @click="invite.open = true"><Icon name="add" :size="14" />Add someone</button></div>
      <div v-for="u in users" :key="u.id" class="listrow">
        <div class="tile-ic" style="border-radius: 50%">{{ (u.name || u.email)[0].toUpperCase() }}</div>
        <div class="grow"><div :style="{ opacity: u.isDisabled ? 0.5 : 1 }">{{ u.name || u.email }}</div><div class="label">{{ u.email }} · {{ u.roleDetail?.key === 'ADMIN' ? 'Admin' : 'Member' }}{{ u.isDisabled ? ' · disabled' : '' }}</div></div>
        <button v-if="u.id !== store.user?.id" class="chip" @click="toggleUser(u)">{{ u.isDisabled ? 'Enable' : 'Disable' }}</button>
      </div>
      <div class="label">Share a vehicle with someone from its edit page.</div>
    </div>

    <div class="card c6 stack" style="gap: 14px; animation-delay: 0.15s">
      <h2 class="ctitle">Import</h2>
      <div class="seg" style="align-self: flex-start"><button :class="{ on: imp.source === 'fuelly' }" @click="imp.source = 'fuelly'">Fuelly CSV</button><button :class="{ on: imp.source === 'drivvo' }" @click="imp.source = 'drivvo'">Drivvo CSV</button></div>
      <label v-if="imp.source === 'drivvo'" class="field">Into vehicle<select v-model="imp.vehicleId"><option v-for="v in store.vehicles" :key="v.id" :value="v.id">{{ vname(v) }}</option></select></label>
      <label class="btn ghost" style="border-style: dashed"><Icon name="upload" :size="18" />{{ imp.file ? imp.file.name : 'Choose CSV file' }}<input type="file" accept=".csv,text/csv" class="sr-only" @change="(e) => (imp.file = e.target.files?.[0] || null)" /></label>
      <div class="label">{{ imp.source === 'fuelly' ? 'Fuelly exports match vehicles by name, so create them here first with the same names.' : 'Drivvo imports everything into the vehicle you pick.' }}</div>
      <button class="btn pri" style="align-self: flex-start" :disabled="imp.busy" @click="runImport">Import</button>
    </div>

    <Modal v-if="invite.open" title="Add someone" @close="invite.open = false">
      <div class="grid2"><label class="field">Name<input v-model="invite.name" /></label><label class="field">Email<input v-model="invite.email" type="email" /></label></div>
      <div class="grid2"><label class="field">Temporary password<input v-model="invite.password" type="text" autocomplete="off" /></label><label class="field">Role<select v-model.number="invite.role"><option :value="1">Member</option><option :value="0">Admin</option></select></label></div>
      <button class="btn pri" @click="addUser">Create account</button>
    </Modal>
  </section>
</template>
