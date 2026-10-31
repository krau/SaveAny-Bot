<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { errorMessage, storagesApi, tasksApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import AppSelect from '../components/AppSelect.vue'
import PageHeader from '../components/PageHeader.vue'
import TaskForm from '../components/TaskForm.vue'
import TaskTable from '../components/TaskTable.vue'
import { useUiStore } from '../stores/ui'
import type { CreateTaskRequest, StorageRecord, TaskItem } from '../types'

const ui = useUiStore()
const tasks = ref<TaskItem[]>([])
const storages = ref<StorageRecord[]>([])
const total = ref(0)
const statusFilter = ref('')
const typeFilter = ref('')
const busyId = ref('')
const error = ref('')
const createOpen = ref(false)
const creating = ref(false)
let timer: number | undefined

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
    error.value = ''
  } catch (err) {
    error.value = errorMessage(err)
  }
}

async function loadStorages() {
  try {
    storages.value = await storagesApi.list()
  } catch (err) {
    error.value = errorMessage(err)
  }
}

function schedule() {
  if (timer) window.clearInterval(timer)
  timer = window.setInterval(() => void load(), 2000)
}

onMounted(async () => {
  schedule()
  await Promise.all([load(), loadStorages()])
})
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})

async function create(payload: CreateTaskRequest) {
  creating.value = true
  try {
    await tasksApi.create(payload)
    ui.setWarnings([])
    createOpen.value = false
    await load()
  } catch (err) {
    ui.setWarnings([errorMessage(err)])
  } finally {
    creating.value = false
  }
}

async function cancel(taskId: string) {
  if (!window.confirm('取消该任务？')) return
  busyId.value = taskId
  try {
    await tasksApi.cancel(taskId)
    await load()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    busyId.value = ''
  }
}

async function remove(taskId: string) {
  if (!window.confirm('删除该任务记录？')) return
  busyId.value = taskId
  try {
    await tasksApi.remove(taskId)
    await load()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    busyId.value = ''
  }
}
</script>

<template>
  <div class="stack fade-in">
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

    <div v-if="error" class="banner error">{{ error }}</div>

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
      <div v-if="createOpen" class="modal-overlay" @click.self="createOpen = false">
        <div class="modal wide" role="dialog" aria-modal="true" aria-labelledby="create-task-title">
          <div class="modal-head">
            <h3 id="create-task-title" class="modal-title">新建任务</h3>
            <button class="btn ghost icon small" type="button" aria-label="关闭" @click="createOpen = false">
              <AppIcon name="close" :size="17" />
            </button>
          </div>
          <TaskForm :storages="storages" :busy="creating" @submit="create" @cancel="createOpen = false" />
        </div>
      </div>
    </Teleport>
  </div>
</template>
