import { defineStore } from 'pinia'
import { ref } from 'vue'
import { devices as devicesApi } from '@/api/index.js'

// sessionStorage keys for the global CP_OP / device selection (README 2.3.1/2.3.2)
const KEY_DEVICE = 'csms.selectedDeviceId'
const KEY_CPOP = 'csms.selectedCpOp'

// Returns true for OCPP 2.x protocols (OCPP201, OCPP2.0.1, OCPP21, ...)
export function isOcpp2(protocol) {
  return (protocol || '').toUpperCase().startsWith('OCPP2')
}

export const useDevicesStore = defineStore('devices', () => {
  const list = ref([])
  const current = ref(null)
  const loading = ref(false)
  // CP_OP selection (only meaningful for CS_Admin): { tenantId, tenantName }
  const cpOp = ref(null)

  function loadPersistedCpOp() {
    try {
      const raw = sessionStorage.getItem(KEY_CPOP)
      if (raw) cpOp.value = JSON.parse(raw)
    } catch { cpOp.value = null }
  }
  loadPersistedCpOp()

  async function fetchAll() {
    loading.value = true
    try {
      const data = await devicesApi.list()
      // Guard: backend may return null (Go nil slice → JSON null)
      list.value = Array.isArray(data) ? data : []
      // Reconcile the persisted selection against the fresh list:
      // refresh the current object (status/online/lastHeartbeat change) or
      // clear it when the device no longer exists. No auto-select.
      const persistedId = sessionStorage.getItem(KEY_DEVICE)
      if (persistedId) {
        current.value = list.value.find(d => d.id === persistedId) || null
      } else {
        current.value = null
      }
    } catch {
      list.value = []
    } finally {
      loading.value = false
    }
  }

  function select(deviceId) {
    current.value = list.value.find(d => d.id === deviceId) || null
    if (current.value) {
      sessionStorage.setItem(KEY_DEVICE, deviceId)
    } else {
      sessionStorage.removeItem(KEY_DEVICE)
    }
  }

  function setCpOp(payload) {
    cpOp.value = payload || null
    if (cpOp.value) {
      sessionStorage.setItem(KEY_CPOP, JSON.stringify(cpOp.value))
    } else {
      sessionStorage.removeItem(KEY_CPOP)
    }
  }

  function reset() {
    list.value = []
    current.value = null
    cpOp.value = null
    loading.value = false
    sessionStorage.removeItem(KEY_DEVICE)
    sessionStorage.removeItem(KEY_CPOP)
  }

  return { list, current, cpOp, loading, fetchAll, select, setCpOp, reset }
})
