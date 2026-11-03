<script setup>
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { store, toast, refreshVehicles, selectVehicle } from '../lib/store.js'
import { get, post, upload } from '../lib/api.js'
import { vfull } from '../lib/format.js'
import { blankVehicle, toPayload } from '../lib/vehicleForm.js'
import Modal from './Modal.vue'
import Icon from './Icon.vue'
import VehicleFields from './VehicleFields.vue'

const router = useRouter()
const step = ref(1)
const vin = ref('')
const decoding = ref(false)
const decoded = ref(null)
const manual = ref(false)
const err = ref('')
const m = reactive(blankVehicle())
const photo = ref(null)
const photoUrl = ref('')
const busy = ref(false)
const recall = ref(null) // { ok, count, error }

const steps = ['Vehicle', 'Photo', 'Recalls & schedule']
const facts = computed(() => decoded.value ? [
  ['Year', decoded.value.year], ['Make', decoded.value.make], ['Model', decoded.value.model], ['Trim', decoded.value.trim || '—'],
  ['Engine', decoded.value.engine || '—'], ['Fuel', decoded.value.fuelLabel || '—'], ['Drivetrain', decoded.value.drivetrain || '—'], ['Body', decoded.value.body || '—'],
] : [])

async function decode() {
  err.value = ''
  const v = vin.value.trim().toUpperCase()
  if (v.length !== 17) { err.value = 'A VIN is 17 characters. Find it on the dashboard by the windshield, the driver door jamb, or your insurance card.'; return }
  decoding.value = true
  try {
    const d = await get('/vin/' + v)
    decoded.value = d
    Object.assign(m, { vin: v, make: d.make, model: d.model, yearOfManufacture: d.year, trim: d.trim, drivetrain: d.drivetrain || '', engineSize: d.engineSize || 0 })
    if (d.fuelType != null) { m.fuelType = d.fuelType; if (d.fuelType === 4) m.fuelUnit = 4 }
    if (!m.nickname) m.nickname = d.model
  } catch (x) {
    err.value = x.message.includes('fetch') ? "Couldn't reach NHTSA from your server. You can enter the details by hand." : x.message
  } finally { decoding.value = false }
}

function pickPhoto(e) {
  const f = e.target.files?.[0]
  if (!f) return
  photo.value = f
  photoUrl.value = URL.createObjectURL(f)
}

const canNext = computed(() => m.make && m.model && m.registration)
const close = () => (store.wizard = false)

async function finish() {
  busy.value = true
  err.value = ''
  try {
    const v = await post('/vehicles', toPayload(m))
    if (photo.value) {
      const fd = new FormData(); fd.append('file', photo.value)
      await upload(`/vehicles/${v.id}/photo`, fd).catch(() => toast('Vehicle added, but the photo upload failed', true))
    }
    post(`/vehicles/${v.id}/recall-check`).catch(() => {})
    await refreshVehicles()
    selectVehicle(v.id)
    toast(`${vfull(v)} added. Log a fillup to start tracking.`)
    close()
    router.push('/garage')
  } catch (x) {
    err.value = x.message
  } finally { busy.value = false }
}

async function checkRecallsPreview() {
  if (!m.make || !m.model || !m.yearOfManufacture) return
  recall.value = { loading: true }
  // Recall check needs a saved vehicle; show what will happen instead of calling NHTSA twice.
  recall.value = { info: true }
}
function next() {
  step.value++
  if (step.value === 3) checkRecallsPreview()
}
</script>

