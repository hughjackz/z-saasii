<template>
  <button
    :class="['btn', `btn-${variant}`, { 'btn-sm': size === 'sm', 'btn-loading': loading }]"
    :disabled="disabled || loading"
    v-bind="$attrs"
  >
    <i v-if="loading" class="ti ti-loader-2 spin"></i>
    <slot v-else />
  </button>
</template>

<script setup>
defineProps({
  variant: { type: String, default: 'default' }, // default | primary | danger | ghost
  size: { type: String, default: 'md' },
  loading: Boolean,
  disabled: Boolean
})
</script>

<style scoped>
.btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 7px 14px; border-radius: var(--radius);
  border: 0.5px solid var(--border-md);
  background: var(--surface); color: var(--text1);
  font-size: 13px; font-weight: 500; font-family: inherit;
  cursor: pointer; transition: all 0.15s; white-space: nowrap;
}
.btn:hover:not(:disabled) { background: var(--bg); }
.btn:active:not(:disabled) { transform: scale(0.98); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-primary { background: var(--accent); color: #fff; border-color: var(--accent); }
.btn-primary:hover:not(:disabled) { opacity: 0.88; background: var(--accent); }
.btn-danger { color: #e24b4a; border-color: #e24b4a; }
.btn-danger:hover:not(:disabled) { background: #fcebeb; }
.btn-ghost { border-color: transparent; background: transparent; }
.btn-ghost:hover:not(:disabled) { background: var(--bg); }
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>