<template>
  <div>
    <PageHeader title="VDV 261" subtitle="VDV profiles, car management and settings" />

    <div class="tabs">
      <button :class="['tab', {active: tab==='profiles'}]" @click="tab='profiles'">VDV Profiles</button>
      <button :class="['tab', {active: tab==='cars'}]" @click="tab='cars'">Car Management</button>
      <button v-if="auth.isCSAdmin" :class="['tab', {active: tab==='settings'}]" @click="tab='settings';loadSettings()">Settings</button>
    </div>

    <!-- VDV Profiles -->
    <div v-if="tab==='profiles'">
      <div class="toolbar">
        <AppButton @click="loadProfiles" :loading="pLoading"><i class="ti ti-refresh"></i> Read</AppButton>
        <AppButton variant="primary" @click="openProfile()"><i class="ti ti-plus"></i> New</AppButton>
      </div>
      <div class="card-table">
        <table>
          <thead><tr><th>Name</th><th>CP_OP</th><th>DriveOff</th><th>Prec.Dsrd</th><th>Prec.HVAC</th><th>Amb.Temp</th><th>Actions</th></tr></thead>
          <tbody>
            <tr v-if="pLoading"><td colspan="7" class="empty-cell">Loading…</td></tr>
            <tr v-for="p in profiles" :key="p.id">
              <td>{{ p.name }}</td>
              <td>{{ p.cpopName || '—' }}</td>
              <td>{{ p.driveoff }}</td>
              <td>{{ p.precDsrd }}</td>
              <td>{{ p.precHvac }}</td>
              <td>{{ p.ambientTemp }}°C</td>
              <td><div class="action-btns"><AppButton size="sm" @click="openProfile(p)">Edit</AppButton><AppButton size="sm" variant="danger" @click="delProfile(p)">Del</AppButton></div></td>
            </tr>
            <tr v-if="!pLoading && !profiles.length"><td colspan="7" class="empty-cell">No profiles. Click Read.</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Car Management -->
    <div v-if="tab==='cars'">
      <div class="toolbar">
        <AppButton @click="loadCars" :loading="cLoading"><i class="ti ti-refresh"></i> Read</AppButton>
        <AppButton v-if="auth.isCSAdmin" variant="primary" @click="openCar()"><i class="ti ti-plus"></i> New</AppButton>
      </div>
      <div class="card-table">
        <table>
          <thead><tr><th>VIN</th><th>CP_OP</th><th>EVCCID</th><th>Odo</th><th>Profile</th><th>Actions</th></tr></thead>
          <tbody>
            <tr v-if="cLoading"><td colspan="6" class="empty-cell">Loading…</td></tr>
            <tr v-for="c in cars" :key="c.id">
              <td><code>{{ c.vin }}</code></td>
              <td>{{ c.cpopName || '—' }}</td>
              <td>{{ c.evccid || '—' }}</td>
              <td>{{ c.odo }}</td>
              <td>{{ c.vdvProfileName || '—' }}</td>
              <td><div class="action-btns"><AppButton v-if="auth.isCSAdmin" size="sm" @click="openCar(c)">Edit</AppButton><AppButton v-if="auth.isCSAdmin" size="sm" variant="danger" @click="delCar(c)">Del</AppButton></div></td>
            </tr>
            <tr v-if="!cLoading && !cars.length"><td colspan="6" class="empty-cell">No cars. Click Read.</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Settings (CS_Admin only) -->
    <div v-if="tab==='settings' && auth.isCSAdmin">
      <div class="card-table">
        <table>
          <thead><tr><th style="width:200px">Setting</th><th>Value</th></tr></thead>
          <tbody>
            <tr><td>Network Mode</td><td>
              <select v-model="sForm.network_mode" style="width:180px">
                <option value="ipv4">IPv4</option>
                <option value="ipv6">IPv6</option>
                <option value="dual">IPv4 & IPv6 Dual</option>
              </select>
            </td></tr>
            <tr><td>VDV Service</td><td>
              <AppBadge :color="sForm.enable ? 'green' : 'gray'">{{ sForm.enable ? 'Enabled' : 'Disabled' }}</AppBadge>
            </td></tr>
            <tr><td>Listen Address</td><td><code>{{ sForm.listen_addr }}</code></td></tr>
            <tr><td>URL Path</td><td><code>{{ sForm.url_path }}</code></td></tr>
            <tr><td>Server Cert</td><td><code>{{ sForm.cert_file }}</code></td></tr>
            <tr><td>Server Key</td><td><code>{{ sForm.key_file }}</code></td></tr>
            <tr><td>Root CA</td><td><code>{{ sForm.root_ca }}</code></td></tr>
          </tbody>
        </table>
      </div>
      <div class="card-table" style="margin-top:14px">
        <table>
          <thead><tr><th>Certificate File</th><th style="width:180px">Action</th></tr></thead>
          <tbody>
            <tr><td>VDV Root CA (VDVroot.pem)</td><td>
              <input type="file" ref="rootCertFile" accept=".pem,.crt,.cer" style="font-size:12px" />
              <AppButton size="sm" @click="uploadCert('vdv-root')" style="margin-left:4px">Upload</AppButton>
              <AppButton size="sm" @click="downloadCert('vdv-root')">Download</AppButton>
            </td></tr>
            <tr><td>VDV Server Cert (VDVserver.pem)</td><td>
              <input type="file" ref="serverCertFile" accept=".pem,.crt,.cer" style="font-size:12px" />
              <AppButton size="sm" @click="uploadCert('vdv-server-cert')" style="margin-left:4px">Upload</AppButton>
              <AppButton size="sm" @click="downloadCert('vdv-server-cert')">Download</AppButton>
            </td></tr>
            <tr><td>VDV Server Key (VDVserver.key)</td><td>
              <input type="file" ref="serverKeyFile" accept=".key,.pem" style="font-size:12px" />
              <AppButton size="sm" @click="uploadCert('vdv-server-key')" style="margin-left:4px">Upload</AppButton>
              <AppButton size="sm" @click="downloadCert('vdv-server-key')">Download</AppButton>
            </td></tr>
          </tbody>
        </table>
      </div>
      <div class="toolbar" style="margin-top:12px">
        <AppButton variant="primary" :loading="sSaving" @click="saveSettings"><i class="ti ti-device-floppy"></i> Save Settings</AppButton>
        <AppButton variant="danger" :loading="sRestarting" @click="doRestart"><i class="ti ti-refresh"></i> Restart VDV Service</AppButton>
        <input v-if="!sForm.enable" type="checkbox" v-model="sForm.enable" id="vdv-enable" style="width:auto;margin-left:10px" />
        <label v-if="!sForm.enable" for="vdv-enable" style="margin:0">Enable VDV Service</label>
      </div>
      <div v-if="sMsg" :class="['result', sMsg.ok ? 'result-ok' : 'result-err']" style="margin-top:10px">{{ sMsg.text }}</div>
    </div>

    <!-- Profile Modal -->
    <AppModal v-model="pShow" :title="pEdit ? 'Edit Profile' : 'New Profile'" width="400px">
      <div v-if="auth.isCSAdmin && !pEdit">
        <label>CP_OP <span class="req">*</span></label>
        <select v-model="pForm.tenantId">
          <option value="">— select CP_OP —</option>
          <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }}</option>
        </select>
      </div>
      <label>Name</label><input v-model="pForm.name" />
      <label>DriveOff (HH:MM)</label><input v-model="pForm.driveoff" placeholder="00:00" />
      <label>Precondition DSrd</label><input v-model.number="pForm.precDsrd" type="number" />
      <label>Precondition HVAC</label><input v-model.number="pForm.precHvac" type="number" />
      <label>Ambient Temp (°C)</label><input v-model.number="pForm.ambientTemp" type="number" />
      <template #footer>
        <AppButton @click="pShow=false">Cancel</AppButton>
        <AppButton variant="primary" :loading="pSaving" :disabled="auth.isCSAdmin && !pEdit && !pForm.tenantId" @click="saveProfile">{{ pEdit ? 'Save' : 'Create' }}</AppButton>
      </template>
    </AppModal>

    <!-- Car Modal -->
    <AppModal v-model="cShow" :title="cEdit ? 'Edit Car' : 'New Car'" width="400px">
      <div v-if="auth.isCSAdmin && !cEdit">
        <label>CP_OP <span class="req">*</span></label>
        <select v-model="cForm.tenantId">
          <option value="">— select CP_OP —</option>
          <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }}</option>
        </select>
      </div>
      <label>VIN</label><input v-model="cForm.vin" :disabled="!!cEdit" />
      <label>Password</label><input v-model="cForm.password" type="password" />
      <label>VDV Profile</label>
      <select v-model="cForm.vdvProfileId">
        <option value="">— select —</option>
        <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
      </select>
      <template #footer>
        <AppButton @click="cShow=false">Cancel</AppButton>
        <AppButton variant="primary" :loading="cSaving" :disabled="auth.isCSAdmin && !cEdit && !cForm.tenantId" @click="saveCar">{{ cEdit ? 'Save' : 'Create' }}</AppButton>
      </template>
    </AppModal>

    <!-- Delete Confirms -->
    <ConfirmModal v-model="pDelShow" title="Delete Profile" :message="`Delete '${pDelTarget?.name}'?`" confirm-text="Delete" :danger="true" :loading="pDeleting" @confirm="doDeleteProfile" />
    <ConfirmModal v-model="cDelShow" title="Delete Car" :message="`Delete car '${cDelTarget?.vin}'?`" confirm-text="Delete" :danger="true" :loading="cDeleting" @confirm="doDeleteCar" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { vdv, users as usersApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppModal from '@/components/AppModal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'

const auth = useAuthStore()
const tab = ref('profiles')
const cpOps = ref([])

// Profiles
const profiles = ref([]);  const pLoading = ref(false)
const pShow = ref(false);  const pEdit = ref(null); const pForm = ref({}); const pSaving = ref(false)
const pDelShow = ref(false); const pDelTarget = ref(null); const pDeleting = ref(false)

// Cars
const cars = ref([]);      const cLoading = ref(false)
const cShow = ref(false);  const cEdit = ref(null); const cForm = ref({}); const cSaving = ref(false)
const cDelShow = ref(false); const cDelTarget = ref(null); const cDeleting = ref(false)

// Settings
const rootCertFile = ref(null); const serverCertFile = ref(null); const serverKeyFile = ref(null)
const sSaving = ref(false); const sRestarting = ref(false); const sMsg = ref(null)
const sForm = ref({ network_mode: 'ipv6', enable: false, listen_addr: ':9443', url_path: '/vdv', cert_file: '', key_file: '', root_ca: '' })
// Sync sForm keys from backend response (backend sends snake_case json tags)
function applySettings(s) {
  sForm.value = {
    network_mode: s.network_mode || 'ipv6',
    enable: s.enable || false,
    listen_addr: s.listen_addr || ':9443',
    url_path: s.url_path || '/vdv',
    cert_file: s.cert_file || '',
    key_file: s.key_file || '',
    root_ca: s.root_ca || ''
  }
}

// ─── Profiles ────────────────────────────────────────────────────────────────
async function loadProfiles() { pLoading.value = true; try { profiles.value = await vdv.listProfiles() } finally { pLoading.value = false } }
function openProfile(p) {
  pEdit.value = p || null
  pForm.value = p ? { ...p } : { name: '', driveoff: '00:00', precDsrd: 0, precHvac: 0, ambientTemp: 22, tenantId: '' }
  pShow.value = true
}
async function saveProfile() {
  pSaving.value = true
  try {
    const data = { ...pForm.value, tenantId: pForm.value.tenantId || auth.tenantId || auth.myid }
    pEdit.value ? await vdv.updateProfile(pEdit.value.id, data) : await vdv.createProfile(data)
    pShow.value = false; await loadProfiles()
  } finally { pSaving.value = false }
}
function delProfile(p) { pDelTarget.value = p; pDelShow.value = true }
async function doDeleteProfile() { pDeleting.value = true; try { await vdv.deleteProfile(pDelTarget.value.id); pDelShow.value = false; await loadProfiles() } finally { pDeleting.value = false } }

// ─── Cars ────────────────────────────────────────────────────────────────────
async function loadCars() { cLoading.value = true; try { cars.value = await vdv.listCarInfos() } finally { cLoading.value = false } }
function openCar(c) {
  cEdit.value = c || null
  cForm.value = c ? { ...c, password: '' } : { vin: '', password: '', vdvProfileId: '', tenantId: '' }
  cShow.value = true
}
async function saveCar() {
  cSaving.value = true
  try {
    const data = { ...cForm.value, tenantId: cForm.value.tenantId || auth.tenantId || auth.myid }
    cEdit.value ? await vdv.updateCarInfo(cEdit.value.id, data) : await vdv.createCarInfo(data)
    cShow.value = false; await loadCars()
  } catch (e) { alert(e?.error || 'Error') }
  finally { cSaving.value = false }
}
function delCar(c) { cDelTarget.value = c; cDelShow.value = true }
async function doDeleteCar() { cDeleting.value = true; try { await vdv.deleteCarInfo(cDelTarget.value.id); cDelShow.value = false; await loadCars() } finally { cDeleting.value = false } }

async function loadSettings() { try { const s = await vdv.getSettings(); applySettings(s) } catch {} }
async function saveSettings() {
  sSaving.value = true; sMsg.value = null
  try {
    await vdv.updateSettings({ network_mode: sForm.value.network_mode, enable: sForm.value.enable })
    sMsg.value = { ok: true, text: 'Settings saved. Restart required to apply.' }
  } catch (e) { sMsg.value = { ok: false, text: e?.error || 'Error' } }
  finally { sSaving.value = false }
}
function getCertFile(type) {
  const el = { 'vdv-root': rootCertFile, 'vdv-server-cert': serverCertFile, 'vdv-server-key': serverKeyFile }[type]
  return el?.value?.files?.[0]
}
async function uploadCert(type) {
  const file = getCertFile(type)
  if (!file) { sMsg.value = { ok: false, text: 'Select a file first' }; return }
  const fd = new FormData(); fd.append('file', file); fd.append('type', type)
  try { await vdv.uploadCert(fd); sMsg.value = { ok: true, text: 'Uploaded successfully.' } }
  catch (e) { sMsg.value = { ok: false, text: e?.error || 'Upload failed' } }
}
function downloadCert(type) { window.open(`/api/vdv/settings/download-cert?type=${type}`) }
async function doRestart() {
  sRestarting.value = true; sMsg.value = null
  try { await vdv.restartService(); sMsg.value = { ok: true, text: 'VDV261 service restart initiated.' } }
  catch (e) { sMsg.value = { ok: false, text: e?.error || 'Error' } }
  finally { sRestarting.value = false }
}

onMounted(() => { usersApi.listCPOPs().then(r => cpOps.value = r).catch(() => {}) })
</script>

<style scoped>
.tabs { display: flex; gap: 2px; margin-bottom: 14px; }
.tab { padding: 7px 18px; border: 0.5px solid var(--border); background: var(--surface); cursor: pointer; border-radius: var(--radius) var(--radius) 0 0; font-size: 13px; }
.tab.active { background: var(--accent); color: #fff; border-color: var(--accent); }
.toolbar { display: flex; gap: 8px; margin-bottom: 10px; }
.card-table { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius-lg); overflow: hidden; }
.empty-cell { text-align: center; padding: 24px; color: var(--text3); }
.action-btns { display: flex; gap: 4px; }
code { font-size: 11px; background: var(--bg); padding: 2px 6px; border-radius: 4px; }
.req { color: var(--accent); }
.result { padding: 8px 12px; border-radius: var(--radius); font-size: 12px; }
.result-ok { background: #e8f5ee; color: #1a6b4a; }
.result-err { background: #fcebeb; color: #a32d2d; }
</style>
