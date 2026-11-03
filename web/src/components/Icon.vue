<script setup>
const props = defineProps({ name: { type: String, required: true }, size: { type: [Number, String], default: 22 } })
const P = {
  garage: 'M3 21V9l9-6 9 6v12 M7 21v-7h10v7 M7 17.5h10',
  overview: 'M4.9 19a9 9 0 1 1 14.2 0 M12 14l3.5-4 M12 12.5a1.5 1.5 0 1 0 0 3a1.5 1.5 0 1 0 0-3',
  log: 'M8 6h13 M8 12h13 M8 18h13 M3.5 6h.01 M3.5 12h.01 M3.5 18h.01',
  bell: 'M6 8a6 6 0 1 1 12 0c0 7 3 9 3 9H3s3-2 3-9 M10.3 21a1.9 1.9 0 0 0 3.4 0',
  plus: 'M12 8v8 M8 12h8 M12 2a10 10 0 1 0 0 20a10 10 0 1 0 0-20',
  add: 'M12 5v14 M5 12h14',
  wrench: 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.8-3.8a6 6 0 0 1-7.9 7.9l-6.9 6.9a2.1 2.1 0 0 1-3-3l6.9-6.9a6 6 0 0 1 7.9-7.9z',
  receipt: 'M4 2v20l3-2 3 2 3-2 3 2 3-2 1 .7V2l-3 2-3-2-3 2-3-2-3 2z M8 8h8 M8 12h8 M8 16h5',
  pump: 'M3 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18 M2 22h14 M3 10h12 M15 8h2a2 2 0 0 1 2 2v6a1.5 1.5 0 0 0 3 0V9l-3-3',
  sun: 'M12 8a4 4 0 1 0 0 8a4 4 0 1 0 0-8 M12 2v2 M12 20v2 M4.9 4.9l1.4 1.4 M17.7 17.7l1.4 1.4 M2 12h2 M20 12h2 M4.9 19.1l1.4-1.4 M17.7 6.3l1.4-1.4',
  moon: 'M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z',
  costs: 'M12 2a10 10 0 1 0 0 20a10 10 0 1 0 0-20 M15 9.5c-.5-1-1.6-1.5-3-1.5-1.7 0-3 .8-3 2s1.3 1.7 3 2 3 .8 3 2-1.3 2-3 2c-1.4 0-2.5-.5-3-1.5 M12 6v2 M12 16v2',
  tire: 'M12 2a10 10 0 1 0 0 20a10 10 0 1 0 0-20 M12 8a4 4 0 1 0 0 8a4 4 0 1 0 0-8 M12 2v6 M12 16v6 M2 12h6 M16 12h6',
  doc: 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M8 13h8 M8 17h5',
  settings: 'M4 21v-7 M4 10V3 M12 21v-9 M12 8V3 M20 21v-5 M20 12V3 M1 14h6 M9 8h6 M17 16h6',
  x: 'M18 6 6 18 M6 6l12 12',
  check: 'M20 6 9 17l-5-5',
  camera: 'M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3z M12 10a3 3 0 1 0 0 6a3 3 0 1 0 0-6',
  upload: 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4 M17 8l-5-5-5 5 M12 3v12',
  shield: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z M9 12l2 2 4-4',
  alert: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z M12 8v5 M12 16h.01',
  card: 'M3 6a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z M3 10h18 M7 15h4',
  edit: 'M12 20h9 M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z',
  trash: 'M3 6h18 M8 6V4h8v2 M19 6l-1 14H6L5 6',
  logout: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4 M16 17l5-5-5-5 M21 12H9',
  chevron: 'M9 6l6 6-6 6',
  spark: 'M12 3l1.9 5.1L19 10l-5.1 1.9L12 17l-1.9-5.1L5 10l5.1-1.9z',
  printer: 'M6 9V2h12v7 M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2 M6 14h12v8H6z',
  refresh: 'M21 12a9 9 0 1 1-3-6.7L21 8 M21 3v5h-5',
  spinner: 'M12 3a9 9 0 1 0 9 9',
}
</script>
<template>
  <svg class="ic" :width="size" :height="size" viewBox="0 0 24 24" aria-hidden="true"><path :d="P[props.name] || ''" /></svg>
</template>
