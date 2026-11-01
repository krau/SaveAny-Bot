<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, MASKED_SECRET, settingsApi, telegramApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import HelperBotsSection from '../components/HelperBotsSection.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageHeader from '../components/PageHeader.vue'
import ServerPathPicker from '../components/ServerPathPicker.vue'
import { useFormFeedback } from '../composables/useFormFeedback'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import { telegramStateLabel, telegramStateTone } from '../utils'
import type { JsonObject, TelegramStatus } from '../types'

const { t } = useI18n()
const toast = useToastStore()
const confirm = useConfirmStore()
const { invalidField, fail, clear } = useFormFeedback()
const status = ref<TelegramStatus | null>(null)
const botBusy = ref(false)
const userbotBusy = ref(false)
const loginBusy = ref(false)
const botAction = ref<'start' | 'stop' | 'restart' | ''>('')
const syncedFlow = ref('')
const phoneInput = ref<HTMLInputElement | null>(null)
const phone = ref('')
const code = ref('')
const password = ref('')
const settingsLoaded = ref(false)
const sessionPickerOpen = ref(false)
let timer: number | undefined

const token = ref('')
const appId = ref(0)
const appHash = ref('')
const rpcRetry = ref(5)
const mediaTimeout = ref(0)
const relayChatID = ref(0)
const relayKeepMessages = ref(true)
const proxyEnable = ref(false)
const proxyURL = ref('')
const userbotEnable = ref(false)
const userbotSession = ref('')

const botState = computed(() => status.value?.bot.state ?? '')
const hasSavedSecret = computed(() => token.value === MASKED_SECRET || appHash.value === MASKED_SECRET)
const canStart = computed(() => !botAction.value && botState.value !== 'running' && botState.value !== 'starting')
const canStop = computed(() => !botAction.value && (botState.value === 'running' || botState.value === 'starting'))
const canRestartBot = computed(() => !botAction.value && (botState.value === 'running' || botState.value === 'error'))

const loginFlow = computed(() => status.value?.userbot.login)
const flowState = computed(() => loginFlow.value?.state ?? '')
const canSubmitCode = computed(() => flowState.value === 'code_required')
const canSubmitPassword = computed(() => flowState.value === 'password_required')
const flowActive = computed(() =>
  ['starting', 'code_required', 'password_required', 'verifying'].includes(flowState.value),
)
const flowWaiting = computed(() => flowState.value === 'starting' || flowState.value === 'verifying')

const isUserbotLoggedIn = computed(() => {
  const userbot = status.value?.userbot
  if (!userbot) return false
  return userbot.state === 'running' || userbot.state === 'starting' || Boolean(userbot.username)
})

const hasUserbotSession = computed(() => {
  const userbot = status.value?.userbot
  if (!userbot) return false
  return Boolean(userbot.username) || (userbot.enabled && userbot.state !== 'login_required')
})

const loginCompleted = computed(() => flowState.value === 'done' || isUserbotLoggedIn.value)
const showFlowBadge = computed(() => Boolean(loginFlow.value) && flowState.value !== 'done')

const userbotIdentity = computed(() => {
  const userbot = status.value?.userbot
  if (!userbot) return t('telegram.userbot.notLoggedIn')
  if (userbot.username) return `@${userbot.username}`
  if (userbot.phone) return userbot.phone
  return userbot.enabled ? t('telegram.userbot.sessionConfigured') : t('telegram.userbot.notLoggedIn')
})

const flowLabels = computed<Record<string, string>>(() => ({
  starting: t('telegram.flow.states.starting'),
  code_required: t('telegram.flow.states.codeRequired'),
  password_required: t('telegram.flow.states.passwordRequired'),
  verifying: t('telegram.flow.states.verifying'),
  failed: t('telegram.flow.states.failed'),
  done: t('telegram.flow.states.done'),
  cancelled: t('telegram.flow.states.cancelled'),
}))
const flowStateLabel = computed(() => flowLabels.value[flowState.value] ?? t('telegram.flow.notStarted'))
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
      toast.push(t('telegram.toasts.loginComplete'), 'success')
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
    relayChatID.value = Number(telegram.relay_chat_id ?? 0)
    relayKeepMessages.value = Boolean(telegram.relay_keep_messages ?? false)
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

