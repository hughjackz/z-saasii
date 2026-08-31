<template>
  <div>
    <PageHeader title="Plug &amp; Charge (PnC)" subtitle="15118 PnC certificate management" />
    <DeviceBanner :device="device" />

    <div class="pnc-grid">
      <!-- 1. Install SECC Leaf Certificate -->
      <AppCard title="1. Install SECC Leaf Certificate">
        <p class="pnc-desc">Select V2G root + CPO sub-certificates as signer, then trigger device to request SECC Leaf signing.</p>
        <label>V2G Root</label>
        <select v-model="seccForm.v2gRoot"><option value="">— select —</option><option v-for="c in certsByType('V2G-root-cert')" :key="c.id" :value="c.name">{{ c.name }}</option></select>
        <label>CPO Sub1</label>
        <select v-model="seccForm.v2gSub1"><option value="">— select —</option><option v-for="c in certsByType('CPO-sub1-cert')" :key="c.id" :value="c.name">{{ c.name }}</option></select>
        <label>CPO Sub2</label>
        <select v-model="seccForm.v2gSub2"><option value="">— select —</option><option v-for="c in certsByType('CPO-sub2-cert')" :key="c.id" :value="c.name">{{ c.name }}</option></select>
        <div class="btn-row">
          <AppButton variant="primary" :loading="seccLoading" :disabled="!seccForm.v2gRoot || !seccForm.v2gSub1 || !seccForm.v2gSub2" @click="doInstallSeccLeaf">
            <i class="ti ti-certificate"></i> Trigger SECC Leaf Install
          </AppButton>
        </div>
        <div v-if="seccResult" :class="['result', seccResult.ok ? 'result-ok' : 'result-err']">{{ seccResult.message }}</div>
      </AppCard>

      <!-- 2. Install Root Certificate -->
      <AppCard title="2. Install Root Certificate">
        <p class="pnc-desc">Select certificate type, then choose certificates to install on the device.</p>
        <label>Certificate type</label>
        <select v-model="installCertType" @change="installSelected = []">
          <option value="">— select type —</option>
          <option value="MO-root-cert">MO-root-cert</option>
          <option value="V2G-root-cert">V2G-root-cert</option>
        </select>
        <div v-if="installCertType && certsByType(installCertType).length" style="margin-top:10px">
          <label>Select certificates (multi-select)</label>
          <div class="multi-list">
            <label v-for="c in certsByType(installCertType)" :key="c.id" class="multi-item">
              <input type="checkbox" :value="c.name" v-model="installSelected" />
              <span>{{ c.name }}</span>
            </label>
          </div>
        </div>
        <div class="btn-row">
          <AppButton variant="primary" :loading="installLoading" :disabled="!installSelected.length" @click="doInstall">
            <i class="ti ti-upload"></i> Install Selected ({{ installSelected.length }})
          </AppButton>
        </div>
        <div v-if="installResult" :class="['result', installResult.ok ? 'result-ok' : 'result-err']">{{ installResult.message }}</div>
      </AppCard>

      <!-- 3. Get Installed Certificates -->
      <AppCard title="3. Get Installed Certificates">
        <label>Certificate type</label>
        <select v-model="getCertType">
          <option value="">All types</option>
          <option value="MO-root-cert">MO-root-cert</option>
          <option value="V2G-root-cert">V2G-root-cert</option>
          <option value="SECC-leaf-cert">SECC-leaf-cert</option>
        </select>
        <div class="btn-row"><AppButton :loading="getLoading" @click="doGetCerts"><i class="ti ti-list"></i> Get Installed Certs</AppButton></div>
        <div v-if="installedCerts.length" class="cert-result">
          <div v-for="c in installedCerts" :key="c.certificateHashData?.issuerNameHash + c.certificateType" class="cert-item">
            <i class="ti ti-certificate"></i>
            <div><strong>{{ c.certificateType }}</strong><br><small>{{ c.certificateHashData?.issuerNameHash || '—' }}</small></div>
          </div>
        </div>
      </AppCard>

      <!-- 4. Delete Certificate -->
      <AppCard title="4. Delete Certificate">
        <label>Certificate type</label>
        <select v-model="delCertType" @change="delSelected = []">
          <option value="">— select type —</option>
          <option value="MO-root-cert">MO-root-cert</option>
          <option value="V2G-root-cert">V2G-root-cert</option>
          <option value="SECC-leaf-cert">SECC-leaf-cert</option>
        </select>
        <div v-if="delCertType && certsByType(delCertType).length" style="margin-top:10px">
          <label>Select certificates to delete</label>
          <div class="multi-list">
            <label v-for="c in certsByType(delCertType)" :key="c.id" class="multi-item">
              <input type="checkbox" :value="c.name" v-model="delSelected" />
              <span>{{ c.name }}</span>
            </label>
          </div>
        </div>
        <div class="btn-row">
          <AppButton variant="danger" :loading="delLoading" :disabled="!delSelected.length" @click="doDelete">
            <i class="ti ti-trash"></i> Delete Selected ({{ delSelected.length }})
          </AppButton>
        </div>
        <div v-if="delResult" :class="['result', delResult.ok ? 'result-ok' : 'result-err']">{{ delResult.message }}</div>
      </AppCard>

      <!-- 5. Contract Certificate Group (2.3.2.4.e) -->
      <AppCard v-if="device" title="5. Contract Certificate Group" class="span-2">
        <p class="pnc-desc">Select one certificate from each type to form the contract certificate group (used by ContractGenerate / 4.2.9.5).</p>
        <div class="contract-grid">
          <div v-for="t in contractCertTypes" :key="t" class="contract-row">
            <label class="contract-label">{{ t }}</label>
            <select v-model="contractGroup[t]" @change="onContractCertChange(t, contractGroup[t])">
              <option value="">— select —</option>
              <option v-for="c in certsByType(t)" :key="c.id" :value="c.name">{{ c.name }}</option>
            </select>
          </div>
        </div>
        <div class="btn-row">
          <AppButton variant="primary" :loading="contractLoading" :disabled="!contractComplete" @click="doSaveContractGroup">
            <i class="ti ti-device-floppy"></i> {{ contractComplete ? 'Save Cert Group (' + filledCount + ')' : 'Select all types above' }}
          </AppButton>
        </div>
        <div v-if="contractResult" :class="['result', contractResult.ok ? 'result-ok' : 'result-err']">{{ contractResult.message }}</div>
      </AppCard>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { pnc, certs as certsApi } from '@/api/index.js'
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import PageHeader from '@/components/PageHeader.vue'
import AppCard from '@/components/AppCard.vue'
import AppButton from '@/components/AppButton.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const { device, deviceId } = useGlobalDevice()
const libraryCerts = ref([])

