<script setup>
import { ref, computed, watch } from 'vue'
import { store, currentVehicle, selectVehicle, openEntry } from '../lib/store.js'
import { get } from '../lib/api.js'
import { money, num, fdate, MO, vname, effUnit, fuelUnit, distUnit, reminderStatusText, LEVEL_COLOR, LEVEL_TEXT, toDate } from '../lib/format.js'
import { useVehicleData } from '../lib/useData.js'
import { withEfficiency, smooth, spark, sum, avg, monthsBack, inMonth, expenseCategory, niceMax } from '../lib/calc.js'
import { useTween } from '../lib/motion.js'
import Icon from '../components/Icon.vue'
import Ring from '../components/Ring.vue'
import VehiclePlate from '../components/VehiclePlate.vue'

const v = computed(() => currentVehicle())
const { fillups, expenses, loading } = useVehicleData(() => v.value?.id)
const period = ref('1Y')
const hover = ref(-1)
const hoverBar = ref(-1)
const tires = ref(null)
const docs = ref([])
watch([() => v.value?.id, () => store.dataVersion], async ([id]) => {
  if (!id) return
  get(`/vehicles/${id}/tires`).then((t) => (tires.value = t)).catch(() => (tires.value = null))
  get(`/vehicles/${id}/documents`).then((d) => (docs.value = d || [])).catch(() => (docs.value = []))
}, { immediate: true })

const now = new Date()
const eff = computed(() => withEfficiency(fillups.value))
const odo = computed(() => Math.max(0, ...fillups.value.map((f) => f.odoReading), ...expenses.value.map((e) => e.odoReading || 0)))
const lastFill = computed(() => eff.value[eff.value.length - 1])

// ---------------- efficiency chart ----------------
const W0 = 40, W1 = 680, H0 = 14, H1 = 222
const chart = computed(() => {
  const months = period.value === '3M' ? 3 : period.value === '6M' ? 6 : period.value === '1Y' ? 12 : 600
  const cutoff = new Date(now); cutoff.setMonth(cutoff.getMonth() - months)
  const pts = eff.value.filter((f) => f.eff && toDate(f.date) >= cutoff)
  if (pts.length < 2) return null
  const t0 = Math.min(cutoff.getTime(), toDate(pts[0].date).getTime()), t1 = now.getTime()
  const start = period.value === 'All' ? toDate(pts[0].date).getTime() : t0
  const effs = pts.map((p) => p.eff), ppus = pts.map((p) => p.perUnitPrice)
  const mMin = Math.floor(Math.min(...effs) - 2), mMax = Math.ceil(Math.max(...effs) + 2)
  const pMin = Math.min(...ppus) * 0.92, pMax = Math.max(...ppus) * 1.08
  const X = (d) => W0 + ((toDate(d).getTime() - start) / (t1 - start)) * (W1 - W0)
  const Y = (m) => H1 - ((m - mMin) / (mMax - mMin || 1)) * (H1 - H0)
  const YP = (p) => H1 - ((p - pMin) / (pMax - pMin || 1)) * (H1 - H0)
  const mp = pts.map((p) => [X(p.date), Y(p.eff)])
  const line = smooth(mp)
  const average = avg(effs)
  const grid = [0, 1, 2, 3].map((i) => {
    const m = mMin + ((mMax - mMin) * i) / 3, p = pMin + ((pMax - pMin) * i) / 3
    return { y: Y(m), l: Math.round(m), r: p.toFixed(2) }
  })
  const ticks = []
  const md = new Date(start); md.setDate(1); md.setMonth(md.getMonth() + 1)
  const span = (t1 - start) / 2.6e9
  const step = span > 24 ? 6 : span > 12 ? 3 : span > 6 ? 2 : 1
  while (md.getTime() <= t1) { ticks.push({ x: X(md), l: MO[md.getMonth()] + (md.getMonth() === 0 && span > 12 ? ` '${String(md.getFullYear()).slice(2)}` : '') }); md.setMonth(md.getMonth() + step) }
  const hit = mp.map((p, i) => {
    const prev = i ? (p[0] + mp[i - 1][0]) / 2 : W0 - 10, next = i < mp.length - 1 ? (p[0] + mp[i + 1][0]) / 2 : W1 + 10
    return { x: prev, w: Math.max(2, next - prev) }
  })
  return { pts, mp, line, area: `${line} L${mp[mp.length - 1][0].toFixed(1)},${H1} L${mp[0][0].toFixed(1)},${H1} Z`, ppu: smooth(pts.map((p) => [X(p.date), YP(p.perUnitPrice)])), grid, ticks, hit, avgY: Y(average), average, best: Math.max(...effs) }
})
const hi = computed(() => (chart.value ? (hover.value >= 0 && hover.value < chart.value.pts.length ? hover.value : chart.value.pts.length - 1) : -1))
const tip = computed(() => {
  if (!chart.value || hi.value < 0) return null
  const p = chart.value.pts[hi.value], [x, y] = chart.value.mp[hi.value]
  const left = (x / 720) * 100
  return { left, top: (y / 250) * 100, tx: left > 70 ? '-100%' : left < 25 ? '0%' : '-50%', p, x }
})
const avgEff = useTween(() => chart.value?.average || 0)

