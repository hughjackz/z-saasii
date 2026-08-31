<template>
  <div>
    <PageHeader title="OCPP 2.0.1 Console" subtitle="Read-only overview for the selected OCPP 2.x charge point">
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
      <div class="info-item"><label>Enabled</label><strong>{{ deviceInfo?.enabled ? 'Yes' : 'No' }}</strong></div>
      <div class="info-item"><label>Heartbeat Interval</label><strong>{{ deviceInfo?.heartbeatInterval }} s</strong></div>
      <div class="info-item"><label>Location</label><strong>{{ deviceInfo?.location || '—' }}</strong></div>
      <div class="info-item"><label>Owner</label><strong>{{ deviceInfo?.ownerName || '—' }}</strong></div>
    </div>

    <!-- Active sessions -->
    <AppCard style="margin-bottom:14px;padding:0;overflow:hidden">
      <template #header>
        <div style="padding:10px 16px;border-bottom:0.5px solid var(--border);display:flex;align-items:center;gap:8px">
          <AppBadge color="green"><span class="dot dot-green"></span>Active Sessions ({{ active.length }})</AppBadge>
        </div>
      </template>
      <table>
        <thead><tr><th>TX ID</th><th>Device</th><th>Connector</th><th>Started</th><th>Start Meter</th><th>ID Tag</th></tr></thead>
        <tbody>
          <tr v-if="!active.length"><td colspan="6" style="text-align:center;color:var(--text3);padding:20px">No active sessions</td></tr>
          <tr v-for="tx in active" :key="tx.transactionId">
            <td><code>{{ tx.transactionId }}</code></td>
            <td>{{ tx.chargePointId }}</td>
            <td>{{ tx.connectorId }}</td>
            <td>{{ fmtTime(tx.startTime) }}</td>
            <td>{{ tx.startMeter.toLocaleString() }} Wh</td>
            <td>{{ tx.idTag }}</td>
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
            <th>TX ID</th><th>Device</th><th>Conn</th><th>Start</th><th>Stop</th>
            <th>Duration</th><th>Start kWh</th><th>Stop kWh</th><th>Energy</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!history.length"><td colspan="9" style="text-align:center;padding:20px;color:var(--text3)">No transactions yet</td></tr>
          <tr v-for="tx in history" :key="tx.transactionId">
            <td><code>{{ tx.transactionId }}</code></td>
            <td>{{ tx.chargePointId }}</td>
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
import { ref, computed, onMounted, watch } from 'vue'
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
const loading = ref(false)

function fmtTime(t) { return t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '—' }
function duration(s, e) {
  if (!s || !e) return '—'
  const m = dayjs(e).diff(dayjs(s), 'minute')
  return `${Math.floor(m/60)}h ${m%60}m`
}

// WS events carry no tenant — filter client-side by device name
const deviceEvents = computed(() =>
  eventsStore.logs.filter(e => e.device === device.value?.name).slice(-50)
)

async function refresh() {
  if (!deviceId.value) return
  loading.value = true
  try {
    const [info, a, h] = await Promise.all([
      devicesApi.get(deviceId.value),
      transactions.active(deviceId.value),
      transactions.list(deviceId.value, { page: 1 })
    ])
    deviceInfo.value = info
    active.value = a
    history.value = h.data
  } catch {
    // keep last-known values
  } finally {
    loading.value = false
  }
}

onMounted(refresh)
watch(deviceId, refresh)
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
</style>
