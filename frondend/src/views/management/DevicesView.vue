<template>
  <div>
    <PageHeader title="Devices" subtitle="Register and manage charge point devices">
      <template #actions>
        <AppButton variant="primary" @click="openCreate"><i class="ti ti-plus"></i>New Device</AppButton>
      </template>
    </PageHeader>

    <TenantSelector v-if="auth.isCSAdmin" v-model="selectedTenant" @change="onTenantChange" />

    <div class="toolbar">
      <input v-model="search" placeholder="Search devices..." style="width:220px" />
      <select v-model="protocolFilter" style="width:130px">
        <option value="">All protocols</option>
        <option value="OCPP16">OCPP 1.6</option>
        <option value="OCPP201">OCPP 2.0.1</option>
        <option value="OCPP21">OCPP 2.1</option>
      </select>
    </div>

    <div class="card-table">
      <table>
        <thead>
          <tr><th>Device Name</th><th>Protocol</th><th>Topology</th><th>Location</th><th>Owner</th><th>Heartbeat</th><th>Status</th><th>Actions</th></tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="8" class="empty-cell">Loading...</td></tr>
          <tr v-for="d in filtered" :key="d.id">
            <td><strong>{{ d.name }}</strong></td>
            <td><code>{{ d.protocol }}</code></td>
            <td>{{ topologyLabel(d) }}</td>
            <td>{{ d.location }}</td>
            <td>{{ d.ownerName || '—' }}</td>
            <td>{{ d.heartbeatInterval }}s</td>
            <td>
              <AppBadge :color="d.online ? 'green' : 'gray'">{{ d.online ? (d.status || 'Online') : 'Offline' }}</AppBadge>
            </td>
            <td>
              <div class="action-btns">
                <AppButton size="sm" @click="openEdit(d)"><i class="ti ti-edit"></i> Edit</AppButton>
                <AppButton size="sm" variant="danger" @click="confirmDel(d)"><i class="ti ti-trash"></i> Delete</AppButton>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <AppModal v-model="showForm" :title="editing ? 'Edit Device' : 'New Device'" width="480px">
      <div class="form-grid">
        <div v-if="auth.isCSAdmin && !editing">
          <label>CP_OP <span class="req">*</span></label>
          <select v-model="form.tenantId">
            <option value="">— select CP_OP —</option>
            <option v-for="op in cpOps" :key="op.id" :value="op.id">{{ op.name }}</option>
          </select>
        </div>
        <div><label>Device Name</label><input v-model="form.name" placeholder="CP-007" /></div>
        <div>
          <label>Protocol</label>
          <select v-model="form.protocol" @change="onProtocolChange">
            <option value="OCPP16">OCPP 1.6</option>
            <option value="OCPP201">OCPP 2.0.1</option>
            <option value="OCPP21">OCPP 2.1</option>
          </select>
        </div>
        <div><label>Location</label><input v-model="form.location" placeholder="Station A" /></div>
        <div>
          <label>Heartbeat Interval (s)</label>
          <input v-model.number="form.heartbeatInterval" type="number" min="10" max="3600" />
        </div>

        <!-- Topology per protocol (README 2.3.4.3) -->
        <div v-if="isOcpp2Protocol(form.protocol)">
          <label>EVSE Count <span class="req">*</span></label>
          <input v-model.number="form.evseNo" type="number" min="1" max="64" />
        </div>
        <div>
          <label>{{ isOcpp2Protocol(form.protocol) ? 'Connectors per EVSE' : 'Connector Count' }} <span class="req">*</span></label>
          <input v-model.number="form.connectorNo" type="number" min="1" max="16" />
        </div>

        <div style="display:flex;align-items:center;gap:8px;margin-top:10px">
          <input type="checkbox" v-model="form.enabled" id="dev-enabled" style="width:auto" />
          <label for="dev-enabled" style="margin:0">Device enabled</label>
        </div>
      </div>
      <template #footer>
        <AppButton @click="showForm = false">Cancel</AppButton>
        <AppButton variant="primary" :loading="saving" :disabled="!form.tenantId && auth.isCSAdmin && !editing" @click="save">
          {{ editing ? 'Save Changes' : 'Create Device' }}
        </AppButton>
      </template>
    </AppModal>

    <ConfirmModal v-model="showConfirm" title="Delete Device" :message="`Delete device '${delTarget?.name}'?`" confirm-text="Delete" :danger="true" :loading="deleting" @confirm="doDelete" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { devices as devicesApi, users as usersApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import { useDevicesStore, isOcpp2 } from '@/stores/devices.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppBadge from '@/components/AppBadge.vue'
import AppModal from '@/components/AppModal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import TenantSelector from '@/components/TenantSelector.vue'

const auth = useAuthStore()
const devicesStore = useDevicesStore()
const list = ref([])
const cpOps = ref([])
const loading = ref(false)
const search = ref('')
const protocolFilter = ref('')
const selectedTenant = ref('')
const showForm = ref(false)
const editing = ref(null)
const form = ref({})
const saving = ref(false)
const showConfirm = ref(false)
const delTarget = ref(null)
const deleting = ref(false)

function isOcpp2Protocol(p) { return isOcpp2(p) }

// "2 connectors" (OCPP16) / "2 EVSE × 1 conn" (OCPP201/21)
function topologyLabel(d) {
  if (isOcpp2(d.protocol)) {
    const ev = d.evseNo > 0 ? d.evseNo : 2
    const cn = d.connectorNo > 0 ? d.connectorNo : 1
    return `${ev} EVSE × ${cn} conn`
  }
  return `${d.connectorNo > 0 ? d.connectorNo : 2} connectors`
}

// Protocol-dependent defaults (README 2.3.4.3):
//   OCPP16: connectorNo default 2
//   OCPP201/OCPP21: evseNo default 2, connectorNo default 1
function onProtocolChange() {
  if (isOcpp2Protocol(form.value.protocol)) {
    if (!form.value.evseNo) form.value.evseNo = 2
    if (!form.value.connectorNo) form.value.connectorNo = 1
  } else {
    form.value.evseNo = 0
    if (!form.value.connectorNo) form.value.connectorNo = 2
  }
}

const filtered = computed(() => {
  let l = list.value
  if (protocolFilter.value) l = l.filter(d => d.protocol === protocolFilter.value)
  if (search.value) l = l.filter(d => d.name?.toLowerCase().includes(search.value.toLowerCase()))
  return l
})

function onTenantChange({ tenantId }) { selectedTenant.value = tenantId; load() }
function getTenantId() { return auth.isCSAdmin ? selectedTenant.value : (auth.tenantId || auth.myid) }

function openCreate() {
  editing.value = null
  form.value = {
    name: '', protocol: 'OCPP16', location: '', heartbeatInterval: 60, enabled: true,
    ownerId: '', tenantId: getTenantId(),
    connectorNo: 2, evseNo: 0
  }
  showForm.value = true
}
function openEdit(d) {
  editing.value = d
  form.value = { ...d }
  // Normalize legacy records without topology fields
  if (!form.value.connectorNo) form.value.connectorNo = isOcpp2(d.protocol) ? 1 : 2
  if (!form.value.evseNo) form.value.evseNo = isOcpp2(d.protocol) ? 2 : 0
  showForm.value = true
}
function confirmDel(d) { delTarget.value = d; showConfirm.value = true }

async function load() {
  loading.value = true
  try { list.value = await devicesApi.list(getTenantId() ? { tenant_id: getTenantId() } : undefined) } finally { loading.value = false }
}

async function save() {
  saving.value = true
  try {
    const payload = { ...form.value, tenantId: form.value.tenantId || getTenantId() }
    // OCPP 1.6 has no EVSEs
    if (!isOcpp2Protocol(payload.protocol)) payload.evseNo = 0
    if (editing.value) await devicesApi.update(editing.value.id, payload)
    else await devicesApi.create(payload)
    showForm.value = false; await load()
  } finally { saving.value = false }
}

async function doDelete() {
  deleting.value = true
  try {
    await devicesApi.remove(delTarget.value.id); showConfirm.value = false; await load()
    // Reconcile the global selection: a deleted device must not stay selected
    await devicesStore.fetchAll()
  } finally { deleting.value = false }
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
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 14px; }
code { font-size: 11px; background: var(--bg); padding: 2px 6px; border-radius: 4px; }
.req { color: var(--accent); }
</style>
