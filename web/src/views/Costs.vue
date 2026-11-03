<script setup>
import { ref, computed } from 'vue'
import { store, currentVehicle } from '../lib/store.js'
import { money, num, distUnit, vname, MO, monthYear, reminderStatusText } from '../lib/format.js'
import { useAllData } from '../lib/useData.js'
import { costOfOwnership, smooth, sum, niceMax } from '../lib/calc.js'
import { useTween } from '../lib/motion.js'
import Icon from '../components/Icon.vue'

const { data, loading } = useAllData()
const sel = ref('all')
const now = new Date()

const per = computed(() => store.vehicles.map((v) => ({ v, ...costOfOwnership(v, data.value[v.id]?.fillups || [], data.value[v.id]?.expenses || [], now) })))
const agg = computed(() => {
  if (sel.value !== 'all') return per.value.find((p) => p.v.id === sel.value)
  const out = { cats: {}, miles: 0, total: 0, monthly: [], v: null }
  for (const p of per.value) {
    for (const [k, x] of Object.entries(p.cats)) out.cats[k] = (out.cats[k] || 0) + x
    out.miles += p.miles; out.total += p.total
    p.monthly.forEach((m, i) => { out.monthly[i] = out.monthly[i] || { m: m.m, cost: 0, miles: 0 }; out.monthly[i].cost += m.cost; out.monthly[i].miles += m.miles })
  }
  out.cpm = out.miles ? out.total / out.miles : 0
  return out
})
const upcoming = computed(() => store.reminders
  .filter((r) => (r.kind === 'service' || r.kind === 'custom') && (sel.value === 'all' || r.vehicleId === sel.value) && r.estCost > 0 && ((r.daysLeft != null && r.daysLeft < 365) || (r.projectedDate && new Date(r.projectedDate) - now < 365 * 864e5)))
  .map((r) => ({ name: sel.value === 'all' ? `${r.title} · ${r.vehicleName}` : r.title, when: (r.milesLeft != null && r.milesLeft < 0) || (r.daysLeft != null && r.daysLeft < 0) ? 'overdue' : reminderStatusText(r).toLowerCase(), cost: r.estCost })))
const runRate = computed(() => { const c = agg.value?.cats || {}; return (c.Fuel || 0) + (c.Insurance || 0) + (c.Registration || 0) + (c.Depreciation || 0) + (c.Other || 0) })
const proj = computed(() => runRate.value + sum(upcoming.value, (u) => u.cost))

const cpmT = useTween(() => agg.value?.cpm || 0)
const totalT = useTween(() => agg.value?.total || 0)
const projT = useTween(() => proj.value)

const cats = computed(() => {
  const arr = Object.entries(agg.value?.cats || {}).filter(([, x]) => x > 0).sort((a, b) => b[1] - a[1])
  const max = arr[0]?.[1] || 1, tot = agg.value?.total || 1
  return arr.map(([k, x], i) => ({ k, x, w: (x / max) * 100, pct: Math.round((x / tot) * 100), op: 1 - i * 0.09, delay: 0.15 + i * 0.06 }))
})
const trend = computed(() => {
  const ms = agg.value?.monthly || []
  const mc = ms.map((m) => (m.miles > 0 ? m.cost / m.miles : 0))
  const roll = mc.map((x, i) => { const w = mc.slice(Math.max(0, i - 2), i + 1).filter((y) => y > 0); return w.length ? sum(w) / w.length : 0 })
  const max = niceMax(Math.max(...roll, 0.01) * 1.1)
  const X = (i) => 48 + (i / 11) * 462, Y = (y) => 196 - (y / max) * 180
  const pts = roll.map((y, i) => [X(i), Y(y)]).filter((p, i) => roll[i] > 0)
  const line = smooth(pts)
  return { line, area: pts.length > 1 ? `${line} L${pts[pts.length - 1][0]},196 L${pts[0][0]},196 Z` : '', grid: [0, 0.5, 1].map((t) => ({ y: Y(max * t), l: money(max * t, 2) })), months: ms.map((m, i) => ({ x: X(i), l: i % 2 ? '' : MO[m.m.getMonth()] })), last: roll[roll.length - 1] || 0, ok: pts.length > 1 }
})
const compare = computed(() => {
  const list = per.value.filter((p) => p.miles > 0)
  const max = Math.max(...list.map((p) => p.cpm), 0.01)
  return list.map((p) => ({ name: vname(p.v), cpm: p.cpm, w: (p.cpm / max) * 100, on: sel.value === 'all' || sel.value === p.v.id }))
})
const pdfVehicle = computed(() => (sel.value !== 'all' ? store.vehicles.find((v) => v.id === sel.value) : currentVehicle()))
</script>

