<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { store, currentVehicle, selectVehicle, toast, dataChanged } from '../lib/store.js'
import { get, post, put, del, upload, fileUrl } from '../lib/api.js'
import { fdate, fdateLong, vname, vfull, isoDay, dayToISO, toDate } from '../lib/format.js'
import Icon from '../components/Icon.vue'
import Modal from '../components/Modal.vue'

const TYPES = { insurance: 'Insurance card', registration: 'Registration', title: 'Title', warranty: 'Warranty', inspection: 'Inspection', other: 'Other' }
const v = computed(() => currentVehicle())
const docs = ref([])
const loading = ref(true)
async function load() {
  if (!v.value) return
  loading.value = true
  try { docs.value = (await get(`/vehicles/${v.value.id}/documents`)) || [] } finally { loading.value = false }
}
watch(() => v.value?.id, load, { immediate: true })

function status(d) {
  if (!d.expiryDate) return { text: 'On file', color: 'var(--s2)' }
  const n = Math.round((toDate(d.expiryDate) - new Date()) / 86400000)
  if (n < 0) return { text: 'Expired ' + fdate(d.expiryDate), color: 'var(--bad)' }
  if (n <= 45) return { text: `Expires in ${n} days`, color: 'var(--warn)' }
  return { text: 'Valid until ' + fdate(d.expiryDate), color: 'var(--s2)' }
}
const missing = computed(() => ['insurance', 'registration'].filter((t) => !docs.value.some((d) => d.type === t)))

// edit / create
const editing = ref(null)
const form = reactive({ type: 'insurance', title: '', provider: '', number: '', expiry: '', notes: '', file: null })
const busy = ref(false)
function openEdit(d, type) {
  editing.value = d || { new: true }
  Object.assign(form, { type: d?.type || type || 'insurance', title: d?.title || '', provider: d?.provider || '', number: d?.number || '', expiry: d?.expiryDate ? isoDay(d.expiryDate) : '', notes: d?.notes || '', file: null })
}
async function save() {
  busy.value = true
  try {
    let attachmentId = editing.value.attachmentId || ''
    if (form.file) {
      const fd = new FormData(); fd.append('file', form.file)
      fd.append("title", form.title || TYPES[form.type])
      const res = await upload(`/vehicles/${v.value.id}/files`, fd)
      attachmentId = res?.id || res?.attachmentId || attachmentId
    }
    const body = { type: form.type, title: form.title || TYPES[form.type], provider: form.provider, number: form.number, expiryDate: form.expiry ? dayToISO(form.expiry) : null, notes: form.notes, attachmentId }
    if (editing.value.new) await post(`/vehicles/${v.value.id}/documents`, body)
    else await put(`/vehicles/${v.value.id}/documents/${editing.value.id}`, body)
    toast(form.expiry ? 'Saved. Hammond will remind you 30 days before it expires.' : 'Saved')
    editing.value = null
    await load(); dataChanged()
  } catch (x) { toast(x.message, true) } finally { busy.value = false }
}
async function remove() {
  if (!confirm('Delete this document?')) return
  await del(`/vehicles/${v.value.id}/documents/${editing.value.id}`)
  editing.value = null
  await load(); dataChanged()
}

// view
const viewing = ref(null)
const fileLink = computed(() => fileUrl(viewing.value?.attachmentId))
</script>

