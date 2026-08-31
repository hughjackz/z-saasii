<template>
  <span :class="['badge', `badge-${color}`]"><slot /></span>
</template>

<script setup>
defineProps({ color: { type: String, default: 'gray' } })
// color: green | amber | red | blue | gray
</script>

<style scoped>
.badge {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 2px 8px; border-radius: 20px;
  font-size: 11px; font-weight: 500; white-space: nowrap;
}
.badge-green  { background: #e8f5ee; color: #1a6b4a; }
.badge-amber  { background: #faeeda; color: #854f0b; }
.badge-red    { background: #fcebeb; color: #a32d2d; }
.badge-blue   { background: #e6f1fb; color: #185fa5; }
.badge-gray   { background: var(--bg); color: var(--text2); border: 0.5px solid var(--border); }
@media (prefers-color-scheme: dark) {
  .badge-green  { background: #1a3028; color: #2daa72; }
  .badge-amber  { background: #2a1e08; color: #ef9f27; }
  .badge-red    { background: #2a1010; color: #e24b4a; }
  .badge-blue   { background: #0c2a44; color: #85b7eb; }
}
</style>