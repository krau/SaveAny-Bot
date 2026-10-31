<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { errorMessage, storagesApi, tasksApi } from '../api'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import AppSelect from '../components/AppSelect.vue'
import PageHeader from '../components/PageHeader.vue'
import TaskForm from '../components/TaskForm.vue'
import TaskTable from '../components/TaskTable.vue'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import type { CreateTaskRequest, StorageRecord, TaskItem } from '../types'

const toast = useToastStore()
const confirm = useConfirmStore()
const route = useRoute()
const router = useRouter()
const tasks = ref<TaskItem[]>([])
const storages = ref<StorageRecord[]>([])
const total = ref(0)
const statusFilter = ref('')
const typeFilter = ref('')
const busyId = ref('')
const createOpen = ref(false)
const creating = ref(false)
const taskModal = ref<HTMLElement | null>(null)
let timer: number | undefined
let taskModalResizeTimer: number | undefined
let taskModalResizeFrom = 0
let taskModalResizeVersion = 0

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'queued', label: '排队中' },
  { value: 'running', label: '进行中' },
  { value: 'completed', label: '已完成' },
  { value: 'failed', label: '失败' },
  { value: 'cancelled', label: '已取消' },
]
const typeOptions = [
  { value: '', label: '全部类型' },
  { value: 'directlinks', label: '直链下载' },
  { value: 'ytdlp', label: 'yt-dlp' },
  { value: 'aria2', label: 'Aria2' },
  { value: 'parseditem', label: '解析链接' },
  { value: 'tgfiles', label: 'Telegram 文件' },
  { value: 'tphpics', label: 'Telegraph 图集' },
  { value: 'transfer', label: '存储迁移' },
]

async function load() {
  try {
    const data = await tasksApi.list({ status: statusFilter.value, type: typeFilter.value, limit: 100 })
    tasks.value = data.tasks ?? []
    total.value = data.total ?? 0
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function loadStorages() {
  try {
    storages.value = await storagesApi.list()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

function schedule() {
  if (timer) window.clearInterval(timer)
  timer = window.setInterval(() => void load(), 2000)
}

async function consumeCreateIntent() {
  if (route.query.create !== '1') return
  createOpen.value = true
  await router.replace({ path: '/tasks' })
}

watch(
  () => route.query.create,
  () => {
    void consumeCreateIntent()
  },
)

onMounted(async () => {
  schedule()
  await Promise.all([load(), loadStorages()])
  await consumeCreateIntent()
})
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
  if (taskModalResizeTimer) window.clearTimeout(taskModalResizeTimer)
})

function startTaskModalResize() {
  const el = taskModal.value
  if (!el) return
  if (taskModalResizeTimer) {
    window.clearTimeout(taskModalResizeTimer)
    taskModalResizeTimer = undefined
  }
  // Freeze the current height before Vue swaps the field set.
  taskModalResizeVersion += 1
  taskModalResizeFrom = el.getBoundingClientRect().height
  el.style.transition = 'none'
  el.style.height = `${taskModalResizeFrom}px`
}

function endTaskModalResize() {
  const el = taskModal.value
  if (!el || !taskModalResizeFrom) return

  const from = taskModalResizeFrom
  const version = taskModalResizeVersion
  el.style.height = 'auto'
  const to = el.getBoundingClientRect().height
  el.style.height = `${from}px`
  void el.offsetHeight

  const finish = () => {
    taskModalResizeFrom = 0
    taskModalResizeTimer = undefined
    if (taskModal.value === el) {
      el.style.transition = ''
      el.style.height = ''
    }
  }

  if (
    Math.abs(to - from) < 1 ||
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  ) {
    finish()
    return
  }

  el.style.transition = 'height var(--duration-fast) var(--ease-smooth-out)'
  requestAnimationFrame(() => {
    if (version === taskModalResizeVersion && taskModal.value === el) {
      el.style.height = `${to}px`
    }
  })
  taskModalResizeTimer = window.setTimeout(finish, 280)
}

async function create(payload: CreateTaskRequest) {
  creating.value = true
  try {
    await tasksApi.create(payload)
    toast.push('任务已创建', 'success')
    createOpen.value = false
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    creating.value = false
  }
}

async function cancel(taskId: string) {
  const confirmed = await confirm.ask({
    title: '取消任务',
    message: '取消该任务？正在下载的内容会被中断。',
    confirmText: '取消任务',
  })
  if (!confirmed) return
  busyId.value = taskId
  try {
    await tasksApi.cancel(taskId)
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busyId.value = ''
  }
}

async function remove(taskId: string) {
  const confirmed = await confirm.ask({
    title: '删除任务记录',
    message: '删除该任务记录？此操作不可撤销。',
    confirmText: '删除记录',
    danger: true,
  })
  if (!confirmed) return
  busyId.value = taskId
  try {
    await tasksApi.remove(taskId)
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busyId.value = ''
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader title="任务" description="创建下载与转存任务，查看进度或清理记录。">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          刷新
        </button>
        <button class="btn primary small" type="button" @click="createOpen = true">
          <AppIcon name="plus" :size="15" />
          新建任务
        </button>
      </template>
    </PageHeader>

    <div class="filter-bar">
      <div class="filter-field">
        <label for="task-status-filter">状态</label>
        <AppSelect id="task-status-filter" v-model="statusFilter" :options="statusOptions" @change="load" />
      </div>
      <div class="filter-field">
        <label for="task-type-filter">类型</label>
        <AppSelect id="task-type-filter" v-model="typeFilter" :options="typeOptions" @change="load" />
      </div>
      <div class="filter-spacer"></div>
      <span class="count">共 {{ total }} 条记录</span>
    </div>

    <TaskTable :tasks="tasks" :busy-id="busyId" @cancel="cancel" @remove="remove" />

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="createOpen" class="modal-overlay" @click.self="createOpen = false">
          <div
            ref="taskModal"
            class="modal wide task-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="create-task-title"
          >
            <div class="modal-head">
              <h3 id="create-task-title" class="modal-title">新建任务</h3>
              <button class="btn ghost icon small" type="button" aria-label="关闭" @click="createOpen = false">
                <AppIcon name="close" :size="17" />
              </button>
            </div>
            <TaskForm
              :storages="storages"
              :busy="creating"
              @submit="create"
              @cancel="createOpen = false"
              @resize-start="startTaskModalResize"
              @resize-end="endTaskModalResize"
            />
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>