// ---------------- KPIs ----------------
const ytd = computed(() => fillups.value.filter((f) => toDate(f.date).getFullYear() === now.getFullYear()))
const fuelYTD = useTween(() => sum(ytd.value, (f) => f.totalAmount))
const avgFill = useTween(() => avg(ytd.value, (f) => f.totalAmount))
const avgPPU = useTween(() => avg(ytd.value, (f) => f.perUnitPrice))
const cpmVal = computed(() => {
  const from = new Date(now); from.setFullYear(from.getFullYear() - 1)
  const fy = fillups.value.filter((f) => toDate(f.date) >= from).sort((a, b) => a.odoReading - b.odoReading)
  const d = fy.length > 1 ? fy[fy.length - 1].odoReading - fy[0].odoReading : 0
  const cost = sum(fy, (f) => f.totalAmount) + sum(expenses.value.filter((e) => toDate(e.date) >= from), (e) => e.amount)
  return d ? cost / d : 0
})
const cpm = useTween(() => cpmVal.value)
const last8 = computed(() => eff.value.slice(-8))
const kpis = computed(() => {
  const lp = last8.value[last8.value.length - 1]
  const ap = avg(ytd.value, (f) => f.perUnitPrice)
  return [
    { label: 'Fuel spend this year', value: money(fuelYTD.value), spark: spark(last8.value.map((f) => f.totalAmount)), sub: `${ytd.value.length} fillups` },
    { label: 'Average fillup', value: money(avgFill.value), spark: spark(last8.value.map((f) => f.fuelQuantity)), sub: `${num(avg(ytd.value, (f) => f.fuelQuantity), 2)} ${fuelUnit(v.value)} on average` },
    { label: `Average price / ${fuelUnit(v.value)}`, value: money(avgPPU.value, 3), spark: spark(last8.value.map((f) => f.perUnitPrice)), sub: lp ? `${lp.perUnitPrice < ap ? '▼' : '▲'} Last paid ${money(lp.perUnitPrice, 3)}` : '', warn: lp && lp.perUnitPrice > ap },
    { label: `Cost per ${distUnit()}`, value: money(cpm.value, 3), spark: spark(last8.value.filter((f) => f.eff).map((f) => f.perUnitPrice / f.eff)), sub: 'Fuel + expenses, last 12 months' },
  ]
})

// ---------------- services ----------------
const vrem = computed(() => store.reminders.filter((r) => r.vehicleId === v.value?.id))
const services = computed(() => vrem.value.filter((r) => r.kind === 'service' || r.kind === 'custom').slice(0, 5))
const recallRem = computed(() => vrem.value.find((r) => r.kind === 'recall'))
const rings = computed(() => vrem.value.filter((r) => (r.kind === 'service' || r.kind === 'custom') && r.status !== 'snoozed').slice(0, 4).map((r, i) => {
  const left = Math.max(0, Math.round((1 - Math.min(1, r.pct)) * 100))
  return { r, left, color: left < 15 ? 'var(--bad)' : left < 35 ? 'var(--warn)' : 'var(--accent)', delay: 0.2 + i * 0.1 }
}))

