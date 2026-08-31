<template>
  <div>
    <PageHeader title="Charging Profiles" subtitle="Import and manage smart charging profiles">
      <template #actions>
        <AppButton variant="primary" @click="openUpload"><i class="ti ti-upload"></i>Import Profile</AppButton>
      </template>
    </PageHeader>

    <TenantSelector v-if="auth.isCSAdmin" v-model="selectedTenant" @change="onTenantChange" />

    <div class="card-table">
      <table>
        <thead><tr><th>Name</th><th>Purpose</th><th>Owner</th><th>Imported</th><th>Actions</th></tr></thead>
        <tbody>
          <tr v-if="loading"><td colspan="5" class="empty-cell">Loading…</td></tr>
          <tr v-for="p in list" :key="p.id">
            <td><i class="ti ti-file-text" style="font-size:14px;color:var(--accent);vertical-align:-2px;margin-right:6px"></i>{{ p.name }}</td>
            <td><AppBadge :color="p.purpose ? 'blue' : 'gray'">{{ p.purpose || 'General' }}</AppBadge></td>
            <td>{{ p.ownerName || '—' }}</td>
            <td>{{ p.importedAt }}</td>
            <td>
              <div class="action-btns">
                <AppButton size="sm" @click="openRename(p)"><i class="ti ti-edit"></i> Rename</AppButton>
                <AppButton size="sm" variant="danger" @click="confirmDel(p)"><i class="ti ti-trash"></i> Delete</AppButton>
              </div>
            </td>
          </tr>
          <tr v-if="!loading && !list.length"><td colspan="5" class="empty-cell">No profiles imported.</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Upload Modal -->
    <AppModal v-model="showUpload" title="Import Charging Profile" width="440px">
      <div v-if="auth.isCSAdmin" style="margin-bottom:12px">
        <label>CP_OP <span class="req">*</span></label>
        <select v-model="uploadForm.tenantId">
          <option value="">— select CP_OP —</option>
          <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }}</option>
        </select>
      </div>
      <label>Profile file (.json)</label>
      <input type="file" ref="profileFile" accept=".json" @change="onFileChange" />
      <label style="margin-top:12px">Purpose</label>
      <select v-model="uploadForm.purpose">
        <option value="">General</option>
        <option value="ChargePointMaxProfile">ChargePointMaxProfile</option>
        <option value="TxDefaultProfile">TxDefaultProfile</option>
        <option value="TxProfile">TxProfile</option>
      </select>
      <label>Name (optional — defaults to filename)</label>
      <input v-model="uploadForm.name" :placeholder="uploadForm.filename" />
      <template #footer>
        <AppButton @click="showUpload = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="uploading" :disabled="!uploadForm.filename || (auth.isCSAdmin && !uploadForm.tenantId)" @click="doUpload">Import</AppButton>
      </template>
    </AppModal>

    <!-- Rename Modal -->
    <AppModal v-model="showRename" title="Rename Profile" width="380px">
      <label>New name</label>
      <input v-model="renameVal" />
      <template #footer>
        <AppButton @click="showRename = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="renaming" @click="doRename">Save</AppButton>
      </template>
    </AppModal>

    <ConfirmModal v-model="showConfirm" title="Delete Profile" :message="`Delete profile '${delTarget?.name}'?`" confirm-text="Delete" :danger="true" :loading="deleting" @confirm="doDelete" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { profiles as profilesApi, users as usersApi } from '@/api/index.js'
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
const showUpload = ref(false)
const uploading = ref(false)
const profileFile = ref(null)
const uploadForm = ref({ name: '', purpose: '', filename: '', tenantId: '' })
const showRename = ref(false)
const renameTarget = ref(null)
const renameVal = ref('')
const renaming = ref(false)
const showConfirm = ref(false)
const delTarget = ref(null)
const deleting = ref(false)

function onTenantChange() { load() }

function openUpload() {
  uploadForm.value = { name: '', purpose: '', filename: '', tenantId: auth.isCSAdmin ? selectedTenant.value : '' }
  showUpload.value = true
}
function onFileChange(e) {
  const f = e.target.files[0]
  uploadForm.value.filename = f?.name || ''
  if (!uploadForm.value.name) uploadForm.value.name = f?.name || ''
}
function openRename(p) { renameTarget.value = p; renameVal.value = p.name; showRename.value = true }
function confirmDel(p) { delTarget.value = p; showConfirm.value = true }

async function load() {
  loading.value = true
  try { list.value = await profilesApi.list() } finally { loading.value = false }
}

async function doUpload() {
  const file = profileFile.value?.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  fd.append('purpose', uploadForm.value.purpose)
  fd.append('name', uploadForm.value.name || file.name)
  const tid = uploadForm.value.tenantId || (auth.isCSAdmin ? selectedTenant.value : '')
  if (tid) fd.append('tenant_id', tid)
  uploading.value = true
  try { await profilesApi.upload(fd); showUpload.value = false; await load() }
  finally { uploading.value = false }
}

async function doRename() {
  renaming.value = true
  try { await profilesApi.rename(renameTarget.value.id, renameVal.value); showRename.value = false; await load() }
  finally { renaming.value = false }
}

async function doDelete() {
  deleting.value = true
  try { await profilesApi.remove(delTarget.value.id); showConfirm.value = false; await load() }
  finally { deleting.value = false }
}

onMounted(async () => {
  await Promise.all([load(), usersApi.listCPOPs().then(r => cpOps.value = r).catch(() => {})])
})
</script>

<style scoped>
.card-table { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius-lg); overflow: hidden; }
.empty-cell { text-align: center; padding: 24px; color: var(--text3); }
.action-btns { display: flex; gap: 4px; }
.req { color: var(--accent); }
</style>
