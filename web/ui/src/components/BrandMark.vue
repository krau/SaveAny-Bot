<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 24 })
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 32 32"
    fill="none"
    aria-hidden="true"
    focusable="false"
  >
    <g
      transform="rotate(-5 16 16)"
      stroke="currentColor"
      stroke-width="2.2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path
        d="M5.4 9.3c0-1.3 1-2.3 2.3-2.3h5.1l2.7 2.7h7.7c1.3 0 2.3 1 2.3 2.3v9.6c0 1.3-1 2.3-2.3 2.3H7.7c-1.3 0-2.3-1-2.3-2.3V9.3Z"
      />
      <path d="M16 13.2v7.6" />
      <path d="m12.4 17.4 3.6 3.6 3.6-3.6" />
    </g>
  </svg>
</template>
