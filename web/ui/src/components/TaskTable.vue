<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatBytes, formatDateTime, taskDisplayTitle, taskStatusLabel, taskStatusTone, taskTypeLabel } from '../utils'
import type { TaskItem } from '../types'
import AppIcon from './AppIcon.vue'

const { t } = useI18n()

withDefaults(
  defineProps<{
    tasks: TaskItem[]
    busyId?: string
    emptyText?: string
  }>(),
  {
    busyId: '',
    emptyText: '',
  },
)

const emit = defineEmits<{
  cancel: [taskId: string]
  remove: [taskId: string]
  retry: [taskId: string]
}>()

function canRetry(task: TaskItem): boolean {
  return task.type === 'tgfiles' && (task.status === 'failed' || task.status === 'partial')
}

function isActive(status: string): boolean {
  return status === 'queued' || status === 'running'
}

function progressPercent(task: TaskItem): number {
  const percent = task.progress?.percent
  if (typeof percent === 'number' && Number.isFinite(percent)) {
    return Math.max(0, Math.min(100, percent))
  }
  return 0
}

function progressText(task: TaskItem): string {
  const progress = task.progress
  if (!progress) {
    return '—'
  }
  const parts: string[] = []
  if (typeof progress.percent === 'number') {
    parts.push(`${progress.percent.toFixed(1)}%`)
  }
  if (progress.total_bytes) {
    parts.push(`${formatBytes(progress.downloaded_bytes)} / ${formatBytes(progress.total_bytes)}`)
  } else if (progress.total_files) {
    parts.push(t('tasks.progress.files', { done: progress.downloaded_files ?? 0, total: progress.total_files }))
  }
  return parts.join(t('tasks.progress.separator')) || '—'
}
</script>

