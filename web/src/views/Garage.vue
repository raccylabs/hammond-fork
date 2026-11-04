<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { store, selectVehicle, openEntry, toast, refreshVehicles } from '../lib/store.js'
import { upload } from '../lib/api.js'
import { money, num, distUnit, fuelUnit, vname, reminderStatusText, LEVEL_COLOR, LEVEL_TEXT } from '../lib/format.js'
import { useAllData } from '../lib/useData.js'
import { sum, inMonth } from '../lib/calc.js'
import Icon from '../components/Icon.vue'
import VehiclePlate from '../components/VehiclePlate.vue'

const router = useRouter()
const { data } = useAllData()

const cards = computed(() => store.vehicles.map((v, i) => {
  const d = data.value[v.id] || { fillups: [], expenses: [] }
  const last = [...d.fillups].sort((a, b) => b.odoReading - a.odoReading)[0]
  const odo = Math.max(last?.odoReading || 0, ...d.expenses.map((e) => e.odoReading || 0))
  const rem = store.reminders.filter((r) => r.vehicleId === v.id && r.status !== 'snoozed')[0]
  const psi = v.tirePressureFront ? (v.tirePressureRear && v.tirePressureRear !== v.tirePressureFront ? `${v.tirePressureFront} / ${v.tirePressureRear} PSI` : `${v.tirePressureFront} PSI`) : '—'
  return { v, delay: i * 0.08, odo, last, psi, rem }
}))

const monthSpend = computed(() => {
  const now = new Date()
  return sum(Object.values(data.value), (d) => sum(d.fillups.filter((f) => inMonth(f.date, now)), (f) => f.totalAmount) + sum(d.expenses.filter((e) => inMonth(e.date, now)), (e) => e.amount))
})
const next = computed(() => store.reminders.find((r) => r.status !== 'snoozed' && (r.kind === 'service' || r.kind === 'custom')))
const recalls = computed(() => store.vehicles.filter((v) => v.recallCount > 0))

function open(v) { selectVehicle(v.id); router.push('/overview') }
async function photo(v, file, credit = '') {
  const fd = new FormData(); fd.append('file', file); fd.append('credit', credit || '')
  try { await upload(`/vehicles/${v.id}/photo`, fd); await refreshVehicles(); toast('Photo updated') } catch (x) { toast(x.message, true) }
}
</script>

