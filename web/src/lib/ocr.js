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

function normalize(text) {
  return text
    .replace(/(?<=[\d.,$])[Oo]|[Oo](?=[\d.,])/g, '0')
    .replace(/(?<=[\d.,$])[Il|]|[Il|](?=[\d.,]\d)/g, '1')
    .replace(/(?<=[\d.,])S|S(?=\d)/g, '5')
    .replace(/(\d) ([.,]) ?(\d)/g, '$1.$3')
}

export function parseReceipt(text) {
  text = normalize(text)
  const lines = text.split(/\n+/).map((l) => l.trim()).filter(Boolean)
  const out = { quantity: null, price: null, total: null, station: null, date: null, raw: text }
  for (const l of lines) {
    const low = l.toLowerCase()
    const n = nums(l)
    if (!n.length) continue
    if (out.quantity == null && /(gal|gallon|litre|liter|\bltr\b|\bl\b|volume|qty|quantity)/.test(low) && !/\/\s*(gal|l)/.test(low)) out.quantity = n[0]
    if (out.price == null && /(price|\/\s*gal|\$\s*\/\s*g|\/\s*l\b|per\s*(gal|l)|ppg|unit|@)/.test(low)) out.price = n.find((x) => x < 20) ?? n[0]
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

// ---------------------------------------------------------------- image cleanup

/** Upscale/downscale, grayscale, auto-contrast, invert dark displays and binarize (Otsu). */
async function preprocess(file) {
  let bmp
  try { bmp = await createImageBitmap(file, { imageOrientation: 'from-image' }) } catch { bmp = await createImageBitmap(file) }
  const long = Math.max(bmp.width, bmp.height)
  const scale = Math.min(3, Math.max(0.4, 2000 / long))
  const w = Math.round(bmp.width * scale), h = Math.round(bmp.height * scale)
  const c = document.createElement('canvas')
  c.width = w; c.height = h
  const ctx = c.getContext('2d', { willReadFrequently: true })
  ctx.drawImage(bmp, 0, 0, w, h)
  const img = ctx.getImageData(0, 0, w, h)
  const d = img.data
  const g = new Uint8ClampedArray(w * h)
  const hist = new Uint32Array(256)
  for (let i = 0, j = 0; i < d.length; i += 4, j++) {
    const y = (d[i] * 299 + d[i + 1] * 587 + d[i + 2] * 114) / 1000
    g[j] = y; hist[g[j]]++
  }
  // contrast stretch between the 2nd and 98th percentile
  const n = w * h
  let lo = 0, hi = 255, acc = 0
  for (let k = 0; k < 256; k++) { acc += hist[k]; if (acc > n * 0.02) { lo = k; break } }
  acc = 0
  for (let k = 255; k >= 0; k--) { acc += hist[k]; if (acc > n * 0.02) { hi = k; break } }
  const range = Math.max(1, hi - lo)
  let mean = 0
  for (let j = 0; j < n; j++) { g[j] = ((g[j] - lo) * 255) / range; mean += g[j] }
  mean /= n
  const invert = mean < 105 // light digits on a dark pump display
  // Otsu threshold
  const h2 = new Uint32Array(256)
  for (let j = 0; j < n; j++) h2[g[j]]++
  let sumAll = 0
  for (let k = 0; k < 256; k++) sumAll += k * h2[k]
  let sumB = 0, wB = 0, best = 0, thr = 128
  for (let k = 0; k < 256; k++) {
    wB += h2[k]; if (!wB) continue
    const wF = n - wB; if (!wF) break
    sumB += k * h2[k]
    const mB = sumB / wB, mF = (sumAll - sumB) / wF
    const between = wB * wF * (mB - mF) * (mB - mF)
    if (between > best) { best = between; thr = k }
  }
  for (let i = 0, j = 0; j < n; i += 4, j++) {
    let v = g[j] > thr ? 255 : 0
    if (invert) v = 255 - v
    d[i] = d[i + 1] = d[i + 2] = v
  }
  ctx.putImageData(img, 0, 0)
  return c
}

const score = (r) => (r.quantity ? 1 : 0) + (r.price ? 1 : 0) + (r.total ? 1 : 0)

export async function readReceipt(file, onProgress) {
  const w = await worker()
  onProgress && onProgress('reading')
  const clean = await preprocess(file).catch(() => file)
  // Receipts read best as one block of text; pump displays as scattered text.
  await w.setParameters({ tessedit_pageseg_mode: '6', preserve_interword_spaces: '1' })
  let r = parseReceipt((await w.recognize(clean)).data.text || '')
  if (score(r) < 3) {
    await w.setParameters({ tessedit_pageseg_mode: '11' })
    const r2 = parseReceipt((await w.recognize(clean)).data.text || '')
    if (score(r2) > score(r)) r = { ...r2, station: r2.station || r.station, date: r2.date || r.date }
  }
  return r
}
