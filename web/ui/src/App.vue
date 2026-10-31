<script setup lang="ts">
import AppToaster from './components/AppToaster.vue'
import ConfirmHost from './components/ConfirmHost.vue'
</script>

<template>
  <RouterView />
  <AppToaster />
  <ConfirmHost />
</template>
