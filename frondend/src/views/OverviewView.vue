<template>
  <div>
    <PageHeader title="Overview" subtitle="All devices under your account">
      <template #actions>
        <AppButton :loading="loading" @click="fetchDevices()"><i class="ti ti-refresh"></i> Refresh</AppButton>
      </template>
    </PageHeader>

    <TenantSelector v-if="auth.isCSAdmin" v-model="selectedTenant" @change="onTenantChange" />

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
    <div v-else class="device-grid">
      <div
        v-for="d in deviceList"
        :key="d.id"
        class="dev-card"
        @click="$router.push('/ocpp/configuration')"
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
          <div class="dev-meta-item">Heartbeat<br><strong>{{ d.lastHeartbeat ? dayjs(d.lastHeartbeat).fromNow() : '—' }}</strong></div>
          <div class="dev-meta-item">Owner<br><strong>{{ d.ownerName ?? '—' }}</strong></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { devices as devicesApi } from '@/api/index.js'
import { useAuthStore } from '@/stores/auth.js'
import PageHeader from '@/components/PageHeader.vue'
import AppBadge from '@/components/AppBadge.vue'
import AppButton from '@/components/AppButton.vue'
import TenantSelector from '@/components/TenantSelector.vue'

dayjs.extend(relativeTime)
const auth = useAuthStore()
const selectedTenant = ref('')
const deviceList = ref([])
const loading = ref(false)

const onlineCount = computed(() => deviceList.value.filter(d => d.online).length)
const activeCount = computed(() => deviceList.value.reduce((s, d) => s + (d.activeTx || 0), 0))
const faultedCount = computed(() => deviceList.value.filter(d => d.status === 'Faulted').length)


async function fetchDevices() {
  loading.value = true
  try {
    const tid = auth.isCSAdmin && selectedTenant.value ? selectedTenant.value : undefined
    deviceList.value = await devicesApi.list(tid ? { tenant_id: tid } : undefined)
  } catch { deviceList.value = [] }
  finally { loading.value = false }
}
function onTenantChange() { fetchDevices() }
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
.dev-card-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.dev-name { font-weight: 500; font-size: 14px; }
.dev-protocol { font-size: 11px; color: var(--text2); margin-bottom: 12px; }
.dev-meta { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 12px; }
.dev-meta-item { font-size: 11px; color: var(--text2); line-height: 1.6; }
.dev-meta-item strong { color: var(--text1); font-weight: 500; display: block; }
.loading { display: flex; align-items: center; gap: 8px; color: var(--text2); padding: 40px 0; justify-content: center; }
.dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }
.dot-green { background: #1d9e75; }
.dot-amber { background: #ef9f27; }
.dot-gray  { background: #888; }
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
