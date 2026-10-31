<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage, storagesApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageHeader from '../components/PageHeader.vue'
import StorageForm from '../components/StorageForm.vue'
import { useModalHeightTransition } from '../composables/useModalHeightTransition'
import { storageTypeIcon, storageTypeLabel } from '../storageFields'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import type { StorageInput, StorageRecord } from '../types'
import { storageLocation } from '../utils'

const toast = useToastStore()
const confirm = useConfirmStore()
const storages = ref<StorageRecord[]>([])
const busy = ref(false)
const testingId = ref(0)
const removingId = ref(0)
const editing = ref<StorageRecord | null>(null)
const formOpen = ref(false)
const storageModal = ref<HTMLElement | null>(null)
const { startResize, endResize } = useModalHeightTransition(storageModal)

async function load() {
  try {
    storages.value = await storagesApi.list()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

onMounted(load)

function openCreate() {
  editing.value = null
  formOpen.value = true
}

function openEdit(storage: StorageRecord) {
  editing.value = storage
  formOpen.value = true
}

async function save(payload: StorageInput) {
  busy.value = true
  try {
    if (editing.value) {
      await storagesApi.update(editing.value.id, payload)
      toast.push('存储已更新', 'success')
    } else {
      await storagesApi.create(payload)
      toast.push('存储已创建', 'success')
    }
    formOpen.value = false
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  } finally {
    busy.value = false
  }
}

async function remove(storage: StorageRecord) {
  const confirmed = await confirm.ask({
    title: '删除存储',
    message: `删除存储 ${storage.name}？已有的文件不会被删除。`,
    confirmText: '删除存储',
    danger: true,
  })
  if (!confirmed) return
  removingId.value = storage.id
  try {
    await storagesApi.remove(storage.id)
    toast.push(`${storage.name} 已删除`, 'success')
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error')
  } finally {
    removingId.value = 0
  }
}

async function test(storage: StorageRecord) {
  testingId.value = storage.id
  try {
    const result = await storagesApi.test(storage.id)
    if (result.ok) {
      toast.push(`${storage.name} 连接正常`, 'success')
    } else {
      toast.push(`${storage.name} 测试失败：${result.error ?? '未知错误'}`, 'error', 6000)
    }
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    testingId.value = 0
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader title="存储">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          刷新
        </button>
        <button class="btn primary small" type="button" @click="openCreate">
          <AppIcon name="plus" :size="15" />
          新建存储
        </button>
      </template>
    </PageHeader>

    <div class="storage-grid">
      <article v-for="storage in storages" :key="storage.id" class="storage-card">
        <div class="storage-card-head">
          <div class="storage-identity">
            <span class="storage-icon"><AppIcon :name="storageTypeIcon(storage.type)" :size="18" /></span>
            <div style="min-width: 0">
              <h2>{{ storage.name }}</h2>
              <div class="storage-kind">{{ storageTypeLabel(storage.type) }}</div>
            </div>
          </div>
          <span class="badge status" :class="storage.enable ? 'success' : 'muted'">
            {{ storage.enable ? '已启用' : '已禁用' }}
          </span>
        </div>

        <div class="storage-detail">{{ storageLocation(storage.config) }}</div>
        <p v-if="storage.error" class="storage-card-error">{{ storage.error }}</p>

        <div class="storage-card-foot">
          <div class="actions">
            <button class="btn small" type="button" @click="openEdit(storage)">
              <AppIcon name="edit" :size="14" />
              编辑
            </button>
            <LoadingButton
              class="small ghost"
              type="button"
              :loading="testingId === storage.id"
              @click="test(storage)"
            >
              <AppIcon name="plug" :size="14" />
              测试
            </LoadingButton>
          </div>
          <button
            class="btn small danger"
            type="button"
            :disabled="removingId === storage.id"
            @click="remove(storage)"
          >
            <AppIcon name="trash" :size="14" />
            删除
          </button>
        </div>
      </article>

      <div v-if="storages.length === 0" class="storage-empty">
        <span class="empty-icon"><AppIcon name="database" :size="18" /></span>
        <strong style="color: var(--text-2); font-size: 14px; font-weight: 500">还没有存储</strong>
        <button class="btn primary small" type="button" @click="openCreate">
          <AppIcon name="plus" :size="14" />
          新建存储
        </button>
      </div>
    </div>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="formOpen" class="modal-overlay" @click.self="formOpen = false">
          <div
            ref="storageModal"
            class="modal wide auto-height-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="storage-dialog-title"
          >
            <div class="modal-head">
              <h3 id="storage-dialog-title" class="modal-title">{{ editing ? '编辑存储' : '新建存储' }}</h3>
              <button class="btn ghost icon small" type="button" aria-label="关闭" @click="formOpen = false">
                <AppIcon name="close" :size="17" />
              </button>
            </div>
            <StorageForm
              :storage="editing"
              :busy="busy"
              @submit="save"
              @cancel="formOpen = false"
              @resize-start="startResize"
              @resize-end="endResize"
            />
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>