<template>
  <div class="task-table">
  <div class="table-wrap task-table-wrap">
    <table>
      <thead>
        <tr>
          <th>{{ t('tasks.table.task') }}</th>
          <th>{{ t('tasks.table.status') }}</th>
          <th>{{ t('tasks.table.storagePath') }}</th>
          <th>{{ t('tasks.table.progress') }}</th>
          <th>{{ t('tasks.table.createdAt') }}</th>
          <th>{{ t('tasks.table.actions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="task in tasks" :key="task.task_id">
          <td>
            <div class="task-cell-title">{{ taskDisplayTitle(task) }}</div>
            <div class="task-cell-meta">
              <span v-if="taskDisplayTitle(task) !== taskTypeLabel(task.type)">{{ taskTypeLabel(task.type) }}</span>
              <span class="mono">{{ task.task_id }}</span>
            </div>
            <div v-if="task.error" class="field-error">{{ task.error }}</div>
            <div v-if="task.failed_items?.length" class="task-failed">
              <div class="task-failed-title">{{ t('tasks.failedFiles') }} ({{ task.failed_count ?? task.failed_items.length }})</div>
              <ul class="task-failed-items">
                <li v-for="item in task.failed_items" :key="item.index">
                  <span class="task-failed-name">{{ item.name }}</span>
                  <span v-if="item.error" class="task-failed-reason">{{ item.error }}</span>
                </li>
              </ul>
            </div>
          </td>
          <td>
            <span class="badge status" :class="taskStatusTone(task.status)">
              {{ taskStatusLabel(task.status) }}
            </span>
          </td>
          <td>
            <div>{{ task.storage_name || '—' }}</div>
            <div class="task-path">{{ task.path || t('tasks.table.rootPath') }}</div>
          </td>
          <td style="min-width: 170px">
            <div v-if="task.status === 'running'" class="progress">
              <div class="progress-bar" :style="{ width: `${progressPercent(task)}%` }"></div>
            </div>
            <div class="task-cell-meta">{{ progressText(task) }}</div>
          </td>
          <td class="muted" style="white-space: nowrap">{{ formatDateTime(task.created_at) }}</td>
          <td>
            <div class="actions">
              <button
                v-if="isActive(task.status)"
                class="btn small"
                type="button"
                :disabled="busyId === task.task_id"
                @click="emit('cancel', task.task_id)"
              >
                <AppIcon name="close" :size="14" />
                {{ t('common.actions.cancel') }}
              </button>
              <button
                v-if="canRetry(task)"
                class="btn small"
                type="button"
                :disabled="busyId === task.task_id"
                @click="emit('retry', task.task_id)"
              >
                <AppIcon name="refresh" :size="14" />
                {{ t('common.actions.retry') }}
              </button>
              <button
                v-if="!isActive(task.status)"
                class="btn small danger"
                type="button"
                :disabled="busyId === task.task_id"
                @click="emit('remove', task.task_id)"
              >
                <AppIcon name="trash" :size="14" />
                {{ t('common.actions.delete') }}
              </button>
            </div>
          </td>
        </tr>
        <tr v-if="tasks.length === 0">
          <td colspan="6" class="empty">
            <span class="empty-icon"><AppIcon name="inbox" :size="17" /></span>
            <strong>{{ emptyText || t('tasks.empty') }}</strong>
          </td>
        </tr>
      </tbody>
    </table>
  </div>

  <div class="task-mobile-list">
    <article v-for="task in tasks" :key="task.task_id" class="task-mobile-card">
      <div class="task-mobile-head">
        <div class="task-mobile-title">{{ taskDisplayTitle(task) }}</div>
        <span class="badge status" :class="taskStatusTone(task.status)">
          {{ taskStatusLabel(task.status) }}
        </span>
      </div>
      <div class="task-mobile-meta">
        <span v-if="taskDisplayTitle(task) !== taskTypeLabel(task.type)">{{ taskTypeLabel(task.type) }}</span>
        <span class="mono">{{ task.task_id }}</span>
      </div>
      <div class="task-mobile-target">
        {{ task.storage_name || t('tasks.table.noStorage') }}
        <span class="task-path">{{ task.path || t('tasks.table.rootPath') }}</span>
      </div>
      <div v-if="task.error" class="field-error">{{ task.error }}</div>
      <div v-if="task.failed_items?.length" class="task-failed">
        <div class="task-failed-title">{{ t('tasks.failedFiles') }} ({{ task.failed_count ?? task.failed_items.length }})</div>
        <ul class="task-failed-items">
          <li v-for="item in task.failed_items" :key="item.index">
            <span class="task-failed-name">{{ item.name }}</span>
            <span v-if="item.error" class="task-failed-reason">{{ item.error }}</span>
          </li>
        </ul>
      </div>
      <div v-if="task.status === 'running'" class="task-mobile-progress">
        <div class="progress">
          <div class="progress-bar" :style="{ width: `${progressPercent(task)}%` }"></div>
        </div>
        <span>{{ progressText(task) }}</span>
      </div>
      <div class="task-mobile-foot">
        <span class="task-mobile-time">{{ formatDateTime(task.created_at) }}</span>
        <div class="actions">
          <button
            v-if="isActive(task.status)"
            class="btn small"
            type="button"
            :disabled="busyId === task.task_id"
            @click="emit('cancel', task.task_id)"
          >
            <AppIcon name="close" :size="14" />
            {{ t('common.actions.cancel') }}
          </button>
          <button
            v-if="canRetry(task)"
            class="btn small"
            type="button"
            :disabled="busyId === task.task_id"
            @click="emit('retry', task.task_id)"
          >
            <AppIcon name="refresh" :size="14" />
            {{ t('common.actions.retry') }}
          </button>
          <button
            v-if="!isActive(task.status)"
            class="btn small danger"
            type="button"
            :disabled="busyId === task.task_id"
            @click="emit('remove', task.task_id)"
          >
            <AppIcon name="trash" :size="14" />
            {{ t('common.actions.delete') }}
          </button>
        </div>
      </div>
    </article>
    <div v-if="tasks.length === 0" class="empty">
      <span class="empty-icon"><AppIcon name="inbox" :size="17" /></span>
      <strong>{{ emptyText || t('tasks.empty') }}</strong>
    </div>
  </div>
</div>
</template>
