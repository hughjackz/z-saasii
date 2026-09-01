import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useDevicesStore, isOcpp2 } from '@/stores/devices.js'

const routes = [
  { path: '/login', name: 'Login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('@/views/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/overview' },
      { path: 'overview', name: 'Overview', component: () => import('@/views/OverviewView.vue'), meta: { title: 'Overview' } },
      { path: 'ocpp/configuration', name: 'OcppConfig', component: () => import('@/views/ocpp/ConfigurationView.vue'), meta: { module: 'ocpp.configuration', title: 'Configuration' } },
      { path: 'ocpp/transactions', name: 'OcppTransactions', component: () => import('@/views/ocpp/TransactionsView.vue'), meta: { module: 'ocpp.transaction', title: 'Transactions' } },
      { path: 'ocpp/actions', name: 'OcppActions', component: () => import('@/views/ocpp/ActionsView.vue'), meta: { module: 'ocpp.action', title: 'Actions' } },
      { path: 'ocpp/maintenance', name: 'OcppMaintenance', component: () => import('@/views/ocpp/MaintenanceView.vue'), meta: { module: 'ocpp.maintenance', title: 'Maintenance' } },
      { path: 'ocpp/pnc', name: 'OcppPnC', component: () => import('@/views/ocpp/PnCView.vue'), meta: { module: 'ocpp.pnc', title: 'PnC' } },
      { path: 'ocpp/smart-charging', name: 'OcppSmartCharging', component: () => import('@/views/ocpp/SmartChargingView.vue'), meta: { module: 'ocpp.smartcharging', title: 'Smart Charging' } },
      { path: 'ocpp/ocpp201', name: 'Ocpp201', component: () => import('@/views/ocpp/Ocpp201View.vue'), meta: { module: 'ocpp.ocpp201', title: 'OCPP 2.0.1 Console' } },
      { path: 'vdv261', name: 'VDV261', component: () => import('@/views/VDV261View.vue'), meta: { module: 'vdv261', title: 'VDV 261' } },
      { path: 'management/users', name: 'Users', component: () => import('@/views/management/UsersView.vue'), meta: { module: 'management.users', title: 'Users' } },
      { path: 'management/certificates', name: 'Certificates', component: () => import('@/views/management/CertificatesView.vue'), meta: { module: 'management.certificates', title: 'Certificates' } },
      { path: 'management/devices', name: 'Devices', component: () => import('@/views/management/DevicesView.vue'), meta: { module: 'management.devices', title: 'Devices' } },
      { path: 'management/idtags', name: 'IdTags', component: () => import('@/views/management/IdTagsView.vue'), meta: { module: 'management.idtags', title: 'ID Tags' } },
      { path: 'management/profiles', name: 'Profiles', component: () => import('@/views/management/ProfilesView.vue'), meta: { module: 'management.profiles', title: 'Profiles' } },
      { path: 'events', name: 'Events', component: () => import('@/views/EventsView.vue'), meta: { title: 'Events' } }
    ]
  },
  { path: '/:catchAll(.*)', redirect: '/' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isLoggedIn) return '/login'
  if (to.path === '/login' && auth.isLoggedIn) return '/'
  // Skip permission check until user data is loaded (fetchMe completes)
  if (!auth.loaded) return
  if (to.meta.module && !auth.hasPermission(to.meta.module)) return '/overview'

  // Protocol-aware routing (README 2.3.1): only acts when a device is
  // selected — the no-selection case is handled per-view after the store
  // finishes loading (see useGlobalDevice).
  const devices = useDevicesStore()
  if (to.path.startsWith('/ocpp/') && devices.current) {
    const targetIs201 = to.path === '/ocpp/ocpp201'
    const deviceIs201 = isOcpp2(devices.current.protocol)
    if (deviceIs201 && !targetIs201) return '/ocpp/ocpp201'
    if (!deviceIs201 && targetIs201) return '/ocpp/configuration'
  }
})
export default router