// All 13 uploadable types (exclude SECC-leaf-cert, which is auto-generated)
const contractCertTypes = [
  'V2G-root-cert', 'CPO-sub1-cert', 'CPO-sub2-cert',
  'CPS-sub1-cert', 'CPS-sub2-cert', 'CPS-leaf-cert',
  'MO-root-cert', 'MO-sub1-cert', 'MO-sub2-cert',
  'Contract-leaf-cert',
  'OEM-root-cert', 'OEM-sub1-cert', 'OEM-sub2-cert',
]

// Operation states
const getCertType = ref('');     const installedCerts = ref([]); const getLoading = ref(false)
const delCertType = ref('');     const delSelected = ref([]);    const delLoading = ref(false); const delResult = ref(null)
const installCertType = ref(''); const installSelected = ref([]); const installLoading = ref(false); const installResult = ref(null)
const seccLoading = ref(false);  const seccResult = ref(null)
const seccForm = ref({ v2gRoot: '', v2gSub1: '', v2gSub2: '' })

// Contract cert group (Card 5)
const contractGroup = ref({})
const contractLoading = ref(false)
const contractResult = ref(null)

const filledCount = computed(() => Object.values(contractGroup.value).filter(v => v).length)
const contractComplete = computed(() => filledCount.value === contractCertTypes.length)

function certsByType(type) {
  return libraryCerts.value.filter(c => c.type === type)
}

