<script setup lang="ts">
import AppToaster from './components/AppToaster.vue'
import ConfirmHost from './components/ConfirmHost.vue'
</script>

<template>
  <RouterView v-slot="{ Component, route }">
    <Transition name="shell">
      <component :is="Component" :key="route.path === '/login' ? 'login' : 'shell'" />
    </Transition>
  </RouterView>
  <AppToaster />
  <ConfirmHost />
</template>
