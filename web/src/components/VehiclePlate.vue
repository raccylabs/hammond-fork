<script setup>
import { computed } from 'vue'
import { fileUrl } from '../lib/api.js'
import { vfull } from '../lib/format.js'
import Brand from './Brand.vue'
import Icon from './Icon.vue'

const props = defineProps({ vehicle: Object, height: { type: Number, default: 200 }, editable: Boolean })
const emit = defineEmits(['upload'])
const src = computed(() => fileUrl(props.vehicle?.photoAttachmentId))
function pick(e) { const f = e.target.files?.[0]; if (f) emit('upload', f); e.target.value = '' }
</script>
<template>
  <div class="plate" :style="{ height: height + 'px' }">
    <img v-if="src" :key="src" :src="src" :alt="vfull(vehicle)" />
    <div v-else style="display: flex; flex-direction: column; align-items: center; gap: 12px; color: #a9a3a0; text-align: center; padding: 16px">
      <Brand kind="mark" :width="52" color="#2C2728" />
      <span style="font-size: 13px">No photo yet</span>
      <label v-if="editable" class="chip" style="color: #f6f3f1; border-color: rgba(255, 255, 255, 0.2)">
        <Icon name="upload" :size="16" />Upload a photo
        <input type="file" accept="image/*" class="sr-only" @change="pick" />
      </label>
    </div>
  </div>
</template>