function isRelayChatID(id: number): boolean {
  return id === 0 || (id < 0 && String(id).startsWith('-100'))
}

function validateBotSettings(): boolean {
  clear()
  if (!token.value.trim()) {
    fail(t('telegram.validation.botToken'), 'tg-token')
    return false
  }
  if (!Number.isInteger(appId.value) || appId.value <= 0) {
    fail(t('telegram.validation.apiId'), 'tg-app-id')
    return false
  }
  if (!appHash.value.trim()) {
    fail(t('telegram.validation.apiHash'), 'tg-app-hash')
    return false
  }
  if (!Number.isInteger(rpcRetry.value) || rpcRetry.value < 0) {
    fail(t('telegram.validation.rpcRetry'), 'tg-rpc-retry')
    return false
  }
  if (!Number.isFinite(mediaTimeout.value) || mediaTimeout.value < 0) {
    fail(t('telegram.validation.mediaTimeout'), 'tg-media-timeout')
    return false
  }
  if (!Number.isInteger(relayChatID.value) || !isRelayChatID(relayChatID.value)) {
    fail(t('telegram.validation.relayChatId'), 'tg-relay-chat-id')
    return false
  }
  if (proxyEnable.value && !proxyURL.value.trim()) {
    fail(t('telegram.validation.proxy'), 'tg-proxy')
    return false
  }
  return true
}

function selectSessionPath(path: string) {
  userbotSession.value = path
  sessionPickerOpen.value = false
}

async function saveBotSettings() {
  if (!validateBotSettings()) return
  botBusy.value = true
  try {
    const patch: JsonObject = {
      token: token.value,
      app_id: appId.value,
      app_hash: appHash.value,
      rpc_retry: rpcRetry.value,
      media_group_timeout: mediaTimeout.value,
      relay_chat_id: relayChatID.value,
      relay_keep_messages: relayKeepMessages.value,
      proxy: { enable: proxyEnable.value, url: proxyURL.value },
    }
    const result = await settingsApi.updateGroup('telegram', patch)
    toast.push(
      result.warnings?.length ? result.warnings.join('; ') : t('telegram.toasts.botSaved'),
      result.warnings?.length ? 'error' : 'success',
    )
    await Promise.all([loadSettings(), loadStatus()])
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  } finally {
    botBusy.value = false
  }
}

function validateUserbotSettings(): boolean {
  clear()
  if (userbotEnable.value && !userbotSession.value.trim()) {
    fail(t('telegram.validation.session'), 'tg-session')
    return false
  }
  return true
}

async function saveUserbotSettings() {
  if (!validateUserbotSettings()) return
  userbotBusy.value = true
  try {
    const patch: JsonObject = {
      userbot: {
        enable: userbotEnable.value,
        session: userbotSession.value,
      },
    }
    const result = await settingsApi.updateGroup('telegram', patch)
    toast.push(
      result.warnings?.length ? result.warnings.join('; ') : t('telegram.toasts.userbotSaved'),
      result.warnings?.length ? 'error' : 'success',
    )
    await Promise.all([loadSettings(), loadStatus()])
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  } finally {
    userbotBusy.value = false
  }
}

