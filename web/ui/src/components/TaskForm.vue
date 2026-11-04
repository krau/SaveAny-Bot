<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, filesystemApi } from '../api'
import { useFormFeedback } from '../composables/useFormFeedback'
import { taskTypeDefs } from '../taskFields'
import AppIcon from './AppIcon.vue'
import AppSelect from './AppSelect.vue'
import LoadingButton from './LoadingButton.vue'
import ServerPathPicker from './ServerPathPicker.vue'
import type { CreateTaskRequest, JsonObject, StorageRecord } from '../types'

const props = defineProps<{
  storages: StorageRecord[]
  busy?: boolean
}>()

const emit = defineEmits<{
  submit: [payload: CreateTaskRequest]
  cancel: []
  'resize-start': []
  'resize-end': []
}>()

const type = ref(taskTypeDefs[0].type)
const storage = ref('')
const path = ref('')
const webhook = ref('')
const body = ref<HTMLElement | null>(null)
const values = reactive<Record<string, string>>({})
const boolValues = reactive<Record<string, boolean>>({})
const pathPickerOpen = ref(false)
const pathPickerLoading = ref(false)
const pathPickerInitial = ref('')
const pathPickerRoot = ref('')
const { invalidField, fail, clear } = useFormFeedback()
const { t } = useI18n()

const fields = computed(() => taskTypeDefs.find((item) => item.type === type.value)?.fields ?? [])
const isTransfer = computed(() => type.value === 'transfer')
const typeOptions = computed(() => taskTypeDefs.map((item) => ({ value: item.type, label: t(item.label) })))
const storageOptions = computed(() => [
  { value: '', label: t('tasks.form.storagePlaceholder') },
  ...props.storages
    .filter((item) => item.enable)
    .map((item) => ({ value: item.name, label: item.name })),
])
const selectedStorage = computed(() => props.storages.find((item) => item.name === storage.value))
const localStorageRoot = computed(() => {
  const selected = selectedStorage.value
  if (!selected || selected.type !== 'local') return ''
  const basePath = selected.config.base_path
  return typeof basePath === 'string' ? basePath.trim() : ''
})

watch(
  () => props.storages,
  (list) => {
    if (storage.value || list.length === 0) {
      return
    }
    const enabled = list.find((item) => item.enable)
    storage.value = (enabled ?? list[0]).name
  },
  { immediate: true },
)

watch(storage, () => {
  pathPickerOpen.value = false
})

watch(
  type,
  () => {
    clear()
    for (const key of Object.keys(values)) {
      delete values[key]
    }
    for (const key of Object.keys(boolValues)) {
      delete boolValues[key]
    }
    for (const field of fields.value) {
      if (field.kind === 'bool') {
        boolValues[field.name] = false
      } else {
        values[field.name] = ''
      }
    }
    void nextTick(() => body.value?.scrollTo({ top: 0 }))
  },
  { immediate: true },
)

function changeType(next: string) {
  if (next === type.value) return
  emit('resize-start')
  type.value = next
  void nextTick(() => emit('resize-end'))
}

function splitList(raw: string): string[] {
  return raw
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function buildParams(): JsonObject {
  const params: JsonObject = {}
  for (const field of fields.value) {
    const raw = (values[field.name] ?? '').trim()
    switch (field.kind) {
      case 'list': {
        const list = splitList(raw)
        if (field.required && list.length === 0) {
          invalidField.value = `task-field-${field.name}`
          throw new Error(t('tasks.form.required', { field: t(field.label) }))
        }
        if (list.length > 0) {
          params[field.name] = list
        }
        break
      }
      case 'json': {
        if (!raw) {
          break
        }
        let parsed: unknown
        try {
          parsed = JSON.parse(raw)
        } catch {
          invalidField.value = `task-field-${field.name}`
          throw new Error(t('tasks.form.invalidJson', { field: t(field.label) }))
        }
        if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
          invalidField.value = `task-field-${field.name}`
          throw new Error(t('tasks.form.jsonObject', { field: t(field.label) }))
        }
        const options: JsonObject = {}
        for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
          options[key] = typeof value === 'string' ? value : String(value)
        }
        params[field.name] = options
        break
      }
      case 'number': {
        if (!raw) {
          break
        }
        const num = Number(raw)
        if (!Number.isFinite(num)) {
          invalidField.value = `task-field-${field.name}`
          throw new Error(t('tasks.form.notNumber', { field: t(field.label) }))
        }
        params[field.name] = num
        break
      }
      case 'bool': {
        params[field.name] = boolValues[field.name] ?? false
        break
      }
      default: {
        if (!raw && field.required) {
          invalidField.value = `task-field-${field.name}`
          throw new Error(t('tasks.form.required', { field: t(field.label) }))
        }
        if (raw) {
          params[field.name] = raw
        }
      }
    }
  }
  return params
}

async function openTaskPathPicker() {
  if (!localStorageRoot.value) return
  pathPickerLoading.value = true
  try {
    const listing = await filesystemApi.list(localStorageRoot.value)
    pathPickerRoot.value = listing.path
    const relative = path.value.trim().replace(/^[/\\]+/, '')
    pathPickerInitial.value = relative ? `${listing.path}/${relative}` : listing.path
    pathPickerOpen.value = true
  } catch (err) {
    fail(errorMessage(err))
  } finally {
    pathPickerLoading.value = false
  }
}

