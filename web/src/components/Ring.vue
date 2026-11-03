<script setup>
defineProps({ pct: { type: Number, default: 0 }, label: String, color: { type: String, default: 'var(--accent)' }, delay: { type: Number, default: 0 } })
</script>
<template>
  <svg viewBox="0 0 64 64" width="58" height="58" aria-hidden="true" style="flex: none">
    <circle class="ring-bg" cx="32" cy="32" r="26" />
    <circle class="ring-fg" cx="32" cy="32" r="26" pathLength="100"
      :style="{ stroke: color, strokeDasharray: 100, strokeDashoffset: 100 - Math.max(0, Math.min(100, pct)), animationDelay: delay + 's' }" />
    <text x="32" y="37" text-anchor="middle" style="fill: var(--text); font-size: 14px; font-weight: 600; font-family: inherit">{{ label }}</text>
  </svg>
</template>
