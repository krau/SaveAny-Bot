<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { setLocale } from '../i18n'
import type { Locale } from '../i18n'

const { locale, t } = useI18n()

const options = computed<{ value: Locale; label: string; short: string }[]>(() => [
  { value: 'zh-Hans', label: '中文', short: 'ZH' },
  { value: 'en', label: 'English', short: 'EN' },
])
</script>

<template>
  <div class="locale-switch" role="group" :aria-label="t('common.language')">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :class="{ active: locale === option.value }"
      :aria-label="option.label"
      :aria-pressed="locale === option.value"
      :title="option.label"
      @click="setLocale(option.value)"
    >
      {{ option.short }}
    </button>
  </div>
</template>
