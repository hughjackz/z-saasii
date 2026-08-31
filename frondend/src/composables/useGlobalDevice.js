import { computed, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { useDevicesStore } from '@/stores/devices.js'

// Global device selection (README 2.3.2): CP_OP and device are chosen in the
// OVERVIEW page and shared by all OCPP views.
//
// Views that require a selected device redirect to /overview when none is
// selected — but only after the store finishes loading, so a page reload
// doesn't bounce while the device list is still being fetched.
export function useGlobalDevice() {
  const router = useRouter()
  const store = useDevicesStore()

  const device = computed(() => store.current)
  const deviceId = computed(() => store.current?.id ?? null)

  watchEffect(() => {
    if (!store.current && !store.loading) {
      if (router.currentRoute.value.path !== '/overview') {
        router.replace('/overview')
      }
    }
  })

  return { device, deviceId }
}
