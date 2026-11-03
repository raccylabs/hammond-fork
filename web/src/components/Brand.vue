<script setup>
import { H_MARK, WORDMARK } from '../lib/brand.js'
const props = defineProps({ kind: { type: String, default: 'wordmark' }, width: { type: [Number, String], default: 196 }, color: { type: String, default: 'var(--text)' } })
const g = props.kind === 'mark' ? H_MARK : WORDMARK
const [, , vw, vh] = g.viewBox.split(' ').map(Number)
</script>
<template>
  <svg :viewBox="g.viewBox" :width="width" :height="(Number(width) * vh) / vw" role="img" aria-label="Hammond" :style="{ fill: color, display: 'block' }">
    <path :d="g.d" fill-rule="evenodd" />
  </svg>
</template>
