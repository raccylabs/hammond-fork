import { store } from './store.js'

export const MO = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

const FUEL_SHORT = { 0: 'L', 1: 'gal', 2: 'gal', 3: 'kg', 4: 'kWh', 5: 'min' }
const FUEL_LONG = { 0: 'Litre', 1: 'Imperial gallon', 2: 'US gallon', 3: 'Kilogram', 4: 'Kilowatt hour', 5: 'Minutes' }
export const FUEL_TYPES = { 0: 'Gasoline', 1: 'Diesel', 2: 'Ethanol', 3: 'CNG', 4: 'Electric', 5: 'LPG' }
export const FUEL_UNITS = FUEL_LONG

export const distUnit = () => (store.user?.distanceUnit === 1 ? 'km' : 'mi')
export const fuelUnit = (v) => FUEL_SHORT[v?.fuelUnit ?? 2] || 'gal'
export function effUnit(v) {
  const d = distUnit(), f = fuelUnit(v)
  if (d === 'mi' && f === 'gal') return 'mpg'
  return `${d}/${f}`
}

export function money(n, digits = 2) {
  const cur = store.user?.currency || 'USD'
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: cur, minimumFractionDigits: digits, maximumFractionDigits: digits }).format(n || 0)
  } catch {
    return (n || 0).toFixed(digits)
  }
}
export const num = (n, d = 0) => Number(n || 0).toLocaleString(undefined, { minimumFractionDigits: d, maximumFractionDigits: d })
export const dist = (n) => `${num(n)} ${distUnit()}`

export function toDate(d) { return d instanceof Date ? d : new Date(d) }
export function fdate(d) {
  if (!d) return '—'
  const x = toDate(d)
  return x.toLocaleDateString(undefined, { month: 'numeric', day: 'numeric', year: '2-digit' })
}
export function fdateLong(d) {
  if (!d) return '—'
  return toDate(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
export const monthYear = (d) => `${MO[toDate(d).getMonth()]} ${toDate(d).getFullYear()}`
export function isoDay(d = new Date()) {
  const x = toDate(d)
  return `${x.getFullYear()}-${String(x.getMonth() + 1).padStart(2, '0')}-${String(x.getDate()).padStart(2, '0')}`
}
// Noon local time avoids dates shifting a day when the server stores UTC.
export const dayToISO = (s) => new Date(s + 'T12:00:00').toISOString()
export const daysBetween = (a, b) => Math.round((toDate(b) - toDate(a)) / 86400000)

export function vname(v) { return v ? v.nickname || `${v.make} ${v.model}` : '' }
export function vfull(v) { return v ? [v.yearOfManufacture || '', v.make, v.model].join(' ').trim() : '' }

export function reminderStatusText(r) {
  if (r.status === 'snoozed') return 'Snoozed to ' + fdate(r.snoozedUntil)
  if (r.kind === 'recall') return 'Book the repair'
  const mi = r.milesLeft, dl = r.daysLeft
  const miT = mi == null ? null : mi < 0 ? `Overdue ${num(-mi)} ${distUnit()}` : `In ${num(mi)} ${distUnit()}`
  const dT = dl == null ? null : dl < 0 ? `Overdue ${-dl} d` : dl === 0 ? 'Due today' : `In ${dl} days`
  if (miT && dT) {
    if (mi < 0) return miT
    if (dl < 0) return dT
    // whichever limit arrives first: the projected date for the distance vs the calendar due date
    if (r.projectedDate && r.dueDate) return new Date(r.projectedDate) < new Date(r.dueDate) ? miT : dT
    return miT
  }
  return miT || dT || 'Scheduled'
}
export const LEVEL_COLOR = ['var(--good)', 'var(--warn)', 'var(--bad)']
export const LEVEL_TEXT = ['var(--text)', 'var(--warn)', 'var(--bad)']
