<script setup lang="ts">
import { useToastStore } from '../stores/toasts'
import type { ToastTone } from '../stores/toasts'
import AppIcon from './AppIcon.vue'

const toasts = useToastStore()

const icons: Record<ToastTone, string> = {
  success: 'check',
  error: 'alert',
  info: 'file',
}
</script>

<template>
  <div class="toast-stack" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <div
        v-for="item in toasts.items"
        :key="item.id"
        class="toast"
        :class="item.tone"
        :role="item.tone === 'error' ? 'alert' : 'status'"
      >
        <span class="toast-icon"><AppIcon :name="icons[item.tone]" :size="17" /></span>
        <p>{{ item.message }}</p>
        <button class="toast-close" type="button" aria-label="关闭通知" @click="toasts.dismiss(item.id)">
          <AppIcon name="close" :size="15" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>
