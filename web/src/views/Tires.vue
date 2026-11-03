<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { store, currentVehicle, selectVehicle, toast, dataChanged } from '../lib/store.js'
import { get, post, put, del } from '../lib/api.js'
import { num, fdate, vname, distUnit, MO, isoDay, dayToISO, toDate, monthYear } from '../lib/format.js'
import { smooth } from '../lib/calc.js'
import { useTween } from '../lib/motion.js'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'

const v = computed(() => currentVehicle())
const t = ref(null)
async function load() { if (v.value) t.value = await get(`/vehicles/${v.value.id}/tires`) }
watch([() => v.value?.id, () => store.dataVersion], load, { immediate: true })

const mounted = computed(() => t.value?.sets.find((s) => s.isMounted))
const readings = computed(() => (t.value?.readings || []).filter((r) => r.tireSetId === mounted.value?.id))
const latest = computed(() => readings.value[readings.value.length - 1])
const corners = computed(() => (latest.value ? [latest.value.fl, latest.value.fr, latest.value.rl, latest.value.rr] : null))
const avgT = computed(() => (corners.value ? corners.value.reduce((a, b) => a + b, 0) / 4 : null))

function setMiles(s) {
  return s.isMounted ? (s.milesBefore || 0) + Math.max(0, (t.value?.odometer || 0) - s.installedOdo) : s.milesBefore || 0
}
function setTread(s) {
  const rs = (t.value?.readings || []).filter((r) => r.tireSetId === s.id)
  const r = rs[rs.length - 1]
  return r ? (r.fl + r.fr + r.rl + r.rr) / 4 : null
}
const sets = computed(() => (t.value?.sets || []).map((s) => {
  const tr = setTread(s)
  return { s, miles: setMiles(s), tread: tr, pct: tr == null ? 0 : Math.max(4, Math.min(100, ((tr - 2) / ((s.treadNew || 10) - 2)) * 100)), color: tr == null ? 'var(--s3)' : tr <= 4 ? 'var(--bad)' : tr <= 5.5 ? 'var(--warn)' : 'var(--accent)' }
}))

// rotation pattern drawing
const ARROWS = {
  'Forward cross': ['M31,128 L31,76', 'M129,128 L129,76', 'M44,62 L114,138', 'M116,62 L46,138'],
  'Front to rear, same side': ['M25,74 L25,126', 'M37,126 L37,74', 'M123,74 L123,126', 'M135,126 L135,74'],
  'Rearward cross': ['M31,72 L31,124', 'M129,72 L129,124', 'M44,138 L114,62', 'M116,138 L46,62'],
}
const rotationItem = computed(() => store.reminders.find((r) => r.vehicleId === v.value?.id && /rotation/i.test(r.title)))

// tread chart + projection
const chart = computed(() => {
  if (!mounted.value || readings.value.length < 1) return null
  const pts = readings.value.map((r) => ({ t: toDate(r.date).getTime(), odo: r.odoReading, avg: (r.fl + r.fr + r.rl + r.rr) / 4, min: Math.min(r.fl, r.fr, r.rl, r.rr) }))
  const start = mounted.value.installedDate ? Math.min(toDate(mounted.value.installedDate).getTime(), pts[0].t) : pts[0].t
  const newT = mounted.value.treadNew || 10
  const series = [{ t: start, odo: mounted.value.installedOdo, avg: newT, min: newT }, ...pts.filter((p) => p.t > start)]
  const last = series[series.length - 1]
  const used = last.odo - mounted.value.installedOdo
  const wear = used > 200 ? (newT - last.avg) / used : 0
  const mpd = t.value?.milesPerDay || 30
  let rep = null
  if (wear > 0 && last.avg > 4) rep = new Date(last.t + ((last.avg - 4) / wear / mpd) * 864e5)
  const end = Math.max(rep ? Math.min(rep.getTime(), Date.now() + 4 * 365 * 864e5) : 0, Date.now() + 90 * 864e5, last.t + 30 * 864e5)
  const X = (x) => 40 + ((x - start) / (end - start || 1)) * 640, Y = (d) => 216 - (d / (newT + 1)) * 200
  const ticks = [0, 1, 2, 3].map((i) => { const x = start + ((end - start) * i) / 3; const d = new Date(x); return { x: X(x), l: `${MO[d.getMonth()]} '${String(d.getFullYear()).slice(2)}` } })
  return {
    avg: smooth(series.map((p) => [X(p.t), Y(p.avg)])), min: smooth(series.map((p) => [X(p.t), Y(p.min)])),
    proj: rep ? `M${X(last.t)},${Y(last.avg)} L${X(rep.getTime())},${Y(4)}` : '', rep, repX: rep ? X(rep.getTime()) : 0,
    y4: Y(4), y2: Y(2), grid: [0, 2, 4, 6, 8, 10].filter((d) => d <= newT).map((d) => ({ y: Y(d), l: d })), ticks, mpd, last,
  }
})
const projText = computed(() => {
  if (!chart.value) return 'Log a tread reading to start the chart.'
  if (chart.value.last.avg <= 4) return 'Average tread is at or below 4/32". Time to shop for tires; wet-road grip drops off quickly from here.'
  if (!chart.value.rep) return 'Add another reading after a few thousand miles to project when these will need replacing.'
  return `At your pace (about ${num(chart.value.mpd * 30)} ${distUnit()} a month) the average reaches 4/32" around ${monthYear(chart.value.rep)}.`
})
const health = useTween(() => (avgT.value == null ? 0 : Math.round(Math.max(0, (avgT.value - 2) / ((mounted.value?.treadNew || 10) - 2)) * 100)))