<template>
  <section aria-label="Garage">
    <div v-if="!store.vehicles.length" class="card empty">
      <Icon name="garage" :size="40" />
      <div class="disp" style="font-size: 22px; color: var(--text)">Your garage is empty</div>
      <div>Add a vehicle with its VIN and Hammond fills in the details.</div>
      <button class="btn pri" @click="store.wizard = true">Add a vehicle</button>
    </div>
    <template v-else>
      <div class="card summary">
        <div class="sum-item"><div class="label">Vehicles</div><div class="sum-val">{{ store.vehicles.length }}</div></div>
        <div class="sum-item"><div class="label">This month</div><div class="sum-val">{{ money(monthSpend, 0) }}</div></div>
        <RouterLink to="/reminders" class="sum-item" style="color: inherit">
          <div class="label">Next service</div>
          <div class="sum-val sm" :style="{ color: next && next.level > 0 ? LEVEL_TEXT[next.level] : 'var(--text)' }">{{ next ? next.title : 'None' }}</div>
          <div class="label sum-sub">{{ next ? `${next.vehicleName} · ${reminderStatusText(next).toLowerCase()}` : 'Set up in Reminders' }}</div>
        </RouterLink>
        <RouterLink to="/reminders" class="sum-item" style="color: inherit">
          <div class="label">Open recalls</div>
          <div class="sum-val sm row" style="gap: 6px"><Icon :name="recalls.length ? 'alert' : 'shield'" :size="16" />{{ recalls.length ? `${recalls.reduce((a, v) => a + v.recallCount, 0)} open` : 'None' }}</div>
          <div class="label sum-sub">{{ recalls.length ? recalls.map(vname).join(', ') : 'Checked weekly' }}</div>
        </RouterLink>
      </div>

      <div class="scroller">
        <article v-for="c in cards" :key="c.v.id" class="card" :style="{ flex: '0 0 min(400px, 86vw)', padding: '14px', display: 'flex', flexDirection: 'column', gap: '14px', borderColor: c.v.id === store.vid ? 'var(--accent)' : 'var(--line)', animationDelay: c.delay + 's' }">
          <div style="background: var(--card2); border-radius: 16px; padding: 18px 18px 20px">
            <div class="label">Vehicle · {{ c.v.yearOfManufacture || '' }}</div>
            <VehiclePlate :vehicle="c.v" :height="190" editable style="margin: 12px -4px 16px" @upload="(f, cr) => photo(c.v, f, cr)" />
            <div class="gfacts">
              <div class="label" style="grid-area: mk">{{ c.v.make }}</div>
              <div class="disp gname" style="grid-area: nm">{{ c.v.nickname && c.v.nickname !== c.v.make + ' ' + c.v.model ? c.v.nickname : c.v.model }}</div>
              <div class="label" style="grid-area: l1">Mileage</div>
              <div class="val" style="grid-area: v1; font-size: 17px; line-height: 1">{{ c.odo ? num(c.odo) : '—' }}</div>
              <div class="label" style="grid-area: l2">Latest fillup</div>
              <div class="val" style="grid-area: v2; font-size: 17px; line-height: 1">{{ c.last ? `${num(c.last.fuelQuantity, 1)} ${fuelUnit(c.v)}` : 'None yet' }}</div>
            </div>
          </div>
          <div style="background: var(--card2); border-radius: 16px; padding: 18px">
            <h3 class="ctitle">Vehicle info</h3>
            <div class="grid3" style="margin-top: 14px">
              <div><div class="label">Oil type</div><div style="margin-top: 4px">{{ c.v.oilType || '—' }}</div></div>
              <div><div class="label">Tire size</div><div style="margin-top: 4px">{{ c.v.tireSize || '—' }}</div></div>
              <div><div class="label">Tire pressure</div><div style="margin-top: 4px">{{ c.psi }}</div></div>
            </div>
            <div v-if="c.rem" class="row" style="margin-top: 16px; gap: 8px; font-size: 13px"><span class="dot" :style="{ background: LEVEL_COLOR[c.rem.level] }"></span><span class="muted">{{ c.rem.title }} · {{ reminderStatusText(c.rem).toLowerCase() }}</span></div>
          </div>
          <div class="row" style="padding: 0 4px 4px; gap: 10px">
            <button class="btn out grow" @click="openEntry('fillup', { vehicleId: c.v.id })"><Icon name="pump" :size="20" />Quick add</button>
            <button class="btn ghost grow" @click="open(c.v)">Full details</button>
          </div>
        </article>
        <button class="card" style="flex: 0 0 min(300px, 80vw); border: 1.5px dashed var(--line2); background: transparent; color: var(--muted); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; cursor: pointer; font: inherit; text-align: center; animation-delay: 0.3s" @click="store.wizard = true">
          <Icon name="plus" :size="36" />
          <span style="font-size: 17px; color: var(--text); font-weight: 600">Add a vehicle</span>
          <span style="font-size: 13px; max-width: 220px">Enter a VIN and Hammond fills in the make, model, engine and drivetrain.</span>
        </button>
      </div>
    </template>
  </section>
</template>

<style scoped>
.gfacts { display: grid; grid-template-columns: minmax(0, 1fr) auto auto; grid-template-areas: 'mk l1 l2' 'nm v1 v2'; column-gap: 18px; row-gap: 4px; align-items: baseline; }
.gname { font-size: 32px; font-weight: 600; line-height: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
@media (max-width: 820px) {
  .gfacts { grid-template-columns: 1fr 1fr; grid-template-areas: 'mk mk' 'nm nm' 'l1 l2' 'v1 v2'; }
  .gfacts > :nth-child(3), .gfacts > :nth-child(5) { margin-top: 10px; }
  .gname { font-size: 30px; }
}
.summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); padding: 0; margin-bottom: 22px; }
.sum-item { padding: 16px 20px; min-width: 0; border-left: 1px solid var(--line); }
.sum-item:first-child { border-left: none; }
.sum-val { font-size: 24px; font-weight: 600; font-stretch: 118%; margin-top: 4px; line-height: 1.15; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sum-val.sm { font-size: 17px; font-stretch: 100%; }
.sum-sub { margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
@media (max-width: 820px) {
  .summary { grid-template-columns: repeat(2, minmax(0, 1fr)); margin-bottom: 16px; }
  .sum-item { padding: 12px 14px; }
  .sum-item:nth-child(3) { border-left: none; }
  .sum-item:nth-child(n + 3) { border-top: 1px solid var(--line); }
  .sum-val { font-size: 20px; }
  .sum-val.sm { font-size: 15px; }
}
</style>
