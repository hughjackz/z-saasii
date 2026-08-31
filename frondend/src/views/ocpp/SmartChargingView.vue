<template>
  <div>
    <PageHeader title="Smart Charging" subtitle="Charging profiles and power management" />
    <DeviceBanner :device="device" />
    <div class="empty-state">
      <i class="ti ti-bolt"></i>
      <p>Smart Charging module — coming in next release</p>
    </div>
  </div>
</template>

<script setup>
import { useGlobalDevice } from '@/composables/useGlobalDevice.js'
import PageHeader from '@/components/PageHeader.vue'
import DeviceBanner from '@/components/DeviceBanner.vue'

const { device } = useGlobalDevice()
</script>

<style scoped>
.empty-state { text-align: center; padding: 60px; color: var(--text3); }
.empty-state i { font-size: 40px; display: block; margin-bottom: 12px; }
.empty-state p { font-size: 14px; }
</style>