// ---------------- spending ----------------
const spend = computed(() => {
  const ms = monthsBack(12, now).map((m) => {
    const fuel = sum(fillups.value.filter((f) => inMonth(f.date, m)), (f) => f.totalAmount)
    const ex = expenses.value.filter((e) => inMonth(e.date, m))
    const service = sum(ex.filter((e) => ['Maintenance', 'Repairs', 'Tires'].includes(expenseCategory(e))), (e) => e.amount)
    const other = sum(ex, (e) => e.amount) - service + (v.value?.insurancePremium || 0)
    return { m, fuel, service, other, tot: fuel + service + other }
  })
  const max = niceMax(Math.max(...ms.map((x) => x.tot), 1))
  const SB = 180, ST = 8, bw = 38
  const bars = ms.map((x, i) => {
    const h = (n) => (n / max) * (SB - ST)
    const h1 = h(x.fuel), h2 = h(x.service), h3 = h(x.other)
    const bx = 52 + i * ((716 - 52) / 12) + 6
    return { ...x, x: bx, w: bw, cx: bx + bw / 2, y1: SB - h1, h1: Math.max(0, h1 - 1), y2: SB - h1 - h2, h2: Math.max(0, h2 - (h2 > 2 ? 2 : 0)), y3: SB - h1 - h2 - h3, h3: Math.max(0, h3 - 2), l: MO[x.m.getMonth()], delay: i * 0.04 }
  })
  const total = sum(ms, (x) => x.tot)
  const pct = (k) => (total ? Math.round((sum(ms, (x) => x[k]) / total) * 100) : 0)
  return { bars, max, total, grid: [0, 0.5, 1].map((t) => ({ y: SB - t * (SB - ST), l: money(max * t, 0) })), breakdown: `Fuel ${pct('fuel')}% · Service ${pct('service')}% · Other ${pct('other')}%` }
})
const spendTotal = useTween(() => spend.value.total)
const hb = computed(() => (hoverBar.value >= 0 ? spend.value.bars[hoverBar.value] : null))

// ---------------- tires ----------------
const tireView = computed(() => {
  const t = tires.value
  if (!t) return null
  const set = t.sets.find((s) => s.isMounted)
  const rs = t.readings.filter((r) => r.tireSetId === set?.id)
  const r = rs[rs.length - 1]
  const tread = r ? [r.fl, r.fr, r.rl, r.rr] : null
  const pf = v.value?.tirePressureFront, pr = v.value?.tirePressureRear || pf
  const psi = r && r.pressureFl ? [r.pressureFl, r.pressureFr, r.pressureRl, r.pressureRr] : [pf, pf, pr, pr]
  const col = (x) => (x == null ? 'var(--s3)' : x <= 3 ? 'var(--bad)' : x <= 5 ? 'var(--warn)' : 'var(--s2)')
  const pos = [[40, 52, 30, 'end'], [164, 52, 190, 'start'], [40, 140, 30, 'end'], [164, 140, 190, 'start']]
  const list = pos.map((p, i) => ({ x: p[0] + (i % 2 ? -10 : 14), y: p[1], tx: p[2], ty: p[1] + 18, ty2: p[1] + 34, anchor: p[3], tread: tread ? `${tread[i]}/32"` : '—', psi: psi[i] ? `${psi[i]} psi` : '', color: col(tread?.[i]) }))
  const health = tread ? Math.round((tread.reduce((a, x) => a + Math.max(0, (x - 2) / ((set?.treadNew || 10) - 2)), 0) / 4) * 100) : null
  return { list, health, set, reading: r, minT: tread ? Math.min(...tread) : null }
})
const healthT = useTween(() => tireView.value?.health || 0)

