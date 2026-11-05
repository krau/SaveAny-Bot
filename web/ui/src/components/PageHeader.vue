<script setup lang="ts">
defineProps<{
  title: string
}>()
</script>

<template>
  <header class="page-head">
    <div>
      <h1>{{ title }}</h1>
    </div>
    <div v-if="$slots.actions" class="page-actions">
      <slot name="actions" />
    </div>
  </header>
</template>
