import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useEventsStore = defineStore('events', () => {
  const logs = ref([])
  const paused = ref(false)
  const maxLogs = 500
  let ws = null
  let reconnectTimer = null
  let stopped = true            // true = 主动断开，禁止自动重连

  function connect(token) {
    stopped = false             // 允许重连
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) ws.close()
    const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
    ws = new WebSocket(`${protocol}://${location.host}/api/events/ws?token=${token}`)

    ws.onmessage = (e) => {
      if (paused.value) return
      try {
        const msg = JSON.parse(e.data)
        logs.value.unshift({ ...msg, id: Date.now() + Math.random() })
        if (logs.value.length > maxLogs) logs.value.splice(maxLogs)
      } catch {}
    }

    ws.onclose = () => {
      if (stopped) return       // 主动断开，不再重连
      reconnectTimer = setTimeout(() => { if (token) connect(token) }, 3000)
    }
  }

  function disconnect() {
    stopped = true              // 阻止 onclose 触发重连
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) { ws.close(); ws = null }
  }

  function clear() { logs.value = [] }
  function togglePause() { paused.value = !paused.value }
  function reset() {
    disconnect()
    clear()
    paused.value = false
  }

  return { logs, paused, connect, disconnect, clear, togglePause, reset }
})
