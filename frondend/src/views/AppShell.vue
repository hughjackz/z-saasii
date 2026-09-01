<template>
  <div class="shell" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <!-- Topbar -->
    <header class="topbar">
      <span class="topbar-collapse" @click="sidebarCollapsed = !sidebarCollapsed" :title="sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'">
        <i :class="sidebarCollapsed ? 'ti ti-menu-2' : 'ti ti-menu-2'"></i>
      </span>
      <span class="topbar-logo"><i class="ti ti-bolt"></i> CSMS</span>
      <div class="topbar-sep"></div>
      <span class="topbar-module">{{ currentRouteName }}</span>
      <span v-if="devicesStore.current" class="topbar-device">
        <i class="ti ti-charging-pile"></i>
        {{ devicesStore.current.name }}
        <span :class="['dot', devicesStore.current.online ? 'dot-green' : 'dot-gray']"></span>
        <span class="topbar-device-protocol">{{ devicesStore.current.protocol }}</span>
      </span>
      <div class="topbar-right">
        <span class="topbar-chip"><i class="ti ti-clock"></i> {{ clock }}</span>
        <div class="user-badge" @click="showUserMenu = !showUserMenu">
          <i class="ti ti-user"></i>
          <span>{{ auth.user?.username }}</span>
          <AppBadge :color="roleColor">{{ auth.role }}</AppBadge>
          <!-- User menu -->
          <div v-if="showUserMenu" class="user-menu" @click.stop>
            <div class="user-menu-item" @click="doLogout">
              <i class="ti ti-logout"></i> Sign out
            </div>
          </div>
        </div>
      </div>
    </header>

    <!-- Sidebar -->
    <nav class="sidebar">
      <div class="nav-section">
        <RouterLink to="/overview" class="nav-item" active-class="active">
          <i class="ti ti-layout-dashboard"></i> Overview
        </RouterLink>
      </div>

      <div v-if="hasAnyOcpp" class="nav-section">
        <div class="nav-label">OCPP</div>
        <!-- OCPP 2.x device selected: only the 2.0.1 console is offered -->
        <RouterLink v-if="ocpp2Selected && auth.hasPermission('ocpp.ocpp201')" to="/ocpp/ocpp201" class="nav-item" active-class="active">
          <i class="ti ti-terminal-2"></i> OCPP 2.0.1 Console
        </RouterLink>
        <template v-else>
          <RouterLink v-if="auth.hasPermission('ocpp.configuration')" to="/ocpp/configuration" class="nav-item" active-class="active">
            <i class="ti ti-settings"></i> Configuration
          </RouterLink>
          <RouterLink v-if="auth.hasPermission('ocpp.transaction')" to="/ocpp/transactions" class="nav-item" active-class="active">
            <i class="ti ti-receipt"></i> Transactions
          </RouterLink>
          <RouterLink v-if="auth.hasPermission('ocpp.action')" to="/ocpp/actions" class="nav-item" active-class="active">
            <i class="ti ti-player-play"></i> Actions
          </RouterLink>
          <RouterLink v-if="auth.hasPermission('ocpp.maintenance')" to="/ocpp/maintenance" class="nav-item" active-class="active">
            <i class="ti ti-tool"></i> Maintenance
          </RouterLink>
          <RouterLink v-if="auth.hasPermission('ocpp.pnc')" to="/ocpp/pnc" class="nav-item" active-class="active">
            <i class="ti ti-certificate"></i> PnC
          </RouterLink>
          <RouterLink v-if="auth.hasPermission('ocpp.smartcharging')" to="/ocpp/smart-charging" class="nav-item" active-class="active">
            <i class="ti ti-bolt"></i> Smart Charging
          </RouterLink>
        </template>
      </div>

      <div v-if="auth.hasPermission('vdv261')" class="nav-section">
        <div class="nav-label">VDV</div>
        <RouterLink to="/vdv261" class="nav-item" active-class="active">
          <i class="ti ti-bus"></i> VDV 261
        </RouterLink>
      </div>

      <div class="nav-section">
        <div class="nav-label">Management</div>
        <RouterLink v-if="auth.hasPermission('management.users')" to="/management/users" class="nav-item" active-class="active">
          <i class="ti ti-users"></i> Users
        </RouterLink>
        <RouterLink v-if="auth.hasPermission('management.devices')" to="/management/devices" class="nav-item" active-class="active">
          <i class="ti ti-cpu"></i> Devices
        </RouterLink>
        <RouterLink v-if="auth.hasPermission('management.certificates')" to="/management/certificates" class="nav-item" active-class="active">
          <i class="ti ti-shield"></i> Certificates
        </RouterLink>
        <RouterLink v-if="auth.hasPermission('management.idtags')" to="/management/idtags" class="nav-item" active-class="active">
          <i class="ti ti-id"></i> ID Tags
        </RouterLink>
        <RouterLink v-if="auth.hasPermission('management.profiles')" to="/management/profiles" class="nav-item" active-class="active">
          <i class="ti ti-file-plus"></i> Profiles
        </RouterLink>
      </div>

      <div class="nav-section">
        <div class="nav-label">System</div>
        <RouterLink to="/events" class="nav-item" active-class="active">
          <i class="ti ti-activity"></i> Events
          <span v-if="events.logs.length" class="nav-badge">{{ Math.min(events.logs.length, 99) }}</span>
        </RouterLink>
      </div>
    </nav>

    <!-- Main -->
    <main class="main" @click="showUserMenu = false">
      <RouterView v-slot="{ Component }">
        <Transition name="fade" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>

    <!-- Footer -->
    <footer class="botbar">CSMS SaaS Platform &nbsp;·&nbsp; © 2026</footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useEventsStore } from '@/stores/events.js'
