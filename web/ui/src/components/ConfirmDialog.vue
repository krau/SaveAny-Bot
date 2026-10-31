<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { translate } from '../i18n'
import AppIcon from './AppIcon.vue'

withDefaults(
  defineProps<{
    title?: string
    message?: string
    confirmText?: string
    cancelText?: string
    danger?: boolean
    busy?: boolean
  }>(),
  {
    title: translate('common.confirmTitle'),
    message: '',
    confirmText: translate('common.confirm'),
    cancelText: translate('common.cancel'),
    danger: false,
    busy: false,
  },
)

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const cancelButton = ref<HTMLButtonElement | null>(null)

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  emit('cancel')
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  void nextTick(() => cancelButton.value?.focus())
})

onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="modal-overlay confirm-overlay" @click.self="emit('cancel')">
    <div
      class="modal confirm-modal"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
    >
      <div class="confirm-heading">
        <span class="confirm-icon" :class="{ danger }">
          <AppIcon :name="danger ? 'alert' : 'check'" :size="18" />
        </span>
        <div>
          <h3 id="confirm-dialog-title" class="modal-title">{{ title }}</h3>
          <p class="confirm-message">{{ message }}</p>
        </div>
      </div>
      <div class="modal-actions">
        <button
          ref="cancelButton"
          class="btn ghost"
          type="button"
          :disabled="busy"
          @click="emit('cancel')"
        >
          {{ cancelText }}
        </button>
        <button
          class="btn"
          :class="danger ? 'danger-solid' : 'primary'"
          type="button"
          :disabled="busy"
          @click="emit('confirm')"
        >
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>
