<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError, errorMessage, systemApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageHeader from '../components/PageHeader.vue'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import { formatUptime } from '../utils'
import type { SystemInfo, UpdateInfo } from '../types'

const toast = useToastStore()
const { t } = useI18n()
const confirm = useConfirmStore()
const info = ref<SystemInfo | null>(null)
const error = ref('')
const busy = ref(false)
const updateInfo = ref<UpdateInfo | null>(null)
const checking = ref(false)
const updating = ref(false)
const host = window.location.host

async function load() {
  try {
    info.value = await systemApi.info()
    error.value = ''
  } catch (err) {
    error.value = 'load_failed'
    toast.push(errorMessage(err), 'error', 6000)
  }
}

onMounted(load)

async function checkUpdate() {
  checking.value = true
  try {
    updateInfo.value = await systemApi.checkUpdate()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    checking.value = false
  }
}

async function installUpdate() {
  if (!updateInfo.value?.latest) return
  const confirmed = await confirm.ask({
    title: t('system.update.installTitle'),
    message: t('system.update.installMessage', { version: updateInfo.value.latest }),
    confirmText: t('system.update.install'),
    danger: true,
  })
  if (!confirmed) return
  updating.value = true
  try {
    await systemApi.applyUpdate()
    toast.push(t('system.update.updated'), 'info')
  } catch (err) {
    toast.push(t('system.update.failed') + ': ' + errorMessage(err), 'error', 6000)
  } finally {
    updating.value = false
  }
}

async function restart(force = false) {
  if (
    !force &&
    !(await confirm.ask({
      title: t('system.confirm.restartTitle'),
      message: t('system.confirm.restartMessage'),
      confirmText: t('system.restart'),
      danger: true,
    }))
  ) {
    return
  }
  busy.value = true
  try {
    await systemApi.restart(force)
    toast.push(t('system.toast.restarting'), 'info')
    await new Promise((resolve) => setTimeout(resolve, 1500))
    for (let i = 0; i < 20; i += 1) {
      try {
        info.value = await systemApi.info()
        toast.push(t('system.toast.restarted'), 'success')
        break
      } catch {
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }
    }
  } catch (err) {
    if (
      !force &&
      err instanceof ApiError &&
      err.status === 409 &&
      (await confirm.ask({
        title: t('system.confirm.forceTitle'),
        message: t('system.confirm.forceMessage'),
        confirmText: t('common.actions.forceRestart'),
        danger: true,
      }))
    ) {
      await restart(true)
      return
    }
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('nav.system')">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </button>
        <LoadingButton class="small danger" type="button" :loading="busy" @click="restart()">
          <AppIcon name="refresh" :size="15" />
          {{ t('system.restart') }}
        </LoadingButton>
      </template>
    </PageHeader>

    <div v-if="info" class="process-panel">
      <div>
        <span class="badge status success">{{ t('system.process.running') }}</span>
        <div class="process-version">{{ info.version }}</div>
        <div class="process-meta">{{ info.commit || t('system.info.commitUnknown') }}</div>
        <div class="process-meta">{{ info.build_time }}</div>
      </div>
      <div class="process-metrics">
        <div>
          <div class="stat-label">{{ t('system.process.uptime') }}</div>
          <div class="stat-value">{{ formatUptime(info.uptime_seconds) }}</div>
        </div>
        <div>
          <div class="stat-label">{{ t('system.process.tasks') }}</div>
          <div class="stat-value">{{ info.running_tasks }}</div>
          <div class="stat-sub">{{ t('system.process.queued', { n: info.queued_tasks }) }}</div>
        </div>
        <div>
          <div class="stat-label">{{ t('system.process.concurrency') }}</div>
          <div class="stat-value">{{ info.workers }}</div>
        </div>
      </div>
    </div>

    <div v-else-if="!error" class="panel loading">{{ t('system.process.loading') }}</div>

    <div v-if="info" class="panel">
      <div class="panel-head">
        <h2 class="panel-title">{{ t('system.info.title') }}</h2>
        <span class="badge" :class="info.docker ? 'info' : 'muted'">
          {{ info.docker ? t('system.info.docker') : t('system.info.local') }}
        </span>
      </div>
      <dl class="kv">
        <dt>{{ t('system.info.version') }}</dt><dd class="mono">{{ info.version }}</dd>
        <dt>{{ t('system.info.commit') }}</dt><dd class="mono">{{ info.commit || '—' }}</dd>
        <dt>{{ t('system.info.buildTime') }}</dt><dd class="mono">{{ info.build_time }}</dd>
        <dt>{{ t('system.info.dataDir') }}</dt><dd class="mono">{{ info.data_dir }}</dd>
        <dt>{{ t('system.info.database') }}</dt><dd class="mono">{{ info.db_path }}</dd>
        <dt>{{ t('system.info.listenAddress') }}</dt><dd class="mono">{{ host }}</dd>
      </dl>
    </div>
    <div class="panel">
      <div class="panel-head">
        <h2 class="panel-title">{{ t('system.update.title') }}</h2>
        <button class="btn small" type="button" :disabled="checking" @click="checkUpdate">
          <AppIcon name="refresh" :size="15" />
          {{ checking ? t('system.update.checking') : t('system.update.check') }}
        </button>
      </div>
      <div v-if="!updateInfo" class="muted">{{ t('system.update.notChecked') }}</div>
      <template v-else>
        <dl class="kv">
          <dt>{{ t('system.update.current') }}</dt><dd class="mono">{{ updateInfo.current }}</dd>
          <dt v-if="updateInfo.latest">{{ t('system.update.latest') }}</dt><dd v-if="updateInfo.latest" class="mono">{{ updateInfo.latest }}</dd>
          <dt v-if="updateInfo.published_at">{{ t('system.update.published') }}</dt><dd v-if="updateInfo.published_at">{{ new Date(updateInfo.published_at).toLocaleString() }}</dd>
          <dt v-if="updateInfo.size">{{ t('system.update.size') }}</dt><dd v-if="updateInfo.size">{{ updateInfo.size }} B</dd>
        </dl>
        <p v-if="!updateInfo.updatable" class="muted">{{ t('system.update.notUpdatable') }}<span v-if="updateInfo.reason">: {{ updateInfo.reason }}</span></p>
        <p v-else-if="updateInfo.reason_code === 'major_upgrade_required'" class="muted">{{ t('system.update.major') }}</p>
        <p v-else-if="updateInfo.reason_code === 'no_release'" class="muted">{{ t('system.update.noRelease') }}</p>
        <p v-else-if="updateInfo.available" class="badge info">{{ t('system.update.available') }}</p>
        <p v-else class="badge success">{{ t('system.update.upToDate') }}</p>
        <button v-if="updateInfo.available" class="btn primary small" type="button" :disabled="updating" @click="installUpdate">
          {{ updating ? t('system.update.updating') : t('system.update.install') }}
        </button>
        <details v-if="updateInfo.release_notes" style="margin-top: 16px">
          <summary>{{ t('system.update.releaseNotes') }}</summary>
          <pre class="mono" style="white-space: pre-wrap">{{ updateInfo.release_notes }}</pre>
        </details>
      </template>
    </div>
  </div>
</template>
