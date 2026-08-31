import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'

const routes = [
  { path: '/login', name: 'Login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('@/views/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/overview' },
      { path: 'overview', name: 'Overview', component: () => import('@/views/OverviewView.vue') },
      { path: 'ocpp/configuration', name: 'OcppConfig', component: () => import('@/views/ocpp/ConfigurationView.vue'), meta: { module: 'ocpp.configuration' } },
      { path: 'ocpp/transactions', name: 'OcppTransactions', component: () => import('@/views/ocpp/TransactionsView.vue'), meta: { module: 'ocpp.transaction' } },
      { path: 'ocpp/actions', name: 'OcppActions', component: () => import('@/views/ocpp/ActionsView.vue'), meta: { module: 'ocpp.action' } },
      { path: 'ocpp/maintenance', name: 'OcppMaintenance', component: () => import('@/views/ocpp/MaintenanceView.vue'), meta: { module: 'ocpp.maintenance' } },
      { path: 'ocpp/pnc', name: 'OcppPnC', component: () => import('@/views/ocpp/PnCView.vue'), meta: { module: 'ocpp.pnc' } },
      { path: 'ocpp/smart-charging', name: 'OcppSmartCharging', component: () => import('@/views/ocpp/SmartChargingView.vue'), meta: { module: 'ocpp.smartcharging' } },
      { path: 'vdv261', name: 'VDV261', component: () => import('@/views/VDV261View.vue'), meta: { module: 'vdv261' } },
      { path: 'management/users', name: 'Users', component: () => import('@/views/management/UsersView.vue'), meta: { module: 'management.users' } },
      { path: 'management/certificates', name: 'Certificates', component: () => import('@/views/management/CertificatesView.vue'), meta: { module: 'management.certificates' } },
      { path: 'management/devices', name: 'Devices', component: () => import('@/views/management/DevicesView.vue'), meta: { module: 'management.devices' } },
      { path: 'management/idtags', name: 'IdTags', component: () => import('@/views/management/IdTagsView.vue'), meta: { module: 'management.idtags' } },
      { path: 'management/profiles', name: 'Profiles', component: () => import('@/views/management/ProfilesView.vue'), meta: { module: 'management.profiles' } },
      { path: 'events', name: 'Events', component: () => import('@/views/EventsView.vue') }
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
})
export default router