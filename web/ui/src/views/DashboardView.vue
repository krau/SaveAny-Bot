<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { errorMessage, storagesApi, tasksApi, telegramApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
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

const status = ref<TelegramStatus | null>(null)
const tasks = ref<TaskItem[]>([])
const storages = ref<StorageRecord[]>([])
const error = ref('')
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
  if (!status.value) return '正在读取连接状态'
  return `Bot ${telegramStateLabel(status.value.bot.state)}，Userbot ${telegramStateLabel(status.value.userbot.state)}`
})
const statusErrors = computed(() =>
  [status.value?.bot.error, status.value?.userbot.error].filter((item): item is string => Boolean(item)),
)

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
    error.value = ''
  } catch (err) {
    error.value = errorMessage(err)
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
  <div class="stack fade-in">
    <PageHeader title="运行概览" description="机器人、任务队列与存储的当前状态。">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          刷新
        </button>
        <RouterLink class="btn primary small" to="/tasks">
          <AppIcon name="plus" :size="15" />
          新建任务
        </RouterLink>
      </template>
    </PageHeader>

    <div v-if="error" class="banner error">{{ error }}</div>
    <div v-for="message in statusErrors" :key="message" class="banner warning">{{ message }}</div>

    <section class="relay-board">
      <div class="relay-status">
        <span class="live-dot" :class="dotClass(status?.bot.state)"></span>
        <span>{{ statusLine }}</span>
      </div>

      <div class="relay-flow" aria-label="当前中继链路">
        <div class="relay-node">
          <div class="relay-node-label">
            <AppIcon name="send" :size="14" />
            入口
          </div>
          <strong>{{ telegramStateLabel(status?.bot.state) }}</strong>
          <small>{{ status?.bot.username ? `@${status.bot.username}` : 'Bot 未连接' }}</small>
        </div>
        <div class="relay-link" :class="{ active: activeTasks > 0 }" aria-hidden="true">
          <AppIcon name="arrowRight" :size="17" />
        </div>
        <div class="relay-node">
          <div class="relay-node-label">
            <AppIcon name="tasks" :size="14" />
            调度队列
          </div>
          <strong>{{ activeTasks }} 个任务</strong>
          <small>{{ activeTasks > 0 ? '正在处理或等待 Worker' : '队列空闲' }}</small>
        </div>
        <div class="relay-link" :class="{ active: activeTasks > 0 }" aria-hidden="true">
          <AppIcon name="arrowRight" :size="17" />
        </div>
        <div class="relay-node">
          <div class="relay-node-label">
            <AppIcon name="database" :size="14" />
            归档存储
          </div>
          <strong>{{ enabledStorages }} 个可用</strong>
          <small>共 {{ storages.length }} 个已配置存储</small>
        </div>
      </div>

      <div class="relay-metrics">
        <div class="relay-metric">
          <span>活动任务</span>
          <strong>{{ activeTasks }}</strong>
          <small>排队中或正在运行</small>
        </div>
        <div class="relay-metric">
          <span>近 7 天任务</span>
          <strong>{{ weekTotal }}</strong>
          <small>按创建时间统计</small>
        </div>
        <div class="relay-metric">
          <span>已完成任务</span>
          <strong>{{ completedTasks }}</strong>
          <small>最近加载的 {{ tasks.length }} 条记录</small>
        </div>
      </div>
    </section>

    <div class="dashboard-split">
      <section class="panel">
        <div class="panel-head">
          <div>
            <h2 class="panel-title">近 7 天任务</h2>
            <p class="panel-sub">每天创建的任务数，今天的柱形用信号色标记</p>
          </div>
          <RouterLink class="link-quiet" to="/tasks">查看全部</RouterLink>
        </div>
        <div class="activity-figure">
          <strong>{{ weekTotal }}</strong>
          <span>个任务创建于最近 7 天</span>
        </div>
        <div
          v-if="tasks.length > 0"
          class="activity-chart"
          role="img"
          :aria-label="`最近 7 天共创建 ${weekTotal} 个任务`"
        >
          <div
            v-for="day in activity"
            :key="day.key"
            class="chart-col"
            :class="{ current: day.current }"
          >
            <span class="chart-count">{{ day.count }}</span>
            <div
              class="chart-bar"
              :style="{ height: `${day.height}px` }"
              :title="`${day.label} 创建 ${day.count} 个任务`"
            ></div>
            <span class="chart-label">{{ day.label }}</span>
          </div>
        </div>
        <div v-else class="chart-empty">还没有任务记录<br />创建任务后，这里会显示最近 7 天的分布</div>
      </section>

      <section class="panel">
        <div class="panel-head">
          <div>
            <h2 class="panel-title">存储</h2>
            <p class="panel-sub">可用目标与位置</p>
          </div>
          <RouterLink class="link-quiet" to="/storages">管理</RouterLink>
        </div>
        <div v-if="recentStorages.length > 0" class="storage-mini-list">
          <div v-for="storage in recentStorages" :key="storage.id" class="storage-mini">
            <span class="storage-mini-icon"><AppIcon :name="storageTypeIcon(storage.type)" :size="15" /></span>
            <div class="storage-mini-main">
              <strong>{{ storage.name }}</strong>
              <span>{{ storageTypeLabel(storage.type) }}，{{ storageLocation(storage.config) }}</span>
            </div>
            <span class="badge status" :class="storage.enable ? 'success' : 'muted'">
              {{ storage.enable ? '启用' : '禁用' }}
            </span>
          </div>
        </div>
        <div v-else class="empty">
          <span class="empty-icon"><AppIcon name="database" :size="17" /></span>
          <strong>还没有存储</strong>
          添加一个目标后，任务才能写入文件
          <div class="actions" style="justify-content: center; margin-top: 12px">
            <RouterLink class="btn small" to="/storages">
              <AppIcon name="plus" :size="14" />
              新建存储
            </RouterLink>
          </div>
        </div>
      </section>
    </div>

    <section class="panel">
      <div class="panel-head">
        <div>
          <h2 class="panel-title">最近任务</h2>
          <p class="panel-sub">最新创建的任务及其归档目标</p>
        </div>
        <RouterLink class="link-quiet" to="/tasks">全部任务</RouterLink>
      </div>
      <div v-if="recentTasks.length > 0" class="recent-list">
        <div v-for="task in recentTasks" :key="task.task_id" class="recent-task">
          <div class="recent-task-main">
            <div class="recent-task-title">{{ taskDisplayTitle(task) }}</div>
            <div class="recent-task-meta">
              <span v-if="taskDisplayTitle(task) !== taskTypeLabel(task.type)">{{ taskTypeLabel(task.type) }}，</span>
              <span>{{ task.storage || '未指定存储' }}，</span>
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
        <strong>还没有任务</strong>
        从任务页创建直链下载、存储迁移或 Telegram 文件任务
      </div>
    </section>
  </div>
</template>
