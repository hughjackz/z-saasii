<template>
  <div class="login-page">
    <div class="login-box">
      <div class="login-logo">
        <i class="ti ti-bolt"></i>
        <span>CSMS SaaS</span>
      </div>
      <p class="login-sub">Charge Point Management System</p>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="field">
          <label>Username</label>
          <input v-model="form.username" type="text" autocomplete="username" placeholder="Enter username" required />
        </div>
        <div class="field">
          <label>Password</label>
          <input v-model="form.password" type="password" autocomplete="current-password" placeholder="Enter password" required />
        </div>
        <div v-if="error" class="error-msg">{{ error }}</div>
        <button class="login-btn" :disabled="loading" type="submit">
          <i v-if="loading" class="ti ti-loader-2 spin"></i>
          <span v-else>Sign in</span>
        </button>
      </form>
    </div>
    <div class="login-footer">© 2026 CSMS Platform</div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useEventsStore } from '@/stores/events.js'

const router = useRouter()
const auth = useAuthStore()
const events = useEventsStore()

const form = ref({ username: '', password: '' })
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(form.value.username, form.value.password)
    events.connect(auth.token)
    await router.push('/overview')
  } catch (e) {
    error.value = e?.message || 'Invalid username or password'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh; display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  background: var(--bg); padding: 24px;
}
.login-box {
  background: var(--surface); border: 0.5px solid var(--border);
  border-radius: var(--radius-lg); padding: 36px 40px;
  width: 100%; max-width: 380px;
  box-shadow: 0 4px 24px rgba(0,0,0,0.08);
}
.login-logo {
  display: flex; align-items: center; gap: 10px;
  font-size: 20px; font-weight: 600; color: var(--accent); margin-bottom: 4px;
}
.login-logo i { font-size: 24px; }
.login-sub { font-size: 12px; color: var(--text2); margin-bottom: 28px; }
.login-form { display: flex; flex-direction: column; gap: 0; }
.field { margin-bottom: 14px; }
.error-msg {
  font-size: 12px; color: #e24b4a; background: #fcebeb;
  padding: 8px 12px; border-radius: var(--radius); margin-bottom: 12px;
}
.login-btn {
  width: 100%; padding: 10px; border-radius: var(--radius);
  background: var(--accent); color: #fff; border: none;
  font-size: 14px; font-weight: 500; cursor: pointer;
  transition: opacity 0.15s; margin-top: 4px;
}
.login-btn:hover:not(:disabled) { opacity: 0.88; }
.login-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.login-footer { margin-top: 32px; font-size: 11px; color: var(--text3); }
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>