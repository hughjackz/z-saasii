<template>
  <div class="tenant-bar">
    <label class="tenant-label"><i class="ti ti-building"></i> CP_OP</label>
    <select v-model="selected" @change="onChange">
      <option value="">— select CP_OP —</option>
      <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }} ({{ op.username }})</option>
    </select>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { users as usersApi } from '@/api/index.js'

const props = defineProps({
  modelValue: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue', 'change'])

const selected = ref(props.modelValue)
const cpOps = ref([])

watch(() => props.modelValue, v => { selected.value = v })

function onChange() {
  emit('update:modelValue', selected.value)
  const op = cpOps.value.find(o => o.id === selected.value)
  emit('change', { tenantId: selected.value, tenantName: op?.username || '', cpOp: op })
}

onMounted(async () => {
  try { cpOps.value = await usersApi.listCPOPs() } catch {}
})
</script>

<style scoped>
.tenant-bar {
  display: flex; align-items: center; gap: 10px;
  margin-bottom: 14px; padding: 8px 14px;
  background: var(--surface); border: 0.5px solid var(--border);
  border-radius: var(--radius);
}
.tenant-label {
  font-size: 12px; font-weight: 500; color: var(--text2);
  display: flex; align-items: center; gap: 5px; white-space: nowrap;
}
select { flex: 1; max-width: 300px; }
</style>