// actions
const busy = ref(false)
async function act(fn, msg) {
  busy.value = true
  try { t.value = await fn(); toast(msg); dataChanged() } catch (x) { toast(x.message, true) } finally { busy.value = false }
}
const rot = reactive({ open: false, date: isoDay(), odo: '' })
function openRotation() { Object.assign(rot, { open: true, date: isoDay(), odo: t.value?.odometer || '' }) }
async function saveRotation() {
  await act(() => post(`/vehicles/${v.value.id}/rotations`, { date: dayToISO(rot.date), odoReading: parseInt(rot.odo) || 0 }), `Rotation logged · ${t.value.pattern.toLowerCase()}`)
  rot.open = false
}
const tread = reactive({ fl: '', fr: '', rl: '', rr: '', pfl: '', pfr: '', prl: '', prr: '', date: isoDay(), odo: '' })
watch(latest, (r) => { if (r) Object.assign(tread, { fl: r.fl, fr: r.fr, rl: r.rl, rr: r.rr }) }, { immediate: true })
async function saveTread() {
  const n = (x) => parseFloat(x) || 0
  if (![tread.fl, tread.fr, tread.rl, tread.rr].every((x) => n(x) > 0)) { toast('Enter all four corners', true); return }
  await act(() => post(`/vehicles/${v.value.id}/treadReadings`, { date: dayToISO(tread.date), odoReading: parseInt(tread.odo) || 0, fl: n(tread.fl), fr: n(tread.fr), rl: n(tread.rl), rr: n(tread.rr), pressureFl: n(tread.pfl), pressureFr: n(tread.pfr), pressureRl: n(tread.prl), pressureRr: n(tread.prr) }), `Tread reading saved for ${vname(v.value)}`)
}
async function mount(s) { await act(() => post(`/vehicles/${v.value.id}/tires/${s.id}/mount`, { odoReading: t.value.odometer }), `${s.name} mounted`) }

const setForm = reactive({ open: false, id: '', name: '', brand: '', size: '', storedAt: '', milesBefore: '', treadNew: 10 })
function openSet(s) { Object.assign(setForm, { open: true, id: s?.id || '', name: s?.name || '', brand: s?.brand || '', size: s?.size || v.value?.tireSize || '', storedAt: s?.storedAt || '', milesBefore: s?.milesBefore || '', treadNew: s?.treadNew || 10 }) }
async function saveSet() {
  const body = { name: setForm.name || 'Tire set', brand: setForm.brand, size: setForm.size, storedAt: setForm.storedAt, milesBefore: parseInt(setForm.milesBefore) || 0, treadNew: parseFloat(setForm.treadNew) || 10 }
  busy.value = true
  try {
    if (setForm.id) await put(`/vehicles/${v.value.id}/tires/${setForm.id}`, body)
    else await post(`/vehicles/${v.value.id}/tires`, body)
    setForm.open = false; await load(); toast('Tire set saved')
  } catch (x) { toast(x.message, true) } finally { busy.value = false }
}
async function removeSet() {
  if (!confirm('Delete this tire set and its readings?')) return
  try { await del(`/vehicles/${v.value.id}/tires/${setForm.id}`); setForm.open = false; await load() } catch (x) { toast(x.message, true) }
}
</script>

