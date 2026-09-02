<template>
  <div>
    <PageHeader title="OCPP 2.0.1 Console" subtitle="Overview for the selected OCPP 2.x charge point">
      <template #actions>
        <AppButton :loading="loading" @click="refresh"><i class="ti ti-refresh"></i> Refresh</AppButton>
      </template>
    </PageHeader>

    <DeviceBanner :device="deviceInfo" />

    <!-- Device info -->
    <div class="info-grid">
      <div class="info-item"><label>Device ID</label><strong>{{ deviceInfo?.id }}</strong></div>
      <div class="info-item"><label>Name</label><strong>{{ deviceInfo?.name }}</strong></div>
      <div class="info-item"><label>Protocol</label><strong>{{ deviceInfo?.protocol }}</strong></div>
      <div class="info-item"><label>Status</label>
        <AppBadge :color="deviceInfo?.online ? 'green' : 'gray'">{{ deviceInfo?.online ? deviceInfo.status : 'Offline' }}</AppBadge>
      </div>
      <div class="info-item"><label>Boot Reason</label><strong>{{ deviceInfo?.bootReason || '—' }}</strong></div>
      <div class="info-item"><label>Enabled</label><strong>{{ deviceInfo?.enabled ? 'Yes' : 'No' }}</strong></div>
      <div class="info-item"><label>Heartbeat Interval</label><strong>{{ deviceInfo?.heartbeatInterval }} s</strong></div>
      <div class="info-item"><label>Location</label><strong>{{ deviceInfo?.location || '—' }}</strong></div>
      <div class="info-item"><label>Owner</label><strong>{{ deviceInfo?.ownerName || '—' }}</strong></div>
      <div class="info-item"><label>Topology</label><strong>{{ evseList.length }} EVSE × {{ connectorList.length }} conn</strong></div>
    </div>

    <!-- EVSE / connector statuses (README 2.3.1) -->
    <AppCard style="margin-bottom:14px;padding:0;overflow:hidden">
      <template #header>
        <div style="padding:10px 16px;border-bottom:0.5px solid var(--border);display:flex;align-items:center;gap:8px">
          <span style="font-size:13px;font-weight:500">EVSE / Connector Status</span>
        </div>
      </template>
      <div class="evse-list">
        <div v-if="!evseList.length" style="text-align:center;color:var(--text3);padding:20px">No EVSE configured</div>
        <div v-for="evse in evseList" :key="'evse-'+evse" class="evse-row">
          <span class="evse-label">EVSE {{ evse }}</span>
          <div class="conn-row">
            <span
              v-for="c in connectorList"
              :key="'c-'+evse+'-'+c"
              :class="['conn-chip', statusTone(connectorStatus(evse, c))]"
            >
              C{{ c }} · {{ connectorStatus(evse, c) }}
            </span>
          </div>
        </div>
      </div>
    </AppCard>

    <!-- Active sessions -->
    <AppCard style="margin-bottom:14px;padding:0;overflow:hidden">
      <template #header>
        <div style="padding:10px 16px;border-bottom:0.5px solid var(--border);display:flex;align-items:center;gap:8px">
          <AppBadge color="green"><span class="dot dot-green"></span>Active Sessions ({{ active.length }})</AppBadge>
        </div>
      </template>
      <table>
        <thead><tr><th>TX ID</th><th>Device</th><th>EVSE</th><th>Connector</th><th>Started</th><th>Start Meter</th><th>ID Tag</th></tr></thead>
        <tbody>
          <tr v-if="!active.length"><td colspan="7" style="text-align:center;color:var(--text3);padding:20px">No active sessions</td></tr>
          <tr v-for="tx in active" :key="txKey(tx)">
            <td><code>{{ tx.transactionId }}</code></td>
            <td>{{ tx.chargePointId }}</td>
            <td>{{ tx.evseId }}</td>
            <td>{{ tx.connectorId }}</td>
            <td>{{ fmtTime(tx.startTime) }}</td>
            <td>{{ (tx.startMeter/1000).toFixed(2) }} kWh</td>
            <td>{{ tx.idTag }}</td>
          </tr>
        </tbody>
      </table>
    </AppCard>

    <!-- Transaction events (non-meter events are displayed; MeterValueClock /
         MeterValuePeriodic go to the event log only — README 4.3.5) -->
    <AppCard style="margin-bottom:14px;padding:0;overflow:hidden">
      <template #header>
        <div style="padding:10px 16px;border-bottom:0.5px solid var(--border);display:flex;align-items:center;gap:8px">
          <span style="font-size:13px;font-weight:500">Transaction Events</span>
          <span style="margin-left:auto;font-size:11px;color:var(--text3)">meter-sampling events are logged only</span>
        </div>
      </template>
      <table>
        <thead><tr><th>Time</th><th>TX ID</th><th>EVSE</th><th>Conn</th><th>Event</th><th>Trigger</th><th>Seq</th></tr></thead>
        <tbody>
          <tr v-if="!txEvents.length"><td colspan="7" style="text-align:center;color:var(--text3);padding:20px">No transaction events yet</td></tr>
          <tr v-for="e in txEvents" :key="e.id">
            <td>{{ fmtTime(e.createdAt) }}</td>
            <td><code>{{ e.transactionId }}</code></td>
            <td>{{ e.evseId }}</td>
            <td>{{ e.connectorId }}</td>
            <td><AppBadge :color="eventColor(e.eventType)">{{ e.eventType }}</AppBadge></td>
            <td>{{ e.triggerReason || '—' }}</td>
            <td>{{ e.seqNo }}</td>
          </tr>
        </tbody>
      </table>
    </AppCard>

    <!-- History -->
    <AppCard style="padding:0;overflow:hidden;margin-bottom:14px">
      <template #header>
        <div style="padding:10px 16px;border-bottom:0.5px solid var(--border);display:flex;align-items:center;gap:8px">
          <span style="font-size:13px;font-weight:500">Transaction History</span>
        </div>
      </template>
      <table>
        <thead>
          <tr>
            <th>TX ID</th><th>Device</th><th>EVSE</th><th>Conn</th><th>Start</th><th>Stop</th>
            <th>Duration</th><th>Start kWh</th><th>Stop kWh</th><th>Energy</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!history.length"><td colspan="10" style="text-align:center;padding:20px;color:var(--text3)">No transactions yet</td></tr>
          <tr v-for="tx in history" :key="txKey(tx)">
            <td><code>{{ tx.transactionId }}</code></td>
            <td>{{ tx.chargePointId }}</td>
            <td>{{ tx.evseId }}</td>
            <td>{{ tx.connectorId }}</td>
            <td>{{ fmtTime(tx.startTime) }}</td>
            <td>{{ fmtTime(tx.stopTime) }}</td>
            <td>{{ duration(tx.startTime, tx.stopTime) }}</td>
            <td>{{ (tx.startMeter/1000).toFixed(2) }}</td>
            <td>{{ (tx.stopMeter/1000).toFixed(2) }}</td>
            <td><strong>{{ ((tx.stopMeter - tx.startMeter)/1000).toFixed(2) }} kWh</strong></td>
          </tr>
        </tbody>
      </table>
    </AppCard>

    <!-- Events for this device -->
    <AppCard style="padding:0;overflow:hidden">
      <template #header>
        <div style="padding:10px 16px;border-bottom:0.5px solid var(--border)">
          <span style="font-size:13px;font-weight:500">Device Events</span>
        </div>
      </template>
      <div class="event-list">
        <div v-if="!deviceEvents.length" style="text-align:center;color:var(--text3);padding:20px">No events</div>
        <div v-for="(e, i) in deviceEvents" :key="i" class="event-row">
          <span class="event-time">{{ fmtTime(e.time) }}</span>
          <AppBadge :color="e.level === 'error' ? 'red' : e.level === 'warning' ? 'amber' : 'gray'">{{ e.level }}</AppBadge>
          <span class="event-msg">{{ e.message }}</span>
        </div>
      </div>
    </AppCard>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import dayjs from 'dayjs'
