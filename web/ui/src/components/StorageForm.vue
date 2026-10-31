<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useFormFeedback } from '../composables/useFormFeedback'
import { storageDefaults, storageTypeDef, storageTypeDefs } from '../storageFields'
import AppSelect from './AppSelect.vue'
import LoadingButton from './LoadingButton.vue'
import type { JsonObject, StorageInput, StorageRecord } from '../types'

const props = defineProps<{
  storage?: StorageRecord | null
  busy?: boolean
}>()

const emit = defineEmits<{
  submit: [payload: StorageInput]
  cancel: []
  'resize-start': []
  'resize-end': []
}>()

const isEdit = computed(() => props.storage !== null && props.storage !== undefined)

const name = ref('')
const type = ref(storageTypeDefs[0].type)
const enable = ref(true)
const values = reactive<Record<string, string>>({})
const bools = reactive<Record<string, boolean>>({})
const { invalidField, fail, clear } = useFormFeedback()

const fields = computed(() => storageTypeDef(type.value)?.fields ?? [])
const storageTypeOptions = storageTypeDefs.map((item) => ({ value: item.type, label: item.label }))

function clearFields() {
  for (const key of Object.keys(values)) {
    delete values[key]
  }
  for (const key of Object.keys(bools)) {
    delete bools[key]
  }
}

function applyDefaults() {
  clearFields()
  const defaults = storageDefaults(type.value)
  for (const field of fields.value) {
    const defaultValue = defaults[field.name]
    if (field.kind === 'bool') {
      bools[field.name] = Boolean(defaultValue)
    } else if (Array.isArray(defaultValue)) {
      values[field.name] = defaultValue.map(String).join('\n')
    } else {
      values[field.name] = defaultValue === undefined || defaultValue === null ? '' : String(defaultValue)
    }
  }
}

function loadStorage(storage: StorageRecord) {
  name.value = storage.name
  type.value = storage.type
  enable.value = storage.enable
  clearFields()
  for (const field of fields.value) {
    const raw = storage.config[field.name]
    if (field.kind === 'bool') {
      bools[field.name] = raw === true
    } else if (field.kind === 'list') {
      values[field.name] = Array.isArray(raw) ? raw.map((item) => String(item)).join('\n') : ''
    } else {
      values[field.name] = raw === undefined || raw === null ? '' : String(raw)
    }
  }
}

watch(
  () => props.storage,
  (storage) => {
    clear()
    if (storage) {
      loadStorage(storage)
    } else {
      name.value = ''
      type.value = storageTypeDefs[0].type
      enable.value = true
      applyDefaults()
    }
  },
  { immediate: true },
)

watch(type, () => {
  clear()
  if (!isEdit.value) {
    applyDefaults()
  }
})

function changeType(next: string) {
  if (next === type.value) return
  emit('resize-start')
  type.value = next
  void nextTick(() => emit('resize-end'))
}

function submit() {
  clear()
  const storageName = name.value.trim()
  if (!storageName) {
    fail('请填写存储名称', 'storage-name')
    return
  }
  const config: JsonObject = {}
  for (const field of fields.value) {
    const raw = values[field.name] ?? ''
    switch (field.kind) {
      case 'bool':
        config[field.name] = bools[field.name] ?? false
        break
      case 'number': {
        const num = Number(raw.trim())
        config[field.name] = raw.trim() !== '' && Number.isFinite(num) ? num : 0
        break
      }
      case 'list':
        config[field.name] = raw
          .split(/\r?\n/)
          .map((item) => item.trim())
          .filter((item) => item.length > 0)
        break
      default:
        config[field.name] = raw
    }
  }
  emit('submit', {
    name: storageName,
    type: type.value,
    enable: enable.value,
    config,
  })
}
</script>

<template>
  <form class="stack modal-form" @submit.prevent="submit" novalidate>
    <div class="modal-form-body">
      <div class="form-grid">
      <div class="field" :class="{ 'is-invalid': invalidField === 'storage-name' }">
        <label for="storage-name">名称</label>
        <input
          id="storage-name"
          v-model="name"
          :disabled="isEdit"
          placeholder="例如 my-webdav"
          :aria-invalid="invalidField === 'storage-name' ? 'true' : undefined"
          required
          @input="clear('storage-name')"
        />
        <span v-if="isEdit" class="hint">名称不可修改</span>
      </div>
      <div class="field">
        <label for="storage-type">类型</label>
        <AppSelect
          id="storage-type"
          :model-value="type"
          :options="storageTypeOptions"
          :disabled="isEdit"
          @update:model-value="changeType"
        />
      </div>
      <label class="checkbox">
        <input v-model="enable" type="checkbox" />
        <span>启用该存储</span>
      </label>
    </div>

    <div class="form-grid">
      <template v-for="field in fields" :key="field.name">
        <label v-if="field.kind === 'bool'" class="checkbox">
          <input v-model="bools[field.name]" type="checkbox" />
          <span>{{ field.label }}</span>
        </label>
        <div v-else class="field">
          <label :for="`storage-field-${field.name}`">{{ field.label }}</label>
          <textarea
            v-if="field.kind === 'list'"
            :id="`storage-field-${field.name}`"
            v-model="values[field.name]"
            :placeholder="field.placeholder"
          ></textarea>
          <input
            v-else
            :id="`storage-field-${field.name}`"
            v-model="values[field.name]"
            :type="field.kind === 'number' ? 'number' : field.kind === 'secret' ? 'password' : 'text'"
            :placeholder="field.placeholder"
            autocomplete="off"
          />
          <span v-if="field.hint" class="hint">{{ field.hint }}</span>
        </div>
      </template>
    </div>

      <div v-if="isEdit" class="hint muted">
        已保存的密钥显示为 <code>__MASKED__</code>，保持不变即可继续使用原值。
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn ghost" type="button" @click="emit('cancel')">取消</button>
      <LoadingButton type="submit" class="primary" :loading="props.busy">
        {{ isEdit ? '保存修改' : '创建存储' }}
      </LoadingButton>
    </div>
  </form>
</template>