<template>
  <section v-if="!v" class="card empty">Add a vehicle first.</section>
  <section v-else aria-label="Documents">
    <div class="row wrap" style="margin-bottom: 18px; gap: 8px">
      <button v-for="x in store.vehicles" :key="x.id" class="chip" :class="{ on: x.id === v.id }" @click="selectVehicle(x.id)">{{ vname(x) }}</button>
      <div class="grow"></div>
      <button class="btn pri" @click="openEdit(null)"><Icon name="upload" :size="18" />Add document</button>
    </div>
    <div class="g12" :key="v.id">
      <article v-for="(d, i) in docs" :key="d.id" class="card c3 stack" style="gap: 12px" :style="{ animationDelay: i * 0.06 + 's' }">
        <button class="tile-ic" style="all: unset; height: 110px; border-radius: 14px; background: var(--card2); display: flex; align-items: center; justify-content: center; cursor: pointer; color: var(--faint)" :aria-label="'View ' + (d.title || d.type)" @click="viewing = d">
          <Icon :name="d.type === 'insurance' ? 'card' : 'doc'" :size="40" />
        </button>
        <div><div style="font-size: 16px; font-weight: 600">{{ d.title || TYPES[d.type] }}</div><div class="label" style="margin-top: 2px">{{ [d.provider, d.number].filter(Boolean).join(' · ') || TYPES[d.type] }}</div></div>
        <div class="row" style="gap: 8px; font-size: 13px"><span class="dot" :style="{ background: status(d).color }"></span>{{ status(d).text }}</div>
        <div class="row" style="gap: 8px"><button class="btn ghost grow" style="min-height: 40px" @click="viewing = d">View</button><button class="btn ghost grow" style="min-height: 40px" @click="openEdit(d)">Edit</button></div>
      </article>
      <button v-for="t in missing" :key="t" class="card c3 stack" style="gap: 10px; border: 1.5px dashed var(--line2); background: transparent; color: var(--muted); cursor: pointer; font: inherit; align-items: center; justify-content: center; min-height: 260px" @click="openEdit(null, t)">
        <Icon name="plus" :size="30" /><span style="color: var(--text); font-weight: 600">Add {{ TYPES[t].toLowerCase() }}</span>
      </button>
      <div class="card c12 row wrap" style="animation-delay: 0.3s">
        <Icon name="bell" :size="26" style="color: var(--accentText)" />
        <div class="grow" style="min-width: 240px"><div style="font-weight: 600">Expiry dates turn into reminders</div><div class="label">Add the expiry date and Hammond nudges you 30 days out. Files stay on your own server.</div></div>
        <RouterLink to="/reminders" class="chip">See reminders</RouterLink>
      </div>
    </div>

    <Modal v-if="editing" :title="editing.new ? 'Add document' : 'Edit document'" @close="editing = null">
      <div class="grid2">
        <label class="field">Type<select v-model="form.type"><option v-for="(l, k) in TYPES" :key="k" :value="k">{{ l }}</option></select></label>
        <label class="field">Title<input v-model="form.title" :placeholder="TYPES[form.type]" /></label>
      </div>
      <div class="grid2">
        <label class="field">{{ form.type === 'insurance' ? 'Insurer' : 'Issued by' }}<input v-model="form.provider" /></label>
        <label class="field">{{ form.type === 'insurance' ? 'Policy number' : form.type === 'registration' ? 'Plate' : 'Number' }}<input v-model="form.number" /></label>
      </div>
      <label class="field">Expires<input v-model="form.expiry" type="date" /></label>
      <label class="field">Notes<input v-model="form.notes" /></label>
      <label class="btn ghost" style="border-style: dashed"><Icon name="upload" :size="18" />{{ form.file ? form.file.name : editing.attachmentId ? 'Replace the file' : 'Attach a photo or PDF' }}<input type="file" accept="image/*,application/pdf" class="sr-only" @change="(e) => (form.file = e.target.files?.[0] || null)" /></label>
      <div class="row"><button v-if="!editing.new" class="btn danger" @click="remove"><Icon name="trash" :size="18" />Delete</button><button class="btn pri grow" :disabled="busy" @click="save">Save</button></div>
    </Modal>

    <Modal v-if="viewing" :title="`${viewing.title || TYPES[viewing.type]} · ${vname(v)}`" @close="viewing = null">
      <div v-if="viewing.type === 'insurance'" class="stack" style="border-radius: 18px; padding: 22px; background: #231f20; color: #f6f3f1; border-top: 6px solid #fe4611; gap: 16px; animation: rise 0.5s both">
        <div class="between"><span style="font-weight: 700; font-size: 17px">{{ viewing.provider || 'Insurer' }}</span><span style="font-size: 12px; color: #a9a3a0">Proof of insurance</span></div>
        <div class="grid2" style="font-size: 13px">
          <div><div style="color: #a9a3a0; font-size: 11px">Policy</div>{{ viewing.number || '—' }}</div>
          <div><div style="color: #a9a3a0; font-size: 11px">Expires</div>{{ fdateLong(viewing.expiryDate) }}</div>
          <div><div style="color: #a9a3a0; font-size: 11px">Vehicle</div>{{ vfull(v) }}</div>
          <div><div style="color: #a9a3a0; font-size: 11px">VIN</div>{{ v.vin || '—' }}</div>
          <div><div style="color: #a9a3a0; font-size: 11px">Insured</div>{{ store.user?.name }}</div>
        </div>
      </div>
      <img v-if="fileLink && viewing.attachmentId" :src="fileLink" alt="" style="max-width: 100%; border-radius: 12px; background: var(--card2)" @error="(e) => (e.target.style.display = 'none')" />
      <a v-if="fileLink" :href="fileLink" target="_blank" rel="noopener" class="btn ghost">Open the file</a>
      <div v-if="!viewing.attachmentId && viewing.type !== 'insurance'" class="empty"><Icon name="doc" :size="30" />No file attached yet.</div>
      <div class="between"><span class="label">{{ status(viewing).text }}</span><button class="btn pri" @click="openEdit(viewing); viewing = null">Edit</button></div>
    </Modal>
  </section>
</template>
