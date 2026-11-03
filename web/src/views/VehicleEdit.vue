<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { store, toast, refreshVehicles, isAdmin } from '../lib/store.js'
import { get, put, del, post, upload } from '../lib/api.js'
import { vname } from '../lib/format.js'
import { fromVehicle, toPayload } from '../lib/vehicleForm.js'
import VehicleFields from '../components/VehicleFields.vue'
import VehiclePlate from '../components/VehiclePlate.vue'
import Icon from '../components/Icon.vue'

const route = useRoute()
const router = useRouter()
const id = route.params.id
const v = ref(null)
const m = reactive({})
const busy = ref(false)
const sharing = ref([])
const users = ref([])
const shareWith = ref('')

onMounted(async () => {
  v.value = await get(`/vehicles/${id}`)
  Object.assign(m, fromVehicle(v.value))
  sharing.value = (await get(`/vehicles/${id}/users`)) || []
  if (isAdmin()) users.value = ((await get('/users')) || []).filter((u) => !sharing.value.some((s) => s.userId === u.id))
})

async function save() {
  busy.value = true
  try { await put(`/vehicles/${id}`, toPayload(m)); await refreshVehicles(); toast('Vehicle saved'); router.push('/overview') } catch (x) { toast(x.message, true) } finally { busy.value = false }
}
async function photo(file) {
  const fd = new FormData(); fd.append('file', file)
  try { const r = await upload(`/vehicles/${id}/photo`, fd); v.value = { ...v.value, photoAttachmentId: r.photoAttachmentId }; await refreshVehicles(); toast('Photo updated') } catch (x) { toast(x.message, true) }
}
async function removePhoto() { await del(`/vehicles/${id}/photo`); v.value = { ...v.value, photoAttachmentId: '' }; refreshVehicles() }
async function share() {
  if (!shareWith.value) return
  try { await post(`/vehicles/${id}/users/${shareWith.value}`); sharing.value = await get(`/vehicles/${id}/users`); users.value = users.value.filter((u) => u.id !== shareWith.value); shareWith.value = ''; toast('Shared') } catch (x) { toast(x.message, true) }
}
async function unshare(s) {
  try { await del(`/vehicles/${id}/users/${s.userId}`); sharing.value = await get(`/vehicles/${id}/users`) } catch (x) { toast(x.message, true) }
}
async function remove() {
  if (!confirm(`Delete ${vname(v.value)} and all its fillups, expenses and reminders? This cannot be undone.`)) return
  try { await del(`/vehicles/${id}`); await refreshVehicles(); toast('Vehicle deleted'); router.push('/garage') } catch (x) { toast(x.message, true) }
}
</script>

<template>
  <section v-if="v" class="g12">
    <div class="card c8 stack">
      <h2 class="ctitle">{{ vname(v) }}</h2>
      <VehicleFields :model="m" />
      <div class="row"><RouterLink to="/overview" class="btn ghost">Cancel</RouterLink><button class="btn pri grow" :disabled="busy" @click="save">Save vehicle</button></div>
    </div>
    <div class="c4 stack" style="gap: 18px">
      <div class="card stack">
        <h2 class="ctitle">Photo</h2>
        <VehiclePlate :vehicle="v" :height="180" editable @upload="photo" />
        <div class="row">
          <label class="btn ghost grow"><Icon name="upload" :size="18" />{{ v.photoAttachmentId ? 'Replace' : 'Upload' }}<input type="file" accept="image/*" class="sr-only" @change="(e) => e.target.files?.[0] && photo(e.target.files[0])" /></label>
          <button v-if="v.photoAttachmentId" class="btn ghost" @click="removePhoto">Remove</button>
        </div>
        <div class="label">A cut-out PNG with a transparent background sits best on the dark plate.</div>
      </div>
      <div class="card stack" style="animation-delay: 0.05s">
        <h2 class="ctitle">Shared with</h2>
        <div v-for="s in sharing" :key="s.id" class="listrow"><span class="grow">{{ s.name || s.userId }}</span><span v-if="s.isOwner" class="badge">Owner</span><button v-else class="chip" @click="unshare(s)">Remove</button></div>
        <div v-if="isAdmin() && users.length" class="row"><select v-model="shareWith" class="chip grow" style="appearance: auto"><option value="">Choose someone…</option><option v-for="u in users" :key="u.id" :value="u.id">{{ u.name || u.email }}</option></select><button class="btn ghost" @click="share">Share</button></div>
      </div>
      <div class="card stack" style="animation-delay: 0.1s">
        <h2 class="ctitle">Danger zone</h2>
        <button class="btn danger" @click="remove"><Icon name="trash" :size="18" />Delete this vehicle</button>
      </div>
    </div>
  </section>
</template>
