<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { errorMessage } from '../api'
import { useFormFeedback } from '../composables/useFormFeedback'
import { useAuthStore } from '../stores/auth'
import AppIcon from './AppIcon.vue'
import LoadingButton from './LoadingButton.vue'
import BrandMark from './BrandMark.vue'
import LocaleSwitch from './LocaleSwitch.vue'
import ThemeSwitch from './ThemeSwitch.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()

const navigation = [
  {
    labelKey: 'nav.workspace',
    items: [
      { path: '/', labelKey: 'nav.overview', icon: 'dashboard' },
      { path: '/guide', labelKey: 'nav.guide', icon: 'bookOpen' },
      { path: '/tasks', labelKey: 'nav.tasks', icon: 'tasks' },
    ],
  },
  {
    labelKey: 'nav.resources',
    items: [
      { path: '/storages', labelKey: 'nav.storages', icon: 'database' },
      { path: '/users', labelKey: 'nav.users', icon: 'users' },
      { path: '/automation', labelKey: 'nav.automation', icon: 'sliders' },
    ],
  },
  {
    labelKey: 'nav.connections',
    items: [{ path: '/telegram', labelKey: 'nav.telegram', icon: 'send' }],
  },
  {
    labelKey: 'nav.administration',
    items: [
      { path: '/settings', labelKey: 'nav.settings', icon: 'sliders' },
      { path: '/parsers', labelKey: 'nav.parsers', icon: 'plug' },
      { path: '/system', labelKey: 'nav.system', icon: 'server' },
    ],
  },
]

const sidebarOpen = ref(false)
const passwordOpen = ref(false)
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordBusy = ref(false)
const { invalidField, fail, clear, success } = useFormFeedback()

watch(
  () => route.fullPath,
  () => {
    sidebarOpen.value = false
  },
)

function openPasswordDialog() {
  currentPassword.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
  clear()
  passwordOpen.value = true
}

