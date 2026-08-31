<template>
  <div>
    <PageHeader title="ID Tags" subtitle="Manage RFID tags for charge point authorization">
      <template #actions>
        <AppButton variant="primary" @click="openCreate"><i class="ti ti-plus"></i>New ID Tag</AppButton>
      </template>
    </PageHeader>

    <TenantSelector v-if="auth.isCSAdmin" v-model="selectedTenant" @change="onTenantChange" />

    <div class="toolbar">
      <input v-model="search" placeholder="Search tag ID…" style="width:220px" />
      <select v-model="statusFilter" style="width:130px">
        <option value="">All status</option>
        <option value="Valid">Valid</option>
        <option value="Blocked">Blocked</option>
        <option value="Expired">Expired</option>
      </select>
    </div>

    <div class="card-table">
      <table>
        <thead>
          <tr><th>Tag ID</th><th>Parent Tag</th><th>Status</th><th>Expiry</th><th>Owner</th><th>Actions</th></tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="6" class="empty-cell">Loading…</td></tr>
          <tr v-for="t in filtered" :key="t.id">
            <td><code>{{ t.tagId }}</code></td>
            <td><code v-if="t.parentTagId">{{ t.parentTagId }}</code><span v-else class="muted">—</span></td>
            <td><AppBadge :color="statusColor(t.status)">{{ t.status }}</AppBadge></td>
            <td>{{ t.expiryTime ? fmtDate(t.expiryTime) : '—' }}</td>
            <td>{{ t.ownerName || '—' }}</td>
            <td>
              <div class="action-btns">
                <AppButton size="sm" @click="openEdit(t)"><i class="ti ti-edit"></i> Edit</AppButton>
                <AppButton size="sm" variant="danger" @click="confirmDel(t)"><i class="ti ti-trash"></i> Delete</AppButton>
              </div>
            </td>
          </tr>
          <tr v-if="!loading && !filtered.length"><td colspan="6" class="empty-cell">No ID tags found.</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit Modal -->
    <AppModal v-model="showForm" :title="editing ? 'Edit ID Tag' : 'New ID Tag'" width="460px">
      <div class="form-grid">
        <div v-if="auth.isCSAdmin && !editing">
          <label>CP_OP <span class="req">*</span></label>
          <select v-model="form.tenantId">
            <option value="">— select CP_OP —</option>
            <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }}</option>
          </select>
        </div>
        <div>
          <label>Tag ID <span class="req">*</span></label>
          <input v-model="form.tagId" placeholder="e.g. RFID-001" :disabled="!!editing" />
        </div>
        <div>
          <label>Parent Tag ID</label>
          <input v-model="form.parentTagId" placeholder="Optional parent tag" />
        </div>
        <div>
          <label>Status</label>
          <select v-model="form.status">
            <option value="Valid">Valid</option>
            <option value="Blocked">Blocked</option>
            <option value="Expired">Expired</option>
          </select>
        </div>
        <div>
          <label>Expiry Time</label>
          <input v-model="form.expiryTime" type="datetime-local" />
        </div>
      </div>
      <template #footer>
        <AppButton @click="showForm = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="saving" :disabled="auth.isCSAdmin && !editing && !form.tenantId" @click="save">
          {{ editing ? 'Save Changes' : 'Create Tag' }}
        </AppButton>
      </template>
    </AppModal>

    <ConfirmModal v-model="showConfirm" title="Delete ID Tag" :message="`Delete tag '${delTarget?.tagId}'?`" confirm-text="Delete" :danger="true" :loading="deleting" @confirm="doDelete" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { idtags as api, users as usersApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppBadge from '@/components/AppBadge.vue'
import AppModal from '@/components/AppModal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import TenantSelector from '@/components/TenantSelector.vue'

const auth = useAuthStore()
const list = ref([])
const cpOps = ref([])
const loading = ref(false)
const search = ref('')
const statusFilter = ref('')
const selectedTenant = ref('')
const showForm = ref(false)
const editing = ref(null)
const form = ref({})
const saving = ref(false)
const showConfirm = ref(false)
const delTarget = ref(null)
const deleting = ref(false)

const filtered = computed(() => {
  let l = list.value
  if (statusFilter.value) l = l.filter(t => t.status === statusFilter.value)
  if (search.value) l = l.filter(t => t.tagId?.toLowerCase().includes(search.value.toLowerCase()))
  return l
})

function statusColor(s) { return { Valid: 'green', Blocked: 'red', Expired: 'amber' }[s] || 'gray' }
function fmtDate(d) { return d ? new Date(d).toLocaleString('en-GB') : '' }

function onTenantChange({ tenantId }) { selectedTenant.value = tenantId; load() }
function getTenantId() { return auth.isCSAdmin ? selectedTenant.value : (auth.tenantId || auth.myid) }

function openCreate() {
  editing.value = null
  form.value = { tagId: '', parentTagId: '', status: 'Valid', expiryTime: '', tenantId: getTenantId() }
  showForm.value = true
}
function openEdit(t) {
  editing.value = t
  const exp = t.expiryTime ? new Date(t.expiryTime).toISOString().slice(0, 16) : ''
  form.value = { ...t, expiryTime: exp }
  showForm.value = true
}
function confirmDel(t) { delTarget.value = t; showConfirm.value = true }

async function load() {
  loading.value = true
  try { list.value = await api.list() } finally { loading.value = false }
}

async function save() {
  saving.value = true
  try {
    const exp = form.value.expiryTime ? form.value.expiryTime + ':00Z' : null
    const payload = {
      ...form.value,
      tenantId: form.value.tenantId || getTenantId(),
      parentTagId: form.value.parentTagId || null,
      expiryTime: exp
    }
    if (editing.value) await api.update(editing.value.id, payload)
    else await api.create(payload)
    showForm.value = false; await load()
  } finally { saving.value = false }
}

async function doDelete() {
  deleting.value = true
  try { await api.remove(delTarget.value.id); showConfirm.value = false; await load() }
  finally { deleting.value = false }
}

onMounted(async () => {
  await Promise.all([load(), usersApi.listCPOPs().then(r => cpOps.value = r).catch(() => {})])
})
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 12px; }
.card-table { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius-lg); overflow: hidden; }
.empty-cell { text-align: center; padding: 24px; color: var(--text3); }
.action-btns { display: flex; gap: 4px; }
code { font-size: 11px; background: var(--bg); padding: 2px 6px; border-radius: 4px; }
.muted { color: var(--text3); font-size: 12px; }
.req { color: var(--accent); }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 14px; }
</style>
