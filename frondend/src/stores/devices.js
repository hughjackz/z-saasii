import { defineStore } from 'pinia'
import { ref } from 'vue'
import { devices as devicesApi } from '@/api/index.js'

export const useDevicesStore = defineStore('devices', () => {
  const list = ref([])
  const current = ref(null)
  const loading = ref(false)

  async function fetchAll() {
    loading.value = true
    try {
      const data = await devicesApi.list()
      // Guard: backend may return null (Go nil slice → JSON null)
      list.value = Array.isArray(data) ? data : []
    } catch {
      list.value = []
    } finally {
      loading.value = false
    }
  }

  function select(deviceId) {
    current.value = list.value.find(d => d.id === deviceId) || null
  }

  function reset() {
    list.value = []
    current.value = null
    loading.value = false
  }

  return { list, current, loading, fetchAll, select, reset }
})