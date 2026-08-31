<template>
  <div class="page-header">
    <div>
      <h1 class="page-title">{{ title }}</h1>
      <p v-if="subtitle" class="page-sub">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.actions" class="page-actions"><slot name="actions" /></div>
  </div>
</template>

<script setup>
defineProps({ title: String, subtitle: String })
</script>

<style scoped>
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 20px; gap: 12px; }
.page-title { font-size: 17px; font-weight: 500; color: var(--text1); }
.page-sub { font-size: 12px; color: var(--text2); margin-top: 3px; }
.page-actions { display: flex; gap: 8px; flex-shrink: 0; }
</style>