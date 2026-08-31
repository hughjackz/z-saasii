<template>
  <div>
    <PageHeader title="Users" subtitle="Manage user accounts and role permissions">
      <template #actions>
        <AppButton v-if="auth.isCSAdmin || auth.isCpOp" variant="primary" @click="openCreate"><i class="ti ti-plus"></i>New User</AppButton>
      </template>
    </PageHeader>

    <div class="toolbar">
      <input v-model="search" placeholder="Search users…" style="width:220px" />
      <select v-model="roleFilter" style="width:130px">
        <option value="">All roles</option>
        <option v-if="auth.isCSAdmin">CS_Admin</option>
        <option v-if="auth.isCSAdmin">CP_OP</option>
        <option>CP_OM</option>
      </select>
    </div>

    <div class="card-table">
      <table>
        <thead>
          <tr><th>Name</th><th>Username</th><th>Role</th><th>Company</th><th>Email</th><th>Contact</th><th>Status</th><th>Actions</th></tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="8" class="empty-cell">Loading…</td></tr>
          <tr v-for="u in filtered" :key="u.id">
            <td>{{ u.name }}</td>
            <td><code>{{ u.username }}</code></td>
            <td><AppBadge :color="roleColor(u.role)">{{ u.role }}</AppBadge></td>
            <td>{{ u.company }}</td>
            <td>{{ u.email }}</td>
            <td>{{ u.contact }}</td>
            <td><AppBadge :color="u.enabled ? 'green' : 'gray'">{{ u.enabled ? 'Active' : 'Disabled' }}</AppBadge></td>
            <td>
              <div class="action-btns">
                <AppButton size="sm" @click="openEdit(u)"><i class="ti ti-edit"></i> Edit</AppButton>
                <AppButton v-if="canManagePermissions(u)" size="sm" @click="openPermissions(u)"><i class="ti ti-shield"></i> Perms</AppButton>
                <AppButton v-if="canDelete(u)" size="sm" variant="danger" @click="confirmDeleteUser(u)"><i class="ti ti-trash"></i> Delete</AppButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit Modal -->
    <AppModal v-model="showForm" :title="editingUser ? 'Edit User' : 'New User'" width="500px">
      <div class="form-grid">
        <div>
          <label>Full Name</label>
          <input v-model="form.name" placeholder="Jane Doe" />
        </div>
        <div>
          <label>Username</label>
          <input v-model="form.username" placeholder="jane.doe" :disabled="!!editingUser" />
        </div>
        <div v-if="!editingUser">
          <label>Password</label>
          <input v-model="form.password" type="password" placeholder="Initial password" />
        </div>
        <div>
          <label>Role</label>
          <select v-model="form.role" :disabled="!!editingUser">
            <option v-if="auth.isCSAdmin">CP_OP</option>
            <option v-if="auth.isCSAdmin || auth.isCpOp">CP_OM</option>
          </select>
        </div>
        <div v-if="form.role === 'CP_OM' && auth.isCSAdmin">
          <label>Parent CP_OP</label>
          <select v-model="form.parentId">
            <option value="">— select CP_OP —</option>
            <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }}</option>
          </select>
        </div>
        <div>
          <label>Company</label>
          <input v-model="form.company" placeholder="Company name" />
        </div>
        <div>
          <label>Email</label>
          <input v-model="form.email" type="email" placeholder="user@company.com" />
        </div>
        <div>
          <label>Contact</label>
          <input v-model="form.contact" placeholder="+49 123 456789" />
        </div>
        <div style="display:flex;align-items:center;gap:8px;margin-top:10px">
          <input type="checkbox" v-model="form.enabled" id="chk-enabled" />
          <label for="chk-enabled" style="margin:0">Account enabled</label>
        </div>
      </div>
      <template #footer>
        <AppButton @click="showForm = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="saving" @click="saveUser">
          {{ editingUser ? 'Save Changes' : 'Create User' }}
        </AppButton>
      </template>
    </AppModal>

    <!-- Permissions Modal -->
    <AppModal v-model="showPermissions" :title="`Permissions — ${permUser?.name}`" width="480px">
      <p style="font-size:12px;color:var(--text2);margin-bottom:14px">Toggle which modules this user can access.</p>
      <div class="perm-list">
        <label v-for="m in visibleModules" :key="m.key" class="perm-item">
          <input type="checkbox" v-model="permForm[m.key]" />
          <span>{{ m.label }}</span>
        </label>
      </div>
      <template #footer>
        <AppButton @click="showPermissions = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="savingPerms" @click="savePermissions">Save</AppButton>
      </template>
    </AppModal>

    <!-- Delete Confirm -->
    <ConfirmModal
      v-model="confirmDelete"
      title="Delete User"
      :message="`Delete user '${deleteTarget?.name}'? This action cannot be undone.`"
      confirm-text="Delete"
      :danger="true"
      :loading="deleting"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { users as usersApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppBadge from '@/components/AppBadge.vue'
import AppModal from '@/components/AppModal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'

const auth = useAuthStore()
const list = ref([])
const loading = ref(false)
const search = ref('')
const roleFilter = ref('')
const showForm = ref(false)
const editingUser = ref(null)
const form = ref({})
const saving = ref(false)
const showPermissions = ref(false)
const permUser = ref(null)
const permForm = ref({})
const savingPerms = ref(false)
const confirmDelete = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)

