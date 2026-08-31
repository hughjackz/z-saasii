import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { auth as authApi, setToken, clearToken, onUnauthorized, getToken } from '@/api/index.js'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const token = ref(null)
  const loaded = ref(false)  // true after fetchMe() completes

  const isLoggedIn = computed(() => !!token.value && !!getToken())
  const role = computed(() => user.value?.role || '')
  const myid = computed(() => user.value?.id || '')
  const myparentId = computed(() => user.value?.parentId || '')
  const myusername = computed(() => user.value?.username || '')
  const tenantId = computed(() => user.value?.tenantId || '')
  const isCSAdmin = computed(() => role.value === 'CS_Admin')
  const isCpOp = computed(() => role.value === 'CP_OP')
  const isCpOm = computed(() => role.value === 'CP_OM')

  // 当 Axios 收到 401 时，同步清理 Pinia 状态，避免 isLoggedIn 与 _token 不一致
  onUnauthorized(() => {
    token.value = null
    user.value = null
  })

  // Permissions from backend per module
  const permissions = computed(() => user.value?.permissions || [])

  function hasPermission(module) {
    if (isCSAdmin.value) return true
    return permissions.value.includes(module)
  }

  async function login(username, password) {
    const res = await authApi.login(username, password)
    token.value = res.token
    user.value = res.user
    setToken(res.token)
    localStorage.setItem('auth_token', res.token)
  }

  async function logout() {
    try { await authApi.logout() } catch {}
    token.value = null
    user.value = null
    clearToken()
    localStorage.removeItem('auth_token')
  }

  async function fetchMe() {
    try {
      const res = await authApi.me()
      user.value = res
    } finally {
      loaded.value = true
    }
  }
  return { user, token, loaded, isLoggedIn, role, myid, myparentId, myusername, tenantId, isCSAdmin, isCpOp, isCpOm, permissions, hasPermission, login, logout, fetchMe }
})