// ---------------- activity ----------------
const recent = computed(() => [
  ...fillups.value.map((f) => ({ date: f.date, icon: 'pump', title: `${f.fillingStation || 'Fillup'} · ${num(f.fuelQuantity, 2)} ${fuelUnit(v.value)}`, who: f.user?.name, amount: f.totalAmount })),
  ...expenses.value.map((e) => ({ date: e.date, icon: expenseCategory(e) === 'Maintenance' ? 'wrench' : 'receipt', title: e.expenseType || 'Expense', who: e.user?.name, amount: e.amount })),
].sort((a, b) => toDate(b.date) - toDate(a.date)).slice(0, 4))
const docBadges = computed(() => docs.value.map((d) => {
  const left = d.expiryDate ? Math.round((toDate(d.expiryDate) - now) / 86400000) : null
  return { name: d.title || d.type, exp: d.expiryDate ? 'exp ' + fdate(d.expiryDate) : 'on file', color: left == null ? 'var(--s2)' : left < 0 ? 'var(--bad)' : left < 45 ? 'var(--warn)' : 'var(--s2)' }
}))
</script>

<template>
  <section v-if="!v" class="card empty"><Icon name="garage" :size="40" /><div>Add a vehicle to see its overview.</div><button class="btn pri" @click="store.wizard = true">Add a vehicle</button></section>
  <section v-else aria-label="Vehicle overview">
    <div class="chips" style="margin-bottom: 18px">
      <button v-for="x in store.vehicles" :key="x.id" class="chip" :class="{ on: x.id === v.id }" @click="selectVehicle(x.id)">{{ vname(x) }}</button>
    </div>

    <div v-if="recallRem" class="card row wrap" style="margin-bottom: 18px; border-color: var(--accent)">
      <Icon name="alert" :size="28" style="color: var(--accentText)" />
      <div class="grow" style="min-width: 220px"><div style="font-weight: 600">{{ recallRem.title }} for this model year</div><div class="label">From NHTSA. Repairs are free at any dealer. Dismiss any that don't apply or are already fixed.</div></div>
      <RouterLink to="/reminders" class="btn pri">Review recalls</RouterLink>
    </div>

    <div class="g12" :key="v.id">
      <div class="card c5 stack" style="gap: 16px">
        <div class="between"><h2 class="ctitle">Vehicle</h2><RouterLink :to="`/vehicles/${v.id}/edit`" class="chip"><Icon name="edit" :size="14" />Edit</RouterLink></div>
        <VehiclePlate :vehicle="v" :height="230" />
        <div style="display: grid; grid-template-columns: 1.4fr 1fr 1fr; gap: 12px; align-items: end">
          <div><div class="label">{{ v.make }} · {{ v.yearOfManufacture }}</div><div class="disp" style="font-size: 28px; font-weight: 600; line-height: 1.05">{{ v.model }}</div></div>
          <div><div class="label">Mileage</div><div class="val">{{ odo ? num(odo) : '—' }}</div></div>
          <div><div class="label">Latest fillup</div><div class="val">{{ lastFill ? `${num(lastFill.fuelQuantity, 1)} ${fuelUnit(v)}` : '—' }}</div></div>
        </div>
        <div class="grid3" style="padding-top: 14px; border-top: 1px solid var(--line)">
          <div><div class="label">Oil</div><div style="margin-top: 4px">{{ v.oilType || '—' }}</div></div>
          <div><div class="label">Tires</div><div style="margin-top: 4px">{{ v.tireSize || '—' }}</div></div>
          <div><div class="label">Pressure</div><div style="margin-top: 4px">{{ v.tirePressureFront ? `${v.tirePressureFront}${v.tirePressureRear && v.tirePressureRear !== v.tirePressureFront ? ' / ' + v.tirePressureRear : ''} PSI` : '—' }}</div></div>
        </div>
      </div>

      <div class="card c7 stack" style="animation-delay: 0.05s">
        <div class="between wrap">
          <h2 class="ctitle">Fuel efficiency</h2>
          <div class="seg" role="group" aria-label="Time range"><button v-for="p in ['3M', '6M', '1Y', 'All']" :key="p" :class="{ on: period === p }" @click="period = p; hover = -1">{{ p }}</button></div>
        </div>
        <template v-if="chart">
          <div class="row wrap" style="gap: 28px">
            <div><div class="label">Average</div><div class="big">{{ avgEff.toFixed(1) }} <span style="font-size: 14px; color: var(--muted); font-weight: 400">{{ effUnit(v) }}</span></div></div>
            <div><div class="label">Best</div><div class="val" style="margin-top: 8px">{{ chart.best.toFixed(1) }}</div></div>
            <div><div class="label">Last</div><div class="val" style="margin-top: 8px">{{ chart.pts[chart.pts.length - 1].eff.toFixed(1) }}</div></div>
            <div class="row" style="margin-left: auto; gap: 16px; font-size: 12px; color: var(--muted)">
              <span class="row" style="gap: 6px"><span style="width: 18px; height: 3px; border-radius: 2px; background: var(--accent)"></span>{{ effUnit(v) }}</span>
              <span class="row" style="gap: 6px"><span style="width: 18px; border-top: 2px dashed var(--s2)"></span>Price / {{ fuelUnit(v) }}</span>
            </div>
          </div>
          <div style="position: relative" @mouseleave="hover = -1">
            <svg :key="period + v.id + store.dataVersion" viewBox="0 0 720 250" class="chart" role="img" :aria-label="`Fuel efficiency per fillup, average ${chart.average.toFixed(1)} ${effUnit(v)} over ${chart.pts.length} fillups`">
              <defs><linearGradient id="effFill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#FE4611" stop-opacity=".32" /><stop offset="1" stop-color="#FE4611" stop-opacity="0" /></linearGradient></defs>
              <g v-for="g in chart.grid" :key="g.y"><line class="gl" x1="40" x2="680" :y1="g.y" :y2="g.y" /><text class="ax" x="30" :y="g.y + 4" text-anchor="end">{{ g.l }}</text><text class="ax" x="688" :y="g.y + 4">{{ g.r }}</text></g>
              <text v-for="t in chart.ticks" :key="t.x" class="ax" :x="t.x" y="246" text-anchor="middle">{{ t.l }}</text>
              <path class="area" :d="chart.area" fill="url(#effFill)" />
              <line class="avgl" x1="40" x2="680" :y1="chart.avgY" :y2="chart.avgY" />
              <path class="line-alt" :d="chart.ppu" />
              <path class="line-main" pathLength="1" :d="chart.line" />
              <line v-if="tip" :x1="tip.x" :x2="tip.x" y1="10" y2="226" style="stroke: var(--line2)" />
              <circle v-for="(p, i) in chart.mp" :key="i" class="pt" :cx="p[0]" :cy="p[1]" :r="i === hi ? 6 : chart.mp.length > 40 ? 0 : 3" />
              <rect v-for="(h, i) in chart.hit" :key="'h' + i" :x="h.x" y="0" :width="h.w" height="230" fill="transparent" style="cursor: crosshair" @mouseenter="hover = i" @touchstart.passive="hover = i" />
            </svg>
            <div v-if="tip" class="tip" :style="{ left: tip.left + '%', top: tip.top + '%', transform: `translate(${tip.tx}, -120%)` }">
              <div style="font-weight: 700; font-size: 14px">{{ tip.p.eff.toFixed(1) }} {{ effUnit(v) }}</div>
              <div style="opacity: 0.75">{{ fdate(tip.p.date) }} · {{ tip.p.fillingStation || 'Fillup' }}</div>
              <div style="opacity: 0.75">{{ num(tip.p.fuelQuantity, 2) }} {{ fuelUnit(v) }} @ {{ money(tip.p.perUnitPrice, 3) }} = {{ money(tip.p.totalAmount) }}</div>
            </div>
          </div>
        </template>
        <div v-else class="empty" style="padding: 40px 10px">
          <Icon name="pump" :size="32" />
          <div>{{ loading ? 'Loading…' : 'Log two full-tank fillups and your efficiency chart appears here.' }}</div>
          <button v-if="!loading" class="btn out" @click="openEntry('fillup')">Add a fillup</button>
        </div>
      </div>

      <div v-for="(t, i) in kpis" :key="t.label" class="card c3 stack" style="gap: 8px" :style="{ animationDelay: 0.05 + i * 0.05 + 's' }">
        <div class="label">{{ t.label }}</div>
        <div class="between" style="align-items: flex-end">
          <div class="disp" style="font-size: 26px; font-weight: 600">{{ t.value }}</div>
          <svg viewBox="0 0 100 32" width="96" height="32" aria-hidden="true"><path :key="store.dataVersion + v.id" class="line-main" pathLength="1" :d="t.spark" style="stroke-width: 2" /></svg>
        </div>
        <div style="font-size: 12px" :style="{ color: t.warn ? 'var(--accentText)' : 'var(--muted)' }">{{ t.sub }}</div>
      </div>

      <div class="card c5 stack" style="gap: 16px; animation-delay: 0.1s">
        <div class="between"><h2 class="ctitle">Upcoming service</h2><button class="chip" @click="openEntry('service')">Log service</button></div>
        <div v-for="(s, i) in services" :key="s.key" class="stack" style="gap: 8px">
          <div class="between" style="font-size: 14px"><span class="row" style="gap: 8px"><span class="dot" :style="{ background: s.status === 'snoozed' ? 'var(--line2)' : LEVEL_COLOR[s.level] }"></span>{{ s.title }}</span><span :style="{ color: s.status === 'snoozed' ? 'var(--muted)' : LEVEL_TEXT[s.level], fontWeight: 600 }">{{ reminderStatusText(s) }}</span></div>
          <div class="bar"><div :style="{ width: Math.min(100, s.pct * 100) + '%', background: LEVEL_COLOR[s.level], animationDelay: 0.2 + i * 0.08 + 's' }"></div></div>
        </div>
        <div v-if="!services.length" class="label">No maintenance schedule yet. <RouterLink to="/reminders">Set one up</RouterLink>.</div>
      </div>

      <div class="card c7 stack" style="animation-delay: 0.15s">
        <div class="between wrap">
          <h2 class="ctitle">Spending · last 12 months</h2>
          <div class="row" style="gap: 14px; font-size: 12px; color: var(--muted)">
            <span class="row" style="gap: 6px"><span class="dot" style="background: var(--s1)"></span>Fuel</span>
            <span class="row" style="gap: 6px"><span class="dot" style="background: var(--s2)"></span>Service</span>
            <span class="row" style="gap: 6px"><span class="dot" style="background: var(--s3)"></span>Other</span>
          </div>
        </div>
        <div class="row wrap" style="gap: 28px; align-items: baseline">
          <div><div class="label">{{ hb ? `${MO[hb.m.getMonth()]} ${hb.m.getFullYear()}` : 'Total, 12 months' }}</div><div class="big">{{ hb ? money(hb.tot) : money(spendTotal, 0) }}</div></div>
          <div class="muted" style="font-size: 13px">{{ hb ? `Fuel ${money(hb.fuel)} · Service ${money(hb.service)} · Other ${money(hb.other)}` : spend.breakdown }}</div>
        </div>
        <svg :key="'s' + v.id + store.dataVersion" viewBox="0 0 720 200" class="chart" role="img" aria-label="Monthly spending stacked by fuel, service and other" @mouseleave="hoverBar = -1">
          <g v-for="g in spend.grid" :key="g.y"><line class="gl" x1="48" x2="716" :y1="g.y" :y2="g.y" /><text class="ax" x="42" :y="g.y + 4" text-anchor="end">{{ g.l }}</text></g>
          <g v-for="(b, i) in spend.bars" :key="i" :style="{ opacity: hoverBar < 0 || hoverBar === i ? 1 : 0.35, transition: 'opacity .2s' }" @mouseenter="hoverBar = i">
            <rect :x="b.x" y="0" :width="b.w" height="180" fill="transparent" />
            <rect class="col" :x="b.x" :y="b.y1" :width="b.w" :height="b.h1" rx="3" style="fill: var(--s1)" :style="{ animationDelay: b.delay + 's' }" />
            <rect class="col" :x="b.x" :y="b.y2" :width="b.w" :height="b.h2" rx="3" style="fill: var(--s2)" :style="{ animationDelay: b.delay + 's' }" />
            <rect class="col" :x="b.x" :y="b.y3" :width="b.w" :height="b.h3" rx="3" style="fill: var(--s3)" :style="{ animationDelay: b.delay + 's' }" />
            <text class="ax" :x="b.cx" y="196" text-anchor="middle">{{ b.l }}</text>
          </g>
        </svg>
      </div>

      <RouterLink to="/tires" class="card c4 stack" style="gap: 12px; animation-delay: 0.2s; color: inherit">
        <div class="between" style="align-items: baseline"><h2 class="ctitle">Tires</h2><span class="label">Tread / pressure</span></div>
        <div v-if="tireView" class="row" style="gap: 18px">
          <svg viewBox="-30 0 280 230" width="200" role="img" :aria-label="tireView.reading ? 'Tread depth per corner' : 'No tread readings yet'" style="flex: none">
            <rect x="70" y="20" width="80" height="190" rx="34" style="fill: var(--card2); stroke: var(--line2)" />
            <rect x="82" y="62" width="56" height="40" rx="10" style="fill: var(--bg)" />
            <rect x="82" y="140" width="56" height="30" rx="10" style="fill: var(--bg)" />
            <g v-for="(t, i) in tireView.list" :key="i">
              <rect :x="t.x" :y="t.y" width="16" height="44" rx="5" :style="{ fill: t.color, transition: 'fill .4s' }" />
              <text class="ax" :x="t.tx" :y="t.ty" :text-anchor="t.anchor" style="fill: var(--text); font-size: 13px; font-weight: 600">{{ t.tread }}</text>
              <text class="ax" :x="t.tx" :y="t.ty2" :text-anchor="t.anchor">{{ t.psi }}</text>
            </g>
          </svg>
          <div class="stack" style="gap: 8px">
            <div class="label">Tire health</div>
            <div class="disp" style="font-size: 34px; font-weight: 600">{{ tireView.health == null ? '—' : Math.round(healthT) + '%' }}</div>
            <div class="label">{{ tireView.reading ? (tireView.minT <= 3 ? 'Time to replace soon' : tireView.minT <= 5 ? 'Keep an eye on the lowest corner' : 'Plenty of tread left') : 'Log a tread reading to see this' }}</div>
          </div>
        </div>
      </RouterLink>

      <div class="card c4 stack" style="animation-delay: 0.25s">
        <h2 class="ctitle">Systems life</h2>
        <div v-if="rings.length" style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
          <div v-for="g in rings" :key="g.r.key" class="row">
            <Ring :pct="g.left" :label="g.left + '%'" :color="g.color" :delay="g.delay" />
            <div style="min-width: 0"><div style="font-size: 14px">{{ g.r.title }}</div><div class="label">{{ reminderStatusText(g.r) }}</div></div>
          </div>
        </div>
        <div v-else class="label">Life left on each maintenance item shows here once a schedule is set.</div>
      </div>

      <div class="card c4 stack" style="gap: 10px; animation-delay: 0.3s">
        <div class="between"><h2 class="ctitle">Recent activity</h2><RouterLink to="/log" class="chip">All</RouterLink></div>
        <div v-for="(r, i) in recent" :key="i" class="listrow">
          <div class="tile-ic"><Icon :name="r.icon" :size="18" /></div>
          <div class="grow"><div style="font-size: 14px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis">{{ r.title }}</div><div class="label">{{ fdate(r.date) }}{{ r.who ? ' · ' + r.who : '' }}</div></div>
          <div style="font-weight: 600; font-size: 14px">{{ money(r.amount) }}</div>
        </div>
        <div v-if="!recent.length" class="label">Nothing logged yet.</div>
        <div class="label" style="margin-top: 6px">Documents</div>
        <div class="chips">
          <RouterLink v-for="d in docBadges" :key="d.name" to="/documents" class="badge"><span class="dot" :style="{ background: d.color }"></span>{{ d.name }} · {{ d.exp }}</RouterLink>
          <RouterLink v-if="!docBadges.length" to="/documents" class="badge">Add insurance &amp; registration</RouterLink>
        </div>
      </div>
    </div>
  </section>
</template>