<template>
  <Modal title="Add a vehicle" wide @close="close">
    <div class="row" style="gap: 8px">
      <div v-for="(s, i) in steps" :key="s" style="flex: 1; display: flex; flex-direction: column; gap: 6px">
        <div :style="{ height: '4px', borderRadius: '4px', background: i < step ? 'var(--accent)' : 'var(--line2)', transition: 'background .3s' }"></div>
        <span class="label">{{ s }}</span>
      </div>
    </div>

    <template v-if="step === 1">
      <div v-if="!manual || !decoded" class="row" style="align-items: flex-end">
        <label class="field grow">VIN (17 characters)<input v-model="vin" placeholder="Dashboard, door jamb or insurance card" style="letter-spacing: 0.06em; text-transform: uppercase" @keyup.enter="decode" /></label>
        <button class="btn pri" style="min-height: 46px" :disabled="decoding" @click="decode">Decode</button>
      </div>
      <div v-if="decoding" role="status" class="row muted" style="font-size: 14px"><Icon name="spinner" class="spin" :size="20" />Looking up the VIN with NHTSA…</div>
      <div v-if="!decoded && !manual" class="label">No VIN handy? <a href="#" @click.prevent="manual = true">Enter the details by hand</a>.</div>
      <div v-if="decoded" class="grid3" style="padding: 16px; border-radius: 16px; background: var(--card2); grid-template-columns: repeat(4, minmax(0, 1fr))">
        <div v-for="(f, i) in facts" :key="f[0]" :style="{ animation: 'rise .4s both', animationDelay: i * 0.05 + 's' }"><div class="label">{{ f[0] }}</div><div style="font-size: 15px; margin-top: 3px">{{ f[1] }}</div></div>
      </div>
      <div v-if="decoded?.warning" class="label">NHTSA note: {{ decoded.warning }}</div>
      <VehicleFields v-if="decoded || manual" :model="m" show-odo />
      <div v-if="decoded" class="label">Tire size and oil type come from the trim, not the VIN. Check them against the sticker in your door jamb.</div>
      <div v-if="err" class="error" role="alert">{{ err }}</div>
      <button v-if="decoded || manual" class="btn pri" style="min-height: 52px" :disabled="!canNext" @click="next">Next: photo</button>
    </template>

    <template v-if="step === 2">
      <div class="plate" style="height: 220px">
        <img v-if="photoUrl" :src="photoUrl" alt="Vehicle photo preview" />
        <div v-else class="stack" style="align-items: center; color: #a9a3a0; font-size: 14px"><Icon name="camera" :size="30" />Add a photo so you can spot it at a glance</div>
      </div>
      <div class="row wrap">
        <label class="btn ghost"><Icon name="upload" :size="18" />{{ photoUrl ? 'Choose another' : 'Choose a photo' }}<input type="file" accept="image/*" class="sr-only" @change="pickPhoto" /></label>
        <span class="label grow">Side or three-quarter shots look best on the dark plate. A cut-out PNG with a transparent background looks best of all.</span>
      </div>
      <div class="row"><button class="btn ghost" @click="step--">Back</button><button class="btn pri grow" @click="next">{{ photoUrl ? 'Next: recalls & schedule' : 'Skip for now' }}</button></div>
    </template>

    <template v-if="step === 3">
      <div class="row" style="padding: 16px; border-radius: 16px; background: var(--card2); animation: rise 0.4s both">
        <Icon name="shield" :size="28" style="color: var(--accentText)" />
        <div><div style="font-weight: 600">Recalls are checked when you add it</div><div class="label">Hammond looks up {{ m.yearOfManufacture }} {{ m.make }} {{ m.model }} recalls with NHTSA, then re-checks weekly and tells you if a new one appears.</div></div>
      </div>
      <div class="label">Starting maintenance schedule (edit any time from Reminders)</div>
      <div>
        <div v-for="s in [['Oil change', 'Every 5,000 mi or 6 mo'], ['Tire rotation', 'Every 6,000 mi or 12 mo'], ['Engine air filter', 'Every 15,000 mi or 24 mo'], ['Cabin air filter', 'Every 15,000 mi or 12 mo'], ['Brake fluid', 'Every 30,000 mi or 36 mo'], ['Wiper blades', 'Every 12 months']]" :key="s[0]" class="between" style="padding: 10px 0; border-bottom: 1px solid var(--line); font-size: 14px"><span>{{ s[0] }}</span><span class="muted">{{ s[1] }}</span></div>
      </div>
      <div v-if="err" class="error" role="alert">{{ err }}</div>
      <div class="row"><button class="btn ghost" @click="step--">Back</button><button class="btn pri grow" :disabled="busy" @click="finish"><Icon v-if="busy" name="spinner" class="spin" :size="18" />Add to garage</button></div>
    </template>
  </Modal>
</template>