async function submitPassword() {
  clear()
  if (!currentPassword.value) {
    fail(t('login.currentPasswordRequired'), 'current-password')
    return
  }
  if (newPassword.value.length < 8) {
    fail(t('login.passwordTooShort'), 'new-password')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    fail(t('login.passwordMismatch'), 'confirm-password')
    return
  }
  passwordBusy.value = true
  try {
    await auth.changePassword(currentPassword.value, newPassword.value)
    success(t('login.passwordUpdated'))
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (err) {
    fail(errorMessage(err))
  } finally {
    passwordBusy.value = false
  }
}

async function handleLogout() {
  try {
    await auth.logout()
  } catch {
    // The local session is cleared regardless of the server response.
  }
  await router.replace({ path: '/login' })
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  if (passwordOpen.value) {
    passwordOpen.value = false
  } else if (sidebarOpen.value) {
    sidebarOpen.value = false
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">{{ t('common.skipToContent') }}</a>
    <Transition name="backdrop">
      <div v-if="sidebarOpen" class="sidebar-backdrop" @click="sidebarOpen = false"></div>
    </Transition>

    <div class="layout">
      <aside id="primary-navigation" class="sidebar" :class="{ open: sidebarOpen }">
        <div class="brand">
          <span class="brand-mark"><BrandMark :size="22" /></span>
          <div class="brand-copy">
            <strong>SaveAny-Bot</strong>
            <span>{{ t('common.fileRelayConsole') }}</span>
          </div>
          <button class="sidebar-close" type="button" :aria-label="t('common.closeNavigation')" @click="sidebarOpen = false">
            <AppIcon name="close" :size="18" />
          </button>
        </div>

        <nav class="nav" :aria-label="t('common.mainNavigation')">
          <div v-for="group in navigation" :key="group.labelKey" class="nav-group">
            <div class="nav-label">{{ t(group.labelKey) }}</div>
            <RouterLink
              v-for="item in group.items"
              :key="item.path"
              :to="item.path"
              active-class="nav-prefix"
              :class="{ 'is-active': route.path === item.path }"
            >
              <AppIcon :name="item.icon" :size="17" />
              <span>{{ t(item.labelKey) }}</span>
            </RouterLink>
          </div>
        </nav>

        <div class="sidebar-foot">
          <div class="sidebar-status">
            <span class="live-dot"></span>
            <span>{{ t('common.adminConnected') }}</span>
          </div>
          <div class="theme-control">
            <span class="theme-control-label">{{ t('common.language') }}</span>
            <LocaleSwitch />
          </div>
          <div class="theme-control">
            <span class="theme-control-label">{{ t('common.appearance') }}</span>
            <ThemeSwitch />
          </div>
          <div class="sidebar-account">
            <button
              class="sidebar-action"
              type="button"
              :aria-label="t('login.changePassword')"
              :title="t('login.changePassword')"
              @click="openPasswordDialog"
            >
              <AppIcon name="key" :size="14" />
              {{ t('common.actions.changePassword') }}
            </button>
            <button class="sidebar-action" type="button" @click="handleLogout">
              <AppIcon name="logout" :size="14" />
              {{ t('common.actions.logout') }}
            </button>
          </div>
        </div>
      </aside>

      <div class="main">
        <header class="topbar">
          <button
            class="btn ghost icon menu-button"
            type="button"
            :aria-label="t('common.openNavigation')"
            aria-controls="primary-navigation"
            :aria-expanded="sidebarOpen"
            @click="sidebarOpen = true"
          >
            <AppIcon name="menu" :size="19" />
          </button>
          <span class="topbar-mobile-title">SaveAny-Bot</span>
        </header>

        <main id="main-content" class="content" tabindex="-1">
          <RouterView v-slot="{ Component, route: currentRoute }">
            <Transition name="route" mode="out-in" appear>
              <component :is="Component" :key="currentRoute.path" />
            </Transition>
          </RouterView>
        </main>
      </div>
    </div>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="passwordOpen" class="modal-overlay" @click.self="passwordOpen = false">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="password-dialog-title">
          <div class="modal-head">
            <h3 id="password-dialog-title" class="modal-title">{{ t('login.changePassword') }}</h3>
            <button class="btn ghost icon small" type="button" :aria-label="t('common.close')" @click="passwordOpen = false">
              <AppIcon name="close" :size="17" />
            </button>
          </div>
          <form class="stack" @submit.prevent="submitPassword" novalidate>
            <div class="field" :class="{ 'is-invalid': invalidField === 'current-password' }">
              <label for="current-password">{{ t('login.currentPassword') }}</label>
              <input
                id="current-password"
                v-model="currentPassword"
                type="password"
                autocomplete="current-password"
                :aria-invalid="invalidField === 'current-password' ? 'true' : undefined"
                required
                @input="clear('current-password')"
              />
            </div>
            <div class="field" :class="{ 'is-invalid': invalidField === 'new-password' }">
              <label for="new-password">{{ t('login.newPassword') }}</label>
              <input
                id="new-password"
                v-model="newPassword"
                type="password"
                autocomplete="new-password"
                :aria-invalid="invalidField === 'new-password' ? 'true' : undefined"
                required
                @input="clear('new-password')"
              />
              <span class="hint">{{ t('login.passwordHint') }}</span>
            </div>
            <div class="field" :class="{ 'is-invalid': invalidField === 'confirm-password' }">
              <label for="confirm-password">{{ t('login.confirmPassword') }}</label>
              <input
                id="confirm-password"
                v-model="confirmPassword"
                type="password"
                autocomplete="new-password"
                :aria-invalid="invalidField === 'confirm-password' ? 'true' : undefined"
                required
                @input="clear('confirm-password')"
              />
            </div>
            <div class="modal-actions">
              <button class="btn ghost" type="button" @click="passwordOpen = false">{{ t('common.actions.cancel') }}</button>
              <LoadingButton type="submit" class="primary" :loading="passwordBusy">
                {{ t('login.saveNewPassword') }}
              </LoadingButton>
            </div>
          </form>
        </div>
      </div>
      </Transition>
    </Teleport>
  </div>
</template>
