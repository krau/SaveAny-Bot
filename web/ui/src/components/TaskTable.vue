<script setup lang="ts">
import { formatBytes, formatDateTime, taskDisplayTitle, taskStatusLabel, taskStatusTone, taskTypeLabel } from '../utils'
import type { TaskItem } from '../types'
import AppIcon from './AppIcon.vue'

withDefaults(
  defineProps<{
    tasks: TaskItem[]
    busyId?: string
    emptyText?: string
  }>(),
  {
    busyId: '',
    emptyText: '暂无任务',
  },
)

const emit = defineEmits<{
  cancel: [taskId: string]
  remove: [taskId: string]
}>()

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
    parts.push(`${progress.downloaded_files ?? 0} / ${progress.total_files} 个文件`)
  }
  return parts.join('，') || '—'
}
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>任务</th>
          <th>状态</th>
          <th>存储 / 路径</th>
          <th>进度</th>
          <th>创建时间</th>
          <th>操作</th>
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
          </td>
          <td>
            <span class="badge status" :class="taskStatusTone(task.status)">
              {{ taskStatusLabel(task.status) }}
            </span>
          </td>
          <td>
            <div>{{ task.storage || '—' }}</div>
            <div class="task-path">{{ task.path || '—' }}</div>
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
                取消
              </button>
              <button
                v-else
                class="btn small danger"
                type="button"
                :disabled="busyId === task.task_id"
                @click="emit('remove', task.task_id)"
              >
                <AppIcon name="trash" :size="14" />
                删除
              </button>
            </div>
          </td>
        </tr>
        <tr v-if="tasks.length === 0">
          <td colspan="6" class="empty">
            <span class="empty-icon"><AppIcon name="inbox" :size="17" /></span>
            <strong>{{ emptyText }}</strong>
            创建任务后，下载状态会显示在这里
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
