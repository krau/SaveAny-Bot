<script setup lang="ts">
import { useI18n } from 'vue-i18n'

withDefaults(
  defineProps<{
    variant?: 'rows' | 'cards'
    rows?: number
  }>(),
  {
    variant: 'rows',
    rows: 4,
  },
)

const { t } = useI18n()
</script>

<template>
  <div
    class="page-loading"
    :class="`page-loading--${variant}`"
    role="status"
    :aria-label="t('common.loading')"
  >
    <template v-if="variant === 'cards'">
      <div v-for="row in rows" :key="row" class="page-loading-card">
        <span class="skeleton-block page-loading-icon"></span>
        <span class="skeleton-block page-loading-title"></span>
        <span class="skeleton-block page-loading-text"></span>
        <span class="skeleton-block page-loading-text short"></span>
        <span class="skeleton-block page-loading-footer"></span>
      </div>
    </template>
    <template v-else>
      <div v-for="row in rows" :key="row" class="page-loading-row">
        <span class="skeleton-block page-loading-icon"></span>
        <span class="page-loading-copy">
          <span class="skeleton-block page-loading-title"></span>
          <span class="skeleton-block page-loading-text"></span>
        </span>
        <span class="skeleton-block page-loading-action"></span>
      </div>
    </template>
  </div>
</template>
