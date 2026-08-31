<template>
  <AppModal :modelValue="modelValue" :title="title" width="380px" @update:modelValue="$emit('update:modelValue', $event)">
    <p style="font-size:13px;color:var(--text2);line-height:1.6">{{ message }}</p>
    <template #footer>
      <AppButton @click="$emit('update:modelValue', false)">Cancel</AppButton>
      <AppButton :variant="danger ? 'danger' : 'primary'" :loading="loading" @click="$emit('confirm')">
        {{ confirmText || 'Confirm' }}
      </AppButton>
    </template>
  </AppModal>
</template>

<script setup>
import AppModal from './AppModal.vue'
import AppButton from './AppButton.vue'
defineProps({ modelValue: Boolean, title: String, message: String, confirmText: String, danger: Boolean, loading: Boolean })
defineEmits(['update:modelValue', 'confirm'])
</script>