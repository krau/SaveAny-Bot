<script setup lang="ts">
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
    title: '确认操作',
    message: '',
    confirmText: '确认',
    cancelText: '取消',
    danger: false,
    busy: false,
  },
)

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()
</script>

<template>
  <Teleport to="body">
    <div class="modal-overlay" @click.self="emit('cancel')">
      <div class="modal" role="dialog" aria-modal="true" style="max-width: 420px">
        <h3 class="modal-title">{{ title }}</h3>
        <p class="muted" style="margin: 0; white-space: pre-line">{{ message }}</p>
        <div class="modal-actions">
          <button class="btn ghost" type="button" :disabled="busy" @click="emit('cancel')">
            {{ cancelText }}
          </button>
          <button
            class="btn"
            :class="danger ? 'danger' : 'primary'"
            type="button"
            :disabled="busy"
            @click="emit('confirm')"
          >
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