const allModules = [
  { key: 'ocpp.configuration', label: 'OCPP · Configuration' },
  { key: 'ocpp.transaction', label: 'OCPP · Transactions' },
  { key: 'ocpp.action', label: 'OCPP · Actions' },
  { key: 'ocpp.maintenance', label: 'OCPP · Maintenance' },
  { key: 'ocpp.pnc', label: 'OCPP · PnC' },
  { key: 'ocpp.smartcharging', label: 'OCPP · Smart Charging' },
  { key: 'vdv261', label: 'VDV 261' },
  { key: 'management.users', label: 'Management · Users' },
  { key: 'management.devices', label: 'Management · Devices' },
  { key: 'management.certificates', label: 'Management · Certificates' },
  { key: 'management.idtags', label: 'Management · ID Tags' },
  { key: 'management.profiles', label: 'Management · Profiles' },
]

const cpOps = computed(() => list.value.filter(u => u.role === 'CP_OP'))

// CS_Admin sees all modules; CP_OP sees only what they have
const visibleModules = computed(() => {
  if (auth.isCSAdmin) return allModules
  return allModules.filter(m => auth.hasPermission(m.key))
})

const filtered = computed(() => {
  let l = list.value
  if (roleFilter.value) l = l.filter(u => u.role === roleFilter.value)
  if (search.value) l = l.filter(u =>
    u.name.toLowerCase().includes(search.value.toLowerCase()) ||
    u.username.toLowerCase().includes(search.value.toLowerCase())
  )
  return l
})

function roleColor(r) { return { CS_Admin: 'red', CP_OP: 'amber', CP_OM: 'green' }[r] || 'gray' }

// CS_Admin can manage permissions for CP_OP; CP_OP can manage for their CP_OM
// CS_Admin can manage permissions for CP_OP and CP_OM; CP_OP for their CP_OM
function canManagePermissions(u) {
  if (auth.isCSAdmin) return u.role === 'CP_OP' || u.role === 'CP_OM'
  if (auth.isCpOp) return u.role === 'CP_OM' && u.parentId === auth.myid
  return false
}

// CS_Admin can delete CP_OP and CP_OM; CP_OP can delete their CP_OM
function canDelete(u) {
  if (auth.isCSAdmin) return u.role === 'CP_OP' || u.role === 'CP_OM'
  if (auth.isCpOp) return u.role === 'CP_OM' && u.parentId === auth.myid
  return false
}

function openCreate() {
  editingUser.value = null
  const defaultRole = auth.isCSAdmin ? 'CP_OP' : 'CP_OM'
  form.value = {
    name: '', username: '', password: '', role: defaultRole,
    company: '', email: '', contact: '', enabled: true,
    parentId: auth.isCpOp ? auth.myid : ''
  }
  showForm.value = true
}
function openEdit(u) {
  editingUser.value = u
  form.value = { ...u }
  showForm.value = true
}
function openPermissions(u) {
  permUser.value = u
  permForm.value = Object.fromEntries(visibleModules.value.map(m => [m.key, u.permissions?.includes(m.key)]))
  showPermissions.value = true
}
function confirmDeleteUser(u) { deleteTarget.value = u; confirmDelete.value = true }

async function loadUsers() {
  loading.value = true
  try { list.value = await usersApi.list() } finally { loading.value = false }
}

async function saveUser() {
  saving.value = true
  try {
    if (editingUser.value) await usersApi.update(editingUser.value.id, form.value)
    else await usersApi.create(form.value)
    showForm.value = false
    await loadUsers()
  } finally { saving.value = false }
}

async function savePermissions() {
  savingPerms.value = true
  try {
    let perms
    if (auth.isCSAdmin) {
      perms = allModules.filter(m => permForm.value[m.key]).map(m => m.key)
    } else {
      const existingPerms = permUser.value.permissions || []
      const myScope = visibleModules.value.map(m => m.key)
      const keptPerms = existingPerms.filter(p => !myScope.includes(p))
      const newPerms = visibleModules.value.filter(m => permForm.value[m.key]).map(m => m.key)
      perms = [...keptPerms, ...newPerms]
    }
    await usersApi.updatePermissions(permUser.value.id, perms)
    showPermissions.value = false
    await loadUsers()
  } finally { savingPerms.value = false }
}

async function doDelete() {
  deleting.value = true
  try {
    await usersApi.remove(deleteTarget.value.id)
    confirmDelete.value = false
    await loadUsers()
  } finally { deleting.value = false }
}

onMounted(loadUsers)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 12px; }
.card-table { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius-lg); overflow: hidden; }
.empty-cell { text-align: center; padding: 24px; color: var(--text3); }
.action-btns { display: flex; gap: 4px; }
code { font-size: 11px; background: var(--bg); padding: 2px 6px; border-radius: 4px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 14px; }
.perm-list { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; }
.perm-item { display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: var(--radius); font-size: 13px; cursor: pointer; }
.perm-item:hover { background: var(--bg); }
.perm-item input { width: auto; }
</style>
