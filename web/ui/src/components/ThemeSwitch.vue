<script setup lang="ts">
import { useThemeStore } from '../stores/theme'
import type { ThemePreference } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const theme = useThemeStore()

const options: { value: ThemePreference; label: string; icon: string }[] = [
  { value: 'system', label: '跟随系统', icon: 'monitor' },
  { value: 'light', label: '浅色', icon: 'sun' },
  { value: 'dark', label: '深色', icon: 'moon' },
]
</script>

<template>
  <div class="theme-switch" role="group" aria-label="主题">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :class="{ active: theme.preference === option.value }"
      :aria-label="option.label"
      :aria-pressed="theme.preference === option.value"
      :title="option.label"
      @click="theme.setPreference(option.value)"
    >
      <AppIcon :name="option.icon" :size="15" />
    </button>
  </div>
</template>
