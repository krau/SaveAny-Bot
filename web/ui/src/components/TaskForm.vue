<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { taskTypeDefs } from '../taskFields'
import AppSelect from './AppSelect.vue'
import type { CreateTaskRequest, JsonObject, StorageRecord } from '../types'

const props = defineProps<{
  storages: StorageRecord[]
  busy?: boolean
}>()

const emit = defineEmits<{
  submit: [payload: CreateTaskRequest]
  cancel: []
}>()

const type = ref(taskTypeDefs[0].type)
const storage = ref('')
const path = ref('')
const webhook = ref('')
const values = reactive<Record<string, string>>({})
const boolValues = reactive<Record<string, boolean>>({})
const error = ref('')

const fields = computed(() => taskTypeDefs.find((item) => item.type === type.value)?.fields ?? [])
const description = computed(
  () => taskTypeDefs.find((item) => item.type === type.value)?.description ?? '',
)
const isTransfer = computed(() => type.value === 'transfer')
const typeOptions = taskTypeDefs.map((item) => ({ value: item.type, label: item.label }))
const storageOptions = computed(() => [
  { value: '', label: '请选择存储' },
  ...props.storages
    .filter((item) => item.enable)
    .map((item) => ({ value: item.name, label: item.name })),
])

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

watch(
  type,
  () => {
    error.value = ''
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
  },
  { immediate: true },
)

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
          throw new Error(`请填写${field.label}`)
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
          throw new Error(`${field.label} 不是合法的 JSON`)
        }
        if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
          throw new Error(`${field.label} 必须是 JSON 对象`)
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
          throw new Error(`${field.label} 必须是数字`)
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
          throw new Error(`请填写${field.label}`)
        }
        if (raw) {
          params[field.name] = raw
        }
      }
    }
  }
  return params
}

function submit() {
  error.value = ''
  try {
    const params = buildParams()
    let targetStorage = storage.value
    let targetPath = path.value.trim()
    if (isTransfer.value) {
      const target = params.target_storage
      targetStorage = typeof target === 'string' ? target : ''
      const transferPath = params.target_path
      targetPath = typeof transferPath === 'string' ? transferPath : ''
    }
    if (!targetStorage) {
      throw new Error('请选择存储')
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
    error.value = err instanceof Error ? err.message : String(err)
  }
}
</script>

<template>
  <form class="stack" @submit.prevent="submit">
    <div class="form-grid">
      <div class="field">
        <label for="task-type">任务类型</label>
        <AppSelect id="task-type" v-model="type" :options="typeOptions" />
        <span class="hint">{{ description }}</span>
      </div>
      <div v-if="!isTransfer" class="field">
        <label for="task-storage">存储</label>
        <AppSelect id="task-storage" v-model="storage" :options="storageOptions" />
      </div>
      <div v-if="!isTransfer" class="field">
        <label for="task-path">目标路径</label>
        <input id="task-path" v-model="path" placeholder="可留空，例如 downloads/anime" />
      </div>
      <div class="field">
        <label for="task-webhook">Webhook（可选）</label>
        <input id="task-webhook" v-model="webhook" placeholder="https://example.com/hook" />
      </div>
    </div>

    <template v-for="field in fields" :key="field.name">
      <label v-if="field.kind === 'bool'" class="checkbox">
        <input v-model="boolValues[field.name]" type="checkbox" />
        <span>{{ field.label }}</span>
      </label>
      <div v-else class="field">
        <label :for="`task-field-${field.name}`">
          {{ field.label }}<span v-if="field.required"> *</span>
        </label>
        <AppSelect
          v-if="field.storageSelect"
          :id="`task-field-${field.name}`"
          v-model="values[field.name]"
          :options="storageOptions"
        />
        <textarea
          v-else-if="field.kind === 'list' || field.kind === 'json'"
          :id="`task-field-${field.name}`"
          v-model="values[field.name]"
          :placeholder="field.placeholder"
        ></textarea>
        <input
          v-else
          :id="`task-field-${field.name}`"
          v-model="values[field.name]"
          :type="field.kind === 'number' ? 'number' : 'text'"
          :placeholder="field.placeholder"
        />
        <span v-if="field.hint" class="hint">{{ field.hint }}</span>
      </div>
    </template>

    <div v-if="error" class="form-error">{{ error }}</div>

    <div class="modal-actions">
      <button class="btn ghost" type="button" @click="emit('cancel')">取消</button>
      <button class="btn primary" type="submit" :disabled="props.busy">
        {{ props.busy ? '创建中…' : '创建任务' }}
      </button>
    </div>
  </form>
</template>
