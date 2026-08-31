<template>
  <div class="card">
    <div v-if="title || $slots.header" class="card-header">
      <span v-if="title" class="card-title">{{ title }}</span>
      <slot name="header" />
    </div>
    <slot />
  </div>
</template>

<script setup>
defineProps({ title: String })
</script>

<style scoped>
.card { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius-lg); padding: 16px 18px; }
.card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
.card-title { font-size: 14px; font-weight: 500; color: var(--text1); }
</style>