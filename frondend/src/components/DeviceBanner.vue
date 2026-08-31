<template>
  <div v-if="device" class="device-banner">
    <div class="db-left">
      <i class="ti ti-charging-pile" :style="{ color: statusColor }"></i>
      <div class="db-info">
        <div class="db-name">{{ device.name }}</div>
        <div class="db-sub">
          <span v-if="device.location">{{ device.location }} · </span>
          <span>{{ device.protocol }}</span>
          <span v-if="device.ownerName"> · {{ device.ownerName }}</span>
        </div>
      </div>
    </div>
    <div class="db-right">
      <AppBadge :color="device.online ? 'green' : 'gray'">
        <span :class="['dot', device.online ? 'dot-green' : 'dot-gray']"></span>
        {{ device.online ? device.status : 'Offline' }}
      </AppBadge>
      <span class="db-heartbeat">
        <i class="ti ti-heartbeat"></i>
        {{ device.lastHeartbeat ? dayjs(device.lastHeartbeat).fromNow() : '—' }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import AppBadge from '@/components/AppBadge.vue'

dayjs.extend(relativeTime)

const props = defineProps({
  device: { type: Object, default: null }
})

// Status color logic mirrors DeviceSelector: gray offline; green Available;
// amber Charging/Preparing/Finishing; red Faulted.
const statusColor = computed(() => {
  const d = props.device
  if (!d || !d.online) return '#888'
  if (['Charging', 'Preparing', 'Finishing'].includes(d.status)) return '#ef9f27'
  if (d.status === 'Faulted') return '#e24b4a'
  return '#1d9e75'
})
</script>

<style scoped>
.device-banner {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--surface); border: 0.5px solid var(--border);
  border-radius: var(--radius); padding: 10px 14px; margin-bottom: 16px;
}
.db-left { display: flex; align-items: center; gap: 10px; }
.db-left > i { font-size: 22px; }
.db-name { font-size: 13px; font-weight: 600; color: var(--text1); }
.db-sub { font-size: 11px; color: var(--text2); margin-top: 1px; }
.db-right { display: flex; align-items: center; gap: 14px; }
.db-heartbeat { font-size: 12px; color: var(--text2); display: flex; align-items: center; gap: 5px; }
.db-heartbeat i { font-size: 14px; }
.dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }
.dot-green { background: #1d9e75; }
.dot-gray { background: #888; }
</style>
