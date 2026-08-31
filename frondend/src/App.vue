<template>
  <RouterView />
</template>

<script setup>
import { onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useEventsStore } from '@/stores/events.js'
import { setToken, clearToken } from '@/api/index.js'

const authStore = useAuthStore()
const events = useEventsStore()

onMounted(async () => {
  // 从 localStorage 恢复 token（如果存在）
  const savedToken = localStorage.getItem('auth_token')
  if (savedToken) {
    setToken(savedToken)
    authStore.token = savedToken
    try {
      await authStore.fetchMe()
      events.connect(authStore.token)
    } catch {
      // 如果 token 无效，清除它
      localStorage.removeItem('auth_token')
      clearToken()
      authStore.token = null
      authStore.user = null
    }
  }
})
</script>