<template>
  <section aria-label="Cost of ownership">
    <div class="chips" style="margin-bottom: 18px">
      <button class="chip" :class="{ on: sel === 'all' }" @click="sel = 'all'">All vehicles</button>
      <button v-for="v in store.vehicles" :key="v.id" class="chip" :class="{ on: sel === v.id }" @click="sel = v.id">{{ vname(v) }}</button>
    </div>
    <div v-if="loading && !per.length" class="card empty">Loading…</div>
    <div v-else class="g12" :key="sel">
      <div class="card c4"><div class="label">Cost per {{ distUnit() }}, all-in</div><div class="disp" style="font-size: 40px; font-weight: 600; margin-top: 6px">{{ money(cpmT, 2) }}</div><div class="label" style="margin-top: 4px">Fuel alone is {{ money((agg?.cats.Fuel || 0) / (agg?.miles || 1), 3) }} / {{ distUnit() }} · {{ num(agg?.miles) }} {{ distUnit() }} driven</div></div>
      <div class="card c4" style="animation-delay: 0.05s"><div class="label">Spent, last 12 months</div><div class="disp" style="font-size: 40px; font-weight: 600; margin-top: 6px">{{ money(totalT, 0) }}</div><div class="label" style="margin-top: 4px">{{ agg?.v?.purchasePrice ? `Bought ${monthYear(agg.v.purchaseDate)} for ${money(agg.v.purchasePrice, 0)}` : sel === 'all' ? `${store.vehicles.length} vehicles combined` : 'Includes insurance and depreciation when set' }}</div></div>
      <div class="card c4" style="animation-delay: 0.1s"><div class="label">Projected, next 12 months</div><div class="disp" style="font-size: 40px; font-weight: 600; margin-top: 6px">{{ money(projT, 0) }}</div><div class="label" style="margin-top: 4px">Current run rate plus {{ upcoming.length }} scheduled service{{ upcoming.length === 1 ? '' : 's' }}</div></div>

      <div class="card c7 stack" style="animation-delay: 0.15s">
        <h2 class="ctitle">Where the money goes · last 12 months</h2>
        <div v-for="c in cats" :key="c.k" style="display: grid; grid-template-columns: 110px 1fr 90px 40px; gap: 12px; align-items: center; font-size: 14px">
          <span>{{ c.k }}</span>
          <div class="bar" style="height: 10px"><div :style="{ width: c.w + '%', background: 'var(--accent)', opacity: c.op, animationDelay: c.delay + 's' }"></div></div>
          <span style="text-align: right; font-weight: 600">{{ money(c.x, 0) }}</span>
          <span class="label" style="text-align: right">{{ c.pct }}%</span>
        </div>
        <div v-if="!cats.length" class="label">Log fillups and expenses to see the breakdown.</div>
        <div class="label">Depreciation uses the purchase price and the value estimate on each vehicle. Insurance uses the monthly premium. <RouterLink v-if="sel !== 'all'" :to="`/vehicles/${sel}/edit`">Edit these</RouterLink></div>
      </div>

      <div class="card c5 stack" style="animation-delay: 0.2s">
        <h2 class="ctitle">Cost per {{ distUnit() }}, by month</h2>
        <svg v-if="trend.ok" viewBox="0 0 520 220" class="chart" role="img" :aria-label="`Cost per ${distUnit()} by month, 3-month rolling average, latest ${money(trend.last, 2)}`">
          <defs><linearGradient id="cpmFill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#FE4611" stop-opacity=".28" /><stop offset="1" stop-color="#FE4611" stop-opacity="0" /></linearGradient></defs>
          <g v-for="g in trend.grid" :key="g.y"><line class="gl" x1="48" x2="510" :y1="g.y" :y2="g.y" /><text class="ax" x="42" :y="g.y + 4" text-anchor="end">{{ g.l }}</text></g>
          <path class="area" :d="trend.area" fill="url(#cpmFill)" />
          <path class="line-main" pathLength="1" :d="trend.line" />
          <text v-for="m in trend.months" :key="m.x" class="ax" :x="m.x" y="214" text-anchor="middle">{{ m.l }}</text>
        </svg>
        <div v-else class="label">Needs a few months of fillups.</div>
        <div class="label">3-month rolling average. Bumps are months with a big repair.</div>
      </div>

      <div class="card c6 stack" style="gap: 10px; animation-delay: 0.25s">
        <h2 class="ctitle">Known costs coming up</h2>
        <div v-for="u in upcoming.slice(0, 8)" :key="u.name" class="listrow between" style="font-size: 14px"><span>{{ u.name }} <span class="label">· {{ u.when }}</span></span><span style="font-weight: 600">~{{ money(u.cost, 0) }}</span></div>
        <div v-if="!upcoming.length" class="label">Nothing with a cost estimate due in the next year.</div>
        <div class="label">Estimates come from each schedule item's typical cost. Edit them in Reminders.</div>
      </div>

      <div class="card c6 stack" style="animation-delay: 0.3s">
        <h2 class="ctitle">Compare vehicles · cost per {{ distUnit() }}</h2>
        <div v-for="c in compare" :key="c.name" style="display: grid; grid-template-columns: 110px 1fr 64px; gap: 12px; align-items: center; font-size: 14px">
          <span>{{ c.name }}</span>
          <div class="bar" style="height: 10px"><div :style="{ width: c.w + '%', background: c.on ? 'var(--accent)' : 'var(--s3)' }"></div></div>
          <span style="text-align: right; font-weight: 600">{{ money(c.cpm, 2) }}</span>
        </div>
        <div style="border-top: 1px solid var(--line); padding-top: 14px" class="stack">
          <div style="font-weight: 600">Selling or trading in?</div>
          <div class="label">{{ agg?.v?.purchasePrice ? `Bought for ${money(agg.v.purchasePrice, 0)} · estimated value ${money(agg.v.estimatedValue, 0)} · losing about ${money(agg.cats.Depreciation, 0)} a year` : 'A dated record of every service helps at resale.' }}</div>
          <RouterLink v-if="pdfVehicle" :to="`/report/${pdfVehicle.id}`" target="_blank" class="btn pri" style="align-self: flex-start"><Icon name="printer" :size="18" />Service history PDF · {{ vname(pdfVehicle) }}</RouterLink>
        </div>
      </div>
    </div>
  </section>
</template>