function onContractCertChange(type, name) {
  // Load certs of this type if not already loaded (lazy, triggered by dropdown open)
  // The certs are already in libraryCerts, just filter by type
}

// 1. Install SECC Leaf
async function doInstallSeccLeaf() {
  seccLoading.value = true; seccResult.value = null
  try {
    await pnc.signCertificate(deviceId.value, { ...seccForm.value })
    seccResult.value = { ok: true, message: 'SECC Leaf signing triggered. Device will request certificate.' }
  } catch (e) { seccResult.value = { ok: false, message: e?.message || 'Error' } }
  finally { seccLoading.value = false }
}

// 2. Install Root Certificate
async function doInstall() {
  installLoading.value = true; installResult.value = null
  try {
    const res = await pnc.installCert(deviceId.value, [...installSelected.value], installCertType.value)
    const sent = res.results?.filter(r => r.status === 'Sent').length || installSelected.value.length
    installResult.value = { ok: true, message: `${sent} certificate(s) sent one-by-one.` }
    installSelected.value = []
  } catch (e) { installResult.value = { ok: false, message: e?.message || 'Install failed' } }
  finally { installLoading.value = false }
}

// 3. Get Installed Certificates
async function doGetCerts() {
  if (!deviceId.value) return; getLoading.value = true
  try {
    const res = await pnc.getInstalledCerts(deviceId.value, getCertType.value || undefined)
    installedCerts.value = res?.certificateHashDataChain || (Array.isArray(res) ? res : [])
  }
  finally { getLoading.value = false }
}

// 4. Delete Certificate
async function doDelete() {
  delLoading.value = true; delResult.value = null
  try {
    for (const name of delSelected.value) {
      await pnc.deleteCert(deviceId.value, name)
    }
    delResult.value = { ok: true, message: `${delSelected.value.length} certificate(s) deleted.` }
    delSelected.value = []
  } catch (e) { delResult.value = { ok: false, message: e?.message || 'Delete failed' } }
  finally { delLoading.value = false }
}

// 5. Contract Certificate Group
async function doSaveContractGroup() {
  contractLoading.value = true; contractResult.value = null
  try {
    await pnc.contractCertGroup(deviceId.value, { ...contractGroup.value })
    contractResult.value = { ok: true, message: 'Contract certificate group saved.' }
  } catch (e) { contractResult.value = { ok: false, message: e?.error || 'Save failed' } }
  finally { contractLoading.value = false }
}

onMounted(async () => {
  await Promise.all([certsApi.list().then(r => libraryCerts.value = r).catch(() => {})])
})
</script>

<style scoped>
.pnc-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.span-2 { grid-column: 1 / -1; }
.pnc-desc { font-size: 12px; color: var(--text2); margin-bottom: 14px; line-height: 1.6; }
.btn-row { margin-top: 12px; }
.cert-result { margin-top: 12px; display: flex; flex-direction: column; gap: 6px; max-height: 200px; overflow-y: auto; }
.cert-item { display: flex; align-items: flex-start; gap: 8px; padding: 8px; background: var(--bg); border-radius: var(--radius); font-size: 12px; }
.cert-item i { font-size: 16px; color: var(--accent); margin-top: 1px; }
.cert-item small { color: var(--text3); font-family: monospace; }
.multi-list { max-height: 160px; overflow-y: auto; border: 0.5px solid var(--border); border-radius: var(--radius); padding: 6px; }
.multi-item { display: flex; align-items: center; gap: 8px; padding: 4px 6px; border-radius: 4px; font-size: 13px; cursor: pointer; }
.multi-item:hover { background: var(--bg); }
.multi-item input { width: auto; }
.result { margin-top: 10px; padding: 8px 12px; border-radius: var(--radius); font-size: 12px; }
.result-ok { background: #e8f5ee; color: #1a6b4a; }
.result-err { background: #fcebeb; color: #a32d2d; }

.contract-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 14px; max-height: 360px; overflow-y: auto; }
.contract-row { display: flex; align-items: center; gap: 8px; }
.contract-label { font-size: 12px; white-space: nowrap; min-width: 150px; color: var(--text2); }
.contract-row select { flex: 1; font-size: 12px; }
</style>
