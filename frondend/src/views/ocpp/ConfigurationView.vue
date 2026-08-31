<template>
  <div>
    <PageHeader title="Configuration" subtitle="Read and set charge point configuration keys">
      <template #actions>
        <AppButton @click="readAll" :loading="loadingAll"><i class="ti ti-refresh"></i>Read All</AppButton>
        <AppButton @click="readSelected" :loading="loadingSelected" :disabled="!hasSelection"><i class="ti ti-download"></i>Read Selected</AppButton>
        <AppButton variant="primary" @click="setSelected" :loading="loadingSet" :disabled="!hasDirtySelected"><i class="ti ti-device-floppy"></i>Set Selected</AppButton>
      </template>
    </PageHeader>

    <DeviceBanner :device="device" />

    <div class="toolbar">
      <input v-model="search" placeholder="Search key…" style="width:240px" />
      <label style="display:flex;align-items:center;gap:6px;margin:0;font-size:12px;color:var(--text2)">
        <input type="checkbox" v-model="showReadonly" style="width:auto" />
        Show read-only
      </label>
      <span v-if="!configInitialized" class="hint-fetch">
        <i class="ti ti-info-circle"></i> First time — click "Read All" to load configuration
      </span>
      <span style="margin-left:auto;font-size:12px;color:var(--text3)">
        {{ fetchedCount }}/{{ configs.length }} keys loaded · {{ dirtyCount }} modified
      </span>
    </div>

    <div class="cfg-table">
      <div class="cfg-head">
        <input type="checkbox" @change="toggleAll" :checked="allSelected" />
        <span class="col-key">Key</span>
        <span class="col-val">Value</span>
        <span class="col-ro">Status</span>
      </div>
      <div v-if="loading && !configInitialized" class="empty"><i class="ti ti-loader-2 spin"></i> Loading configuration…</div>
      <div
        v-for="cfg in filtered"
        :key="cfg.key"
        :class="['cfg-row', {
          'cfg-row--dirty': cfg.dirty,
          'cfg-row--unfetched': !cfg.fetched
        }]"
      >
        <input type="checkbox" v-model="cfg.selected" />
        <span class="col-key">{{ cfg.key }}</span>
        <input
          v-if="!cfg.readonly && cfg.fetched"
          class="col-val-input"
          v-model="cfg.pendingValue"
          @input="cfg.dirty = cfg.pendingValue !== cfg.value"
        />
        <span v-else-if="!cfg.fetched" class="col-val-ro col-val-placeholder">— not loaded —</span>
        <span v-else class="col-val-ro">{{ cfg.value }}</span>
        <span class="col-ro">
          <AppBadge v-if="cfg.readonly" color="gray">read-only</AppBadge>
          <AppBadge v-else-if="cfg.dirty" color="amber">modified</AppBadge>
          <AppBadge v-else-if="cfg.fetched" color="green">loaded</AppBadge>
          <AppBadge v-else color="gray">pending</AppBadge>
        </span>
      </div>
      <div v-if="!loading && !filtered.length && configInitialized" class="empty">No configuration keys match your search.</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ocppConfig } from '@/api/index.js'
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import PageHeader from '@/components/PageHeader.vue'
import AppButton from '@/components/AppButton.vue'
import AppBadge from '@/components/AppBadge.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const { device, deviceId } = useGlobalDevice()
const configs = ref([])
const configInitialized = ref(false)   // true once we've done at least one fetchAll
const loading = ref(false)
const loadingAll = ref(false)
const loadingSelected = ref(false)
const loadingSet = ref(false)
const search = ref('')
const showReadonly = ref(true)

const filtered = computed(() => {
  let list = configs.value
  if (!showReadonly.value) list = list.filter(c => !c.readonly)
  if (search.value) list = list.filter(c => c.key.toLowerCase().includes(search.value.toLowerCase()))
  return list
})

const allSelected = computed(() => filtered.value.length > 0 && filtered.value.every(c => c.selected))
const hasSelection = computed(() => configs.value.some(c => c.selected))
const hasDirtySelected = computed(() => configs.value.some(c => c.selected && c.dirty && !c.readonly))
const fetchedCount = computed(() => configs.value.filter(c => c.fetched).length)
const dirtyCount = computed(() => configs.value.filter(c => c.dirty).length)

function toggleAll(e) { filtered.value.forEach(c => c.selected = e.target.checked) }

