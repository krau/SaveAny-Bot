<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 24 })
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 28 28"
    fill="none"
    aria-hidden="true"
    focusable="false"
  >
    <path
      d="M5.5 7.5h6.2a4 4 0 0 1 4 4v5a4 4 0 0 0 4 4h3.3"
      stroke="currentColor"
      stroke-width="1.7"
      stroke-linecap="round"
    />
    <circle cx="5.5" cy="7.5" r="2.2" fill="currentColor" />
    <circle cx="23" cy="20.5" r="2.2" fill="currentColor" />
    <circle cx="15.7" cy="14" r="2.2" fill="var(--accent)" />
  </svg>
</template>