import { useDevicesStore, isOcpp2 } from '@/stores/devices.js'
import AppBadge from '@/components/AppBadge.vue'

const auth = useAuthStore()
const events = useEventsStore()
const devicesStore = useDevicesStore()
const route = useRoute()
const router = useRouter()

const clock = ref('')
const showUserMenu = ref(false)
const sidebarCollapsed = ref(false)

const currentRouteName = computed(() => route.meta.title || route.name || 'Overview')

const roleColor = computed(() => ({ CS_Admin: 'red', CP_OP: 'amber', CP_OM: 'green' }[auth.role] || 'gray'))

const ocppModules = ['ocpp.configuration','ocpp.transaction','ocpp.action','ocpp.maintenance','ocpp.pnc','ocpp.smartcharging','ocpp.ocpp201']
const hasAnyOcpp = computed(() => ocppModules.some(m => auth.hasPermission(m)))

// OCPP section swaps by selected device protocol (README 2.3.1/2.3.2)
const ocpp2Selected = computed(() => !!devicesStore.current && isOcpp2(devicesStore.current.protocol))

function tick() {
  clock.value = new Date().toLocaleTimeString('en-GB')
}

async function doLogout() {
  await auth.logout()
  events.reset()
  devicesStore.reset()
  router.push('/login')
}

let timer
onMounted(async () => {
  tick()
  timer = setInterval(tick, 1000)
  await devicesStore.fetchAll()
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: var(--sidebar-w) 1fr;
  grid-template-rows: var(--topbar-h) 1fr var(--botbar-h);
  height: 100vh; overflow: hidden;
}
.sidebar-collapsed {
  grid-template-columns: 56px 1fr;
}
.topbar {
  grid-column: 1 / -1;
  background: var(--surface); border-bottom: 0.5px solid var(--border);
  display: flex; align-items: center; padding: 0 18px; gap: 14px; z-index: 100;
}
.topbar-logo { font-size: 15px; font-weight: 600; color: var(--accent); display: flex; align-items: center; gap: 6px; }
.topbar-sep { width: 0.5px; height: 18px; background: var(--border-md); }
.topbar-module { font-size: 13px; font-weight: 500; color: var(--text1); }
.topbar-device {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 500; color: var(--text2);
  padding: 3px 10px; border-radius: 14px;
  border: 0.5px solid var(--border-md); background: var(--bg);
}
.topbar-device i { font-size: 14px; color: var(--accent); }
.topbar-device .dot { width: 6px; height: 6px; border-radius: 50%; }
.topbar-device .dot-green { background: #1d9e75; }
.topbar-device .dot-gray { background: #888; }
.topbar-device-protocol { font-size: 10px; color: var(--text3); }
.topbar-right { margin-left: auto; display: flex; align-items: center; gap: 14px; }
.topbar-chip { font-size: 12px; color: var(--text2); display: flex; align-items: center; gap: 5px; }
.topbar-chip i { font-size: 14px; }
.user-badge {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 10px; border-radius: 20px;
  border: 0.5px solid var(--border-md);
  font-size: 12px; font-weight: 500; cursor: pointer;
  position: relative; user-select: none;
}
.user-badge:hover { background: var(--bg); }
.user-menu {
  position: absolute; top: calc(100% + 6px); right: 0;
  background: var(--surface); border: 0.5px solid var(--border-md);
  border-radius: var(--radius); min-width: 140px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.12); z-index: 200;
}
.user-menu-item {
  display: flex; align-items: center; gap: 8px;
  padding: 9px 14px; font-size: 13px; cursor: pointer; color: var(--text1);
}
.user-menu-item:hover { background: var(--bg); }

/* Sidebar collapse toggle */
.topbar-collapse {
  cursor: pointer; padding: 4px; border-radius: 6px; color: var(--text2);
  display: flex; align-items: center; transition: color 0.15s;
}
.topbar-collapse:hover { color: var(--accent); }
.topbar-collapse i { font-size: 18px; }

/* Sidebar */
.sidebar {
  background: var(--surface); border-right: 0.5px solid var(--border);
  padding: 10px 0; overflow-y: auto; display: flex; flex-direction: column; gap: 2px;
  transition: width 0.2s;
}
.sidebar-collapsed .sidebar {
  width: 56px; overflow: hidden;
}
.sidebar-collapsed .nav-label,
.sidebar-collapsed .nav-item span,
.sidebar-collapsed .nav-badge { display: none; }
.sidebar-collapsed .nav-item { justify-content: center; padding: 7px 10px; margin: 1px 4px; }
.nav-section { margin-bottom: 2px; }
.nav-label {
  font-size: 10px; font-weight: 500; color: var(--text3);
  letter-spacing: 0.08em; text-transform: uppercase;
  padding: 10px 14px 4px;
}
.nav-item {
  display: flex; align-items: center; gap: 8px;
  padding: 7px 14px; cursor: pointer; border-radius: 7px;
  margin: 1px 7px; color: var(--text2); transition: all 0.13s;
  font-size: 13px; text-decoration: none;
}
.nav-item:hover { background: var(--bg); color: var(--text1); }
.nav-item.active { background: var(--accent-light); color: var(--accent); font-weight: 500; }
.nav-item i { font-size: 16px; flex-shrink: 0; }
.nav-badge {
  margin-left: auto; background: var(--accent); color: #fff;
  font-size: 10px; padding: 1px 6px; border-radius: 10px; font-weight: 600;
}

/* Main */
.main { overflow-y: auto; background: var(--bg); padding: 22px 24px; }
.botbar {
  grid-column: 1 / -1;
  background: var(--surface); border-top: 0.5px solid var(--border);
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; color: var(--text3);
}
</style>