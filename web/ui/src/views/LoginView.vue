<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import BrandMark from '../components/BrandMark.vue'

const auth = useAuthStore()
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
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="login-shell">
    <section class="login-story">
      <div class="login-brand">
        <span class="brand-mark"><BrandMark :size="22" /></span>
        <span>SaveAny-Bot</span>
      </div>

      <div class="login-copy">
        <h1>把 Telegram 里的文件<br />送到该去的地方</h1>
        <p>
          在本机接收链接和文件，按设定的规则归档到本地磁盘、WebDAV、S3 或 Telegram。
          任务由你的服务调度，文件写入你配置的存储。
        </p>

        <div class="login-route" aria-label="工作方式">
          <div class="route-step">
            <span class="route-node"></span>
            <strong>接收</strong>
            <span>消息、直链与图集</span>
          </div>
          <span class="route-line" aria-hidden="true"></span>
          <div class="route-step">
            <span class="route-node"></span>
            <strong>调度</strong>
            <span>按类型分发下载</span>
          </div>
          <span class="route-line" aria-hidden="true"></span>
          <div class="route-step">
            <span class="route-node"></span>
            <strong>归档</strong>
            <span>写入目标存储</span>
          </div>
        </div>
      </div>

      <p class="login-note">自托管，会话与配置保存在本机</p>
    </section>

    <section class="login-panel">
      <form class="login-form" @submit.prevent="submit">
        <div class="login-form-head">
          <h2>管理员登录</h2>
          <p>使用服务启动时生成的管理员密码进入控制台。</p>
        </div>

        <div class="field">
          <label for="login-password">管理员密码</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            autofocus
            required
          />
        </div>

        <div v-if="error" class="form-error">{{ error }}</div>

        <button class="btn primary" type="submit" :disabled="busy || password.length === 0">
          {{ busy ? '正在登录…' : '登录控制台' }}
        </button>

        <p class="login-help">
          首次启动的密码会写入服务日志。也可以通过
          <code>SAVEANY_ADMIN_PASSWORD</code>
          在第一次初始化前指定。
        </p>
      </form>
    </section>
  </div>
</template>
