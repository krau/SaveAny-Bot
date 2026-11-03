<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, storagesApi, tasksApi } from '../api'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import AppSelect from '../components/AppSelect.vue'
import PageHeader from '../components/PageHeader.vue'
import PageLoading from '../components/PageLoading.vue'
import TaskForm from '../components/TaskForm.vue'
import TaskTable from '../components/TaskTable.vue'
import { useModalHeightTransition } from '../composables/useModalHeightTransition'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import type { CreateTaskRequest, StorageRecord, TaskItem } from '../types'

const { t } = useI18n()
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
const loaded = ref(false)
const createOpen = ref(false)
const creating = ref(false)
const taskModal = ref<HTMLElement | null>(null)
const { startResize: startTaskModalResize, endResize: endTaskModalResize } =
  useModalHeightTransition(taskModal)
let timer: number | undefined

const statusOptions = computed(() => [
  { value: '', label: t('tasks.filters.allStatuses') },
  { value: 'queued', label: t('tasks.status.queued') },
  { value: 'running', label: t('tasks.status.running') },
  { value: 'completed', label: t('tasks.status.completed') },
  { value: 'partial', label: t('tasks.status.partial') },
  { value: 'failed', label: t('tasks.status.failed') },
  { value: 'cancelled', label: t('tasks.status.cancelled') },
])
const typeOptions = computed(() => [
  { value: '', label: t('tasks.filters.allTypes') },
  { value: 'directlinks', label: t('tasks.types.directlinks') },
  { value: 'ytdlp', label: t('tasks.types.ytdlp') },
  { value: 'aria2', label: t('tasks.types.aria2') },
  { value: 'parseditem', label: t('tasks.types.parseditem') },
  { value: 'tgfiles', label: t('tasks.types.tgfiles') },
  { value: 'tphpics', label: t('tasks.types.tphpics') },
  { value: 'transfer', label: t('tasks.types.transfer') },
])

async function load() {
  try {
    const data = await tasksApi.list({ status: statusFilter.value, type: typeFilter.value, limit: 100 })
    tasks.value = data.tasks ?? []
    total.value = data.total ?? 0
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    loaded.value = true
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
})

async function create(payload: CreateTaskRequest) {
  creating.value = true
  try {
    await tasksApi.create(payload)
    toast.push(t('tasks.toast.created'), 'success')
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
    title: t('tasks.confirm.cancelTitle'),
    message: t('tasks.confirm.cancelMessage'),
    confirmText: t('tasks.confirm.cancelConfirm'),
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

async function retry(taskId: string) {
  const confirmed = await confirm.ask({
    title: t('tasks.confirm.retryTitle'),
    message: t('tasks.confirm.retryMessage'),
    confirmText: t('tasks.confirm.retryConfirm'),
  })
  if (!confirmed) return
  busyId.value = taskId
  try {
    await tasksApi.retry(taskId)
    toast.push(t('tasks.toast.retried'), 'success')
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busyId.value = ''
  }
}

async function remove(taskId: string) {
  const confirmed = await confirm.ask({
    title: t('tasks.confirm.removeTitle'),
    message: t('tasks.confirm.removeMessage'),
    confirmText: t('tasks.confirm.removeConfirm'),
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
    <PageHeader :title="t('nav.tasks')">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </button>
        <button class="btn primary small" type="button" @click="createOpen = true">
          <AppIcon name="plus" :size="15" />
          {{ t('common.actions.createTask') }}
        </button>
      </template>
    </PageHeader>

    <div class="filter-bar">
      <div class="filter-field">
        <label for="task-status-filter">{{ t('tasks.filters.status') }}</label>
        <AppSelect id="task-status-filter" v-model="statusFilter" :options="statusOptions" @change="load" />
      </div>
      <div class="filter-field">
        <label for="task-type-filter">{{ t('tasks.filters.type') }}</label>
        <AppSelect id="task-type-filter" v-model="typeFilter" :options="typeOptions" @change="load" />
      </div>
      <div class="filter-spacer"></div>
      <span v-if="loaded" class="count">{{ t('tasks.count', { total }) }}</span>
    </div>

    <PageLoading v-if="!loaded" :rows="4" />
    <TaskTable v-else :tasks="tasks" :busy-id="busyId" @cancel="cancel" @remove="remove" @retry="retry" />

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="createOpen" class="modal-overlay" @click.self="createOpen = false">
          <div
            ref="taskModal"
            class="modal wide auto-height-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="create-task-title"
          >
            <div class="modal-head">
              <h3 id="create-task-title" class="modal-title">{{ t('tasks.createTitle') }}</h3>
              <button class="btn ghost icon small" type="button" :aria-label="t('common.close')" @click="createOpen = false">
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