function selectTaskPath(selected: string) {
  const root = pathPickerRoot.value
  if (!root) return
  if (selected !== root && !selected.startsWith(`${root}/`) && !selected.startsWith(`${root}\\`)) {
    fail(t('tasks.form.pathOutsideStorage'))
    return
  }
  path.value = selected === root ? '' : selected.slice(root.length + 1)
  pathPickerOpen.value = false
}

function submit() {
  clear()
  try {
    const params = buildParams()
    if (type.value === 'tgfiles') {
      const links = (values.message_links ?? '').trim()
      const chatID = (values.chat_id ?? '').trim()
      const messageRange = (values.message_range ?? '').trim()
      if (!links && (!chatID || !messageRange)) {
        invalidField.value = 'task-field-message_links'
        throw new Error(t('tasks.form.tgfilesSourceRequired'))
      }
    }
    let targetStorage = storage.value
    let targetPath = path.value.trim()
    if (isTransfer.value) {
      const target = params.target_storage
      targetStorage = typeof target === 'string' ? target : ''
      const transferPath = params.target_path
      targetPath = typeof transferPath === 'string' ? transferPath : ''
    }
    if (!targetStorage) {
      invalidField.value = isTransfer.value ? 'task-field-target_storage' : 'task-storage'
      throw new Error(t('tasks.form.storageRequired'))
    }
    const payload: CreateTaskRequest = {
      type: type.value,
      storage: targetStorage,
      path: targetPath,
      params,
    }
    if (webhook.value.trim()) {
      payload.webhook = webhook.value.trim()
    }
    emit('submit', payload)
  } catch (err) {
    fail(err instanceof Error ? err.message : String(err), invalidField.value)
  }
}
</script>

<template>
  <form class="stack modal-form" @submit.prevent="submit" novalidate>
    <div ref="body" class="modal-form-body">
      <div class="form-grid">
        <div class="field">
          <label for="task-type">{{ t('tasks.form.taskType') }}</label>
          <AppSelect
            id="task-type"
            :model-value="type"
            :options="typeOptions"
            @update:model-value="changeType"
          />
        </div>
        <div v-if="!isTransfer" class="field">
          <label for="task-storage">{{ t('tasks.form.storage') }}</label>
          <AppSelect
            id="task-storage"
            v-model="storage"
            :options="storageOptions"
            @update:model-value="clear('task-storage')"
          />
        </div>
        <div v-if="!isTransfer" class="field">
          <label for="task-path">{{ t('tasks.form.targetPath') }}</label>
          <div class="path-input-row">
            <input id="task-path" v-model="path" :placeholder="t('tasks.form.targetPathPlaceholder')" />
            <LoadingButton
              v-if="localStorageRoot"
              class="small"
              :loading="pathPickerLoading"
              @click="openTaskPathPicker"
            >
              <AppIcon name="folder" :size="14" />
              {{ t('common.pathPicker.browse') }}
            </LoadingButton>
          </div>
        </div>
        <div class="field">
          <label for="task-webhook">{{ t('tasks.form.webhook') }}</label>
          <input id="task-webhook" v-model="webhook" :placeholder="t('tasks.form.webhookPlaceholder')" />
        </div>
      </div>

      <template v-for="field in fields" :key="field.name">
        <label v-if="field.kind === 'bool'" class="checkbox">
          <input v-model="boolValues[field.name]" type="checkbox" />
          <span>{{ t(field.label) }}</span>
        </label>
        <div
          v-else
          class="field"
          :class="{ 'is-invalid': invalidField === `task-field-${field.name}` }"
        >
          <label :for="`task-field-${field.name}`">
            {{ t(field.label) }}<span v-if="field.required"> *</span>
          </label>
          <AppSelect
            v-if="field.storageSelect"
            :id="`task-field-${field.name}`"
            v-model="values[field.name]"
            :options="storageOptions"
            @update:model-value="clear(`task-field-${field.name}`)"
          />
          <textarea
            v-else-if="field.kind === 'list' || field.kind === 'json'"
            :id="`task-field-${field.name}`"
            v-model="values[field.name]"
            :placeholder="field.placeholder"
            :aria-invalid="invalidField === `task-field-${field.name}` ? 'true' : undefined"
            @input="clear(`task-field-${field.name}`)"
          ></textarea>
          <input
            v-else
            :id="`task-field-${field.name}`"
            v-model="values[field.name]"
            :type="field.kind === 'number' ? 'number' : 'text'"
            :placeholder="field.placeholder"
            :aria-invalid="invalidField === `task-field-${field.name}` ? 'true' : undefined"
            @input="clear(`task-field-${field.name}`)"
          />
          <span v-if="field.hint" class="hint">{{ t(field.hint) }}</span>
        </div>
      </template>
    </div>

    <div class="modal-actions">
      <button class="btn ghost" type="button" @click="emit('cancel')">{{ t('common.actions.cancel') }}</button>
      <LoadingButton type="submit" class="primary" :loading="props.busy">{{ t('common.actions.createTask') }}</LoadingButton>
    </div>
  </form>

  <ServerPathPicker
    :open="pathPickerOpen"
    mode="directory"
    :initial-path="pathPickerInitial"
    :root-path="pathPickerRoot"
    @close="pathPickerOpen = false"
    @select="selectTaskPath"
  />
</template>
