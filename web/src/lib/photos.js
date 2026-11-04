// Vehicle photos: find candidates on Wikimedia Commons (free, no API key, openly
// licensed) and cut the background out in the browser with @imgly/background-removal.

const SKIP = /interior|dashboard|cockpit|engine|motor|badge|logo|emblem|wheel|rim|seat|trunk|boot|steering|gauge|tail ?light|headlight|key|brochure|crash|wreck|damaged|accident|police|taxi|race|rally|concept|prototype|diagram|drawing|map|\.svg|\.gif|\.tif/i
const PREFER = /front|side|three.?quarter|3\/4|profile|left|right|exterior/i

function stripHtml(s) {
  const d = document.createElement('div')
  d.innerHTML = s || ''
  return (d.textContent || '').replace(/\s+/g, ' ').trim()
}

async function commonsSearch(query, limit = 30) {
  const u = new URL('https://commons.wikimedia.org/w/api.php')
  Object.entries({
    action: 'query', format: 'json', origin: '*', generator: 'search', gsrnamespace: '6',
    gsrsearch: `${query} filetype:bitmap`, gsrlimit: String(limit), prop: 'imageinfo',
    iiprop: 'url|size|mime|extmetadata', iiurlwidth: '1400', iiextmetadatafilter: 'Artist|LicenseShortName|ImageDescription',
  }).forEach(([k, v]) => u.searchParams.set(k, v))
  const r = await fetch(u)
  if (!r.ok) throw new Error('Wikimedia Commons did not answer')
  const j = await r.json()
  return Object.values(j.query?.pages || {}).sort((a, b) => (a.index || 0) - (b.index || 0))
}

/** Returns up to 8 photo candidates for a vehicle: { thumb, url, title, credit, license, page }. */
export async function findCarPhotos(v) {
  const year = v.yearOfManufacture || v.year || ''
  const base = `${v.make || ''} ${v.model || ''}`.trim()
  if (!base) return []
  const seen = new Set()
  const out = []
  for (const q of [`${year} ${base}`.trim(), base]) {
    let pages = []
    try { pages = await commonsSearch(q) } catch (e) { if (!out.length) throw e }
    const ranked = pages
      .filter((p) => p.imageinfo?.[0] && /jpe?g|png|webp/.test(p.imageinfo[0].mime) && p.imageinfo[0].width >= 800 && !SKIP.test(p.title))
      .filter((p) => p.title.toLowerCase().includes((v.model || '').toLowerCase().split(' ')[0]))
      .sort((a, b) => (PREFER.test(b.title) ? 1 : 0) - (PREFER.test(a.title) ? 1 : 0))
    for (const p of ranked) {
      if (seen.has(p.pageid) || out.length >= 8) continue
      seen.add(p.pageid)
      const ii = p.imageinfo[0], md = ii.extmetadata || {}
      out.push({
        thumb: ii.thumburl || ii.url, url: ii.thumburl || ii.url, title: p.title.replace(/^File:/, '').replace(/\.\w+$/, ''),
        credit: `${stripHtml(md.Artist?.value) || 'Unknown author'} · ${md.LicenseShortName?.value || 'see source'} · Wikimedia Commons`,
        page: ii.descriptionurl,
      })
    }
    if (out.length >= 6) break
  }
  return out
}

export async function fetchImage(url) {
  const r = await fetch(url, { mode: 'cors' })
  if (!r.ok) throw new Error("Couldn't download that photo")
  return r.blob()
}

/** Cuts the background out. onProgress(text) reports model download / processing. */
export async function removeBackground(blob, onProgress) {
  const { removeBackground: rb } = await import('@imgly/background-removal')
  const seen = {}
  const out = await rb(blob, {
    model: 'isnet_fp16',
    output: { format: 'image/png', quality: 0.9 },
    progress: (key, current, total) => {
      if (!onProgress) return
      if (key.startsWith('fetch:')) {
        seen[key] = [current, total]
        const [c, t] = Object.values(seen).reduce((a, [x, y]) => [a[0] + x, a[1] + y], [0, 0])
        onProgress(t ? `Downloading the cut-out model (first time only) · ${Math.round((c / t) * 100)}%` : 'Downloading the cut-out model…')
      } else onProgress('Cutting out the background…')
    },
  })
  return trimTransparent(out)
}

/** Crops transparent margins so the car fills the plate, with a little breathing room. */
async function trimTransparent(blob) {
  const bmp = await createImageBitmap(blob)
  const c = document.createElement('canvas')
  c.width = bmp.width; c.height = bmp.height
  const ctx = c.getContext('2d', { willReadFrequently: true })
  ctx.drawImage(bmp, 0, 0)
  const { data, width, height } = ctx.getImageData(0, 0, c.width, c.height)
  let x0 = width, y0 = height, x1 = -1, y1 = -1
  for (let y = 0; y < height; y += 2) for (let x = 0; x < width; x += 2) {
    if (data[(y * width + x) * 4 + 3] > 24) { if (x < x0) x0 = x; if (x > x1) x1 = x; if (y < y0) y0 = y; if (y > y1) y1 = y }
  }
  if (x1 < 0) return blob
  const pad = Math.round(Math.max(x1 - x0, y1 - y0) * 0.04)
  x0 = Math.max(0, x0 - pad); y0 = Math.max(0, y0 - pad); x1 = Math.min(width, x1 + pad); y1 = Math.min(height, y1 + pad)
  const o = document.createElement('canvas')
  o.width = x1 - x0; o.height = y1 - y0
  o.getContext('2d').drawImage(c, x0, y0, o.width, o.height, 0, 0, o.width, o.height)
  return new Promise((res) => o.toBlob((b) => res(b || blob), 'image/png'))
}
