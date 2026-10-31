<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toasts'
import BrandMark from '../components/BrandMark.vue'
import LoadingButton from '../components/LoadingButton.vue'

const auth = useAuthStore()
const toast = useToastStore()
const route = useRoute()
const router = useRouter()

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
        error.value = '密码错误'
      } else if (err.status === 429) {
        error.value = '尝试次数过多，请稍后再试'
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
    <svg class="login-trace" viewBox="0 0 1440 900" preserveAspectRatio="none" aria-hidden="true">
      <path d="M-40 300H320L458 360H516" fill="none" stroke="currentColor" vector-effect="non-scaling-stroke" />
      <path d="M924 360H982L1120 300H1480" fill="none" stroke="currentColor" vector-effect="non-scaling-stroke" />
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
        <h1>管理员登录</h1>
        <p>输入服务启动时生成的管理员密码。</p>
      </div>

      <form class="login-form" @submit.prevent="submit" novalidate>
        <div class="field">
          <label for="login-password">管理员密码</label>
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
          登录控制台
        </LoadingButton>
      </form>

      <p class="login-help">
        首次启动的密码会写入服务日志。也可以在初始化前设置
        <code>SAVEANY_ADMIN_PASSWORD</code>。
      </p>
    </section>

    <p class="login-page-footer">自托管，配置与会话保存在本机</p>
  </div>
</template>
