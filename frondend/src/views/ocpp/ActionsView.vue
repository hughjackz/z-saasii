<template>
  <div>
    <PageHeader title="Remote Actions" subtitle="Start and stop charging sessions remotely" />
    <DeviceBanner :device="device" />

    <div class="actions-grid">
      <!-- Remote Start -->
      <AppCard title="Remote Start">
        <label>Connector ID</label>
        <input v-model.number="startForm.connectorId" type="number" min="1" placeholder="e.g. 1" />
        <label>ID Tag</label>
        <input v-model="startForm.idTag" placeholder="e.g. RFID-001" />
        <label>Charging Profile (optional)</label>
        <select v-model="startForm.profileId">
          <option value="">None</option>
          <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <div class="btn-row">
          <AppButton variant="primary" :loading="startLoading" @click="doStart">
            <i class="ti ti-player-play"></i> Start Transaction
          </AppButton>
        </div>
        <div v-if="startResult" :class="['result', startResult.ok ? 'result-ok' : 'result-err']">
          {{ startResult.message }}
        </div>
      </AppCard>

      <!-- Remote Stop -->
      <AppCard title="Remote Stop">
        <label>Transaction ID</label>
        <input v-model.number="stopForm.transactionId" type="number" placeholder="e.g. 10042" />
        <div class="btn-row">
          <AppButton variant="danger" :loading="stopLoading" @click="doStop">
            <i class="ti ti-player-stop"></i> Stop Transaction
          </AppButton>
        </div>
        <div v-if="stopResult" :class="['result', stopResult.ok ? 'result-ok' : 'result-err']">
          {{ stopResult.message }}
        </div>
      </AppCard>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { transactions, profiles as profilesApi } from '@/api/index.js'
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import PageHeader from '@/components/PageHeader.vue'
import AppCard from '@/components/AppCard.vue'
import AppButton from '@/components/AppButton.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const { device, deviceId } = useGlobalDevice()
const profiles = ref([])
const startForm = ref({ connectorId: 1, idTag: '', profileId: '' })
const stopForm = ref({ transactionId: null })
const startLoading = ref(false)
const stopLoading = ref(false)
const startResult = ref(null)
const stopResult = ref(null)

async function doStart() {
  if (!deviceId.value) return
  startLoading.value = true; startResult.value = null
  try {
    const res = await transactions.remoteStart(deviceId.value, startForm.value)
    startResult.value = { ok: res.status === 'Accepted', message: `Status: ${res.status}` }
  } catch (e) {
    startResult.value = { ok: false, message: e?.message || 'Error' }
  } finally { startLoading.value = false }
}

async function doStop() {
  if (!deviceId.value || !stopForm.value.transactionId) return
  stopLoading.value = true; stopResult.value = null
  try {
    const res = await transactions.remoteStop(deviceId.value, stopForm.value)
    stopResult.value = { ok: res.status === 'Accepted', message: `Status: ${res.status}` }
  } catch (e) {
    stopResult.value = { ok: false, message: e?.message || 'Error' }
  } finally { stopLoading.value = false }
}

onMounted(async () => {
  await Promise.all([
    profilesApi.list().then(r => profiles.value = r).catch(() => {})
  ])
})
</script>

<style scoped>
.actions-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.btn-row { display: flex; gap: 8px; margin-top: 14px; }
.result { margin-top: 10px; padding: 8px 12px; border-radius: var(--radius); font-size: 12px; }
.result-ok { background: #e8f5ee; color: #1a6b4a; }
.result-err { background: #fcebeb; color: #a32d2d; }
</style>
