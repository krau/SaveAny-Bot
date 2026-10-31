<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, settingsApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import AppSelect from '../components/AppSelect.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageHeader from '../components/PageHeader.vue'
import { settingsGroups } from '../settingsFields'
import type { SettingsField } from '../settingsFields'
import { useToastStore } from '../stores/toasts'
import type { JsonObject } from '../types'

const { t } = useI18n()
const toast = useToastStore()
const values = reactive<Record<string, Record<string, unknown>>>({})
const busyGroup = ref('')
const loaded = ref(false)
const activeGroup = ref(settingsGroups[0]?.key ?? '')
const translatedGroups = computed(() =>
  settingsGroups.map((group) => ({
    ...group,
    title: t(group.title),
    fields: group.fields.map((field) => ({
      ...field,
      label: t(field.label),
      hint: field.hint ? t(field.hint) : undefined,
      options: field.options?.map((option) => ({ value: option.value, label: t(option.label) })),
    })),
  })),
)
const settingsGroupOptions = computed(() =>
  translatedGroups.value.map((group) => ({ value: group.key, label: group.title })),
)
const invalid = reactive<Record<string, string>>({})
let scrollFrame = 0
let navLock = false

function fieldKey(group: string, path: string): string {
  return `${group}:${path}`
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

async function load() {
  try {
    const all = await settingsApi.getAll()
    for (const group of settingsGroups) {
      values[group.key] = clone((all[group.key] ?? {}) as Record<string, unknown>)
      for (const key of Object.keys(invalid)) {
        if (key.startsWith(`${group.key}:`)) delete invalid[key]
      }
    }
    loaded.value = true
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

function updateActiveGroup() {
  if (!loaded.value) return
  const threshold = window.matchMedia('(max-width: 960px)').matches ? 132 : 88
  let current = settingsGroups[0]?.key ?? ''
  for (const group of settingsGroups) {
    const element = document.getElementById(`settings-${group.key}`)
    if (element && element.getBoundingClientRect().top <= threshold) {
      current = group.key
    }
  }
  if (window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 4) {
    current = settingsGroups[settingsGroups.length - 1]?.key ?? current
  }
  activeGroup.value = current
}

function scheduleActiveGroupUpdate() {
  if (navLock || scrollFrame) return
  scrollFrame = window.requestAnimationFrame(() => {
    scrollFrame = 0
    if (navLock) return
    updateActiveGroup()
  })
}

function releaseNavLock() {
  if (!navLock) return
  navLock = false
  scheduleActiveGroupUpdate()
}

onMounted(async () => {
  await load()
  await nextTick()
  updateActiveGroup()
  window.addEventListener('scroll', scheduleActiveGroupUpdate, { passive: true })
  window.addEventListener('resize', scheduleActiveGroupUpdate)
  window.addEventListener('wheel', releaseNavLock, { passive: true })
  window.addEventListener('touchmove', releaseNavLock, { passive: true })
  window.addEventListener('keydown', releaseNavLock)
  window.addEventListener('pointerdown', releaseNavLock)
})

onUnmounted(() => {
  window.removeEventListener('scroll', scheduleActiveGroupUpdate)
  window.removeEventListener('resize', scheduleActiveGroupUpdate)
  window.removeEventListener('wheel', releaseNavLock)
  window.removeEventListener('touchmove', releaseNavLock)
  window.removeEventListener('keydown', releaseNavLock)
  window.removeEventListener('pointerdown', releaseNavLock)
  if (scrollFrame) window.cancelAnimationFrame(scrollFrame)
})

function jumpToGroup(key: string) {
  activeGroup.value = key
  navLock = true
  document.getElementById(`settings-${key}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function readPath(source: Record<string, unknown>, path: string): unknown {
  let current: unknown = source
  for (const key of path.split('.')) {
    if (current === null || typeof current !== 'object') return undefined
    current = (current as Record<string, unknown>)[key]
  }
  return current
}

function writePath(target: Record<string, unknown>, path: string, value: unknown) {
  const keys = path.split('.')
  let current = target
  for (let i = 0; i < keys.length - 1; i += 1) {
    const key = keys[i]
    if (!current[key] || typeof current[key] !== 'object' || Array.isArray(current[key])) {
      current[key] = {}
    }
    current = current[key] as Record<string, unknown>
  }
  current[keys[keys.length - 1]] = value
}

function textValue(group: string, field: SettingsField): string {
  const value = readPath(values[group] ?? {}, field.path)
  if (value === null || value === undefined) return ''
  if (Array.isArray(value)) return value.join('\n')
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

function boolValue(group: string, field: SettingsField): boolean {
  return Boolean(readPath(values[group] ?? {}, field.path))
}

function setTextField(group: string, field: SettingsField, raw: string) {
  const key = fieldKey(group, field.path)
  if (field.kind !== 'json') delete invalid[key]
  let value: unknown = raw
  if (field.kind === 'number') {
    value = raw === '' ? 0 : Number(raw)
  } else if (field.kind === 'list') {
    value = raw.split(/\r?\n/).map((item) => item.trim()).filter(Boolean)
  } else if (field.kind === 'json') {
    if (raw.trim() === '') {
      value = {}
    } else {
      try {
        value = JSON.parse(raw)
        delete invalid[fieldKey(group, field.path)]
      } catch {
        invalid[key] = t('settings.errors.invalidJson', { field: field.label })
        toast.push(invalid[key], 'error')
        return
      }
    }
  }
  writePath(values[group], field.path, value)
}

async function save(group: string) {
  busyGroup.value = group
  try {
    const def = translatedGroups.value.find((item) => item.key === group)
    if (!def) return
    for (const field of def.fields) {
      const key = fieldKey(group, field.path)
      if (invalid[key]) {
        toast.push(invalid[key], 'error')
        return
      }
      if (field.kind === 'number') {
        const value = Number(readPath(values[group], field.path) ?? 0)
        const minimum = field.min
        if (!Number.isFinite(value) || (minimum !== undefined && value < minimum)) {
          invalid[key] =
            minimum === undefined
              ? t('settings.errors.numberRequired', { field: field.label })
              : t('settings.errors.minValue', { field: field.label, min: minimum })
          toast.push(invalid[key], 'error')
          document.getElementById(`${group}-${field.path}`)?.focus()
          return
        }
      }
    }
    const patch: JsonObject = {}
    for (const field of def.fields) {
      const value = readPath(values[group], field.path)
      if (field.kind === 'json' && (value === undefined || value === null)) {
        continue
      }
      writePath(patch, field.path, (value ?? (field.kind === 'bool' ? false : field.kind === 'number' ? 0 : '')) as never)
    }
    const result = await settingsApi.updateGroup(group, patch)
    toast.push(t('settings.toast.saved', { group: def.title }), 'success')
    if (result.restart_required?.length) {
      toast.push(
        t('settings.toast.restartRequired', {
          items: result.restart_required.join(t('settings.listSeparator')),
        }),
        'info',
        8000,
      )
    }
    if (result.warnings?.length) {
      toast.push(result.warnings.join(t('settings.warningSeparator')), 'error', 8000)
    }
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busyGroup.value = ''
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('nav.settings')">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.reload') }}
        </button>
      </template>
    </PageHeader>

    <div v-if="!loaded" class="loading">{{ t('settings.loading') }}</div>

    <div v-show="loaded" class="settings-mobile-nav">
      <AppSelect
        :model-value="activeGroup"
        :options="settingsGroupOptions"
        :aria-label="t('settings.groupAria')"
        @update:model-value="jumpToGroup"
      />
    </div>

    <div v-show="loaded" class="settings-layout">
      <nav class="settings-nav" :aria-label="t('settings.groupAria')">
        <button
          v-for="group in translatedGroups"
          :key="group.key"
          type="button"
          :class="{ active: activeGroup === group.key }"
          @click="jumpToGroup(group.key)"
        >
          {{ group.title }}
        </button>
      </nav>

      <div class="settings-sections">
        <form
          v-for="group in translatedGroups"
          novalidate
          :id="`settings-${group.key}`"
          :key="group.key"
          class="panel settings-group"
          @submit.prevent="save(group.key)"
        >
          <div class="settings-group-head">
            <h2>{{ group.title }}</h2>
          </div>
          <div class="form-grid">
            <template v-for="field in group.fields" :key="field.path">
              <label v-if="field.kind === 'bool'" class="checkbox">
                <input
                  type="checkbox"
                  :checked="boolValue(group.key, field)"
                  @change="writePath(values[group.key], field.path, ($event.target as HTMLInputElement).checked)"
                />
                <span>{{ field.label }}</span>
              </label>
              <div
                v-else
                class="field"
                :class="{ 'is-invalid': invalid[fieldKey(group.key, field.path)] }"
              >
                <label :for="`${group.key}-${field.path}`">{{ field.label }}</label>
                <AppSelect
                  v-if="field.kind === 'select'"
                  :id="`${group.key}-${field.path}`"
                  :model-value="textValue(group.key, field)"
                  :options="field.options ?? []"
                  @update:model-value="setTextField(group.key, field, $event)"
                />
                <textarea
                  v-else-if="field.kind === 'list' || field.kind === 'json'"
                  :id="`${group.key}-${field.path}`"
                  :value="textValue(group.key, field)"
                  :placeholder="field.placeholder"
                  :aria-invalid="invalid[fieldKey(group.key, field.path)] ? 'true' : undefined"
                  @input="setTextField(group.key, field, ($event.target as HTMLTextAreaElement).value)"
                ></textarea>
                <input
                  v-else
                  :id="`${group.key}-${field.path}`"
                  :value="textValue(group.key, field)"
                  :type="field.kind === 'number' ? 'number' : field.kind === 'password' ? 'password' : 'text'"
                  :placeholder="field.placeholder"
                  autocomplete="off"
                  :aria-invalid="invalid[fieldKey(group.key, field.path)] ? 'true' : undefined"
                  @input="setTextField(group.key, field, ($event.target as HTMLInputElement).value)"
                />
                <span v-if="field.hint" class="hint">{{ field.hint }}</span>
              </div>
            </template>
          </div>
          <div class="modal-actions">
            <LoadingButton type="submit" class="primary" :loading="busyGroup === group.key">
              {{ t('common.actions.save') }}
            </LoadingButton>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
