<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { store } from '../lib/store.js'
import { get } from '../lib/api.js'
import { money, num, fdate, fdateLong, vfull, distUnit, monthYear, toDate } from '../lib/format.js'
import { expenseCategory } from '../lib/calc.js'
import Brand from '../components/Brand.vue'
import Icon from '../components/Icon.vue'

const route = useRoute()
const v = ref(null)
const expenses = ref([])
const fills = ref([])
const rotations = ref([])
onMounted(async () => {
  const id = route.params.id
  v.value = store.vehicles.find((x) => x.id === id) || (await get(`/vehicles/${id}`))
  const [e, f, t] = await Promise.all([get(`/vehicles/${id}/expenses`), get(`/vehicles/${id}/fillups`), get(`/vehicles/${id}/tires`).catch(() => null)])
  expenses.value = e || []
  fills.value = f || []
  rotations.value = t?.rotations || []
})
const rows = computed(() => [
  ...expenses.value.filter((e) => ['Maintenance', 'Repairs', 'Tires'].includes(expenseCategory(e))).map((e) => ({ date: e.date, odo: e.odoReading, title: e.expenseType, type: expenseCategory(e), notes: e.comments, cost: e.amount })),
  ...rotations.value.map((r) => ({ date: r.date, odo: r.odoReading, title: 'Tire rotation · ' + r.pattern.toLowerCase(), type: 'Tires', notes: '', cost: null })),
].sort((a, b) => toDate(b.date) - toDate(a.date)))
const odo = computed(() => Math.max(0, ...fills.value.map((f) => f.odoReading), ...expenses.value.map((e) => e.odoReading || 0)))
const total = computed(() => rows.value.reduce((a, r) => a + (r.cost || 0), 0))
</script>

<template>
  <div style="max-width: 860px; margin: 0 auto; padding: 24px 16px 60px" class="stack">
    <div class="between no-print"><RouterLink to="/costs" class="btn ghost">Back</RouterLink><button class="btn pri" onclick="window.print()"><Icon name="printer" :size="18" />Print or save as PDF</button></div>
    <div v-if="v" class="paper">
      <div class="between" style="align-items: flex-start; border-bottom: 2px solid #fe4611; padding-bottom: 14px; margin-bottom: 16px">
        <Brand :width="140" color="#231F20" />
        <div style="text-align: right; color: #5f5957">Service history report<br />Generated {{ fdateLong(new Date()) }}</div>
      </div>
      <div style="display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 18px">
        <div><div style="color: #5f5957">Vehicle</div><div style="font-weight: 600; font-size: 13px">{{ vfull(v) }} {{ v.trim }}</div></div>
        <div><div style="color: #5f5957">VIN</div><div style="font-weight: 600; font-size: 13px">{{ v.vin || '—' }}</div></div>
        <div><div style="color: #5f5957">Odometer</div><div style="font-weight: 600; font-size: 13px">{{ num(odo) }} {{ distUnit() }}</div></div>
        <div><div style="color: #5f5957">Owned since</div><div style="font-weight: 600; font-size: 13px">{{ v.purchaseDate && !String(v.purchaseDate).startsWith('0001') ? monthYear(v.purchaseDate) : '—' }}</div></div>
      </div>
      <table>
        <thead><tr><th>Date</th><th>Odometer</th><th>Work performed</th><th>Type</th><th>Notes</th><th style="text-align: right">Cost</th></tr></thead>
        <tbody>
          <tr v-for="(r, i) in rows" :key="i"><td>{{ fdate(r.date) }}</td><td>{{ r.odo ? num(r.odo) : '—' }}</td><td>{{ r.title }}</td><td>{{ r.type }}</td><td>{{ r.notes }}</td><td style="text-align: right">{{ r.cost != null ? money(r.cost) : '—' }}</td></tr>
          <tr v-if="!rows.length"><td colspan="6" style="color: #8a8482">No service records yet.</td></tr>
        </tbody>
      </table>
      <div class="between" style="margin-top: 16px; font-weight: 600"><span>{{ rows.length }} service records · {{ fills.length }} fillups logged</span><span>Total {{ money(total) }}</span></div>
      <div style="margin-top: 18px; color: #77706d; font-size: 10px">Records entered by the owner in Hammond, a self-hosted vehicle log.</div>
    </div>
  </div>
</template>
