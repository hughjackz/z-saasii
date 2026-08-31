<template>
  <div>
    <PageHeader title="Certificate Library" subtitle="Upload and manage PnC certificates" />

    <!-- Filters + Read button -->
    <div class="toolbar">
      <TenantSelector v-if="auth.isCSAdmin" v-model="selectedTenant" @change="onTenantChange" />
      <select v-model="typeFilter" style="width:200px">
        <option value="">All types</option>
        <option v-for="t in allCertTypes" :key="t" :value="t">{{ t }}</option>
      </select>
      <AppButton @click="load" :loading="loading"><i class="ti ti-list"></i> Read</AppButton>
      <AppButton variant="primary" @click="openUpload"><i class="ti ti-upload"></i> Upload</AppButton>
      <span v-if="list.length" style="margin-left:auto;font-size:12px;color:var(--text3)">{{ filtered.length }} certificates</span>
    </div>

    <div class="card-table">
      <table>
        <thead><tr><th>Name</th><th>Type</th><th>CP_OP</th><th>Uploaded</th><th>Actions</th></tr></thead>
        <tbody>
          <tr v-if="loading"><td colspan="5" class="empty-cell">Loading…</td></tr>
          <tr v-if="!loading && !filtered.length && list.length"><td colspan="5" class="empty-cell">No certificates match filter.</td></tr>
          <tr v-if="!loading && !list.length"><td colspan="5" class="empty-cell">Click "Read" to load certificates.</td></tr>
          <tr v-for="c in filtered" :key="c.id">
            <td>{{ c.name }}</td>
            <td><AppBadge :color="typeColor(c.type)">{{ c.type }}</AppBadge></td>
            <td>{{ c.ownerName || '—' }}</td>
            <td>{{ fmtDate(c.uploadedAt) }}</td>
            <td>
              <div style="display:flex;gap:4px">
                <AppButton size="sm" @click="viewDetail(c)"><i class="ti ti-eye"></i> Check</AppButton>
                <AppButton size="sm" @click="viewContent(c)"><i class="ti ti-file-text"></i> Content</AppButton>
                <AppButton v-if="c.type !== 'SECC-leaf-cert'" size="sm" variant="danger" @click="confirmDel(c)"><i class="ti ti-trash"></i> Delete</AppButton>
                <AppBadge v-else color="purple">Auto</AppBadge>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Detail Modal -->
    <AppModal v-model="showDetail" :title="`Certificate: ${detailCert?.name}`" width="500px">
      <dl class="detail-grid">
        <dt>Name</dt><dd>{{ detailCert?.name }}</dd>
        <dt>Type</dt><dd><AppBadge :color="typeColor(detailCert?.type)">{{ detailCert?.type }}</AppBadge></dd>
        <dt>Group</dt><dd>{{ detailCert?.certGroup || '—' }}</dd>
        <dt>Serial</dt><dd><code>{{ detailCert?.serialNumber || '—' }}</code></dd>
        <dt>Issuer</dt><dd>{{ detailCert?.issuerName || '—' }}</dd>
        <dt>Subject</dt><dd>{{ detailCert?.subjectName || '—' }}</dd>
        <dt>Algorithm</dt><dd>{{ detailCert?.signatureAlgorithm || '—' }}</dd>
        <dt>Hash (issuer)</dt><dd><code>{{ detailCert?.issuerNameHash || '—' }}</code></dd>
        <dt>Hash (key)</dt><dd><code>{{ detailCert?.issuerKeyHash || '—' }}</code></dd>
        <dt>Valid from</dt><dd>{{ detailCert?.validFrom || '—' }}</dd>
        <dt>Valid to</dt><dd>{{ detailCert?.validTo || '—' }}</dd>
        <dt>Enabled</dt><dd>{{ detailCert?.enabled ? 'Yes' : 'No' }}</dd>
      </dl>
      <template #footer>
        <AppButton @click="showDetail = false">Close</AppButton>
      </template>
    </AppModal>

    <!-- Content Modal -->
    <AppModal v-model="showContent" :title="`Certificate Content: ${contentCert?.name}`" width="700px">
      <div v-if="contentLoading" style="text-align:center;padding:20px">Loading…</div>
      <div v-else>
        <label>Certificate (PEM)</label>
        <pre class="content-pre">{{ contentData }}</pre>
        <label v-if="contentKey" style="margin-top:10px">Private Key (PEM)</label>
        <pre v-if="contentKey" class="content-pre key-pre">{{ contentKey }}</pre>
      </div>
      <template #footer>
        <AppButton @click="copyContent"><i class="ti ti-copy"></i> Copy Certificate</AppButton>
        <AppButton @click="showContent = false">Close</AppButton>
      </template>
    </AppModal>

    <!-- Upload Modal (single cert) -->
    <AppModal v-model="showUpload" title="Upload Certificate" width="480px">
      <div v-if="auth.isCSAdmin" style="margin-bottom:12px">
        <label>CP_OP <span class="req">*</span></label>
        <select v-model="uploadForm.tenantId">
          <option value="">— select CP_OP —</option>
          <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }}</option>
        </select>
      </div>
      <label>Certificate Name <span class="req">*</span></label>
      <input v-model="uploadForm.name" placeholder="e.g. MyCompany V2G Root" />
      <label>Certificate Type <span class="req">*</span></label>
      <select v-model="uploadForm.type">
        <option value="">— select type —</option>
        <option v-for="t in uploadableTypes" :key="t" :value="t">{{ t }}</option>
      </select>
      <label style="margin-top:8px">Certificate File <span class="req">*</span></label>
      <input type="file" ref="certFile" accept=".pem,.crt,.cer,.der" />
      <div v-if="needsKey" style="margin-top:8px">
        <label>Private Key File <span class="req">*</span></label>
        <input type="file" ref="keyFile" accept=".key,.pem" />
      </div>
      <div v-if="needsKey" style="margin-top:8px">
        <label>Key Passphrase (if encrypted)</label>
        <input v-model="uploadForm.keyPassphrase" type="password" placeholder="Leave empty if not encrypted" />
      </div>
      <template #footer>
        <AppButton @click="showUpload = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="uploading" @click="doUpload">Upload</AppButton>
      </template>
    </AppModal>

    <ConfirmModal v-model="showConfirm" title="Delete Certificate" :message="`Delete '${delTarget?.name}'? This removes the file and DB record.`" confirm-text="Delete" :danger="true" :loading="deleting" @confirm="doDelete" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { certs as certsApi, users as usersApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppBadge from '@/components/AppBadge.vue'
import AppModal from '@/components/AppModal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import TenantSelector from '@/components/TenantSelector.vue'

const auth = useAuthStore()
const selectedTenant = ref('')
const cpOps = ref([])
const list = ref([])
const loading = ref(false)
const typeFilter = ref('')

// All 14 certificate types from README 2.3.4.2
const allCertTypes = [
  'V2G-root-cert', 'CPO-sub1-cert', 'CPO-sub2-cert',
  'CPS-sub1-cert', 'CPS-sub2-cert', 'CPS-leaf-cert',
  'MO-root-cert', 'MO-sub1-cert', 'MO-sub2-cert',
  'Contract-leaf-cert',
  'OEM-root-cert', 'OEM-sub1-cert', 'OEM-sub2-cert',
  'SECC-leaf-cert',
]

// Uploadable types (exclude SECC-leaf-cert which is auto-generated)
const uploadableTypes = allCertTypes.filter(t => t !== 'SECC-leaf-cert')

// Types that require a private key (READMe 2.3.4.2 item 3)
const keyRequiredTypes = ['CPO-sub2-cert', 'CPS-leaf-cert', 'Contract-leaf-cert']

// Upload
const showUpload = ref(false); const uploading = ref(false)
const certFile = ref(null); const keyFile = ref(null)
const uploadForm = ref({ name: '', type: '', tenantId: '', keyPassphrase: '' })
const needsKey = computed(() => keyRequiredTypes.includes(uploadForm.value.type))

// Detail
const showDetail = ref(false); const detailCert = ref(null)

// Content
const showContent = ref(false); const contentCert = ref(null)
const contentData = ref(''); const contentKey = ref(''); const contentLoading = ref(false)

// Delete
const showConfirm = ref(false); const delTarget = ref(null); const deleting = ref(false)

const filtered = computed(() => {
  if (!typeFilter.value) return list.value
  return list.value.filter(c => c.type === typeFilter.value)
})

function typeColor(t) {
  if (!t) return 'gray'
  if (t === 'SECC-leaf-cert') return 'purple'
  if (t.startsWith('MO')) return 'amber'
  if (t.startsWith('V2G') || t.startsWith('OEM')) return 'green'
  if (t.startsWith('CPO')) return 'blue'
  if (t.startsWith('CPS')) return 'cyan'
  if (t.startsWith('Contract')) return 'red'
  return 'gray'
}

function fmtDate(d) { return d ? d.slice(0, 10) : '—' }

function onTenantChange() { load() }
function getTenantId() { return auth.isCSAdmin ? selectedTenant.value : (auth.tenantId || auth.myid) }

function viewDetail(c) { detailCert.value = c; showDetail.value = true }
async function viewContent(c) {
  contentCert.value = c; contentData.value = ''; contentKey.value = ''
  showContent.value = true; contentLoading.value = true
  try {
    const res = await certsApi.getContent(c.id)
    contentData.value = res.content || ''
    contentKey.value = res.privateKey || ''
  } catch { contentData.value = '(failed to load)' }
  finally { contentLoading.value = false }
}
function copyContent() {
  navigator.clipboard.writeText(contentData.value).then(() => alert('Copied!')).catch(() => {})
}
function confirmDel(c) { delTarget.value = c; showConfirm.value = true }
function openUpload() {
  uploadForm.value = { name: '', type: '', tenantId: getTenantId(), keyPassphrase: '' }
  showUpload.value = true
}

async function load() {
  loading.value = true
  try { list.value = await certsApi.list() } finally { loading.value = false }
}

async function doUpload() {
  const name = uploadForm.value.name.trim()
  const type = uploadForm.value.type
  const file = certFile.value?.files?.[0]
  if (!name) { alert('Name is required'); return }
  if (!type) { alert('Type is required'); return }
  if (!file) { alert('Select a certificate file'); return }
  if (needsKey.value && !keyFile.value?.files?.[0]) {
    alert(`${type} requires a private key file`)
    return
  }

  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    fd.append('type', type)
    fd.append('name', name)
    fd.append('cert_group', type) // use type as group
    const key = keyFile.value?.files?.[0]
    if (key) fd.append('private_key', key)
    if (uploadForm.value.keyPassphrase) fd.append('key_passphrase', uploadForm.value.keyPassphrase)
    const tid = uploadForm.value.tenantId || getTenantId()
    if (tid) fd.append('tenant_id', tid)
    await certsApi.upload(fd)
    showUpload.value = false
    await load()
  } finally { uploading.value = false }
}

async function doDelete() {
  deleting.value = true
  try { await certsApi.remove(delTarget.value.id); showConfirm.value = false; await load() }
  finally { deleting.value = false }
}

onMounted(() => { usersApi.listCPOPs().then(r => cpOps.value = r).catch(() => {}) })
</script>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; flex-wrap: wrap; }
.card-table { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius-lg); overflow: hidden; }
.empty-cell { text-align: center; padding: 24px; color: var(--text3); }
.req { color: var(--accent); }
.detail-grid { display: grid; grid-template-columns: 120px 1fr; gap: 4px 12px; font-size: 13px; }
.detail-grid dt { color: var(--text2); font-weight: 500; text-align: right; }
.detail-grid dd { margin: 0; word-break: break-all; }
.detail-grid code { font-size: 11px; background: var(--bg); padding: 2px 6px; border-radius: 4px; }
.content-pre { font-size: 11px; font-family: monospace; background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: var(--radius); max-height: 400px; overflow: auto; white-space: pre-wrap; word-break: break-all; }
.key-pre { color: #ce9178; }
</style>
