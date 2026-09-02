<template>
  <div>
    <PageHeader title="Overview" subtitle="All devices under your account">
      <template #actions>
        <AppButton :loading="loading" @click="fetchDevices()"><i class="ti ti-refresh"></i> Refresh</AppButton>
      </template>
    </PageHeader>

    <TenantSelector
      v-if="auth.isCSAdmin"
      :model-value="devicesStore.cpOp?.tenantId || ''"
      @change="onTenantChange"
    />

    <!-- Stats row -->
    <div class="stats-grid">
      <div class="stat">
        <div class="stat-label">Total Devices</div>
        <div class="stat-val">{{ deviceList.length }}</div>
      </div>
      <div class="stat">
        <div class="stat-label">Online</div>
        <div class="stat-val" style="color:#1d9e75">{{ onlineCount }}</div>
      </div>
      <div class="stat">
        <div class="stat-label">Active Sessions</div>
        <div class="stat-val">{{ activeCount }}</div>
      </div>
      <div class="stat">
        <div class="stat-label">Faulted</div>
        <div class="stat-val" :style="faultedCount ? 'color:#e24b4a' : ''">{{ faultedCount }}</div>
      </div>
    </div>

    <!-- Device cards -->
    <div v-if="loading" class="loading">
      <i class="ti ti-loader-2 spin"></i> Loading devices…
    </div>
    <div v-else-if="!deviceList.length" class="empty">
      No devices found. Select a CP_OP above (CS_Admin) or add devices under Management → Devices.
    </div>
    <div v-else class="device-grid">
      <div
        v-for="d in deviceList"
        :key="d.id"
        class="dev-card"
        :class="{ selected: devicesStore.current?.id === d.id }"
        @click="openDevice(d)"
      >
        <div class="dev-card-top">
          <span class="dev-name">{{ d.name }}</span>
          <AppBadge :color="d.online ? 'green' : 'gray'">
            <span :class="['dot', d.online ? 'dot-green' : 'dot-gray']"></span>
            {{ d.online ? d.status : 'Offline' }}
          </AppBadge>
        </div>
        <div class="dev-protocol">{{ d.protocol }} · {{ d.location }}</div>
        <div class="dev-meta">
          <div class="dev-meta-item">Connectors<br><strong>{{ d.connectors ?? '—' }}</strong></div>
          <div class="dev-meta-item">Active TX<br><strong :style="d.activeTx ? 'color:#1d9e75' : ''">{{ d.activeTx ?? 0 }}</strong></div>
          <div class="dev-meta-item">Last seen<br><strong>{{ d.lastHeartbeat ? dayjs(d.lastHeartbeat).fromNow() : '—' }}</strong></div>
          <div class="dev-meta-item">Owner<br><strong>{{ d.ownerName ?? '—' }}</strong></div>
        </div>

        <!-- Per-connector / per-EVSE status (README 2.3.1) -->
        <div class="conn-grid" @click.stop>
          <template v-if="isOcpp2Device(d)">
            <div v-if="d.bootReason" class="boot-chip" title="Last boot reason">
              <i class="ti ti-power"></i> {{ d.bootReason }}
            </div>
            <div v-for="evse in evseList(d)" :key="'evse-'+evse" class="conn-evse">
              <span class="conn-evse-label">EVSE {{ evse }}</span>
              <div class="conn-row">
                <span
                  v-for="c in connectorList(d)"
                  :key="'c-'+evse+'-'+c"
                  :class="['conn-chip', statusTone(connectorStatus(d, evse, c))]"
                  :title="`EVSE ${evse} / Connector ${c}`"
                >
                  C{{ c }} · {{ connectorStatus(d, evse, c) }}
                </span>
              </div>
            </div>
          </template>
          <template v-else>
            <div class="conn-row">
              <span
                v-for="c in connectorList(d)"
                :key="'c-'+c"
                :class="['conn-chip', statusTone(connectorStatus(d, 0, c))]"
                :title="`Connector ${c}`"
              >
                C{{ c }} · {{ connectorStatus(d, 0, c) }}
              </span>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { devices as devicesApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import { useDevicesStore, isOcpp2 } from '@/stores/devices.js'
import { useEventsStore } from '@/stores/events.js'
import PageHeader from '@/components/PageHeader.vue'
import AppBadge from '@/components/AppBadge.vue'
import AppButton from '@/components/AppButton.vue'
import TenantSelector from '@/components/TenantSelector.vue'

dayjs.extend(relativeTime)
const auth = useAuthStore()
const devicesStore = useDevicesStore()
const eventsStore = useEventsStore()
const router = useRouter()
const deviceList = ref([])
const loading = ref(false)

const onlineCount = computed(() => deviceList.value.filter(d => d.online).length)
const activeCount = computed(() => deviceList.value.reduce((s, d) => s + (d.activeTx || 0), 0))
const faultedCount = computed(() => deviceList.value.filter(d => d.status === 'Faulted').length)

// ── Connector / EVSE status helpers (README 2.3.1) ─────────────────────────
function isOcpp2Device(d) { return isOcpp2(d.protocol) }

// OCPP 1.6: connectors 1..connectorNo (default 2)
// OCPP 2.x: connectors 1..connectorNo (default 1) within each EVSE
function connectorList(d) {
  const n = d.connectorNo > 0 ? d.connectorNo : (isOcpp2Device(d) ? 1 : 2)
  return Array.from({ length: n }, (_, i) => i + 1)
}

// OCPP 2.x: EVSEs 1..evseNo (default 2)
function evseList(d) {
  const n = d.evseNo > 0 ? d.evseNo : 2
  return Array.from({ length: n }, (_, i) => i + 1)
}

function connectorStatus(d, evseId, connectorId) {
  const s = (d.connectorStatuses || []).find(
    cs => cs.evseId === evseId && cs.connectorId === connectorId
  )
  return s ? s.status : (d.online ? 'Unknown' : 'Offline')
}

function statusTone(status) {
  if (status === 'Offline') return 'tone-offline'
  if (status === 'Available') return 'tone-ok'
  if (status === 'Occupied' || status === 'Reserved' || status === 'Charging') return 'tone-busy'
  if (status === 'Faulted') return 'tone-fault'
  return 'tone-unknown'
}

async function fetchDevices() {
  loading.value = true
  try {
    const tid = auth.isCSAdmin && devicesStore.cpOp?.tenantId ? devicesStore.cpOp.tenantId : undefined
    deviceList.value = await devicesApi.list(tid ? { tenant_id: tid } : undefined)
  } catch { deviceList.value = [] }
  finally { loading.value = false }
}

function onTenantChange(payload) {
  devicesStore.setCpOp(payload?.tenantId ? payload : null)
  // A previously selected device from another tenant no longer applies
  if (devicesStore.current && (!payload?.tenantId || devicesStore.current.tenantId !== payload.tenantId)) {
    devicesStore.select(null)
  }
  fetchDevices()
}

// Enter the device's operation interface, routed by OCPP protocol version
// (README 2.3.1).
function openDevice(d) {
  devicesStore.select(d.id)
  router.push(isOcpp2(d.protocol) ? '/ocpp/ocpp201' : '/ocpp/configuration')
}

// Lightweight auto-refresh when device-related events arrive on the WS hub.
let refreshTimer
watch(() => eventsStore.logs.length, () => {
  if (refreshTimer) return
  refreshTimer = setTimeout(() => {
    refreshTimer = null
    fetchDevices()
  }, 2000)
})

onMounted(fetchDevices)
</script>

<style scoped>
.stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin-bottom: 20px; }
.stat { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius); padding: 14px 16px; }
.stat-label { font-size: 11px; color: var(--text2); margin-bottom: 6px; }
.stat-val { font-size: 24px; font-weight: 500; }
.device-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
.dev-card {
  background: var(--surface); border: 0.5px solid var(--border);
  border-radius: var(--radius-lg); padding: 14px 16px; cursor: pointer; transition: all 0.15s;
}
.dev-card:hover { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-light); }
.dev-card.selected { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-light); }
.dev-card-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.dev-name { font-weight: 500; font-size: 14px; }
.dev-protocol { font-size: 11px; color: var(--text2); margin-bottom: 12px; }
.dev-meta { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 12px; }
.dev-meta-item { font-size: 11px; color: var(--text2); line-height: 1.6; }
.dev-meta-item strong { color: var(--text1); font-weight: 500; display: block; }
.loading { display: flex; align-items: center; gap: 8px; color: var(--text2); padding: 40px 0; justify-content: center; }
.empty { color: var(--text2); padding: 40px 0; text-align: center; font-size: 13px; }
.dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }
.dot-green { background: #1d9e75; }
.dot-amber { background: #ef9f27; }
.dot-gray  { background: #888; }
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Connector / EVSE status chips */
.conn-grid { margin-top: 12px; display: flex; flex-direction: column; gap: 6px; }
.conn-row { display: flex; flex-wrap: wrap; gap: 5px; }
.conn-evse { display: flex; flex-direction: column; gap: 3px; }
.conn-evse-label { font-size: 10px; color: var(--text3); text-transform: uppercase; letter-spacing: 0.06em; }
.conn-chip {
  font-size: 10.5px; padding: 2px 8px; border-radius: 10px;
  border: 0.5px solid var(--border-md); background: var(--bg); color: var(--text2);
  white-space: nowrap;
}
.boot-chip {
  display: inline-flex; align-items: center; gap: 5px; align-self: flex-start;
  font-size: 10.5px; padding: 2px 8px; border-radius: 10px;
  border: 0.5px solid var(--border-md); background: var(--bg); color: var(--text2);
}
.tone-ok { color: #1a6b4a; background: #e8f5ee; border-color: #bde3d0; }
.tone-busy { color: #8a5a00; background: #fdf3dd; border-color: #f0dcb2; }
.tone-fault { color: #a32d2d; background: #fcebeb; border-color: #f2c9c9; }
.tone-offline { color: var(--text3); background: var(--bg); }
.tone-unknown { color: var(--text2); }
</style>
