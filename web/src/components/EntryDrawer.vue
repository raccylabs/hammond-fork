<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { store, toast, dataChanged } from '../lib/store.js'
import { get, post, put, del } from '../lib/api.js'
import { money, fuelUnit, distUnit, isoDay, dayToISO, vname, effUnit, num } from '../lib/format.js'
import { readReceipt } from '../lib/ocr.js'
import Icon from './Icon.vue'

const e = store.entry
const kind = ref(e.kind)
const editing = e.record || null
const busy = ref(false)
const err = ref('')
const items = ref([])
const odometer = ref(0)
const lastFill = ref(null)

const f = reactive({
  vehicleId: e.vehicleId,
  date: isoDay(editing?.date || new Date()),
  odo: editing?.odoReading ?? '',
  qty: editing?.fuelQuantity ?? '',
  price: editing?.perUnitPrice ?? '',
  total: editing?.totalAmount ?? '',
  station: editing?.fillingStation ?? '',
  full: editing ? editing.isTankFull !== false : true,
  missed: editing?.hasMissedFillup === true,
  comments: editing?.comments ?? '',
  amount: editing?.amount ?? '',
  expenseType: editing?.expenseType ?? '',
  serviceId: e.serviceId || '',
  otherService: '',
  // reminder
  name: e.item?.name ?? '',
  trigger: e.item?.trigger ?? 'first',
  everyMi: e.item?.intervalMiles ?? '',
  everyMo: e.item?.intervalMonths ?? '',
  leadDays: e.item?.leadDays ?? 14,
  estCost: e.item?.estCost ?? '',
})
const totalTouched = ref(!!editing)

const vehicle = computed(() => store.vehicles.find((v) => v.id === f.vehicleId))
const title = computed(() => {
  if (kind.value === 'reminder') return e.item ? 'Edit schedule item' : 'New reminder'
  const verb = editing ? 'Edit' : kind.value === 'service' ? 'Log' : 'Add'
  return `${verb} ${kind.value}`
})

async function loadVehicle() {
  if (!f.vehicleId) return
  try {
    const s = await get(`/vehicles/${f.vehicleId}/services`)
    items.value = s.items.map((x) => x.item)
    odometer.value = s.odometer
    // Leave the odometer blank for new fillups: a pre-filled last reading is easy to save unchanged.
    if (!editing && f.odo === '' && kind.value !== 'fillup') f.odo = s.odometer ? String(s.odometer) : ''
    if (kind.value === 'service' && !f.serviceId) f.serviceId = items.value.find((i) => i.kind !== 'custom')?.id || 'other'
    const fills = await get(`/vehicles/${f.vehicleId}/fillups`)
    const others = (fills || []).filter((x) => !editing || x.id !== editing.id)
    // When editing, compare against the fillup just before this one, not the newest.
    const before = editing ? others.filter((x) => new Date(x.date) <= new Date(editing.date)) : others
    lastFill.value = before.sort((a, b) => b.odoReading - a.odoReading)[0] || null
  } catch {}
}
onMounted(loadVehicle)
watch(() => f.vehicleId, () => { f.odo = ''; f.serviceId = ''; loadVehicle() })

watch(() => [f.qty, f.price], () => {
  if (!totalTouched.value && f.qty && f.price) f.total = (parseFloat(f.qty) * parseFloat(f.price)).toFixed(2)
})
const confirmOdo = ref(false)
watch(() => f.odo, () => (confirmOdo.value = false))
const odoWarn = computed(() => {
  if (kind.value !== 'fillup' || !lastFill.value || f.odo === '') return ''
  const o = parseInt(f.odo)
  if (!(o > 0)) return ''
  if (o <= lastFill.value.odoReading) return `That's not more than your last fillup (${num(lastFill.value.odoReading)}).`
  if (o - lastFill.value.odoReading > 2500) return `That's ${num(o - lastFill.value.odoReading)} ${distUnit()} since your last fillup. Double-check it.`
  return ''
})
const effPreview = computed(() => {
  if (kind.value !== 'fillup' || !lastFill.value || !f.full || f.missed) return ''
  const d = parseInt(f.odo) - lastFill.value.odoReading
  const q = parseFloat(f.qty)
  if (!(d > 0 && q > 0)) return ''
  return `That works out to ${(d / q).toFixed(1)} ${effUnit(vehicle.value)} since your last fillup.`
})

