<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toasts'
import BrandMark from '../components/BrandMark.vue'
import LoadingButton from '../components/LoadingButton.vue'
import LocaleSwitch from '../components/LocaleSwitch.vue'
import ThemeSwitch from '../components/ThemeSwitch.vue'

const auth = useAuthStore()
const toast = useToastStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await auth.login(password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.status === 401) {
        error.value = t('login.invalidPassword')
      } else if (err.status === 429) {
        error.value = t('login.tooManyAttempts')
      } else {
        error.value = err.message
      }
    } else {
      error.value = err instanceof Error ? err.message : String(err)
    }
    toast.push(error.value, 'error', 5000)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-preferences"><LocaleSwitch /><ThemeSwitch /></div>
    <svg class="login-trace" viewBox="0 0 1440 900" preserveAspectRatio="none" aria-hidden="true">
      <path pathLength="1" d="M-40 300H320L458 360H516" fill="none" stroke="currentColor" vector-effect="non-scaling-stroke" />
      <path pathLength="1" d="M924 360H982L1120 300H1480" fill="none" stroke="currentColor" vector-effect="non-scaling-stroke" />
      <circle cx="320" cy="300" r="3" fill="currentColor" />
      <circle cx="1120" cy="300" r="3" fill="currentColor" />
      <circle cx="982" cy="360" r="3" fill="var(--accent)" />
    </svg>

    <section class="login-card">
      <div class="login-brand">
        <span class="brand-mark login-brand-mark"><BrandMark :size="20" /></span>
        <span>SaveAny-Bot</span>
      </div>

      <div class="login-heading">
        <h1>{{ t('login.title') }}</h1>
      </div>

      <form class="login-form" @submit.prevent="submit" novalidate>
        <div class="field">
          <label for="login-password">{{ t('login.password') }}</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            autofocus
            required
            :aria-invalid="error ? 'true' : undefined"
            @input="error = ''"
          />
        </div>

        <LoadingButton
          type="submit"
          class="primary"
          :loading="busy"
          :disabled="password.length === 0"
        >
          {{ t('login.submit') }}
        </LoadingButton>
      </form>

      <p class="login-help">
        {{ t('login.helpBefore') }}<code>SAVEANY_ADMIN_PASSWORD</code>{{ t('login.helpAfter') }}
      </p>
    </section>

  </div>
</template>
