<template>
  <div class="device-selector">
    <IconChargingPile :size="20" :stroke="2" :class="statusColorClass(currentDevice?.status, currentDevice?.online)" />
    <select :value="modelValue || deviceList[0]?.id" @change="select($event.target.value)" class="ds-select">
      <option value="" disabled v-if="!deviceList.length">— no devices —</option>
      <option v-for="d in deviceList" :key="d.id" :value="d.id">
        {{ d.name }}
      </option>
    </select>
  </div>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useDevicesStore } from '@/stores/devices.js'
import { IconChargingPile } from '@tabler/icons-vue';

const props = defineProps({ modelValue: String, devices: Array })
const emit = defineEmits(['update:modelValue'])
const store = useDevicesStore()

const deviceList = computed(() => props.devices || store.list)

const currentDevice = computed(() => {
  const id = props.modelValue || deviceList.value[0]?.id
  return deviceList.value.find(d => d.id === id) || null
})

function select(id) { emit('update:modelValue', id) }

// Auto-emit first device on mount so parent's v-model is synced
onMounted(() => {
  if (!props.modelValue && deviceList.value.length) {
    emit('update:modelValue', deviceList.value[0].id)
  }
})
// Also emit when devices list changes (e.g. after tenant switch)
watch(deviceList, (list) => {
  if (list.length && !props.modelValue) {
    emit('update:modelValue', list[0].id)
  }
})

function statusIcon(s) {
  return 'ti-charging-pile'
}

function statusColorClass(s, online) {
  if (!online) return 'clr-gray'  // offline
  if (s === 'Available') return 'clr-green'
  if (s === 'Charging' || s === 'Preparing' || s === 'Finishing') return 'clr-amber'
  if (s === 'Faulted') return 'clr-red'
  return 'clr-gray'
}
</script>

<style>
.device-selector {
  display: inline-flex; align-items: center; gap: 8px;
  background: var(--surface); border: 0.5px solid var(--border);
  border-radius: var(--radius); padding: 5px 10px; margin-bottom: 18px;
}
.device-selector i.ti {
  font-size: 16px; flex-shrink: 0; transition: color 0.2s;
}
.device-selector .clr-green { color: #1d9e75; }
.device-selector .clr-amber { color: #ef9f27; }
.device-selector .clr-red   { color: #e24b4a; }
.device-selector .clr-gray  { color: #6b6b65; }
.device-selector .ds-select {
  appearance: none; -webkit-appearance: none;
  border: none; background: transparent;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239a9a94' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 2px center;
  padding-right: 16px;
  font-size: 13px; font-weight: 500; color: var(--text1);
  cursor: pointer; outline: none; line-height: 1.4;
}
.device-selector .ds-select:focus { outline: none; }
</style>
