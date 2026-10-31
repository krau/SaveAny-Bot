<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '../stores/theme'
import type { ThemePreference } from '../stores/theme'
import AppIcon from './AppIcon.vue'

const theme = useThemeStore()
const { t } = useI18n()

const options = computed<{ value: ThemePreference; label: string; icon: string }[]>(() => [
  { value: 'system', label: t('common.theme.system'), icon: 'monitor' },
  { value: 'light', label: t('common.theme.light'), icon: 'sun' },
  { value: 'dark', label: t('common.theme.dark'), icon: 'moon' },
])
</script>

<template>
  <div class="theme-switch" role="group" :aria-label="t('common.theme.label')">
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
