<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { errorMessage, storagesApi, tasksApi, telegramApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import { useToastStore } from '../stores/toasts'
import { storageTypeIcon, storageTypeLabel } from '../storageFields'
import type { StorageRecord, TaskItem, TelegramStatus } from '../types'
import {
  formatRelativeTime,
  storageLocation,
  taskDisplayTitle,
  taskStatusLabel,
  taskStatusTone,
  taskTypeLabel,
  telegramStateLabel,
} from '../utils'

const toast = useToastStore()
const { t } = useI18n()
const status = ref<TelegramStatus | null>(null)
const tasks = ref<TaskItem[]>([])
const storages = ref<StorageRecord[]>([])
let timer: number | undefined

const activeTasks = computed(
  () => tasks.value.filter((task) => task.status === 'queued' || task.status === 'running').length,
)
const enabledStorages = computed(() => storages.value.filter((storage) => storage.enable).length)
const recentTasks = computed(() => tasks.value.slice(0, 6))
const recentStorages = computed(() => storages.value.slice(0, 4))

function localDayKey(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

const activity = computed(() => {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const days = Array.from({ length: 7 }, (_, index) => {
    const date = new Date(today)
    date.setDate(today.getDate() - (6 - index))
    return {
      key: localDayKey(date),
      label: `${date.getMonth() + 1}/${date.getDate()}`,
      count: 0,
      current: index === 6,
      height: 0,
    }
  })
  const byDay = new Map(days.map((day) => [day.key, day]))
  for (const task of tasks.value) {
    const created = new Date(task.created_at)
    if (Number.isNaN(created.getTime())) continue
    const day = byDay.get(localDayKey(created))
    if (day) day.count += 1
  }
  const max = Math.max(1, ...days.map((day) => day.count))
  for (const day of days) {
    day.height = day.count === 0 ? 2 : Math.max(8, Math.round((day.count / max) * 128))
  }
  return days
})
const weekTotal = computed(() => activity.value.reduce((sum, day) => sum + day.count, 0))
const completedTasks = computed(() => tasks.value.filter((task) => task.status === 'completed').length)
const dotClass = (state?: string): string => {
  if (state === 'running') return 'running'
  if (state === 'starting') return 'accent'
  if (state === 'error' || state === 'login_required') return 'error'
  return 'idle'
}
const statusLine = computed(() => {
  if (!status.value) return t('dashboard.status.reading')
  return t('dashboard.status.line', {
    bot: telegramStateLabel(status.value.bot.state),
    userbot: telegramStateLabel(status.value.userbot.state),
  })
})
async function load() {
  try {
    const [statusData, taskData, storageData] = await Promise.all([
      telegramApi.status(),
      tasksApi.list({ limit: 200 }),
      storagesApi.list(),
    ])
    status.value = statusData
    tasks.value = taskData.tasks ?? []
    storages.value = storageData
    for (const message of [statusData.bot.error, statusData.userbot.error]) {
      if (message) toast.push(message, 'error', 6000)
    }
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

onMounted(() => {
  void load()
  timer = window.setInterval(() => void load(), 10000)
})
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('nav.overview')">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </button>
        <RouterLink class="btn primary small" :to="{ path: '/tasks', query: { create: '1' } }">
          <AppIcon name="plus" :size="15" />
          {{ t('common.actions.createTask') }}
        </RouterLink>
      </template>
    </PageHeader>

    <section class="relay-board">
      <div class="relay-status">
        <span class="live-dot" :class="dotClass(status?.bot.state)"></span>
        <span>{{ statusLine }}</span>
      </div>

      <div class="relay-flow" :aria-label="t('dashboard.relay.ariaLabel')">
        <div class="relay-node">
          <div class="relay-node-label">
            <AppIcon name="send" :size="14" />
            {{ t('dashboard.relay.entry') }}
          </div>
          <strong>{{ telegramStateLabel(status?.bot.state) }}</strong>
          <small>{{ status?.bot.username ? `@${status.bot.username}` : t('dashboard.status.botDisconnected') }}</small>
        </div>
        <div class="relay-link" :class="{ active: activeTasks > 0 }" aria-hidden="true">
          <AppIcon name="arrowRight" :size="17" />
        </div>
        <div class="relay-node">
          <div class="relay-node-label">
            <AppIcon name="tasks" :size="14" />
            {{ t('dashboard.relay.queue') }}
          </div>
          <strong>{{ t('dashboard.relay.queueCount', { n: activeTasks }) }}</strong>
          <small>{{ activeTasks > 0 ? t('dashboard.relay.queueBusy') : t('dashboard.relay.queueIdle') }}</small>
        </div>
        <div class="relay-link" :class="{ active: activeTasks > 0 }" aria-hidden="true">
          <AppIcon name="arrowRight" :size="17" />
        </div>
        <div class="relay-node">
          <div class="relay-node-label">
            <AppIcon name="database" :size="14" />
            {{ t('dashboard.relay.storage') }}
          </div>
          <strong>{{ t('dashboard.relay.storageAvailable', { n: enabledStorages }) }}</strong>
          <small>{{ t('dashboard.relay.storageTotal', { n: storages.length }) }}</small>
        </div>
      </div>

      <div class="relay-metrics">
        <div class="relay-metric">
          <span>{{ t('dashboard.metrics.activeTasks') }}</span>
          <strong>{{ activeTasks }}</strong>
        </div>
        <div class="relay-metric">
          <span>{{ t('dashboard.metrics.weekTasks') }}</span>
          <strong>{{ weekTotal }}</strong>
        </div>
        <div class="relay-metric">
          <span>{{ t('dashboard.metrics.completedTasks') }}</span>
          <strong>{{ completedTasks }}</strong>
        </div>
      </div>
    </section>

    <div class="dashboard-split">
      <section class="panel">
        <div class="panel-head">
          <h2 class="panel-title">{{ t('dashboard.metrics.weekTasks') }}</h2>
          <RouterLink class="link-quiet" to="/tasks">{{ t('dashboard.activity.viewAll') }}</RouterLink>
        </div>
        <div class="activity-figure">
          <strong>{{ weekTotal }}</strong>
        </div>
        <div
          v-if="tasks.length > 0"
          class="activity-chart"
          role="img"
          :aria-label="t('dashboard.activity.ariaLabel', { n: weekTotal })"
        >
          <div
            v-for="(day, index) in activity"
            :key="day.key"
            class="chart-col"
            :class="{ current: day.current }"
            :style="{ '--bar-index': String(index) }"
          >
            <span class="chart-count">{{ day.count }}</span>
            <div
              class="chart-bar"
              :style="{ height: `${day.height}px` }"
              :title="t('dashboard.activity.barTitle', { label: day.label, n: day.count })"
            ></div>
            <span class="chart-label">{{ day.label }}</span>
          </div>
        </div>
        <div v-else class="chart-empty">{{ t('dashboard.activity.empty') }}</div>
      </section>

      <section class="panel">
        <div class="panel-head">
          <h2 class="panel-title">{{ t('dashboard.storage.title') }}</h2>
          <RouterLink class="link-quiet" to="/storages">{{ t('dashboard.storage.manage') }}</RouterLink>
        </div>
        <div v-if="recentStorages.length > 0" class="storage-mini-list">
          <div v-for="storage in recentStorages" :key="storage.id" class="storage-mini">
            <span class="storage-mini-icon"><AppIcon :name="storageTypeIcon(storage.type)" :size="15" /></span>
            <div class="storage-mini-main">
              <strong>{{ storage.name }}</strong>
              <span>{{ t('dashboard.storage.meta', { type: storageTypeLabel(storage.type), location: storageLocation(storage.config) }) }}</span>
            </div>
            <span class="badge status" :class="storage.enable ? 'success' : 'muted'">
              {{ storage.enable ? t('states.enabled') : t('states.disabled') }}
            </span>
          </div>
        </div>
        <div v-else class="empty">
          <span class="empty-icon"><AppIcon name="database" :size="17" /></span>
          <strong>{{ t('dashboard.storage.empty') }}</strong>
          <div class="actions" style="justify-content: center; margin-top: 12px">
            <RouterLink class="btn small" to="/storages">
              <AppIcon name="plus" :size="14" />
              {{ t('common.actions.createStorage') }}
            </RouterLink>
          </div>
        </div>
      </section>
    </div>

    <section class="panel">
      <div class="panel-head">
        <h2 class="panel-title">{{ t('dashboard.tasks.recentTitle') }}</h2>
        <RouterLink class="link-quiet" to="/tasks">{{ t('dashboard.tasks.all') }}</RouterLink>
      </div>
      <div v-if="recentTasks.length > 0" class="recent-list">
        <div v-for="task in recentTasks" :key="task.task_id" class="recent-task">
          <div class="recent-task-main">
            <div class="recent-task-title">{{ taskDisplayTitle(task) }}</div>
            <div class="recent-task-meta">
              <span v-if="taskDisplayTitle(task) !== taskTypeLabel(task.type)">{{ taskTypeLabel(task.type) }}{{ t('dashboard.tasks.separator') }}</span>
              <span>{{ task.storage || t('dashboard.tasks.noStorage') }}{{ t('dashboard.tasks.separator') }}</span>
              <span>{{ formatRelativeTime(task.created_at) }}</span>
            </div>
          </div>
          <div class="recent-task-side">
            <div v-if="task.status === 'running'" class="recent-task-progress">
              <div class="progress">
                <div
                  class="progress-bar"
                  :style="{ width: `${Math.max(0, Math.min(100, task.progress?.percent ?? 0))}%` }"
                ></div>
              </div>
            </div>
            <span class="badge status" :class="taskStatusTone(task.status)">
              {{ taskStatusLabel(task.status) }}
            </span>
          </div>
        </div>
      </div>
      <div v-else class="empty">
        <span class="empty-icon"><AppIcon name="inbox" :size="17" /></span>
        <strong>{{ t('dashboard.tasks.empty') }}</strong>
      </div>
    </section>
  </div>
</template>