async function startBot() {
  botAction.value = 'start'
  try {
    await telegramApi.startBot()
    toast.push(t('telegram.toasts.botStarting'), 'info')
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
    toast.push(t('telegram.toasts.botStopped'), 'success')
    await loadStatus()
  } catch (err) {
    if (
      !force &&
      (await confirm.ask({
        title: t('telegram.confirm.forceStopTitle'),
        message: t('telegram.confirm.forceStopMessage', { error: errorMessage(err) }),
        confirmText: t('common.actions.forceStop'),
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
    toast.push(t('telegram.toasts.botRestarting'), 'info')
    await loadStatus()
  } catch (err) {
    if (
      !force &&
      (await confirm.ask({
        title: t('telegram.confirm.forceRestartTitle'),
        message: t('telegram.confirm.forceRestartMessage', { error: errorMessage(err) }),
        confirmText: t('common.actions.forceRestart'),
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

async function focusUserbotLogin() {
  const panel = document.getElementById('userbot-panel')
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  panel?.scrollIntoView({ behavior: reduceMotion ? 'auto' : 'smooth', block: 'start' })
  await nextTick()
  phoneInput.value?.focus({ preventScroll: true })
}

async function startLogin() {
  clear()
  const normalizedPhone = phone.value.replace(/[\s-]/g, '')
  if (!/^\+?\d{6,15}$/.test(normalizedPhone)) {
    fail(t('telegram.validation.phone'), 'tg-phone')
    return
  }
  loginBusy.value = true
  try {
    await telegramApi.login(phone.value)
    code.value = ''
    password.value = ''
    toast.push(t('telegram.toasts.loginStarted'), 'info')
    await loadStatus()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  } finally {
    loginBusy.value = false
  }
}

async function submitCode() {
  const flow = loginFlow.value
  if (!flow) return
  clear()
  if (!code.value.trim()) {
    fail(t('telegram.validation.code'), 'tg-code')
    return
  }
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
  clear()
  if (!password.value) {
    fail(t('telegram.validation.password'), 'tg-password')
    return
  }
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
    toast.push(t('telegram.toasts.loginCancelled'), 'info')
    await loadStatus()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  }
}

async function logoutUserbot() {
  const confirmed = await confirm.ask({
    title: t('telegram.confirm.logoutTitle'),
    message: t('telegram.confirm.logoutMessage'),
    confirmText: t('common.actions.logout'),
    danger: true,
  })
  if (!confirmed) return
  try {
    await telegramApi.logout()
    toast.push(t('telegram.toasts.logoutDone'), 'success')
    await Promise.all([loadStatus(), loadSettings()])
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('nav.telegram')">
      <template #actions>
        <button class="btn small" type="button" @click="loadStatus">
          <AppIcon name="refresh" :size="15" />
          {{ t('telegram.actions.refreshStatus') }}
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
            {{ status?.bot.username ? `@${status.bot.username}` : t('telegram.bot.notConnected') }}
          </div>
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
            {{ t('telegram.actions.start') }}
          </LoadingButton>
          <LoadingButton
            class="small"
            :class="{ danger: canStop || botAction === 'stop' }"
            :loading="botAction === 'stop'"
            :disabled="!canStop"
            @click="stopBot()"
          >
            <AppIcon name="stop" :size="14" />
            {{ t('telegram.actions.stop') }}
          </LoadingButton>
          <LoadingButton
            class="small"
            :loading="botAction === 'restart'"
            :disabled="!canRestartBot"
            @click="restartBot()"
          >
            <AppIcon name="refresh" :size="14" />
            {{ t('telegram.actions.restart') }}
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
          <div class="connection-identity">{{ userbotIdentity }}</div>
        </div>
        <div v-if="status?.userbot.error" class="field-error">{{ status.userbot.error }}</div>
        <div class="connection-actions">
          <button v-if="!isUserbotLoggedIn" class="btn small primary" type="button" @click="focusUserbotLogin">
            <AppIcon name="key" :size="14" />
            {{ t('common.actions.signIn') }}
          </button>
          <button v-if="hasUserbotSession" class="btn small danger" type="button" @click="logoutUserbot">
            <AppIcon name="logout" :size="14" />
            {{ t('common.actions.logout') }}
          </button>
        </div>
      </section>
    </div>

    <section class="panel telegram-panel">
      <div class="panel-head">
        <h2 class="panel-title">{{ t('telegram.botPanel.title') }}</h2>
      </div>
      <form v-if="settingsLoaded" class="stack" @submit.prevent="saveBotSettings" novalidate>
        <div class="form-grid telegram-form-grid">
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-token' }">
            <label for="tg-token">Bot Token *</label>
            <input
              id="tg-token"
              v-model="token"
              type="password"
              autocomplete="off"
              :aria-invalid="invalidField === 'tg-token' ? 'true' : undefined"
              @input="clear('tg-token')"
            />
          </div>
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-app-id' }">
            <label for="tg-app-id">API ID *</label>
            <input
              id="tg-app-id"
              v-model.number="appId"
              type="number"
              :aria-invalid="invalidField === 'tg-app-id' ? 'true' : undefined"
              @input="clear('tg-app-id')"
            />
          </div>
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-app-hash' }">
            <label for="tg-app-hash">API Hash *</label>
            <input
              id="tg-app-hash"
              v-model="appHash"
              type="password"
              autocomplete="off"
              :aria-invalid="invalidField === 'tg-app-hash' ? 'true' : undefined"
              @input="clear('tg-app-hash')"
            />
          </div>
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-rpc-retry' }">
            <label for="tg-rpc-retry">{{ t('telegram.botPanel.rpcRetry') }}</label>
            <input
              id="tg-rpc-retry"
              v-model.number="rpcRetry"
              type="number"
              :aria-invalid="invalidField === 'tg-rpc-retry' ? 'true' : undefined"
              @input="clear('tg-rpc-retry')"
            />
          </div>
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-media-timeout' }">
            <label for="tg-media-timeout">{{ t('telegram.botPanel.mediaTimeout') }}</label>
            <input
              id="tg-media-timeout"
              v-model.number="mediaTimeout"
              type="number"
              :aria-invalid="invalidField === 'tg-media-timeout' ? 'true' : undefined"
              @input="clear('tg-media-timeout')"
            />
          </div>
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-relay-chat-id' }">
            <label for="tg-relay-chat-id">{{ t('telegram.botPanel.relayChatId') }}</label>
            <input
              id="tg-relay-chat-id"
              v-model.number="relayChatID"
              type="number"
              placeholder="-100..."
              :aria-invalid="invalidField === 'tg-relay-chat-id' ? 'true' : undefined"
              :aria-describedby="'tg-relay-chat-id-hint'"
              @input="clear('tg-relay-chat-id')"
            />
            <span id="tg-relay-chat-id-hint" class="hint">{{ t('telegram.botPanel.relayHint') }}</span>
            <label class="checkbox">
              <input v-model="relayKeepMessages" type="checkbox" />
              <span>{{ t('telegram.botPanel.relayKeepMessages') }}</span>
            </label>
            <span class="hint">{{ t('telegram.botPanel.relayKeepHint') }}</span>
          </div>
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-proxy' }">
            <label for="tg-proxy">{{ t('telegram.botPanel.proxyURL') }}</label>
            <input
              id="tg-proxy"
              v-model="proxyURL"
              placeholder="socks5://127.0.0.1:1080"
              :disabled="!proxyEnable"
              :aria-invalid="invalidField === 'tg-proxy' ? 'true' : undefined"
              @input="clear('tg-proxy')"
            />
            <label class="checkbox">
              <input v-model="proxyEnable" type="checkbox" />
              <span>{{ t('telegram.botPanel.proxyEnable') }}</span>
            </label>
          </div>
        </div>
        <p v-if="hasSavedSecret" class="hint">{{ t('telegram.botPanel.maskedHint') }}</p>
        <div class="modal-actions">
          <LoadingButton class="primary" type="submit" :loading="botBusy">{{ t('common.actions.saveApply') }}</LoadingButton>
        </div>
      </form>
      <div v-else class="loading">{{ t('telegram.botPanel.loading') }}</div>
    </section>

    <section id="userbot-panel" class="panel telegram-panel">
      <div class="panel-head">
        <h2 class="panel-title">{{ t('telegram.userbotPanel.title') }}</h2>
        <span v-if="showFlowBadge" class="badge status" :class="flowTone">{{ flowStateLabel }}</span>
      </div>

      <div class="userbot-layout">
        <div class="userbot-auth">
          <div v-if="loginCompleted" class="userbot-session-state">
            <span class="userbot-session-icon"><AppIcon name="check" :size="17" /></span>
            <strong>{{ isUserbotLoggedIn ? t('telegram.userbot.loggedIn') : t('telegram.flow.states.done') }}</strong>
          </div>

          <template v-else>
            <div class="flow-steps" aria-hidden="true">
              <span class="flow-step" :class="stepClass(1)">{{ t('telegram.steps.phone') }}</span>
              <span class="flow-step" :class="stepClass(2)">{{ t('telegram.steps.code') }}</span>
              <span class="flow-step" :class="stepClass(3)">{{ t('telegram.steps.password') }}</span>
            </div>

            <p v-if="flowWaiting" class="flow-status">
              <AppIcon name="clock" :size="14" />
              {{ flowState === 'starting' ? t('telegram.flow.connecting') : t('telegram.flow.verifying') }}
            </p>
            <p v-if="flowState === 'failed' && loginFlow?.error" class="field-error">
              {{ loginFlow.error }}
            </p>

            <form v-if="!flowActive" class="stack" @submit.prevent="startLogin" novalidate>
              <div class="field" :class="{ 'is-invalid': invalidField === 'tg-phone' }">
                <label for="tg-phone">{{ t('telegram.login.phoneLabel') }}</label>
                <input
                  id="tg-phone"
                  ref="phoneInput"
                  v-model="phone"
                  placeholder="+8613800000000"
                  autocomplete="tel"
                  :aria-invalid="invalidField === 'tg-phone' ? 'true' : undefined"
                  @input="clear('tg-phone')"
                />
              </div>
              <div>
                <LoadingButton class="primary" type="submit" :loading="loginBusy">
                  <AppIcon name="send" :size="15" />
                  {{ t('common.actions.sendCode') }}
                </LoadingButton>
              </div>
              <p class="hint">{{ t('telegram.login.codeHint') }}</p>
            </form>

            <div v-if="canSubmitCode" class="field" :class="{ 'is-invalid': invalidField === 'tg-code' }">
              <label for="tg-code">{{ t('telegram.login.codeLabel') }}</label>
              <div class="row">
                <input
                  id="tg-code"
                  v-model="code"
                  class="grow"
                  :placeholder="t('telegram.login.codePlaceholder')"
                  :aria-invalid="invalidField === 'tg-code' ? 'true' : undefined"
                  @input="clear('tg-code')"
                />
                <button class="btn primary" type="button" @click="submitCode">{{ t('common.actions.submit') }}</button>
              </div>
            </div>

            <div v-if="canSubmitPassword" class="field" :class="{ 'is-invalid': invalidField === 'tg-password' }">
              <label for="tg-password">{{ t('telegram.login.passwordLabel') }}</label>
              <div class="row">
                <input
                  id="tg-password"
                  v-model="password"
                  class="grow"
                  type="password"
                  :placeholder="t('telegram.login.passwordPlaceholder')"
                  :aria-invalid="invalidField === 'tg-password' ? 'true' : undefined"
                  @input="clear('tg-password')"
                />
                <button class="btn primary" type="button" @click="submitPassword">{{ t('common.actions.submit') }}</button>
              </div>
            </div>

            <div v-if="flowActive" class="userbot-cancel">
              <button class="btn ghost small" type="button" @click="cancelLogin">{{ t('common.actions.cancelLogin') }}</button>
            </div>
          </template>
        </div>

        <form v-if="settingsLoaded" class="userbot-settings" @submit.prevent="saveUserbotSettings" novalidate>
          <div class="userbot-settings-head">
            <h3>{{ t('telegram.session.title') }}</h3>
          </div>
          <div class="field" :class="{ 'is-invalid': invalidField === 'tg-session' }">
            <label for="tg-session">{{ t('telegram.session.file') }}<span v-if="userbotEnable"> *</span></label>
            <div class="path-input-row">
              <input
                id="tg-session"
                v-model="userbotSession"
                placeholder="data/userbot_session.db"
                :aria-invalid="invalidField === 'tg-session' ? 'true' : undefined"
                @input="clear('tg-session')"
              />
              <button class="btn small" type="button" @click="sessionPickerOpen = true">
                <AppIcon name="file" :size="14" />
                {{ t('common.pathPicker.browse') }}
              </button>
            </div>
          </div>
          <label class="checkbox">
            <input v-model="userbotEnable" type="checkbox" />
            <span>{{ t('telegram.session.enable') }}</span>
          </label>
          <div class="userbot-settings-actions">
            <LoadingButton class="primary" type="submit" :loading="userbotBusy">{{ t('common.actions.saveSettings') }}</LoadingButton>
          </div>
        </form>
        <div v-else class="loading">{{ t('telegram.session.loading') }}</div>
      </div>
    </section>

    <HelperBotsSection />

    <ServerPathPicker
      :open="sessionPickerOpen"
      mode="file"
      :initial-path="userbotSession"
      @close="sessionPickerOpen = false"
      @select="selectSessionPath"
    />
  </div>
</template>
