<template>
  <div>
    <PageHeader title="Transactions" subtitle="Active and historical charging sessions" />
    <DeviceBanner :device="device" />

    <!-- Active -->
    <AppCard style="margin-bottom:14px;padding:0;overflow:hidden">
      <template #header>
        <div style="padding:10px 16px;border-bottom:0.5px solid var(--border);display:flex;align-items:center;gap:8px">
          <AppBadge color="green"><span class="dot dot-green"></span>Active Sessions ({{ active.length }})</AppBadge>
        </div>
      </template>
      <table>
        <thead>
          <tr>
            <th>TX ID</th><th>Device</th>
            <th v-if="ocpp2">EVSE</th><th>Connector</th>
            <th>Started</th><th>Start Meter</th><th>ID Tag</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!active.length"><td :colspan="ocpp2 ? 7 : 6" style="text-align:center;color:var(--text3);padding:20px">No active sessions</td></tr>
          <tr v-for="tx in active" :key="txKey(tx)">
            <td><code>{{ tx.transactionId }}</code></td>
            <td>{{ tx.chargePointId }}</td>
            <td v-if="ocpp2">{{ tx.evseId }}</td>
            <td>{{ tx.connectorId }}</td>
            <td>{{ fmtTime(tx.startTime) }}</td>
            <td>{{ fmtMeter(tx.startMeter) }}</td>
            <td>{{ tx.idTag }}</td>
          </tr>
        </tbody>
      </table>
    </AppCard>

    <!-- History -->
    <AppCard style="padding:0;overflow:hidden">
      <template #header>
        <div style="padding:10px 16px 10px;border-bottom:0.5px solid var(--border);display:flex;align-items:center;gap:10px">
          <span style="font-size:13px;font-weight:500">History</span>
          <div style="margin-left:auto;display:flex;gap:8px">
            <input
              v-model="txSearch"
              :placeholder="ocpp2 ? 'Filter transactionId…' : 'Filter transaction ID…'"
              style="width:170px"
            />
            <select v-model="txPage" style="width:80px">
              <option v-for="n in pageOptions" :key="n" :value="n">Page {{ n }}</option>
            </select>
          </div>
        </div>
      </template>
      <table>
        <thead>
          <tr>
            <th>TX ID</th><th>Device</th>
            <th v-if="ocpp2">EVSE</th><th>Conn</th>
            <th>Start</th><th>Stop</th>
            <th>Duration</th><th>Start kWh</th><th>Stop kWh</th><th>Energy</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td :colspan="ocpp2 ? 10 : 9" style="text-align:center;padding:20px;color:var(--text3)">Loading…</td></tr>
          <tr v-for="tx in history" :key="txKey(tx)">
            <td><code>{{ tx.transactionId }}</code></td>
            <td>{{ tx.chargePointId }}</td>
            <td v-if="ocpp2">{{ tx.evseId }}</td>
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
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import dayjs from 'dayjs'
import { transactions } from '@/api/index.js'
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import { isOcpp2 } from '@/stores/devices.js'
import PageHeader from '@/components/PageHeader.vue'
import AppCard from '@/components/AppCard.vue'
import AppBadge from '@/components/AppBadge.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const { device, deviceId } = useGlobalDevice()
const active = ref([])
const history = ref([])
const loading = ref(false)
const txSearch = ref('')
const txPage = ref(1)
const pageOptions = ref([1])

const ocpp2 = computed(() => !!device.value && isOcpp2(device.value.protocol))

function fmtTime(t) { return t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '—' }
function fmtMeter(v) {
  if (v === null || v === undefined) return '—'
  // Wh (v16) / Wh (v201) — display both as Wh with unit, 201 values are already Wh
  return `${Number(v).toLocaleString()} Wh`
}
function duration(s, e) {
  if (!s || !e) return '—'
  const m = dayjs(e).diff(dayjs(s), 'minute')
  return `${Math.floor(m/60)}h ${m%60}m`
}
// v16 rows use numeric txId, v201 rows use device-generated string txId;
// rows keyed by unique DB id to avoid duplicate keys.
function txKey(tx) {
  return tx.id != null ? `${tx.evseId || 0}-${tx.connectorId}-${tx.transactionId}-${tx.id}` : `${tx.evseId || 0}-${tx.connectorId}-${tx.transactionId}`
}

async function load(device) {
  if (!device) return
  loading.value = true
  try {
    // transactionid filter (README 2.3.2.2.1)
    const params = { page: txPage.value }
    if (txSearch.value) params.transaction_id = txSearch.value.trim()
    const [a, h] = await Promise.all([
      transactions.active(device),
      transactions.list(device, params)
    ])
    active.value = a
    history.value = h.data
    pageOptions.value = Array.from({ length: h.totalPages || 1 }, (_, i) => i + 1)
  } finally { loading.value = false }
}

watch(deviceId, d => load(d), { immediate: true })
watch([txPage, txSearch], () => load(deviceId.value))
</script>

<style scoped>
code { font-size: 11px; background: var(--bg); padding: 2px 6px; border-radius: 4px; }
.dot { width:6px;height:6px;border-radius:50%;display:inline-block; }
.dot-green { background:#1d9e75; }
</style>
