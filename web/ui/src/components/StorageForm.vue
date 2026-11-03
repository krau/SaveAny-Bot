<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useFormFeedback } from '../composables/useFormFeedback'
import { MASKED_SECRET } from '../api'
import { storageDefaults, storageTypeDef, storageTypeDefs } from '../storageFields'
import type { StorageField } from '../storageFields'
import AppIcon from './AppIcon.vue'
import AppSelect from './AppSelect.vue'
import LoadingButton from './LoadingButton.vue'
import ServerPathPicker from './ServerPathPicker.vue'
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

const { t } = useI18n()
const isEdit = computed(() => props.storage !== null && props.storage !== undefined)

const name = ref('')
const type = ref(storageTypeDefs[0].type)
const enable = ref(true)
const values = reactive<Record<string, string>>({})
const bools = reactive<Record<string, boolean>>({})
const pathPickerOpen = ref(false)
const pathPickerField = ref<StorageField | null>(null)
const { invalidField, fail, clear } = useFormFeedback()

const fields = computed(() => storageTypeDef(type.value)?.fields ?? [])
const hasSavedSecrets = computed(() =>
  fields.value.some((field) => field.kind === 'secret' && values[field.name] === MASKED_SECRET),
)
const storageTypeOptions = computed(() =>
  storageTypeDefs.map((item) => ({ value: item.type, label: t(item.label) })),
)

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

function openPathPicker(field: StorageField) {
  pathPickerField.value = field
  pathPickerOpen.value = true
}

function closePathPicker() {
  pathPickerOpen.value = false
  pathPickerField.value = null
}

function selectPath(path: string) {
  const field = pathPickerField.value
  if (!field) return
  values[field.name] = path
  clear(`storage-field-${field.name}`)
  closePathPicker()
}

function changeType(next: string) {
  if (next === type.value) return
  emit('resize-start')
  type.value = next
  void nextTick(() => emit('resize-end'))
}

function fieldValue(name: string): string {
  const raw = values[name]
  return raw === undefined || raw === null ? '' : String(raw)
}

function setFieldValue(name: string, event: Event) {
  const target = event.target as HTMLInputElement | HTMLTextAreaElement | null
  values[name] = target?.value ?? ''
  clear(`storage-field-${name}`)
}

function validateFields(): boolean {
  for (const field of fields.value) {
    const raw = fieldValue(field.name).trim()
    if (field.required && raw === '') {
      fail(t('validation.required', { field: t(field.label) }), `storage-field-${field.name}`)
      return false
    }
    if (field.kind === 'number' && raw !== '') {
      const num = Number(raw)
      if (!Number.isFinite(num) || (field.name === 'chat_id' && num === 0)) {
        fail(t('validation.invalid', { field: t(field.label) }), `storage-field-${field.name}`)
        return false
      }
    }
  }
  return true
}

function submit() {
  clear()
  const storageName = name.value.trim()
  if (!storageName) {
    fail(t('storages.form.nameRequired'), 'storage-name')
    return
  }
  if (!validateFields()) return
  const config: JsonObject = {}
  for (const field of fields.value) {
    const raw = fieldValue(field.name)
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
        <label for="storage-name">{{ t('storages.form.name') }} *</label>
        <input
          id="storage-name"
          v-model="name"
          maxlength="64"
          :placeholder="t('storages.form.namePlaceholder')"
          :aria-invalid="invalidField === 'storage-name' ? 'true' : undefined"
          required
          @input="clear('storage-name')"
        />
      </div>
      <div class="field">
        <label for="storage-type">{{ t('storages.form.type') }}</label>
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
        <span>{{ t('storages.form.enable') }}</span>
      </label>
    </div>

    <div class="form-grid">
      <template v-for="field in fields" :key="field.name">
        <label v-if="field.kind === 'bool'" class="checkbox">
          <input v-model="bools[field.name]" type="checkbox" />
          <span>{{ t(field.label) }}</span>
        </label>
        <div
          v-else
          class="field"
          :class="{ 'is-invalid': invalidField === `storage-field-${field.name}` }"
        >
          <label :for="`storage-field-${field.name}`">
            {{ t(field.label) }}<span v-if="field.required"> *</span>
          </label>
          <textarea
            v-if="field.kind === 'list'"
            :id="`storage-field-${field.name}`"
            :value="values[field.name]"
            :placeholder="field.placeholder ? t(field.placeholder) : undefined"
            @input="setFieldValue(field.name, $event)"
          ></textarea>
          <div v-else class="path-input-row">
            <input
              :id="`storage-field-${field.name}`"
              :value="values[field.name]"
              :type="field.kind === 'number' ? 'number' : field.kind === 'secret' ? 'password' : 'text'"
              :placeholder="field.placeholder ? t(field.placeholder) : undefined"
              :aria-invalid="invalidField === `storage-field-${field.name}` ? 'true' : undefined"
              autocomplete="off"
              @input="setFieldValue(field.name, $event)"
            />
            <button
              v-if="field.pathKind"
              class="btn small"
              type="button"
              @click="openPathPicker(field)"
            >
              <AppIcon :name="field.pathKind === 'directory' ? 'folder' : 'file'" :size="14" />
              {{ t('common.pathPicker.browse') }}
            </button>
          </div>
          <span v-if="field.hint" class="hint">{{ t(field.hint) }}</span>
        </div>
      </template>
    </div>

      <div v-if="isEdit && hasSavedSecrets" class="hint muted">
        {{ t('storages.form.maskedHint') }}
      </div>
    </div>
    <div class="modal-actions">
      <button class="btn ghost" type="button" @click="emit('cancel')">{{ t('common.actions.cancel') }}</button>
      <LoadingButton type="submit" class="primary" :loading="props.busy">
        {{ isEdit ? t('common.actions.saveChanges') : t('common.actions.createStorage') }}
      </LoadingButton>
    </div>
  </form>

  <ServerPathPicker
    :open="pathPickerOpen"
    :mode="pathPickerField?.pathKind ?? 'directory'"
    :initial-path="pathPickerField ? values[pathPickerField.name] : ''"
    @close="closePathPicker"
    @select="selectPath"
  />
</template>