import { transactions, devices as devicesApi } from '@/api/index.js'
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import { useEventsStore } from '@/stores/events.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppCard from '@/components/AppCard.vue'
import AppBadge from '@/components/AppBadge.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const { device, deviceId } = useGlobalDevice()
const eventsStore = useEventsStore()

const deviceInfo = ref(null)
const active = ref([])
const history = ref([])
const txEvents = ref([])
const loading = ref(false)
let pollTimer

const connectorList = computed(() => {
  const n = deviceInfo.value?.connectorNo > 0 ? deviceInfo.value.connectorNo : 1
  return Array.from({ length: n }, (_, i) => i + 1)
})
const evseList = computed(() => {
  const n = deviceInfo.value?.evseNo > 0 ? deviceInfo.value.evseNo : 2
  return Array.from({ length: n }, (_, i) => i + 1)
})

function connectorStatus(evseId, connectorId) {
  const s = (deviceInfo.value?.connectorStatuses || []).find(
    cs => cs.evseId === evseId && cs.connectorId === connectorId
  )
  return s ? s.status : (deviceInfo.value?.online ? 'Unknown' : 'Offline')
}
function statusTone(status) {
  if (status === 'Offline') return 'tone-offline'
  if (status === 'Available') return 'tone-ok'
  if (status === 'Occupied' || status === 'Reserved' || status === 'Charging') return 'tone-busy'
  if (status === 'Faulted') return 'tone-fault'
  return 'tone-unknown'
}
function eventColor(t) {
  return { Started: 'green', Ended: 'red', Updated: 'amber' }[t] || 'gray'
}

