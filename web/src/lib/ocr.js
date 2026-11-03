// Reads a pump display or fuel receipt in the browser with Tesseract.js, then
// pulls out quantity, unit price, total, date and station with simple heuristics.
// Everything it finds is a suggestion; the form highlights the fields to check.

const STATIONS = ['Shell', 'Chevron', 'Exxon', 'Mobil', 'BP', 'Costco', 'QuikTrip', 'QT', "Casey's", 'Caseys', 'Phillips 66', 'Marathon', 'Speedway', 'Sunoco', 'Valero', 'Circle K', 'Wawa', 'Sheetz', "Love's", 'Pilot', 'Flying J', 'Kroger', 'Sam\'s Club', 'Murphy', 'Texaco', 'Citgo', 'Arco', 'Esso', 'Total', 'Hy-Vee', 'Kum & Go', 'Buc-ee\'s', '7-Eleven', 'Holiday', 'Maverik', 'RaceTrac']

let workerPromise
async function worker() {
  if (!workerPromise) {
    workerPromise = import('tesseract.js').then(({ createWorker }) => createWorker('eng'))
  }
  return workerPromise
}

const numRe = /(\d{1,4}[.,]\d{1,3})/g

function nums(line) {
  return [...line.matchAll(numRe)].map((m) => parseFloat(m[1].replace(',', '.')))
}

export function parseReceipt(text) {
  const lines = text.split(/\n+/).map((l) => l.trim()).filter(Boolean)
  const out = { quantity: null, price: null, total: null, station: null, date: null, raw: text }
  for (const l of lines) {
    const low = l.toLowerCase()
    const n = nums(l)
    if (!n.length) continue
    if (out.quantity == null && /(gal|gallon|litre|liter|\bltr\b|\bl\b|volume|qty|quantity)/.test(low) && !/\/\s*(gal|l)/.test(low)) out.quantity = n[0]
    if (out.price == null && /(price|\/\s*gal|\/\s*l\b|per\s*(gal|l)|ppg|unit)/.test(low)) out.price = n.find((x) => x < 20) ?? n[0]
    if (/(total|sale|amount|fuel sale|grand|paid|usd|\$)/.test(low)) {
      const t = Math.max(...n)
      if (out.total == null || /total/.test(low)) out.total = t
    }
  }
  // Fall back to arithmetic: the three numbers where a * b ≈ c.
  const all = lines.flatMap(nums).filter((x) => x > 0)
  if (out.quantity == null || out.price == null || out.total == null) {
    outer: for (const a of all) for (const b of all) {
      if (a === b || b > 20 || a < 1) continue
      const c = all.find((x) => Math.abs(x - a * b) < 0.06 && x !== a && x !== b)
      if (c) { out.quantity ??= a; out.price ??= b; out.total ??= c; break outer }
    }
  }
  if (out.quantity && out.price && !out.total) out.total = +(out.quantity * out.price).toFixed(2)
  if (out.total && out.price && !out.quantity) out.quantity = +(out.total / out.price).toFixed(3)
  const st = STATIONS.find((s) => text.toLowerCase().includes(s.toLowerCase()))
  if (st) out.station = st
  const dm = text.match(/(\d{1,2})[/-](\d{1,2})[/-](\d{2,4})/)
  if (dm) {
    let y = +dm[3]; if (y < 100) y += 2000
    const d = new Date(y, +dm[1] - 1, +dm[2])
    if (!isNaN(d) && d <= new Date() && y > 2000) out.date = d
  }
  return out
}

export async function readReceipt(file, onProgress) {
  const w = await worker()
  onProgress && onProgress('reading')
  const { data } = await w.recognize(file)
  return parseReceipt(data.text || '')
}