// ---------------- receipt scan ----------------
const ocr = ref(null) // null | 'scan' | 'done' | 'fail'
const preview = ref('')
const hl = ref(false)
async function scan(ev) {
  const file = ev.target.files?.[0]
  ev.target.value = ''
  if (!file) return
  preview.value = URL.createObjectURL(file)
  ocr.value = 'scan'
  try {
    const r = await readReceipt(file)
    let found = 0
    if (r.quantity) { f.qty = String(r.quantity); found++ }
    if (r.price) { f.price = String(r.price); found++ }
    if (r.total) { f.total = String(r.total); totalTouched.value = true; found++ }
    if (r.station) f.station = r.station
    if (r.date) f.date = isoDay(r.date)
    ocr.value = found ? 'done' : 'fail'
    hl.value = found > 0
  } catch (x) {
    ocr.value = 'fail'
  }
}
onBeforeUnmount(() => preview.value && URL.revokeObjectURL(preview.value))

// ---------------- save ----------------
const close = () => (store.entry = null)
const n = (x) => (x === '' || x == null ? 0 : parseFloat(x))

async function save() {
  err.value = ''
  busy.value = true
  const vid = f.vehicleId
  try {
    if (kind.value === 'fillup') {
      const body = {
        vehicleId: vid, fuelUnit: vehicle.value?.fuelUnit ?? 2, fuelQuantity: n(f.qty), perUnitPrice: n(f.price), totalAmount: n(f.total),
        odoReading: parseInt(f.odo) || 0, isTankFull: !!f.full, HasMissedFillup: !!f.missed, comments: f.comments, fillingStation: f.station,
        userId: store.user.id, date: dayToISO(f.date),
      }
      if (!body.fuelQuantity || !body.totalAmount || !body.odoReading) throw new Error('Quantity, total and odometer are required')
      if (odoWarn.value && !confirmOdo.value) { confirmOdo.value = true; throw new Error(odoWarn.value + ' Press save again to keep it anyway.') }
      if (editing) await put(`/vehicles/${vid}/fillups/${editing.id}`, body)
      else await post(`/vehicles/${vid}/fillups`, body)
      toast(editing ? 'Fillup updated' : 'Fillup saved' + (effPreview.value ? ' · ' + effPreview.value.match(/[\d.]+ \S+/)[0] : ''))
    } else if (kind.value === 'expense' || (kind.value === 'service' && f.serviceId === 'other')) {
      const type = kind.value === 'service' ? f.otherService || 'Service' : f.expenseType || 'Other'
      const body = { vehicleId: vid, amount: n(f.amount), odoReading: parseInt(f.odo) || 0, comments: f.comments, expenseType: type, userId: store.user.id, date: dayToISO(f.date) }
      if (editing) await put(`/vehicles/${vid}/expenses/${editing.id}`, body)
      else await post(`/vehicles/${vid}/expenses`, body)
      toast(`${kind.value === 'service' ? 'Service' : 'Expense'} saved · ${money(body.amount)}`)
    } else if (kind.value === 'service') {
      const item = items.value.find((i) => i.id === f.serviceId)
      await post(`/vehicles/${vid}/services/${f.serviceId}/complete`, { date: dayToISO(f.date), odoReading: parseInt(f.odo) || 0, amount: n(f.amount), comments: f.comments })
      toast(`${item?.name || 'Service'} logged. Its reminder starts over from today.`)
    } else if (kind.value === 'reminder') {
      const body = { name: f.name.trim(), kind: e.item?.kind || 'custom', trigger: f.trigger, intervalMiles: f.trigger === 'date' ? 0 : parseInt(f.everyMi) || 0,
        intervalMonths: f.trigger === 'miles' ? 0 : parseInt(f.everyMo) || 0, leadDays: parseInt(f.leadDays) || 14, estCost: n(f.estCost) }
      if (!body.name) throw new Error('Give the reminder a name')
      if (!body.intervalMiles && !body.intervalMonths) throw new Error('Set a mileage or a month interval')
      if (e.item) await put(`/vehicles/${vid}/services/${e.item.id}`, { ...body, lastOdo: e.item.lastOdo, lastDate: e.item.lastDate })
      else await post(`/vehicles/${vid}/services`, body)
      toast(e.item ? 'Schedule updated' : `Reminder set · ${body.name} on ${vname(vehicle.value)}`)
    }
    dataChanged()
    close()
  } catch (x) {
    err.value = x.message
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (!editing && !e.item) return
  if (!confirm('Delete this entry? This cannot be undone.')) return
  busy.value = true
  try {
    if (kind.value === 'fillup') await del(`/vehicles/${f.vehicleId}/fillups/${editing.id}`)
    else if (kind.value === 'reminder') await del(`/vehicles/${f.vehicleId}/services/${e.item.id}`)
    else await del(`/vehicles/${f.vehicleId}/expenses/${editing.id}`)
    toast('Deleted')
    dataChanged()
    close()
  } catch (x) { err.value = x.message } finally { busy.value = false }
}

const remPreview = computed(() => {
  const mi = parseInt(f.everyMi) || 0, mo = parseInt(f.everyMo) || 0
  const d = new Date(); d.setMonth(d.getMonth() + mo)
  const a = `at ${num(odometer.value + mi)} ${distUnit()}`, b = `on ${d.toLocaleDateString()}`
  const when = f.trigger === 'miles' ? a : f.trigger === 'date' ? b : `${a} or ${b}, whichever comes first`
  return `First due ${when}. You'll get a heads-up ${f.leadDays} days ahead.`
})
const onKey = (ev) => { if (ev.key === 'Escape') close() }
onMounted(() => document.addEventListener('keydown', onKey))
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>

<template>
  <Teleport to="body">
    <div class="scrim" @click="close"></div>
    <aside class="drawer" role="dialog" aria-modal="true" :aria-label="title">
      <div class="between">
        <h2 class="disp" style="font-size: 24px; text-transform: capitalize">{{ title }}</h2>
        <button class="btn ghost icon" aria-label="Close" @click="close"><Icon name="x" :size="20" /></button>
      </div>

      <div v-if="!editing && kind !== 'reminder'" class="seg" role="tablist" aria-label="Entry type">
        <button v-for="k in ['fillup', 'service', 'expense']" :key="k" :class="{ on: kind === k }" role="tab" :aria-selected="kind === k" style="text-transform: capitalize" @click="kind = k">{{ k }}</button>
      </div>

      <label class="field">Vehicle
        <select v-model="f.vehicleId" :disabled="!!editing || !!e.item">
          <option v-for="v in store.vehicles" :key="v.id" :value="v.id">{{ vname(v) }}</option>
        </select>
      </label>

      <template v-if="kind !== 'reminder'">
        <div class="grid2" :class="{ hl }">
          <label class="field">Date<input v-model="f.date" type="date" /></label>
          <label class="field">Odometer ({{ distUnit() }})<input v-model="f.odo" inputmode="numeric" :placeholder="odometer ? 'Last: ' + num(odometer) : ''" /><span v-if="odoWarn" class="hint" style="color: var(--warn)">{{ odoWarn }}</span></label>
        </div>
      </template>

      <!-- fillup -->
      <template v-if="kind === 'fillup'">
        <label v-if="!ocr || ocr === 'fail'" class="btn ghost" style="border-style: dashed; width: 100%">
          <Icon name="camera" :size="20" />Snap pump or receipt to auto-fill
          <input type="file" accept="image/*" capture="environment" class="sr-only" @change="scan" />
        </label>
        <div v-if="ocr === 'fail'" class="label">Couldn't read that one. Try a closer, flatter photo, or type the numbers in.</div>
        <template v-if="ocr === 'scan'">
          <div class="scan"><img :src="preview" alt="" /><div class="scanline"></div></div>
          <div class="label pulse" role="status">Reading gallons, price and total… The first scan downloads the reader, so it can take a moment.</div>
        </template>
        <div v-if="ocr === 'done'" class="row" style="padding: 10px 12px 10px 14px; border-radius: 12px; background: var(--card2); font-size: 13px; animation: rise 0.4s both">
          <Icon name="check" :size="18" style="color: var(--accentText)" /><span class="grow">Read from your receipt. Check the highlighted fields.</span>
          <label class="chip">Rescan<input type="file" accept="image/*" capture="environment" class="sr-only" @change="scan" /></label>
        </div>
        <div class="grid2" :class="{ hl }">
          <label class="field">Quantity ({{ fuelUnit(vehicle) }})<input v-model="f.qty" inputmode="decimal" /></label>
          <label class="field">Price / {{ fuelUnit(vehicle) }}<input v-model="f.price" inputmode="decimal" /></label>
        </div>
        <label class="field" :class="{ hl }">Station<input v-model="f.station" /></label>
        <div class="chips">
          <button class="chip" :class="{ on: f.full }" @click="f.full = !f.full">Filled the tank</button>
          <button class="chip" :class="{ on: f.missed }" @click="f.missed = !f.missed">Missed last fillup</button>
        </div>
        <label class="field">Total<input v-model="f.total" inputmode="decimal" @input="totalTouched = true" /></label>
        <div v-if="effPreview" class="label">{{ effPreview }}</div>
      </template>

      <!-- service -->
      <template v-if="kind === 'service' && !editing">
        <label class="field">Service
          <select v-model="f.serviceId">
            <option v-for="i in items" :key="i.id" :value="i.id">{{ i.name }}</option>
            <option value="other">Something else…</option>
          </select>
        </label>
        <label v-if="f.serviceId === 'other'" class="field">What was done<input v-model="f.otherService" placeholder="e.g. Replaced front brake pads" /></label>
        <label class="field">Cost<input v-model="f.amount" inputmode="decimal" placeholder="0.00" /></label>
        <label class="field">Notes<input v-model="f.comments" placeholder="Shop, parts, anything worth remembering" /></label>
        <div v-if="f.serviceId !== 'other'" class="label">Logging it resets that item's interval, so its reminder starts over.</div>
      </template>

      <!-- expense -->
      <template v-if="kind === 'expense' || (kind === 'service' && editing)">
        <label class="field">Type<input v-model="f.expenseType" list="exp-types" placeholder="Repair, Insurance, Registration…" /></label>
        <datalist id="exp-types"><option v-for="t in ['Repair', 'Tires', 'Insurance', 'Registration', 'Parking & tolls', 'Car wash', 'Accessories', 'Inspection', 'Other']" :key="t" :value="t" /></datalist>
        <label class="field">Amount<input v-model="f.amount" inputmode="decimal" /></label>
        <label class="field">Notes<input v-model="f.comments" /></label>
      </template>

      <!-- reminder -->
      <template v-if="kind === 'reminder'">
        <label class="field">What needs doing<input v-model="f.name" placeholder="Wiper blades" /></label>
        <div class="field">Remind me by
          <div class="seg" style="align-self: flex-start">
            <button v-for="t in [['first', 'Whichever first'], ['miles', 'Distance'], ['date', 'Date']]" :key="t[0]" :class="{ on: f.trigger === t[0] }" @click="f.trigger = t[0]">{{ t[1] }}</button>
          </div>
        </div>
        <div class="grid2">
          <label v-if="f.trigger !== 'date'" class="field">Every ({{ distUnit() }})<input v-model="f.everyMi" inputmode="numeric" /></label>
          <label v-if="f.trigger !== 'miles'" class="field">Every (months)<input v-model="f.everyMo" inputmode="numeric" /></label>
        </div>
        <div class="grid2">
          <label class="field">Heads-up (days before)<input v-model="f.leadDays" inputmode="numeric" /></label>
          <label class="field">Typical cost<input v-model="f.estCost" inputmode="decimal" placeholder="optional" /></label>
        </div>
        <div style="padding: 14px; border-radius: 12px; background: var(--card2); font-size: 13px; color: var(--muted)">{{ remPreview }}</div>
      </template>

      <div v-if="err" class="error" role="alert">{{ err }}</div>
      <div style="flex-grow: 1"></div>
      <div class="row">
        <button v-if="editing || e.item" class="btn danger" :disabled="busy" @click="remove"><Icon name="trash" :size="18" />Delete</button>
        <button class="btn pri grow" style="min-height: 54px; font-size: 16px" :disabled="busy" @click="save">
          <Icon v-if="busy" name="spinner" class="spin" :size="18" />
          {{ kind === 'fillup' ? 'Save fillup' : kind === 'service' ? 'Save service' : kind === 'reminder' ? 'Save reminder' : 'Save expense' }}
        </button>
      </div>
    </aside>
  </Teleport>
</template>
