// Pure calculations shared by the views: efficiency, spending, costs, chart geometry.
import { toDate } from './format.js'

const byDateAsc = (a, b) => toDate(a.date) - toDate(b.date) || a.odoReading - b.odoReading

/**
 * Full-tank-to-full-tank efficiency. Returns fillups (ascending) with `eff`
 * (distance per fuel unit) set on full-tank fills that have a usable previous full fill.
 */
export function withEfficiency(fillups) {
  const list = [...(fillups || [])].sort(byDateAsc).map((f) => ({ ...f, eff: null }))
  let lastFull = null
  let fuelSince = 0
  for (const f of list) {
    fuelSince += f.fuelQuantity || 0
    const missed = f.hasMissedFillup === true
    if (f.isTankFull !== false) {
      if (lastFull && !missed && f.odoReading > lastFull.odoReading && fuelSince > 0) {
        f.eff = (f.odoReading - lastFull.odoReading) / fuelSince
      }
      lastFull = f
      fuelSince = 0
    }
    if (missed) { lastFull = f.isTankFull !== false ? f : null; fuelSince = 0 }
  }
  return list
}

export const sum = (arr, fn = (x) => x) => arr.reduce((a, x) => a + (fn(x) || 0), 0)
export const avg = (arr, fn = (x) => x) => (arr.length ? sum(arr, fn) / arr.length : 0)

export function monthsBack(n, now = new Date()) {
  const out = []
  for (let i = n - 1; i >= 0; i--) out.push(new Date(now.getFullYear(), now.getMonth() - i, 1))
  return out
}
export const inMonth = (d, m) => { const x = toDate(d); return x.getFullYear() === m.getFullYear() && x.getMonth() === m.getMonth() }

export const SERVICE_TYPES = ['Service', 'Maintenance', 'Oil change', 'Tire rotation', 'Repair', 'Brakes']
export function expenseCategory(e) {
  const t = (e.expenseType || '').toLowerCase()
  if (e.serviceItemId || e.source === 'service') return 'Maintenance'
  if (/repair|fix|replace|bearing|thermostat/.test(t)) return 'Repairs'
  if (/tire|tyre|wheel/.test(t)) return 'Tires'
  if (/insur/.test(t)) return 'Insurance'
  if (/regist|licen|tax|plate|inspection/.test(t)) return 'Registration'
  if (/service|oil|filter|fluid|maint|brake|rotation|plug|wiper/.test(t)) return 'Maintenance'
  return 'Other'
}
export const isServiceExpense = (e) => ['Maintenance', 'Repairs', 'Tires'].includes(expenseCategory(e))

/** Distance driven between the first and last odometer reading in a window. */
export function distanceIn(fillups, from, to = new Date()) {
  const f = fillups.filter((x) => toDate(x.date) >= from && toDate(x.date) <= to).sort(byDateAsc)
  if (f.length < 2) return 0
  return Math.max(0, f[f.length - 1].odoReading - f[0].odoReading)
}

/** 12-month cost of ownership for one vehicle. */
export function costOfOwnership(v, fillups, expenses, now = new Date()) {
  const from = new Date(now); from.setFullYear(from.getFullYear() - 1)
  const fy = fillups.filter((f) => toDate(f.date) >= from)
  const ey = expenses.filter((e) => toDate(e.date) >= from)
  const cats = { Fuel: sum(fy, (f) => f.totalAmount), Maintenance: 0, Repairs: 0, Tires: 0, Insurance: (v.insurancePremium || 0) * 12, Registration: 0, Depreciation: 0, Other: 0 }
  for (const e of ey) cats[expenseCategory(e)] += e.amount || 0
  if (v.purchasePrice && v.purchaseDate) {
    const yrs = Math.max(0.5, (now - toDate(v.purchaseDate)) / 31557600000)
    cats.Depreciation = Math.max(0, (v.purchasePrice - (v.estimatedValue || 0)) / yrs)
  }
  const miles = distanceIn(fillups, from, now)
  const total = sum(Object.values(cats))
  const monthly = monthsBack(12, now).map((m) => {
    const mf = fillups.filter((f) => inMonth(f.date, m))
    const me = expenses.filter((e) => inMonth(e.date, m))
    return { m, cost: sum(mf, (f) => f.totalAmount) + sum(me, (e) => e.amount) + (v.insurancePremium || 0) + cats.Depreciation / 12, miles: 0 }
  })
  // distance per month from consecutive odometer readings
  const asc = [...fillups].sort(byDateAsc)
  for (let i = 1; i < asc.length; i++) {
    const d = asc[i].odoReading - asc[i - 1].odoReading
    const slot = monthly.find((x) => inMonth(asc[i].date, x.m))
    if (slot && d > 0) slot.miles += d
  }
  return { cats, miles, total, monthly, cpm: miles ? total / miles : 0 }
}

/* ---------------- chart geometry ---------------- */

export function smooth(pts) {
  if (!pts.length) return ''
  let d = `M${pts[0][0].toFixed(1)},${pts[0][1].toFixed(1)}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i - 1] || pts[i], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2] || p2
    const c1x = p1[0] + (p2[0] - p0[0]) / 6, c1y = p1[1] + (p2[1] - p0[1]) / 6
    const c2x = p2[0] - (p3[0] - p1[0]) / 6, c2y = p2[1] - (p3[1] - p1[1]) / 6
    d += ` C${c1x.toFixed(1)},${c1y.toFixed(1)} ${c2x.toFixed(1)},${c2y.toFixed(1)} ${p2[0].toFixed(1)},${p2[1].toFixed(1)}`
  }
  return d
}

export function spark(vals, w = 100, h = 32) {
  if (vals.length < 2) return ''
  const mn = Math.min(...vals), mx = Math.max(...vals), rg = mx - mn || 1
  return smooth(vals.map((v, i) => [(i / (vals.length - 1)) * (w - 2) + 1, h - 3 - ((v - mn) / rg) * (h - 6)]))
}

export function niceMax(v) {
  if (v <= 0) return 1
  const p = Math.pow(10, Math.floor(Math.log10(v)))
  const n = v / p
  return (n <= 1 ? 1 : n <= 2 ? 2 : n <= 2.5 ? 2.5 : n <= 5 ? 5 : 10) * p
}
