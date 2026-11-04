<script setup>
import { ref, computed } from 'vue'
import { store, currentVehicle, selectVehicle, openEntry } from '../lib/store.js'
import { money, num, fdate, vname, fuelUnit, effUnit, distUnit, toDate } from '../lib/format.js'
import { useVehicleData } from '../lib/useData.js'
import { withEfficiency, expenseCategory } from '../lib/calc.js'
import Icon from '../components/Icon.vue'

const v = computed(() => currentVehicle())
const { fillups, expenses, loading } = useVehicleData(() => v.value?.id)
const filter = ref('All')
const q = ref('')
const limit = ref(60)

const rows = computed(() => {
  const effList = withEfficiency(fillups.value)
  const effMap = new Map(effList.map((f) => [f.id, f.eff]))
  const badSet = new Set(effList.filter((f) => f.bad).map((f) => f.id))
  const all = [
    ...fillups.value.map((f) => ({ id: f.id, kind: 'fillup', type: 'Fuel', rec: f, date: f.date, title: `${f.fillingStation || 'Fillup'} · ${num(f.fuelQuantity, 2)} ${fuelUnit(v.value)}`, odo: f.odoReading, detail: [badSet.has(f.id) ? '⚠ odometer lower than an earlier fillup' : '', effMap.get(f.id) ? `${effMap.get(f.id).toFixed(1)} ${effUnit(v.value)}` : '', `${money(f.perUnitPrice, 3)}/${fuelUnit(v.value)}`, f.isTankFull === false ? 'partial' : ''].filter(Boolean).join(' · '), who: f.user?.name, amount: f.totalAmount, color: 'var(--s1)' })),
    ...expenses.value.map((e) => {
      const cat = expenseCategory(e)
      const isS = ['Maintenance', 'Repairs', 'Tires'].includes(cat)
      return { id: e.id, kind: 'expense', type: isS ? 'Service' : 'Expense', rec: e, date: e.date, title: e.expenseType || 'Expense', odo: e.odoReading, detail: [cat, e.comments].filter(Boolean).join(' · '), who: e.user?.name, amount: e.amount, color: isS ? 'var(--s2)' : 'var(--s3)' }
    }),
  ].sort((a, b) => toDate(b.date) - toDate(a.date) || b.odo - a.odo)
  const s = q.value.trim().toLowerCase()
  return all.filter((r) => (filter.value === 'All' || r.type === filter.value) && (!s || (r.title + ' ' + r.detail).toLowerCase().includes(s)))
})
const counts = computed(() => {
  const c = { All: 0, Fuel: 0, Service: 0, Expense: 0 }
  fillups.value.forEach(() => { c.All++; c.Fuel++ })
  expenses.value.forEach((e) => { c.All++; c[['Maintenance', 'Repairs', 'Tires'].includes(expenseCategory(e)) ? 'Service' : 'Expense']++ })
  return c
})
function edit(r) { openEntry(r.kind === 'fillup' ? 'fillup' : r.type === 'Service' ? 'service' : 'expense', { vehicleId: v.value.id, record: r.rec }) }
</script>

<template>
  <section v-if="!v" class="card empty">Add a vehicle first.</section>
  <section v-else aria-label="Log" class="card stack" style="gap: 18px">
    <div class="row wrap" style="gap: 8px">
      <button v-for="l in ['All', 'Fuel', 'Service', 'Expense']" :key="l" class="chip" :class="{ on: filter === l }" @click="filter = l">{{ l }} <span style="opacity: 0.6">{{ counts[l] }}</span></button>
      <div class="grow"></div>
      <button v-for="x in store.vehicles" :key="x.id" class="chip" :class="{ on: x.id === v.id }" @click="selectVehicle(x.id)">{{ vname(x) }}</button>
    </div>
    <div class="row wrap">
      <label class="field grow" style="max-width: 360px"><span class="sr-only">Search</span><input v-model="q" placeholder="Search stations, repairs, notes…" /></label>
      <button class="btn pri" @click="openEntry('fillup')"><Icon name="add" :size="18" />Add entry</button>
    </div>
    <div class="scrollx log-table">
      <table class="tbl">
        <thead><tr><th scope="col">Date</th><th scope="col">Type</th><th scope="col">Description</th><th scope="col">Odometer</th><th scope="col">Details</th><th scope="col">By</th><th scope="col" style="text-align: right">Amount</th><th scope="col"><span class="sr-only">Edit</span></th></tr></thead>
        <tbody>
          <tr v-for="(r, i) in rows.slice(0, limit)" :key="r.id" :style="{ animationDelay: Math.min(i * 0.02, 0.5) + 's', cursor: 'pointer' }" @click="edit(r)">
            <td>{{ fdate(r.date) }}</td>
            <td><span class="badge"><span class="dot" :style="{ background: r.color }"></span>{{ r.type }}</span></td>
            <td>{{ r.title }}</td>
            <td class="muted">{{ r.odo ? num(r.odo) + ' ' + distUnit() : '—' }}</td>
            <td class="muted">{{ r.detail }}</td>
            <td class="muted">{{ r.who || '' }}</td>
            <td style="text-align: right; font-weight: 600">{{ money(r.amount) }}</td>
            <td style="text-align: right"><button class="btn ghost icon" style="min-height: 36px; width: 36px" :aria-label="'Edit ' + r.title" @click.stop="edit(r)"><Icon name="edit" :size="16" /></button></td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="log-cards">
      <button v-for="r in rows.slice(0, limit)" :key="'c' + r.id" class="rowbtn" :aria-label="'Edit ' + r.title" @click="edit(r)">
        <div class="tile-ic"><Icon :name="r.kind === 'fillup' ? 'pump' : r.type === 'Service' ? 'wrench' : 'receipt'" :size="18" /></div>
        <div class="grow">
          <div style="font-size: 14px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis">{{ r.title }}</div>
          <div class="label">{{ fdate(r.date) }} · {{ r.odo ? num(r.odo) + ' ' + distUnit() : 'no odometer' }}{{ r.detail ? ' · ' + r.detail : '' }}</div>
        </div>
        <div style="font-weight: 600; font-size: 14px">{{ money(r.amount) }}</div>
        <Icon name="edit" :size="16" style="color: var(--faint)" />
      </button>
    </div>
    <div v-if="!rows.length" class="empty">{{ loading ? 'Loading…' : 'Nothing here yet.' }}</div>
    <button v-if="rows.length > limit" class="btn ghost" style="align-self: center" @click="limit += 100">Show more ({{ rows.length - limit }} left)</button>
  </section>
</template>
