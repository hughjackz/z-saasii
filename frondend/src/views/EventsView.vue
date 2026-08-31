<template>
  <div>
    <PageHeader title="Events" subtitle="System logs — real-time and historical" />

    <TenantSelector v-if="auth.isCSAdmin" v-model="selectedTenant" @change="onTenantChange" />

    <div class="toolbar">
      <select v-model="logMode" style="width:140px">
        <option value="realtime">Real-time</option>
        <option value="history">Past logs</option>
      </select>
      <input v-if="logMode === 'history'" v-model="logDate" type="date" style="width:160px" />
      <select v-if="logMode === 'history'" v-model="logLevel" style="width:100px">
        <option value="">All levels</option>
        <option value="info">Info</option>
        <option value="warn">Warn</option>
        <option value="error">Error</option>
      </select>
      <AppButton v-if="logMode === 'history'" @click="loadHistory" :loading="loading">Read</AppButton>
      <AppButton v-if="logMode === 'realtime'" @click="events.togglePause()">
        {{ events.paused ? 'Resume' : 'Pause' }}
      </AppButton>
      <AppButton @click="clearLogs">Clear</AppButton>
      <span class="count">{{ displayLogs.length }} events</span>
    </div>

    <div class="log-table">
      <div class="log-head">
        <span class="col-time">Time</span>
        <span class="col-level">Level</span>
        <span class="col-device">Device</span>
        <span class="col-msg">Message</span>
      </div>
      <div v-for="l in displayLogs" :key="l.id" :class="['log-row', 'log-' + l.level]">
        <span class="col-time">{{ fmtTime(l) }}</span>
        <span class="col-level"><AppBadge :color="levelColor(l.level)" size="sm">{{ l.level }}</AppBadge></span>
        <span class="col-device">{{ l.device || '—' }}</span>
        <span class="col-msg">{{ l.message }}</span>
      </div>
      <div v-if="!displayLogs.length" class="empty">No events.{{ logMode === 'realtime' ? ' Waiting for device activity...' : '' }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import dayjs from 'dayjs'
import { events as eventsApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import { useEventsStore } from '@/stores/events.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppBadge from '@/components/AppBadge.vue'
import TenantSelector from '@/components/TenantSelector.vue'

const auth = useAuthStore()
const events = useEventsStore()
const selectedTenant = ref('')
const logMode = ref('realtime')
const logDate = ref(dayjs().format('YYYY-MM-DD'))
const logLevel = ref('')
const loading = ref(false)
const historyLogs = ref([])

const displayLogs = computed(() =>
  logMode.value === 'realtime' ? events.logs : historyLogs.value
)

function onTenantChange() { if (logMode.value === 'history') loadHistory() }
function clearLogs() { historyLogs.value = []; events.clear() }

async function loadHistory() {
  loading.value = true
  try {
    const params = { date: logDate.value }
    if (logLevel.value) params.level = logLevel.value
    if (auth.isCSAdmin && selectedTenant.value) params.tenant_id = selectedTenant.value
    historyLogs.value = await eventsApi.queryLogs(params)
  } catch { historyLogs.value = [] }
  finally { loading.value = false }
}

function fmtTime(l) { return l.time ? dayjs(l.time).format('MM-DD HH:mm:ss') : '—' }
function levelColor(l) { return { info: 'green', warn: 'amber', error: 'red' }[l] || 'gray' }
</script>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; flex-wrap: wrap; }
.count { margin-left: auto; font-size: 12px; color: var(--text3); }
.log-table { background: var(--surface); border: 0.5px solid var(--border); border-radius: var(--radius-lg); overflow: hidden; font-size: 12px; }
.log-head, .log-row { display: grid; grid-template-columns: 130px 60px 100px 1fr; gap: 8px; padding: 5px 12px; align-items: center; }
.log-head { background: var(--bg); border-bottom: 0.5px solid var(--border-md); font-weight: 500; color: var(--text2); text-transform: uppercase; letter-spacing: 0.06em; font-size: 11px; }
.log-row { border-bottom: 0.5px solid var(--border); }
.log-row:last-child { border-bottom: none; }
.log-error { background: #fcebeb11; }
.log-warn { background: #faeeda11; }
.col-time { font-family: monospace; color: var(--text2); }
.col-level { text-align: center; }
.col-device { font-family: monospace; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.col-msg { color: var(--text1); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.empty { padding: 40px; text-align: center; color: var(--text3); }
</style>
