<template>
  <div>
    <PageHeader title="Remote Actions" subtitle="Start and stop charging sessions remotely" />
    <DeviceBanner :device="device" />

    <div class="actions-grid">
      <!-- Remote Start -->
      <AppCard title="Remote Start">
        <label>Connector ID</label>
        <input v-model.number="startForm.connectorId" type="number" min="1" placeholder="e.g. 1" />
        <label>ID Tag <span v-if="tagsLoading" style="color:var(--text3)">(loading…)</span></label>
        <select v-model="startForm.idTag">
          <option value="">— select ID tag —</option>
          <option v-for="t in idTags" :key="t.id" :value="t.tagId">
            {{ t.tagId }} ({{ t.status }})
          </option>
        </select>
        <div v-if="!tagsLoading && !idTags.length" class="tag-hint">
          No ID tags under this CP_OP — create one in Management → ID Tags.
        </div>
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
import { ref, onMounted, watch } from 'vue'
import { transactions, profiles as profilesApi, idtags as idtagsApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import PageHeader from '@/components/PageHeader.vue'
import AppCard from '@/components/AppCard.vue'
import AppButton from '@/components/AppButton.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const auth = useAuthStore()
const { device, deviceId } = useGlobalDevice()
const profiles = ref([])
const idTags = ref([])
const tagsLoading = ref(false)
const startForm = ref({ connectorId: 1, idTag: '', profileId: '' })
const stopForm = ref({ transactionId: null })
const startLoading = ref(false)
const stopLoading = ref(false)
const startResult = ref(null)
const stopResult = ref(null)

// Load the ID tags owned by the selected device's CP_OP (README 2.3.2.2.2:
// remote start must use an id tag belonging to the device's tenant).
async function loadIdTags() {
  if (!device.value) { idTags.value = []; return }
  tagsLoading.value = true
  try {
    // CS_Admin: filter by the device's tenant; CP_OP/CP_OM: JWT-scoped server-side
    const params = auth.isCSAdmin && device.value.tenantId ? { tenant_id: device.value.tenantId } : undefined
    idTags.value = await idtagsApi.list(params) || []
  } catch { idTags.value = [] }
  finally {
    tagsLoading.value = false
    // Keep only a still-valid selection
    if (startForm.value.idTag && !idTags.value.some(t => t.tagId === startForm.value.idTag)) {
      startForm.value.idTag = ''
    }
  }
}

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

// Device (and thus tenant) may change — reload id tags accordingly
watch([deviceId], loadIdTags, { immediate: true })
</script>

<style scoped>
.actions-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.btn-row { display: flex; gap: 8px; margin-top: 14px; }
.tag-hint { margin-top: 6px; font-size: 11px; color: var(--text3); }
.result { margin-top: 10px; padding: 8px 12px; border-radius: var(--radius); font-size: 12px; }
.result-ok { background: #e8f5ee; color: #1a6b4a; }
.result-err { background: #fcebeb; color: #a32d2d; }
</style>
