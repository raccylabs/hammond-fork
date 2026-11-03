<script setup>
import { ref, computed, onMounted } from 'vue'
import { store, openEntry, toast, refreshReminders, dataChanged, isAdmin, refreshVehicles } from '../lib/store.js'
import { get, post, del } from '../lib/api.js'
import { fdate, fdateLong, money, num, distUnit, vname, reminderStatusText, LEVEL_COLOR, LEVEL_TEXT } from '../lib/format.js'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'

onMounted(refreshReminders)
const vehicleFilter = ref('all')
const list = computed(() => store.reminders.filter((r) => vehicleFilter.value === 'all' || r.vehicleId === vehicleFilter.value))
const top = computed(() => store.reminders.find((r) => r.level > 0 && r.status !== 'snoozed'))
const noSchedule = computed(() => store.vehicles.filter((v) => !store.reminders.some((r) => r.vehicleId === v.id && r.kind === 'service')))

async function snooze(r, days) {
  try { await post(`/vehicles/${r.vehicleId}/services/${r.refId}/snooze`, { days }); toast(days ? 'Snoozed for 2 weeks' : 'Unsnoozed'); refreshReminders() } catch (x) { toast(x.message, true) }
}
async function editItem(r) {
  const s = await get(`/vehicles/${r.vehicleId}/services`)
  const item = s.items.map((x) => x.item).find((i) => i.id === r.refId)
  if (item) openEntry('reminder', { vehicleId: r.vehicleId, item })
}
async function seed(v) {
  try { await post(`/vehicles/${v.id}/default-services`); toast(`Standard schedule added for ${vname(v)}`); dataChanged() } catch (x) { toast(x.message, true) }
}

// recalls
const recallFor = ref(null)
const recalls = ref(null)
const checking = ref(false)
async function openRecalls(vehicleId) {
  recallFor.value = store.vehicles.find((v) => v.id === vehicleId)
  recalls.value = await get(`/vehicles/${vehicleId}/recalls`)
}
async function checkNow() {
  checking.value = true
  try { recalls.value = await post(`/vehicles/${recallFor.value.id}/recall-check`); await refreshVehicles() } catch (x) { toast("Couldn't reach NHTSA: " + x.message, true) } finally { checking.value = false }
}
async function dismiss(r, on) {
  const path = `/vehicles/${recallFor.value.id}/recalls/${encodeURIComponent(r.campaign)}/dismiss`
  recalls.value = on ? await post(path) : await del(path)
  await refreshVehicles()
}

// notifications
const notif = ref(null)
const notifOpen = ref(false)
const testing = ref(false)
const secret = ref({ ntfyToken: '', smtpPassword: '' })
async function openNotif() { notif.value = await get('/notifications/settings'); notifOpen.value = true }
async function saveNotif(test = false) {
  const body = { ...notif.value }
  if (secret.value.ntfyToken) body.ntfyToken = secret.value.ntfyToken
  if (secret.value.smtpPassword) body.smtpPassword = secret.value.smtpPassword
  try {
    notif.value = await post('/notifications/settings', body)
    secret.value = { ntfyToken: '', smtpPassword: '' }
    if (test) {
      testing.value = true
      const r = await post('/notifications/test')
      toast(r.errors?.length ? 'Some channels failed: ' + r.errors.join('; ') : 'Test sent. Check your phone or inbox.', !!r.errors?.length)
    } else { toast('Notification settings saved'); notifOpen.value = false }
  } catch (x) { toast(x.message, true) } finally { testing.value = false }
}
const channels = computed(() => notif.value ? [notif.value.ntfyEnabled && 'ntfy', notif.value.hookEnabled && 'webhook', notif.value.emailEnabled && 'email'].filter(Boolean) : [])
onMounted(async () => { if (isAdmin()) try { notif.value = await get('/notifications/settings') } catch {} })
</script>

