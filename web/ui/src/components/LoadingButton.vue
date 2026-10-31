<script setup lang="ts">
withDefaults(
  defineProps<{
    loading?: boolean
    disabled?: boolean
    type?: 'button' | 'submit' | 'reset'
  }>(),
  {
    loading: false,
    disabled: false,
    type: 'button',
  },
)
</script>

<template>
  <button
    :type="type"
    class="btn"
    :class="{ 'is-loading': loading }"
    :disabled="loading || disabled"
    :aria-busy="loading || undefined"
  >
    <span class="btn-main"><slot /></span>
    <span v-if="loading" class="btn-spinner" aria-hidden="true"></span>
  </button>
</template>
