// Thin fetch wrapper for the Hammond API. The server expects the raw JWT in the
// Authorization header (no "Bearer" prefix), and accepts ?access_token= for files.

const TOKEN = 'hm.token'
const REFRESH = 'hm.refresh'

export const auth = {
  get token() { try { return localStorage.getItem(TOKEN) || '' } catch { return '' } },
  get refresh() { try { return localStorage.getItem(REFRESH) || '' } catch { return '' } },
  set(token, refresh) {
    try {
      localStorage.setItem(TOKEN, token)
      if (refresh) localStorage.setItem(REFRESH, refresh)
    } catch {}
  },
  clear() { try { localStorage.removeItem(TOKEN); localStorage.removeItem(REFRESH) } catch {} },
}

export class ApiError extends Error {
  constructor(status, body) {
    super(messageFrom(body) || `Request failed (${status})`)
    this.status = status
    this.body = body
  }
}

function messageFrom(body) {
  if (!body) return ''
  if (typeof body === 'string') return body
  if (body.error) return typeof body.error === 'string' ? body.error : JSON.stringify(body.error)
  if (body.errors) {
    const e = body.errors
    if (Array.isArray(e)) return e.join(', ')
    return Object.entries(e).map(([k, v]) => `${k}: ${v}`).join(', ')
  }
  return ''
}

let refreshing = null
async function tryRefresh() {
  if (!auth.refresh) return false
  refreshing = refreshing || fetch('/api/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken: auth.refresh }),
  }).then(async (r) => {
    if (!r.ok) return false
    const j = await r.json()
    auth.set(j.token, j.refreshToken)
    return true
  }).catch(() => false).finally(() => { setTimeout(() => (refreshing = null), 0) })
  return refreshing
}

export const onUnauthorized = { handler: null }

export async function api(method, path, body, { form = false, retry = true } = {}) {
  const headers = {}
  if (auth.token) headers.Authorization = auth.token
  let payload
  if (form) payload = body
  else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  const res = await fetch('/api' + path, { method, headers, body: payload })
  if (res.status === 401 && retry && path !== '/login') {
    if (await tryRefresh()) return api(method, path, body, { form, retry: false })
    auth.clear()
    onUnauthorized.handler && onUnauthorized.handler()
  }
  const text = await res.text()
  let data = text
  try { data = text ? JSON.parse(text) : null } catch {}
  if (!res.ok) throw new ApiError(res.status, data)
  return data
}

export const get = (p) => api('GET', p)
export const post = (p, b) => api('POST', p, b ?? {})
export const put = (p, b) => api('PUT', p, b)
export const del = (p) => api('DELETE', p)
export const upload = (p, formData) => api('POST', p, formData, { form: true })

export function fileUrl(attachmentId) {
  return attachmentId ? `/api/attachments/${attachmentId}/file?access_token=${encodeURIComponent(auth.token)}` : ''
}
