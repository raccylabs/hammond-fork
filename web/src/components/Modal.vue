<script setup>
import { onMounted, onBeforeUnmount } from 'vue'
import Icon from './Icon.vue'
const props = defineProps({ title: String, wide: Boolean })
const emit = defineEmits(['close'])
const onKey = (e) => { if (e.key === 'Escape') emit('close') }
onMounted(() => document.addEventListener('keydown', onKey))
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>
<template>
  <Teleport to="body">
    <div class="scrim" @click="emit('close')"></div>
    <div class="modal" :class="{ wide }" role="dialog" aria-modal="true" :aria-label="title">
      <div class="between">
        <h2 class="disp" style="font-size: 22px">{{ title }}</h2>
        <button class="btn ghost icon" aria-label="Close" @click="emit('close')"><Icon name="x" :size="20" /></button>
      </div>
      <slot />
    </div>
  </Teleport>
</template>
