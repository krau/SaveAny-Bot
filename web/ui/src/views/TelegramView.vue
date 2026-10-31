<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { errorMessage, settingsApi, telegramApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageHeader from '../components/PageHeader.vue'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import { telegramStateLabel, telegramStateTone } from '../utils'
import type { JsonObject, TelegramStatus } from '../types'

const toast = useToastStore()
const confirm = useConfirmStore()
const status = ref<TelegramStatus | null>(null)
const busy = ref(false)
const botAction = ref<'start' | 'stop' | 'restart' | ''>('')
const syncedFlow = ref('')
const phone = ref('')
const code = ref('')
const password = ref('')
const settingsLoaded = ref(false)
let timer: number | undefined

const token = ref('')
const appId = ref(0)
const appHash = ref('')
const rpcRetry = ref(5)
const mediaTimeout = ref(0)
const proxyEnable = ref(false)
const proxyURL = ref('')
const userbotEnable = ref(false)
const userbotSession = ref('')

const botState = computed(() => status.value?.bot.state ?? '')
const canStart = computed(() => !botAction.value && botState.value !== 'running' && botState.value !== 'starting')
const canStop = computed(() => !botAction.value && (botState.value === 'running' || botState.value === 'starting'))
const canRestartBot = computed(() => !botAction.value && (botState.value === 'running' || botState.value === 'error'))

const loginFlow = computed(() => status.value?.userbot.login)
const flowState = computed(() => loginFlow.value?.state ?? '')
const canSubmitCode = computed(() => flowState.value === 'code_required')
const canSubmitPassword = computed(() => flowState.value === 'password_required')
const canRestart = computed(() => {
  if (!loginFlow.value) return true
  if (flowState.value === 'failed' || flowState.value === 'cancelled') return true
  return flowState.value === 'done' && (status.value?.userbot.state === 'login_required' || status.value?.userbot.state === 'error')
})

const flowLabels: Record<string, string> = {
  starting: '正在启动',
  code_required: '等待验证码',
  password_required: '等待两步验证密码',
  verifying: '正在验证',
  failed: '登录失败',
  done: '登录完成',
  cancelled: '已取消',
}
const flowStateLabel = computed(() => flowLabels[flowState.value] ?? '未开始')
const flowTone = computed(() => {
  if (flowState.value === 'done') return 'success'
  if (flowState.value === 'failed') return 'danger'
  if (flowState.value === 'code_required' || flowState.value === 'password_required') return 'warning'
  if (flowState.value === 'starting' || flowState.value === 'verifying') return 'info'
  return 'muted'
})

const loginStep = computed(() => {
  if (flowState.value === 'done') return 4
  if (flowState.value === 'password_required') return 3
  if (flowState.value === 'code_required') return 2
  if (flowState.value === 'starting' || flowState.value === 'verifying') return 1
  return 0
})

function stepClass(step: number): string {
  if (loginStep.value > step) return 'done'
  if (loginStep.value === step) return 'active'
  return ''
}

async function loadStatus() {
  try {
    status.value = await telegramApi.status()
    const flow = status.value.userbot.login
    if (flow?.error) toast.push(flow.error, 'error', 6000)
    if (flow?.state === 'done' && flow.flow_id !== syncedFlow.value) {
      syncedFlow.value = flow.flow_id
      toast.push('登录完成，会话已保存', 'success')
      await loadSettings()
    }
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function loadSettings() {
  try {
    const all = await settingsApi.getAll()
    const telegram = (all.telegram ?? {}) as Record<string, unknown>
    const proxy = (telegram.proxy ?? {}) as Record<string, unknown>
    const userbot = (telegram.userbot ?? {}) as Record<string, unknown>
    token.value = String(telegram.token ?? '')
    appId.value = Number(telegram.app_id ?? 0)
    appHash.value = String(telegram.app_hash ?? '')
    rpcRetry.value = Number(telegram.rpc_retry ?? 5)
    mediaTimeout.value = Number(telegram.media_group_timeout ?? 0)
    proxyEnable.value = Boolean(proxy.enable)
    proxyURL.value = String(proxy.url ?? '')
    userbotEnable.value = Boolean(userbot.enable)
    userbotSession.value = String(userbot.session ?? '')
    settingsLoaded.value = true
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

onMounted(async () => {
  timer = window.setInterval(() => void loadStatus(), 2000)
  await Promise.all([loadStatus(), loadSettings()])
})
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})

async function saveCredentials() {
  busy.value = true
  try {
    const patch: JsonObject = {
      token: token.value,
      app_id: appId.value,
      app_hash: appHash.value,
      rpc_retry: rpcRetry.value,
      media_group_timeout: mediaTimeout.value,
      proxy: { enable: proxyEnable.value, url: proxyURL.value },
      userbot: {
        enable: userbotEnable.value,
        session: userbotSession.value,
      },
    }
    const result = await settingsApi.updateGroup('telegram', patch)
    toast.push(
      result.warnings?.length ? result.warnings.join('；') : '已保存并尝试应用',
      result.warnings?.length ? 'error' : 'success',
    )
    await Promise.all([loadSettings(), loadStatus()])
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  } finally {
    busy.value = false
  }
}

async function startBot() {
  botAction.value = 'start'
  try {
    await telegramApi.startBot()
    toast.push('Bot 正在启动', 'info')
    await loadStatus()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  } finally {
    botAction.value = ''
  }
}

async function stopBot(force = false) {
  botAction.value = 'stop'
  try {
    await telegramApi.stopBot(force)
    toast.push('Bot 已停止', 'success')
    await loadStatus()
  } catch (err) {
    if (
      !force &&
      (await confirm.ask({
        title: '强制停止 Bot',
        message: `${errorMessage(err)}\n仍要强制停止吗？`,
        confirmText: '强制停止',
        danger: true,
      }))
    ) {
      await stopBot(true)
      return
    }
    toast.push(errorMessage(err), 'error')
  } finally {
    botAction.value = ''
  }
}

async function restartBot(force = false) {
  botAction.value = 'restart'
  try {
    await telegramApi.restartBot(force)
    toast.push('Bot 正在重启', 'info')
    await loadStatus()
  } catch (err) {
    if (
      !force &&
      (await confirm.ask({
        title: '强制重启 Bot',
        message: `${errorMessage(err)}\n仍要强制重启吗？`,
        confirmText: '强制重启',
        danger: true,
      }))
    ) {
      await restartBot(true)
      return
    }
    toast.push(errorMessage(err), 'error')
  } finally {
    botAction.value = ''
  }
}

async function startLogin() {
  try {
    await telegramApi.login(phone.value)
    code.value = ''
    password.value = ''
    toast.push('登录流程已启动', 'info')
    await loadStatus()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  }
}

async function submitCode() {
  const flow = loginFlow.value
  if (!flow) return
  try {
    await telegramApi.submitCode(flow.flow_id, code.value)
    code.value = ''
    await loadStatus()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  }
}

async function submitPassword() {
  const flow = loginFlow.value
  if (!flow) return
  try {
    await telegramApi.submitPassword(flow.flow_id, password.value)
    password.value = ''
    await loadStatus()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  }
}

async function cancelLogin() {
  const flow = loginFlow.value
  if (!flow) return
  try {
    await telegramApi.cancelLogin(flow.flow_id)
    toast.push('登录已取消', 'info')
    await loadStatus()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  }
}

async function logoutUserbot() {
  const confirmed = await confirm.ask({
    title: '退出 Userbot',
    message: '退出 Userbot 并删除本地会话？',
    confirmText: '退出登录',
    danger: true,
  })
  if (!confirmed) return
  try {
    await telegramApi.logout()
    toast.push('Userbot 已退出登录', 'success')
    await Promise.all([loadStatus(), loadSettings()])
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  }
}
</script>

<template>
  <div class="stack fade-in">
    <PageHeader title="Telegram" description="管理 Bot 与 Userbot 凭据、连接状态和登录会话。">
      <template #actions>
        <button class="btn small" type="button" @click="loadStatus">
          <AppIcon name="refresh" :size="15" />
          刷新状态
        </button>
      </template>
    </PageHeader>


    <div class="connection-grid">
      <section class="connection-card">
        <div class="connection-head">
          <div class="connection-title">
            <AppIcon name="send" :size="17" />
            Bot
          </div>
          <span class="badge status" :class="telegramStateTone(status?.bot.state)">
            {{ telegramStateLabel(status?.bot.state) }}
          </span>
        </div>
        <div>
          <div class="connection-identity">
            {{ status?.bot.username ? `@${status.bot.username}` : '未连接' }}
          </div>
          <div class="connection-meta">处理消息、命令和文件任务</div>
        </div>
        <div v-if="status?.bot.error" class="field-error">{{ status.bot.error }}</div>
        <div class="connection-actions">
          <LoadingButton
            class="small"
            :class="{ primary: canStart || botAction === 'start' }"
            :loading="botAction === 'start' || botState === 'starting'"
            :disabled="!canStart"
            @click="startBot"
          >
            <AppIcon name="play" :size="14" />
            启动
          </LoadingButton>
          <LoadingButton
            class="small"
            :class="{ danger: canStop || botAction === 'stop' }"
            :loading="botAction === 'stop'"
            :disabled="!canStop"
            @click="stopBot()"
          >
            <AppIcon name="stop" :size="14" />
            停止
          </LoadingButton>
          <LoadingButton
            class="small"
            :loading="botAction === 'restart'"
            :disabled="!canRestartBot"
            @click="restartBot()"
          >
            <AppIcon name="refresh" :size="14" />
            重启
          </LoadingButton>
        </div>
      </section>

      <section class="connection-card">
        <div class="connection-head">
          <div class="connection-title">
            <AppIcon name="users" :size="17" />
            Userbot
          </div>
          <span class="badge status" :class="telegramStateTone(status?.userbot.state)">
            {{ telegramStateLabel(status?.userbot.state) }}
          </span>
        </div>
        <div>
          <div class="connection-identity">
            {{ status?.userbot.username ? `@${status.userbot.username}` : status?.userbot.phone || '未登录' }}
          </div>
          <div class="connection-meta">监听消息、下载媒体与自动保存</div>
        </div>
        <div v-if="status?.userbot.error" class="field-error">{{ status.userbot.error }}</div>
        <div class="connection-actions">
          <button class="btn small danger" type="button" @click="logoutUserbot">
            <AppIcon name="logout" :size="14" />
            退出登录
          </button>
        </div>
      </section>
    </div>

    <div class="telegram-columns">
      <section class="panel">
        <div class="panel-head">
          <div>
            <h2 class="panel-title">凭据</h2>
            <p class="panel-sub">Bot Token 与 Telegram API 配置，保存后尝试立即应用</p>
          </div>
        </div>
        <form v-if="settingsLoaded" class="stack" @submit.prevent="saveCredentials" novalidate>
          <div class="form-grid">
            <div class="field">
              <label for="tg-token">Bot Token</label>
              <input id="tg-token" v-model="token" type="password" autocomplete="off" />
            </div>
            <div class="field">
              <label for="tg-app-id">API ID</label>
              <input id="tg-app-id" v-model.number="appId" type="number" />
            </div>
            <div class="field">
              <label for="tg-app-hash">API Hash</label>
              <input id="tg-app-hash" v-model="appHash" type="password" autocomplete="off" />
            </div>
            <div class="field">
              <label for="tg-rpc-retry">RPC 重试</label>
              <input id="tg-rpc-retry" v-model.number="rpcRetry" type="number" />
            </div>
            <div class="field">
              <label for="tg-media-timeout">媒体组超时（秒）</label>
              <input id="tg-media-timeout" v-model.number="mediaTimeout" type="number" />
            </div>
          </div>

          <div class="form-grid">
            <div class="field">
              <label for="tg-proxy">代理地址</label>
              <input id="tg-proxy" v-model="proxyURL" placeholder="socks5://127.0.0.1:1080" :disabled="!proxyEnable" />
              <label class="checkbox" style="padding-top: 2px">
                <input v-model="proxyEnable" type="checkbox" />
                <span>使用代理</span>
              </label>
            </div>
            <div class="field">
              <label for="tg-session">Userbot 会话文件</label>
              <input id="tg-session" v-model="userbotSession" placeholder="data/usersession.db" />
              <label class="checkbox" style="padding-top: 2px">
                <input v-model="userbotEnable" type="checkbox" />
                <span>启用 Userbot</span>
              </label>
            </div>
          </div>

          <div class="hint">密钥显示为 __MASKED__ 时保持不变即可继续使用已保存的值。</div>
          <div class="modal-actions">
            <LoadingButton class="primary" type="submit" :loading="busy">保存并应用</LoadingButton>
          </div>
        </form>
        <div v-else class="loading">正在加载凭据…</div>
      </section>

      <section class="panel">
        <div class="panel-head">
          <div>
            <h2 class="panel-title">Userbot 登录</h2>
            <p class="panel-sub">在网页中完成手机号、验证码与两步验证</p>
          </div>
          <span class="badge status" :class="flowTone">{{ flowStateLabel }}</span>
        </div>

        <div class="flow-steps" aria-hidden="true">
          <span class="flow-step" :class="stepClass(1)">手机号</span>
          <span class="flow-step" :class="stepClass(2)">验证码</span>
          <span class="flow-step" :class="stepClass(3)">两步验证</span>
        </div>

        <div v-if="canRestart" class="stack">
          <div class="field">
            <label for="tg-phone">手机号（含国家码）</label>
            <input id="tg-phone" v-model="phone" placeholder="+8613800000000" />
          </div>
          <div>
            <button class="btn primary" type="button" @click="startLogin">
              <AppIcon name="send" :size="15" />
              发送验证码
            </button>
          </div>
          <p class="hint">验证码会发送到该 Telegram 账号。登录成功后会话保存在本机。</p>
        </div>

        <div v-else class="stack">
          <div v-if="canSubmitCode" class="field">
            <label for="tg-code">Telegram 验证码</label>
            <div class="row">
              <input id="tg-code" v-model="code" class="grow" placeholder="输入收到的验证码" />
              <button class="btn primary" type="button" @click="submitCode">提交</button>
            </div>
          </div>

          <div v-if="canSubmitPassword" class="field">
            <label for="tg-password">两步验证密码</label>
            <div class="row">
              <input id="tg-password" v-model="password" class="grow" type="password" placeholder="输入两步验证密码" />
              <button class="btn primary" type="button" @click="submitPassword">提交</button>
            </div>
          </div>

          <div v-if="flowState !== 'done'" class="modal-actions" style="justify-content: flex-start">
            <button class="btn ghost" type="button" @click="cancelLogin">取消登录</button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
