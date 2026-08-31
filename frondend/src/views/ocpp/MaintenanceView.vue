<template>
  <div>
    <PageHeader title="Maintenance" subtitle="Reset, firmware update, and log collection" />
    <DeviceBanner :device="device" />

    <div class="maint-grid">
      <AppCard title="Hard Reset">
        <p class="maint-desc">Immediately reboots the charge point. All active sessions will be terminated.</p>
        <AppButton variant="danger" @click="confirmReset = true">
          <i class="ti ti-refresh"></i> Reset Device
        </AppButton>
      </AppCard>

      <AppCard title="OTA Firmware Update">
        <label>Firmware file</label>
        <input type="file" ref="fwFile" accept=".bin,.hex,.tar.gz,.zip" @change="onFileChange" />
        <div v-if="fwFilename" class="file-name"><i class="ti ti-file"></i> {{ fwFilename }}</div>
        <div class="btn-row">
          <AppButton variant="primary" :loading="fwLoading" :disabled="!fwFilename" @click="doFirmwareUpdate">
            <i class="ti ti-cloud-upload"></i> Upload &amp; Update
          </AppButton>
        </div>
        <div v-if="fwResult" :class="['result', fwResult.ok ? 'result-ok' : 'result-err']">{{ fwResult.message }}</div>
      </AppCard>

      <AppCard title="Diagnostic Log Download">
        <p class="maint-desc">Trigger the charge point to upload its diagnostic logs to the server for download.</p>
        <div class="form-row">
          <div>
            <label>Log type</label>
            <select v-model="logForm.logType">
              <option value="DiagnosticsLog">Diagnostics</option>
              <option value="SecurityLog">Security</option>
            </select>
          </div>
          <div>
            <label>Retries</label>
            <input v-model.number="logForm.retries" type="number" min="0" max="5" />
          </div>
        </div>
        <div class="btn-row">
          <AppButton :loading="logLoading" @click="confirmLog = true">
            <i class="ti ti-file-download"></i> Request Logs
          </AppButton>
        </div>
        <div v-if="logResult" :class="['result', logResult.ok ? 'result-ok' : 'result-err']">{{ logResult.message }}</div>
      </AppCard>
    </div>

    <ConfirmModal
      v-model="confirmReset"
      title="Confirm Hard Reset"
      message="Send a HardReset command to this device? All active charging sessions will be stopped immediately."
      confirm-text="Reset"
      :danger="true"
      :loading="resetLoading"
      @confirm="doReset"
    />
    <ConfirmModal
      v-model="confirmLog"
      title="Request Diagnostic Logs"
      message="Send GetLog command to the device? The device will upload its log file to the server."
      confirm-text="Request"
      :loading="logLoading"
      @confirm="doGetLog"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { maintenance } from '@/api/index.js'
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import PageHeader from '@/components/PageHeader.vue'
import AppCard from '@/components/AppCard.vue'
import AppButton from '@/components/AppButton.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const { device, deviceId } = useGlobalDevice()
const confirmReset = ref(false)
const resetLoading = ref(false)
const confirmLog = ref(false)
const logLoading = ref(false)
const fwFile = ref(null)
const fwFilename = ref('')
const fwLoading = ref(false)
const fwResult = ref(null)
const logResult = ref(null)
const logForm = ref({ logType: 'DiagnosticsLog', retries: 3 })

function onFileChange(e) { fwFilename.value = e.target.files[0]?.name || ''; fwResult.value = null }

async function doReset() {
  resetLoading.value = true
  try { await maintenance.hardReset(deviceId.value); confirmReset.value = false }
  finally { resetLoading.value = false }
}

async function doFirmwareUpdate() {
  const file = fwFile.value?.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('firmware', file)
  fwLoading.value = true; fwResult.value = null
  try {
    await maintenance.firmwareUpdate(deviceId.value, fd)
    fwResult.value = { ok: true, message: 'Firmware update initiated.' }
  } catch (e) {
    fwResult.value = { ok: false, message: e?.message || 'Upload failed' }
  } finally { fwLoading.value = false }
}

async function doGetLog() {
  logLoading.value = true; logResult.value = null
  try {
    await maintenance.getLog(deviceId.value, logForm.value)
    confirmLog.value = false
    logResult.value = { ok: true, message: 'GetLog request sent.' }
  } catch (e) {
    logResult.value = { ok: false, message: e?.message || 'Request failed' }
  } finally { logLoading.value = false }
}
</script>

<style scoped>
.maint-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 14px; }
.maint-desc { font-size: 12px; color: var(--text2); margin-bottom: 14px; line-height: 1.6; }
.btn-row { margin-top: 12px; }
.file-name { font-size: 12px; color: var(--text2); margin-top: 6px; display: flex; align-items: center; gap: 6px; }
.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.result { margin-top: 10px; padding: 8px 12px; border-radius: var(--radius); font-size: 12px; }
.result-ok { background: #e8f5ee; color: #1a6b4a; }
.result-err { background: #fcebeb; color: #a32d2d; }
</style>