<template>
  <section aria-label="Reminders" class="g12">
    <div class="card c8 stack" style="gap: 4px">
      <div class="between wrap" style="margin-bottom: 10px">
        <h2 class="ctitle">Coming up across your garage</h2>
        <div class="row wrap">
          <select v-model="vehicleFilter" class="chip" aria-label="Filter by vehicle" style="appearance: auto; padding-right: 8px">
            <option value="all">All vehicles</option>
            <option v-for="v in store.vehicles" :key="v.id" :value="v.id">{{ vname(v) }}</option>
          </select>
          <button class="btn pri" @click="openEntry('reminder', { vehicleId: vehicleFilter !== 'all' ? vehicleFilter : undefined })"><Icon name="add" :size="18" />New reminder</button>
        </div>
      </div>
      <div v-for="(r, i) in list" :key="r.key" class="remrow" style="display: grid; grid-template-columns: 1fr auto; gap: 6px 16px; padding: 14px 0; border-bottom: 1px solid var(--line); animation: rise 0.5s both" :style="{ animationDelay: Math.min(i * 0.04, 0.6) + 's' }">
        <div class="row" style="gap: 10px; min-width: 0"><span class="dot" :style="{ background: r.status === 'snoozed' ? 'var(--line2)' : LEVEL_COLOR[r.level] }"></span><span style="font-size: 15px">{{ r.title }}</span><span class="label">· {{ r.vehicleName }}</span></div>
        <div style="font-weight: 600; text-align: right" :style="{ color: r.status === 'snoozed' ? 'var(--muted)' : LEVEL_TEXT[r.level] }">{{ reminderStatusText(r) }}</div>
        <div class="bar" style="grid-column: 1 / -1"><div :style="{ width: (r.status === 'snoozed' ? 0 : Math.min(100, r.pct * 100)) + '%', background: LEVEL_COLOR[r.level] }"></div></div>
        <div class="label" style="align-self: center">
          <template v-if="r.kind === 'document'">Expires {{ fdateLong(r.dueDate) }}</template>
          <template v-else-if="r.kind === 'recall'">From NHTSA · free repair at any dealer</template>
          <template v-else>
            <span v-if="r.dueOdo">Due at {{ num(r.dueOdo) }} {{ distUnit() }}</span><span v-if="r.dueOdo && r.dueDate"> or </span><span v-if="r.dueDate">{{ fdateLong(r.dueDate) }}</span>
            <span v-if="r.projectedDate && r.milesLeft > 0"> · around {{ fdate(r.projectedDate) }} at your pace</span>
            <span v-if="r.estCost"> · ~{{ money(r.estCost, 0) }}</span>
          </template>
        </div>
        <div class="row" style="gap: 6px; justify-content: flex-end">
          <template v-if="r.kind === 'service' || r.kind === 'custom'">
            <button class="chip" @click="snooze(r, r.status === 'snoozed' ? 0 : 14)">{{ r.status === 'snoozed' ? 'Unsnooze' : 'Snooze 2 wk' }}</button>
            <button class="chip" :aria-label="'Edit ' + r.title" @click="editItem(r)"><Icon name="edit" :size="14" /></button>
            <button class="chip" @click="openEntry('service', { vehicleId: r.vehicleId, serviceId: r.refId })">Log it</button>
          </template>
          <RouterLink v-else-if="r.kind === 'document'" to="/documents" class="chip">Update</RouterLink>
          <button v-else-if="r.kind === 'recall'" class="chip" @click="openRecalls(r.vehicleId)">Review</button>
        </div>
      </div>
      <div v-if="!list.length" class="empty">No reminders yet.</div>
      <div v-for="v in noSchedule" :key="v.id" class="row wrap" style="padding: 14px; margin-top: 12px; border-radius: 14px; background: var(--card2)">
        <Icon name="wrench" :size="20" /><span class="grow">{{ vname(v) }} has no maintenance schedule.</span>
        <button class="btn out" @click="seed(v)">Add the standard schedule</button>
      </div>
    </div>

    <div class="card c4 stack" style="gap: 16px; animation-delay: 0.1s">
      <h2 class="ctitle">Notifications</h2>
      <template v-if="isAdmin()">
        <div class="label">{{ channels.length ? `Sending to ${channels.join(', ')}. Hammond checks every hour and won't repeat itself for a week.` : 'No channels on yet. Hook up ntfy, email or Home Assistant to hear about upcoming maintenance.' }}</div>
        <button class="btn ghost" @click="openNotif"><Icon name="bell" :size="18" />Set up channels</button>
      </template>
      <div v-else class="label">Your Hammond admin sets up notification channels.</div>
      <div class="label" style="border-top: 1px solid var(--line); padding-top: 14px">What it looks like</div>
      <div class="row" style="align-items: flex-start; background: var(--card2); border-radius: 16px; padding: 12px 14px; animation: rise 0.6s 0.4s both">
        <div style="width: 34px; height: 34px; border-radius: 9px; background: #fe4611; display: flex; align-items: center; justify-content: center; flex: none; color: #231f20"><Icon name="bell" :size="18" /></div>
        <div style="min-width: 0"><div class="label">ntfy · Hammond · now</div><div style="font-size: 14px; font-weight: 600">{{ top ? `${top.title} · ${top.vehicleName}` : 'Oil change · Sonic' }}</div><div class="muted" style="font-size: 13px">{{ top ? reminderStatusText(top) : 'In 320 mi' }}. Open Hammond to log it or snooze.</div></div>
      </div>
      <div class="label">Reminders fire on whichever comes first, date or distance. Distance is projected from your fillup history.</div>
    </div>

    <Modal v-if="recallFor" :title="`Recalls · ${vname(recallFor)}`" @close="recallFor = null">
      <div class="between wrap"><span class="label">{{ recalls?.checkedAt ? 'Checked ' + fdateLong(recalls.checkedAt) : 'Not checked yet' }} · {{ recallFor.yearOfManufacture }} {{ recallFor.make }} {{ recallFor.model }}</span><button class="btn ghost" :disabled="checking" @click="checkNow"><Icon name="refresh" :size="16" :class="{ spin: checking }" />Check now</button></div>
      <div class="label">NHTSA lists recalls per model year, so some may not apply to your exact car. Dismiss the ones that don't, or that a dealer has already fixed.</div>
      <div v-for="r in recalls?.recalls || []" :key="r.campaign" class="stack" style="gap: 6px; padding: 14px; border-radius: 14px; background: var(--card2)" :style="{ opacity: r.dismissed ? 0.55 : 1 }">
        <div class="between"><strong>{{ r.component }}</strong><span class="label">#{{ r.campaign }}</span></div>
        <div style="font-size: 13px">{{ r.summary }}</div>
        <div v-if="r.remedy" class="label">Remedy: {{ r.remedy }}</div>
        <button class="chip" style="align-self: flex-start" @click="dismiss(r, !r.dismissed)">{{ r.dismissed ? 'Mark as open' : 'Fixed / not mine' }}</button>
      </div>
      <div v-if="recalls && !recalls.recalls.length" class="empty"><Icon name="shield" :size="30" />No recalls on file for this model year.</div>
    </Modal>

    <Modal v-if="notifOpen && notif" title="Notification channels" @close="notifOpen = false">
      <div class="stack">
        <div class="between"><div><strong>ntfy</strong><div class="label">Push to your phone. Works with ntfy.sh or your own server.</div></div><button class="toggle" role="switch" :aria-checked="notif.ntfyEnabled" aria-label="ntfy" @click="notif.ntfyEnabled = !notif.ntfyEnabled"><span></span></button></div>
        <div v-if="notif.ntfyEnabled" class="grid2"><label class="field">Server<input v-model="notif.ntfyUrl" placeholder="https://ntfy.sh" /></label><label class="field">Topic<input v-model="notif.ntfyTopic" placeholder="hammond-lance" /></label></div>
        <label v-if="notif.ntfyEnabled" class="field">Access token {{ notif.hasNtfyToken ? '(saved; leave blank to keep)' : '(optional)' }}<input v-model="secret.ntfyToken" type="password" autocomplete="off" /></label>
      </div>
      <div class="stack">
        <div class="between"><div><strong>Webhook</strong><div class="label">POSTs JSON {title, message}. Use a Home Assistant webhook trigger.</div></div><button class="toggle" role="switch" :aria-checked="notif.hookEnabled" aria-label="Webhook" @click="notif.hookEnabled = !notif.hookEnabled"><span></span></button></div>
        <label v-if="notif.hookEnabled" class="field">URL<input v-model="notif.webhookUrl" placeholder="http://homeassistant.local:8123/api/webhook/hammond" /></label>
      </div>
      <div class="stack">
        <div class="between"><div><strong>Email</strong><div class="label">Any SMTP server.</div></div><button class="toggle" role="switch" :aria-checked="notif.emailEnabled" aria-label="Email" @click="notif.emailEnabled = !notif.emailEnabled"><span></span></button></div>
        <template v-if="notif.emailEnabled">
          <div class="grid2"><label class="field">SMTP host<input v-model="notif.smtpHost" /></label><label class="field">Port<input v-model.number="notif.smtpPort" inputmode="numeric" /></label></div>
          <div class="grid2"><label class="field">Username<input v-model="notif.smtpUser" autocomplete="off" /></label><label class="field">Password {{ notif.hasSmtpPassword ? '(saved)' : '' }}<input v-model="secret.smtpPassword" type="password" autocomplete="new-password" /></label></div>
          <div class="grid2"><label class="field">From<input v-model="notif.smtpFrom" /></label><label class="field">Send to<input v-model="notif.emailTo" placeholder="you@example.com" /></label></div>
        </template>
      </div>
      <div class="row"><button class="btn ghost" :disabled="testing" @click="saveNotif(true)">Save &amp; send test</button><button class="btn pri grow" @click="saveNotif(false)">Save</button></div>
    </Modal>
  </section>
</template>