// Fetch all config keys from the device (first-time init or explicit "Read All")
async function loadConfigs(device) {
  if (!device) {
    configInitialized.value = false
    configs.value = []
    return
  }
  loading.value = true
  try {
    const res = await ocppConfig.getAll(device)
    configs.value = res.configurationKey.map(c => ({
      key: c.key, value: c.value || '', pendingValue: c.value || '',
      readonly: c.readonly, selected: false, dirty: false, fetched: true
    }))
    configInitialized.value = true
  } catch {
    // Keep existing data on error — don't wipe
  } finally { loading.value = false }
}

// "Read All" button
async function readAll() {
  loadingAll.value = true
  try { await loadConfigs(deviceId.value) } finally { loadingAll.value = false }
}

// "Read Selected" — incremental fetch for selected keys
async function readSelected() {
  const keys = configs.value.filter(c => c.selected && !c.fetched || c.selected).map(c => c.key)
  // If no explicit selection, read only unfetched keys
  const targets = keys.length ? keys : configs.value.filter(c => !c.fetched).map(c => c.key)
  if (!targets.length) return
  loadingSelected.value = true
  try {
    const res = await ocppConfig.getKeys(deviceId.value, targets)
    // Also handle unknown keys from response
    const unknownKeys = res.unknownKey || []
    res.configurationKey.forEach(r => {
      let cfg = configs.value.find(c => c.key === r.key)
      if (!cfg) {
        // New key discovered
        cfg = { key: r.key, value: '', pendingValue: '', readonly: r.readonly, selected: false, dirty: false, fetched: false }
        configs.value.push(cfg)
      }
      cfg.value = r.value || ''
      cfg.pendingValue = r.value || ''
      cfg.readonly = r.readonly
      cfg.dirty = false
      cfg.fetched = true
    })
    // Mark unknown keys as fetched too (they don't exist on device)
    unknownKeys.forEach(k => {
      const cfg = configs.value.find(c => c.key === k)
      if (cfg) {
        cfg.fetched = true
        cfg.value = '(unknown)'
        cfg.pendingValue = '(unknown)'
      }
    })
    if (!configInitialized.value && configs.value.length > 0) {
      configInitialized.value = true
    }
  } finally { loadingSelected.value = false }
}

// "Set Selected" — submit dirty + selected + non-readonly configs
async function setSelected() {
  const dirty = configs.value.filter(c => c.selected && c.dirty && !c.readonly)
  if (!dirty.length) return
  loadingSet.value = true
  try {
    await ocppConfig.setKeys(deviceId.value, dirty.map(c => ({ key: c.key, value: c.pendingValue })))
    dirty.forEach(c => { c.value = c.pendingValue; c.dirty = false })
  } finally { loadingSet.value = false }
}

// Clear configs when device changes (user clicks Read All to fetch)
watch(deviceId, () => {
  configInitialized.value = false
  configs.value = []
})
</script>

<style scoped>
.toolbar {
  display: flex; align-items: center; gap: 10px; margin-bottom: 10px; flex-wrap: wrap;
}
.hint-fetch {
  font-size: 12px; color: var(--accent);
  display: flex; align-items: center; gap: 5px;
}
.cfg-table {
  background: var(--surface); border: 0.5px solid var(--border);
  border-radius: var(--radius-lg); overflow: hidden;
}
.cfg-head {
  display: grid; grid-template-columns: 32px 1fr 1fr 100px;
  padding: 8px 14px; background: var(--bg);
  border-bottom: 0.5px solid var(--border-md);
  font-size: 11px; font-weight: 500; color: var(--text2);
  text-transform: uppercase; letter-spacing: 0.06em; align-items: center; gap: 10px;
}
.cfg-row {
  display: grid; grid-template-columns: 32px 1fr 1fr 100px;
  padding: 5px 14px; border-bottom: 0.5px solid var(--border);
  align-items: center; gap: 10px; transition: background 0.1s;
}
.cfg-row:last-child { border-bottom: none; }
.cfg-row:hover { background: var(--bg); }
.cfg-row--dirty { background: #faeeda22; }
.cfg-row--unfetched { opacity: 0.55; }
.col-key { font-size: 12px; font-family: monospace; color: var(--text1); }
.col-val-input {
  font-size: 12px; font-family: monospace; padding: 3px 8px;
  border-radius: 4px; border: 0.5px solid transparent;
  background: var(--bg); color: var(--text1); width: 100%;
}
.col-val-input:focus { border-color: var(--accent); }
.col-val-ro { font-size: 12px; font-family: monospace; color: var(--text2); }
.col-val-placeholder { font-style: italic; color: var(--text3); }
.col-ro { display: flex; }
.empty {
  padding: 40px; text-align: center; color: var(--text3);
  display: flex; align-items: center; justify-content: center; gap: 8px;
}
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