function fmtTime(t) { return t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '—' }
function duration(s, e) {
  if (!s || !e) return '—'
  const m = dayjs(e).diff(dayjs(s), 'minute')
  return `${Math.floor(m/60)}h ${m%60}m`
}
function txKey(tx) {
  return tx.id != null ? `${tx.evseId || 0}-${tx.connectorId}-${tx.transactionId}-${tx.id}` : `${tx.evseId || 0}-${tx.connectorId}-${tx.transactionId}`
}

// WS events carry no tenant — filter client-side by device name
const deviceEvents = computed(() =>
  eventsStore.logs.filter(e => e.device === device.value?.name).slice(-50)
)

async function refresh() {
  if (!deviceId.value) return
  loading.value = true
  try {
    const [info, a, h, ev] = await Promise.all([
      devicesApi.get(deviceId.value),
      transactions.active(deviceId.value),
      transactions.list(deviceId.value, { page: 1 }),
      transactions.events(deviceId.value, { limit: 100 })
    ])
    deviceInfo.value = info
    active.value = a
    history.value = h.data
    txEvents.value = ev
  } catch {
    // keep last-known values
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
  // Live status: poll so connection/connector status reflects the device
  // without a manual Refresh (README 4.1).
  pollTimer = setInterval(refresh, 30000)
})
watch(deviceId, refresh)
onUnmounted(() => clearInterval(pollTimer))

// Auto-refresh when device-related events arrive so live connection status /
// connector status update without pressing Refresh (README 4.1).
let refreshTimer
watch(() => eventsStore.logs.length, () => {
  if (refreshTimer) return
  refreshTimer = setTimeout(() => {
    refreshTimer = null
    refresh()
  }, 2000)
})
</script>

<style scoped>
code { font-size: 11px; background: var(--bg); padding: 2px 6px; border-radius: 4px; }
.dot { width:6px;height:6px;border-radius:50%;display:inline-block; }
.dot-green { background:#1d9e75; }
.info-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin-bottom: 16px;
}
.info-item {
  background: var(--surface); border: 0.5px solid var(--border);
  border-radius: var(--radius); padding: 10px 14px;
}
.info-item label { display: block; font-size: 11px; color: var(--text2); margin-bottom: 4px; }
.info-item strong { font-size: 13px; font-weight: 500; color: var(--text1); word-break: break-all; }
.event-list { max-height: 320px; overflow-y: auto; }
.event-row {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 16px; border-bottom: 0.5px solid var(--border);
  font-size: 12px;
}
.event-time { color: var(--text2); flex-shrink: 0; }
.event-msg { color: var(--text1); }

/* EVSE / connector chips */
.evse-list { padding: 8px 16px 14px; }
.evse-row { display: flex; align-items: center; gap: 12px; padding: 6px 0; border-bottom: 0.5px solid var(--border); }
.evse-row:last-child { border-bottom: none; }
.evse-label { font-size: 11px; font-weight: 500; color: var(--text2); min-width: 60px; text-transform: uppercase; letter-spacing: 0.06em; }
.conn-row { display: flex; flex-wrap: wrap; gap: 5px; }
.conn-chip {
  font-size: 11px; padding: 2px 9px; border-radius: 11px;
  border: 0.5px solid var(--border-md); background: var(--bg); color: var(--text2);
}
.tone-ok { color: #1a6b4a; background: #e8f5ee; border-color: #bde3d0; }
.tone-busy { color: #8a5a00; background: #fdf3dd; border-color: #f0dcb2; }
.tone-fault { color: #a32d2d; background: #fcebeb; border-color: #f2c9c9; }
.tone-offline { color: var(--text3); background: var(--bg); }
.tone-unknown { color: var(--text2); }
</style>
