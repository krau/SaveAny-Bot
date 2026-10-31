<script setup lang="ts">
import { useConfirmStore } from '../stores/confirm'
import ConfirmDialog from './ConfirmDialog.vue'

const confirm = useConfirmStore()
</script>

<template>
  <Transition name="modal">
    <ConfirmDialog
      v-if="confirm.open"
      :title="confirm.options.title"
      :message="confirm.options.message"
      :confirm-text="confirm.options.confirmText"
      :cancel-text="confirm.options.cancelText"
      :danger="confirm.options.danger"
      @confirm="confirm.respond(true)"
      @cancel="confirm.respond(false)"
    />
  </Transition>
</template>
