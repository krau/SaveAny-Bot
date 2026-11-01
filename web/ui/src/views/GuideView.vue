<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { errorMessage, onboardingApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageHeader from '../components/PageHeader.vue'
import { useToastStore } from '../stores/toasts'
import type { OnboardingStatus } from '../types'
import { formatDateTime, telegramStateLabel } from '../utils'

const { t, locale } = useI18n()
const toast = useToastStore()
const status = ref<OnboardingStatus | null>(null)
const busy = ref(false)
const loadError = ref('')

const requiredTotal = 3
const requiredDone = computed(() => {
  const steps = status.value?.steps
  if (!steps) return 0
  return [steps.bot_configured, steps.storage_ready, steps.user_ready].filter(Boolean).length
})
const progressWidth = computed(() => `${(requiredDone.value / requiredTotal) * 100}%`)
const ready = computed(() => Boolean(status.value?.complete))
const botReady = computed(() => ready.value && Boolean(status.value?.steps.bot_running))
const heroTone = computed(() => {
  if (!ready.value) return 'warning'
  return botReady.value ? 'success' : 'info'
})
const heroBadge = computed(() => {
  if (!ready.value) return t('guide.hero.setupBadge')
  return botReady.value ? t('guide.hero.readyBadge') : t('guide.hero.configuredBadge')
})
const heroTitle = computed(() => {
  if (!ready.value) return t('guide.hero.setupTitle')
  return botReady.value ? t('guide.hero.readyTitle') : t('guide.hero.configuredTitle')
})
const heroBody = computed(() => {
  if (!ready.value) return t('guide.hero.setupBody')
  if (botReady.value) return t('guide.hero.readyBody')
  return t('guide.hero.configuredBody', { state: telegramStateLabel(status.value?.bot_state) })
})
const nextPath = computed(() => steps.value.find((step) => step.required && !step.done)?.to ?? '/telegram')
const docsUrl = computed(() => {
  const base = String(locale.value).startsWith('en') ? 'https://sabot.unv.app/en' : 'https://sabot.unv.app'
  return `${base}/deployment/migration/`
})

const steps = computed(() => {
  const current = status.value
  if (!current) return []
  const s = current.steps
  return [
    {
      key: 'bot',
      required: true,
      done: s.bot_configured,
      title: t('guide.setup.steps.bot.title'),
      body: t('guide.setup.steps.bot.body'),
      hint: t('guide.setup.steps.bot.hint'),
      to: '/telegram',
      action: t('guide.setup.open'),
    },
    {
      key: 'storage',
      required: true,
      done: s.storage_ready,
      title: t('guide.setup.steps.storage.title'),
      body: t('guide.setup.steps.storage.body'),
      hint: t('guide.setup.steps.storage.hint'),
      to: '/storages',
      action: t('guide.setup.open'),
    },
    {
      key: 'user',
      required: true,
      done: s.user_ready,
      title: t('guide.setup.steps.user.title'),
      body: t('guide.setup.steps.user.body'),
      hint: t('guide.setup.steps.user.hint'),
      to: '/users',
      action: t('guide.setup.open'),
    },
    {
      key: 'running',
      required: false,
      done: s.bot_running,
      title: t('guide.setup.steps.running.title'),
      body: t('guide.setup.steps.running.body'),
      hint: t('guide.setup.steps.running.hint'),
      to: '/telegram',
      action: t('guide.setup.open'),
    },
    {
      key: 'task',
      required: false,
      done: s.first_task,
      title: t('guide.setup.steps.task.title'),
      body: t('guide.setup.steps.task.body'),
      hint: t('guide.setup.steps.task.hint'),
      to: '/tasks?create=1',
      action: t('common.actions.createTask'),
    },
  ].map((step) => ({ ...step, action: step.done ? t('guide.setup.review') : step.action }))
})

const legacyImported = computed(() => Boolean(status.value?.legacy.imported))
const legacyImportedAt = computed(() =>
  status.value?.legacy.imported_at ? formatDateTime(status.value.legacy.imported_at) : '',
)

async function load() {
  busy.value = true
  loadError.value = ''
  try {
    status.value = await onboardingApi.status()
  } catch (err) {
    loadError.value = errorMessage(err)
    toast.push(loadError.value, 'error', 6000)
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('guide.title')">
      <template #actions>
        <LoadingButton class="small" :loading="busy" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </LoadingButton>
      </template>
    </PageHeader>

    <template v-if="status">
      <section class="guide-hero">
        <div class="guide-hero-copy">
          <span class="badge status" :class="heroTone">{{ heroBadge }}</span>
          <h2>{{ heroTitle }}</h2>
          <p>{{ heroBody }}</p>
          <div class="actions guide-hero-actions">
            <RouterLink v-if="!ready" class="btn primary small" :to="nextPath">
              <AppIcon name="arrowRight" :size="15" />
              {{ t('guide.hero.continueSetup') }}
            </RouterLink>
            <RouterLink v-else-if="botReady" class="btn primary small" to="/tasks">
              <AppIcon name="tasks" :size="15" />
              {{ t('guide.hero.openTasks') }}
            </RouterLink>
            <RouterLink v-else class="btn primary small" to="/telegram">
              <AppIcon name="send" :size="15" />
              {{ t('guide.hero.openTelegram') }}
            </RouterLink>
            <RouterLink v-if="!ready" class="btn small" to="/tasks">
              {{ t('guide.hero.openTasks') }}
            </RouterLink>
            <RouterLink v-else class="btn small" :to="botReady ? '/telegram' : '/tasks'">
              {{ botReady ? t('guide.hero.review') : t('guide.hero.openTasks') }}
            </RouterLink>
          </div>
          <p class="guide-password-note">
            <AppIcon name="key" :size="14" />
            <span>{{ t('guide.hero.passwordNote') }}</span>
          </p>
        </div>

        <div class="guide-progress" :aria-label="t('guide.hero.progress', { done: requiredDone, total: requiredTotal })">
          <strong>{{ requiredDone }}<span>/{{ requiredTotal }}</span></strong>
          <div class="guide-meter" aria-hidden="true"><span :style="{ width: progressWidth }"></span></div>
          <small>{{ botReady ? t('guide.hero.readyBadge') : ready ? t('guide.hero.configuredBadge') : t('guide.hero.requiredSteps') }}</small>
        </div>
      </section>

      <section class="panel guide-section">
        <div class="panel-head">
          <h2 class="panel-title">{{ t('guide.setup.title') }}</h2>
          <span class="guide-count">{{ requiredDone }}/{{ requiredTotal }}</span>
        </div>
        <ol class="setup-steps">
          <li v-for="(step, index) in steps" :key="step.key" class="setup-step" :class="{ done: step.done }">
            <span class="setup-step-marker">
              <AppIcon v-if="step.done" name="check" :size="15" />
              <template v-else>{{ index + 1 }}</template>
            </span>
            <div class="setup-step-copy">
              <div class="setup-step-heading">
                <h3>{{ step.title }}</h3>
                <span class="guide-tag">{{ step.required ? t('guide.setup.required') : t('guide.setup.optional') }}</span>
                <span class="guide-state">{{ step.done ? t('guide.setup.done') : t('guide.setup.pending') }}</span>
              </div>
              <p>{{ step.body }}</p>
              <small>{{ step.hint }}</small>
            </div>
            <RouterLink class="btn small" :to="step.to">{{ step.action }}</RouterLink>
          </li>
        </ol>
      </section>

      <section class="panel guide-section">
        <div class="panel-head">
          <h2 class="panel-title">{{ t('guide.migration.title') }}</h2>
          <a class="link-quiet" :href="docsUrl" target="_blank" rel="noreferrer">{{ t('guide.migration.docs') }}</a>
        </div>
        <div v-if="legacyImported" class="guide-imported">
          <span class="guide-imported-icon"><AppIcon name="check" :size="16" /></span>
          <div>
            <strong>{{ t('guide.migration.importedTitle') }}</strong>
            <p>{{ t('guide.migration.importedBody') }}</p>
            <small v-if="legacyImportedAt">{{ t('guide.migration.importedAt', { time: legacyImportedAt }) }}</small>
          </div>
        </div>
        <template v-else>
          <p class="guide-copy">{{ t('guide.migration.intro') }}</p>
          <ol class="guide-numbered">
            <li>{{ t('guide.migration.step1') }}</li>
            <li>{{ t('guide.migration.step2') }}</li>
            <li>{{ t('guide.migration.step3') }}</li>
            <li>{{ t('guide.migration.step4') }}</li>
          </ol>
          <p class="guide-note">{{ t('guide.migration.note') }}</p>
        </template>
      </section>

      <section class="panel guide-section">
        <div class="panel-head">
          <h2 class="panel-title">{{ t('guide.usage.title') }}</h2>
        </div>
        <div class="guide-usage-grid">
          <article class="guide-usage-item">
            <span class="guide-usage-icon"><AppIcon name="send" :size="17" /></span>
            <div>
              <h3>{{ t('guide.usage.sendTitle') }}</h3>
              <p>{{ t('guide.usage.sendBody') }}</p>
            </div>
          </article>
          <article class="guide-usage-item">
            <span class="guide-usage-icon"><AppIcon name="file" :size="17" /></span>
            <div>
              <h3>{{ t('guide.usage.saveTitle') }}</h3>
              <p>{{ t('guide.usage.saveBody') }}</p>
            </div>
          </article>
          <article class="guide-usage-item">
            <span class="guide-usage-icon"><AppIcon name="tasks" :size="17" /></span>
            <div>
              <h3>{{ t('guide.usage.commandsTitle') }}</h3>
              <p>{{ t('guide.usage.commandsIntro') }}</p>
              <div class="guide-commands">
                <code>{{ t('guide.usage.commands.start') }}</code>
                <code>{{ t('guide.usage.commands.task') }}</code>
                <code>{{ t('guide.usage.commands.config') }}</code>
              </div>
              <p>{{ t('guide.usage.commandsBody') }}</p>
            </div>
          </article>
          <article class="guide-usage-item">
            <span class="guide-usage-icon"><AppIcon name="dashboard" :size="17" /></span>
            <div>
              <h3>{{ t('guide.usage.consoleTitle') }}</h3>
              <p>{{ t('guide.usage.consoleBody') }}</p>
            </div>
          </article>
          <article class="guide-usage-item">
            <span class="guide-usage-icon"><AppIcon name="users" :size="17" /></span>
            <div>
              <h3>{{ t('guide.usage.userbotTitle') }}</h3>
              <p>{{ t('guide.usage.userbotBody') }}</p>
            </div>
          </article>
          <article class="guide-usage-item">
            <span class="guide-usage-icon"><AppIcon name="hardDrive" :size="17" /></span>
            <div>
              <h3>{{ t('guide.data.title') }}</h3>
              <p>{{ t('guide.data.body') }}</p>
            </div>
          </article>
        </div>
      </section>
    </template>
    <div v-else-if="loadError" class="panel">
      <div class="empty">
        <span class="empty-icon"><AppIcon name="alert" :size="17" /></span>
        <strong>{{ loadError }}</strong>
        <LoadingButton class="small" :loading="busy" @click="load">
          <AppIcon name="refresh" :size="14" />
          {{ t('common.actions.refresh') }}
        </LoadingButton>
      </div>
    </div>
    <div v-else class="panel loading">{{ t('guide.title') }}</div>
  </div>
</template>
