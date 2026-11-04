<script setup>
import { ref, computed } from 'vue'
import { fileUrl } from '../lib/api.js'
import { vfull } from '../lib/format.js'
import Brand from './Brand.vue'
import Icon from './Icon.vue'
import PhotoPicker from './PhotoPicker.vue'

const props = defineProps({ vehicle: Object, height: { type: Number, default: 200 }, editable: Boolean })
const emit = defineEmits(['upload'])
const src = computed(() => fileUrl(props.vehicle?.photoAttachmentId))
const picker = ref('')
function picked({ file, credit }) { picker.value = ''; emit('upload', file, credit) }
defineExpose({ open: (tab = 'find') => (picker.value = tab) })
</script>
<template>
  <div class="plate" :style="{ height: height + 'px' }">
    <img v-if="src" :key="src" :src="src" :alt="vfull(vehicle)" />
    <div v-else style="display: flex; flex-direction: column; align-items: center; gap: 12px; color: #a9a3a0; text-align: center; padding: 16px">
      <Brand kind="mark" :width="52" color="#2C2728" />
      <span style="font-size: 13px">No photo yet</span>
      <div v-if="editable" class="row" style="gap: 8px">
        <button class="chip" style="color: #f6f3f1; border-color: rgba(255, 255, 255, 0.2)" @click="picker = 'find'"><Icon name="spark" :size="16" />Find a photo</button>
        <button class="chip" style="color: #f6f3f1; border-color: rgba(255, 255, 255, 0.2)" @click="picker = 'upload'"><Icon name="upload" :size="16" />Upload</button>
      </div>
    </div>
    <span v-if="src && vehicle?.photoCredit" class="credit" :title="'Photo: ' + vehicle.photoCredit">Photo: {{ vehicle.photoCredit }}</span>
    <PhotoPicker v-if="picker" :vehicle="vehicle" :start-tab="picker" @pick="picked" @close="picker = ''" />
  </div>
</template>
<style scoped>
.credit { position: absolute; right: 8px; bottom: 6px; max-width: 70%; font-size: 9px; color: rgba(246, 243, 241, 0.45); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
</style>