<template>
  <section v-if="!v" class="card empty">Add a vehicle first.</section>
  <section v-else aria-label="Tires">
    <div class="chips" style="margin-bottom: 18px"><button v-for="x in store.vehicles" :key="x.id" class="chip" :class="{ on: x.id === v.id }" @click="selectVehicle(x.id)">{{ vname(x) }}</button></div>
    <div v-if="t" class="g12" :key="v.id">
      <div class="card c7 stack">
        <div class="between"><h2 class="ctitle">Tire sets</h2><button class="chip" @click="openSet(null)"><Icon name="add" :size="14" />Add a set</button></div>
        <div v-for="x in sets" :key="x.s.id" class="stack" style="gap: 10px; background: var(--card2); border-radius: 16px; padding: 16px; transition: border-color 0.3s" :style="{ border: '1.5px solid ' + (x.s.isMounted ? 'var(--accent)' : 'transparent') }">
          <div class="between wrap">
            <div><span style="font-size: 17px; font-weight: 600">{{ x.s.name }}</span> <span v-if="x.s.brand" class="label">· {{ x.s.brand }}</span></div>
            <div class="row" style="gap: 6px">
              <span class="badge" style="background: var(--card)">{{ x.s.isMounted ? 'Mounted' : 'Stored' + (x.s.storedAt ? ' · ' + x.s.storedAt : '') }}</span>
              <button v-if="!x.s.isMounted" class="chip" :disabled="busy" @click="mount(x.s)">Mount</button>
              <button class="chip" :aria-label="'Edit ' + x.s.name" @click="openSet(x.s)"><Icon name="edit" :size="14" /></button>
            </div>
          </div>
          <div class="label">{{ x.s.size || 'Size not set' }} · {{ x.s.isMounted && x.s.installedDate ? 'Mounted ' + fdate(x.s.installedDate) + ' · ' : '' }}{{ num(x.miles) }} {{ distUnit() }} on this set</div>
          <div class="row"><div class="bar grow" style="height: 8px; background: var(--card)"><div :style="{ width: x.pct + '%', background: x.color }"></div></div><span style="font-size: 13px; font-weight: 600">{{ x.tread == null ? 'No reading' : x.tread.toFixed(1) + '/32"' }}</span></div>
        </div>
      </div>

      <div class="card c5 stack" style="animation-delay: 0.05s">
        <div class="between"><h2 class="ctitle">Rotation</h2><button class="chip" @click="openRotation">Log rotation</button></div>
        <div class="row" style="gap: 18px">
          <svg :key="t.rotations.length" viewBox="0 0 160 200" width="130" role="img" :aria-label="t.pattern + ' rotation pattern'" style="flex: none">
            <defs><marker id="arw" viewBox="0 0 10 10" refX="7" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" style="fill: var(--accent)" /></marker></defs>
            <rect x="56" y="14" width="48" height="172" rx="20" style="fill: var(--card2)" />
            <rect v-for="(p, i) in [[20, 22], [118, 22], [20, 132], [118, 132]]" :key="i" :x="p[0]" :y="p[1]" width="22" height="46" rx="6" style="fill: var(--s2)" />
            <text class="ax" x="31" y="14" text-anchor="middle">FL</text><text class="ax" x="129" y="14" text-anchor="middle">FR</text><text class="ax" x="31" y="194" text-anchor="middle">RL</text><text class="ax" x="129" y="194" text-anchor="middle">RR</text>
            <path v-for="(d, i) in ARROWS[t.pattern] || ARROWS['Forward cross']" :key="'a' + i" class="line-main" pathLength="1" :d="d" style="stroke-width: 2.4; marker-end: url(#arw)" :style="{ animationDelay: i * 0.15 + 's' }" />
          </svg>
          <div class="stack" style="gap: 6px"><div style="font-weight: 600">{{ t.pattern }}</div><div class="label">{{ v.drivetrain || 'Drivetrain not set' }} · suggested for your drivetrain</div><div class="label">{{ rotationItem ? 'Next rotation ' + (rotationItem.milesLeft != null ? (rotationItem.milesLeft < 0 ? 'overdue' : 'in ' + num(rotationItem.milesLeft) + ' ' + distUnit()) : 'on schedule') : 'Add a rotation reminder in Reminders' }}</div></div>
        </div>
        <div class="label">History</div>
        <div v-for="r in t.rotations.slice(0, 4)" :key="r.id" class="between listrow" style="font-size: 14px"><span>{{ fdate(r.date) }}</span><span class="muted">{{ num(r.odoReading) }} {{ distUnit() }}</span></div>
        <div v-if="!t.rotations.length" class="label">No rotations logged yet.</div>
      </div>

      <div class="card c8 stack" style="animation-delay: 0.1s">
        <div class="between wrap">
          <h2 class="ctitle">Tread depth over time</h2>
          <div class="row" style="gap: 14px; font-size: 12px; color: var(--muted)">
            <span class="row" style="gap: 6px"><span style="width: 18px; height: 3px; border-radius: 2px; background: var(--accent)"></span>Average</span>
            <span class="row" style="gap: 6px"><span style="width: 18px; border-top: 2px dashed var(--s2)"></span>Lowest corner</span>
          </div>
        </div>
        <svg v-if="chart" :key="readings.length" viewBox="0 0 720 240" class="chart" role="img" :aria-label="`Tread depth: average ${avgT?.toFixed(1)}/32`">
          <g v-for="g in chart.grid" :key="g.l"><line class="gl" x1="40" x2="680" :y1="g.y" :y2="g.y" /><text class="ax" x="30" :y="g.y + 4" text-anchor="end">{{ g.l }}</text></g>
          <line x1="40" x2="680" :y1="chart.y4" :y2="chart.y4" style="stroke: var(--warn); stroke-dasharray: 4 4; stroke-width: 1.2" /><text class="ax" x="684" :y="chart.y4 + 4" style="fill: var(--warn)">4/32 shop</text>
          <line x1="40" x2="680" :y1="chart.y2" :y2="chart.y2" style="stroke: var(--bad); stroke-dasharray: 4 4; stroke-width: 1.2" /><text class="ax" x="684" :y="chart.y2 + 4" style="fill: var(--bad)">2/32 legal</text>
          <path class="line-alt" :d="chart.min" />
          <path class="line-main" pathLength="1" :d="chart.avg" />
          <path v-if="chart.proj" class="area" :d="chart.proj" style="fill: none; stroke: var(--accent); stroke-width: 2; stroke-dasharray: 3 6; opacity: 0.8" />
          <circle v-if="chart.rep" class="area" :cx="chart.repX" :cy="chart.y4" r="6" style="fill: var(--card); stroke: var(--accent); stroke-width: 2" />
          <text v-for="x in chart.ticks" :key="x.x" class="ax" :x="x.x" y="236" text-anchor="middle">{{ x.l }}</text>
        </svg>
        <div class="label">{{ projText }}</div>
      </div>

      <div class="card c4 stack" style="animation-delay: 0.15s">
        <div class="between"><h2 class="ctitle">Log tread depth</h2><span class="label">{{ avgT == null ? '' : 'Health ' + Math.round(health) + '%' }}</span></div>
        <div class="label">In 32nds of an inch. A penny or a cheap gauge works. Pressure is optional.</div>
        <div class="grid2" style="gap: 10px">
          <label v-for="c in [['fl', 'pfl', 'FL · front'], ['fr', 'pfr', 'FR · front'], ['rl', 'prl', 'RL · rear'], ['rr', 'prr', 'RR · rear']]" :key="c[0]" class="field">{{ c[2] }}
            <div class="row" style="gap: 6px"><input v-model="tread[c[0]]" inputmode="decimal" :aria-label="c[2] + ' tread'" placeholder="32nds" /><input v-model="tread[c[1]]" inputmode="numeric" :aria-label="c[2] + ' pressure'" placeholder="psi" style="max-width: 70px" /></div>
          </label>
        </div>
        <div class="grid2"><label class="field">Date<input v-model="tread.date" type="date" /></label><label class="field">Odometer<input v-model="tread.odo" inputmode="numeric" :placeholder="num(t.odometer)" /></label></div>
        <button class="btn pri" :disabled="busy" @click="saveTread">Save reading</button>
      </div>
    </div>

    <Modal v-if="rot.open" title="Log rotation" @close="rot.open = false">
      <div class="label">{{ t.pattern }} for a {{ v.drivetrain || 'front-wheel drive' }} car. Hammond moves your last tread reading to the new corners and resets the rotation reminder.</div>
      <div class="grid2"><label class="field">Date<input v-model="rot.date" type="date" /></label><label class="field">Odometer<input v-model="rot.odo" inputmode="numeric" /></label></div>
      <button class="btn pri" :disabled="busy" @click="saveRotation">Save rotation</button>
    </Modal>

    <Modal v-if="setForm.open" :title="setForm.id ? 'Edit tire set' : 'Add a tire set'" @close="setForm.open = false">
      <div class="grid2"><label class="field">Name<input v-model="setForm.name" placeholder="Winter" /></label><label class="field">Brand &amp; model<input v-model="setForm.brand" /></label></div>
      <div class="grid2"><label class="field">Size<input v-model="setForm.size" /></label><label class="field">Stored at<input v-model="setForm.storedAt" placeholder="Garage shelf" /></label></div>
      <div class="grid2"><label class="field">{{ distUnit() }} already on these<input v-model="setForm.milesBefore" inputmode="numeric" /></label><label class="field">Tread when new (32nds)<input v-model="setForm.treadNew" inputmode="decimal" /></label></div>
      <div class="row"><button v-if="setForm.id" class="btn danger" @click="removeSet"><Icon name="trash" :size="18" />Delete</button><button class="btn pri grow" :disabled="busy" @click="saveSet">Save</button></div>
    </Modal>
  </section>
</template>
