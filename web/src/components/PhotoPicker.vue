<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { findCarPhotos, fetchImage, removeBackground } from '../lib/photos.js'
import Modal from './Modal.vue'
import Icon from './Icon.vue'

const props = defineProps({ vehicle: { type: Object, required: true }, inline: Boolean, startTab: { type: String, default: 'find' } })
const emit = defineEmits(['pick', 'close'])

const tab = ref(props.startTab)
const results = ref([])
const searching = ref(false)
const searchErr = ref('')
const cutOut = ref(true)
const source = ref(null) // { blob, credit }
const result = ref(null) // { blob, url }
const working = ref('')
const err = ref('')
const chosen = ref(-1)
const sourceUrl = ref('')

const label = computed(() => [props.vehicle.yearOfManufacture || props.vehicle.year, props.vehicle.make, props.vehicle.model].filter(Boolean).join(' '))

async function search() {
  searching.value = true
  searchErr.value = ''
  try {
    results.value = await findCarPhotos(props.vehicle)
    if (!results.value.length) searchErr.value = `No good photos of a ${label.value} turned up. Upload your own instead.`
  } catch {
    searchErr.value = "Couldn't reach Wikimedia Commons. Upload your own photo instead."
  } finally { searching.value = false }
}
onMounted(() => { if (tab.value === 'find') search() })

function setResult(blob) {
  if (result.value?.url) URL.revokeObjectURL(result.value.url)
  result.value = { blob, url: URL.createObjectURL(blob) }
}

async function process() {
  if (!source.value) return
  err.value = ''
  if (!cutOut.value) { setResult(source.value.blob); return }
  working.value = 'Getting ready…'
  try {
    setResult(await removeBackground(source.value.blob, (t) => (working.value = t)))
  } catch (x) {
    err.value = "Couldn't cut out the background on this device, so the photo is shown as is."
    setResult(source.value.blob)
  } finally { working.value = '' }
}

async function pickResult(r, i) {
  chosen.value = i
  err.value = ''
  working.value = 'Downloading the photo…'
  try {
    const blob = await fetchImage(r.url)
    setSource(blob, r.credit)
    await process()
  } catch (x) { err.value = x.message; working.value = '' }
}
function setSource(blob, credit) {
  if (sourceUrl.value) URL.revokeObjectURL(sourceUrl.value)
  sourceUrl.value = URL.createObjectURL(blob)
  source.value = { blob, credit }
}
async function pickFile(e) {
  const f = e.target.files?.[0]
  e.target.value = ''
  if (!f) return
  chosen.value = -1
  setSource(f, '')
  await process()
}
function toggleCut() { cutOut.value = !cutOut.value; process() }
function use() {
  if (!result.value) return
  const name = cutOut.value ? 'vehicle.png' : (source.value.blob.name || 'vehicle.jpg')
  const file = new File([result.value.blob], name, { type: result.value.blob.type || 'image/png' })
  emit('pick', { file, credit: source.value.credit, previewUrl: URL.createObjectURL(file) })
}
onBeforeUnmount(() => { if (result.value?.url) URL.revokeObjectURL(result.value.url); if (sourceUrl.value) URL.revokeObjectURL(sourceUrl.value) })
</script>

<template>
  <component :is="inline ? 'div' : Modal" v-bind="inline ? { class: 'stack' } : { title: 'Vehicle photo', wide: true }" @close="emit('close')">
    <div class="seg" style="align-self: flex-start">
      <button :class="{ on: tab === 'find' }" @click="tab = 'find'; if (!results.length && !searching) search()"><Icon name="spark" :size="14" /> Find a photo</button>
      <button :class="{ on: tab === 'upload' }" @click="tab = 'upload'"><Icon name="upload" :size="14" /> Upload your own</button>
    </div>

    <template v-if="tab === 'find'">
      <div v-if="searching" class="row muted" role="status" style="font-size: 14px"><Icon name="spinner" class="spin" :size="18" />Looking for photos of a {{ label }}…</div>
      <div v-else-if="searchErr" class="label">{{ searchErr }}</div>
      <div v-if="results.length" class="pp-grid">
        <button v-for="(r, i) in results" :key="r.url" class="opt pp-item" :class="{ on: chosen === i }" :aria-label="'Use photo: ' + r.title" @click="pickResult(r, i)">
          <img :src="r.thumb" :alt="r.title" loading="lazy" />
        </button>
      </div>
      <div v-if="results.length" class="label">Openly licensed photos from Wikimedia Commons. The photographer's credit is saved with the photo.</div>
    </template>
    <label v-else class="btn ghost" style="border-style: dashed; min-height: 80px">
      <Icon name="camera" :size="22" />Choose or take a photo
      <input type="file" accept="image/*" class="sr-only" @change="pickFile" />
    </label>

    <div class="between">
      <span>Cut out the background</span>
      <button class="toggle" role="switch" :aria-checked="cutOut" aria-label="Cut out the background" @click="toggleCut"><span></span></button>
    </div>

    <div v-if="source || working" class="plate" style="height: 230px">
      <img v-if="result && !working" :key="result.url" :src="result.url" alt="Preview" />
      <template v-else>
        <img v-if="sourceUrl" :src="sourceUrl" alt="" style="opacity: 0.35; animation: none" />
        <div class="scanline"></div>
        <div style="position: absolute; bottom: 12px; left: 12px; right: 12px; color: #f6f3f1; font-size: 13px; text-align: center" role="status">{{ working }}</div>
      </template>
    </div>
    <div v-if="err" class="label" style="color: var(--warn)">{{ err }}</div>
    <div v-if="source?.credit" class="label">Photo: {{ source.credit }}</div>

    <div class="row">
      <button v-if="!inline" class="btn ghost" @click="emit('close')">Cancel</button>
      <button class="btn pri grow" :disabled="!result || !!working" @click="use">Use this photo</button>
    </div>
  </component>
</template>

<style scoped>
.pp-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.pp-item { padding: 0; overflow: hidden; aspect-ratio: 4 / 3; border-radius: 12px; }
.pp-item img { width: 100%; height: 100%; object-fit: cover; display: block; }
@media (max-width: 820px) { .pp-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
