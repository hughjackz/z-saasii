<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="modal-overlay" @click.self="$emit('update:modelValue', false)">
        <div class="modal-box" :style="{ width: width || '420px' }">
          <div class="modal-header">
            <span class="modal-title">{{ title }}</span>
            <button class="close-btn" @click="$emit('update:modelValue', false)">
              <i class="ti ti-x"></i>
            </button>
          </div>
          <div class="modal-body"><slot /></div>
          <div v-if="$slots.footer" class="modal-footer"><slot name="footer" /></div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
defineProps({ modelValue: Boolean, title: String, width: String })
defineEmits(['update:modelValue'])
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.45);
  display: flex; align-items: center; justify-content: center; z-index: 200;
}
.modal-box {
  background: var(--surface); border-radius: var(--radius-lg);
  border: 0.5px solid var(--border-md); max-width: 94vw;
  box-shadow: 0 8px 32px rgba(0,0,0,0.15);
}
.modal-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 18px 12px; border-bottom: 0.5px solid var(--border);
}
.modal-title { font-size: 15px; font-weight: 500; }
.close-btn { background: none; border: none; cursor: pointer; color: var(--text2); font-size: 18px; padding: 2px; display: flex; }
.close-btn:hover { color: var(--text1); }
.modal-body { padding: 16px 18px; }
.modal-footer { padding: 12px 18px 16px; border-top: 0.5px solid var(--border); display: flex; justify-content: flex-end; gap: 8px; }
.modal-enter-active, .modal-leave-active { transition: opacity 0.2s; }
.modal-enter-active .modal-box, .modal-leave-active .modal-box { transition: transform 0.2s; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.modal-enter-from .modal-box, .modal-leave-to .modal-box { transform: scale(0.95); }
</style>