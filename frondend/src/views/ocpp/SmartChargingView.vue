<template>
  <div>
    <PageHeader title="Smart Charging" subtitle="Charging profiles and power management" />
    <TenantSelector v-if="auth.isCSAdmin" v-model="selectedTenant" @change="onTenantChange" />
    <DeviceSelector v-model="selectedDevice" :devices="auth.isCSAdmin ? tenantDevices : null" />
    <div class="empty-state">
      <i class="ti ti-bolt"></i>
      <p>Smart Charging module — coming in next release</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { devices as devicesApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import DeviceSelector from '@/components/DeviceSelector.vue'
import PageHeader from '@/components/PageHeader.vue'
import TenantSelector from '@/components/TenantSelector.vue'

const auth = useAuthStore()
const selectedTenant = ref('')
const tenantDevices = ref([])
const selectedDevice = ref(null)

async function fetchDevices() {
  try {
    const tid = auth.isCSAdmin && selectedTenant.value ? selectedTenant.value : undefined
    tenantDevices.value = await devicesApi.list(tid ? { tenant_id: tid } : undefined)
  } catch { tenantDevices.value = [] }
}
function onTenantChange() { selectedDevice.value = null; fetchDevices() }
onMounted(fetchDevices)
</script>

<style scoped>
.empty-state { text-align: center; padding: 60px; color: var(--text3); }
.empty-state i { font-size: 40px; display: block; margin-bottom: 12px; }
.empty-state p { font-size: 14px; }
</style>
