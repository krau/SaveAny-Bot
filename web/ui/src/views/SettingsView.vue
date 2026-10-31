<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { errorMessage, settingsApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import AppSelect from '../components/AppSelect.vue'
import PageHeader from '../components/PageHeader.vue'
import { settingsGroups } from '../settingsFields'
import type { SettingsField } from '../settingsFields'
import { useUiStore } from '../stores/ui'
import type { JsonObject } from '../types'

const ui = useUiStore()
const values = reactive<Record<string, Record<string, unknown>>>({})
const error = ref('')
const notice = ref('')
const busyGroup = ref('')
const loaded = ref(false)
const activeGroup = ref(settingsGroups[0]?.key ?? '')
const settingsNav = ref<HTMLElement | null>(null)
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
    error.value = ''
  } catch (err) {
    error.value = errorMessage(err)
  }
}

function updateActiveGroup() {
  if (!loaded.value) return
  let current = settingsGroups[0]?.key ?? ''
  for (const group of settingsGroups) {
    const element = document.getElementById(`settings-${group.key}`)
    if (element && element.getBoundingClientRect().top <= 112) {
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

watch(activeGroup, async () => {
  if (!window.matchMedia('(max-width: 960px)').matches) return
  await nextTick()
  const active = settingsNav.value?.querySelector<HTMLElement>('button.active')
  if (active) {
    settingsNav.value?.scrollTo({ left: active.offsetLeft - 16, behavior: 'smooth' })
  }
})

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
        invalid[fieldKey(group, field.path)] = `${field.label} 不是合法的 JSON`
        error.value = invalid[fieldKey(group, field.path)]
        return
      }
    }
  }
  error.value = ''
  writePath(values[group], field.path, value)
}

async function save(group: string) {
  busyGroup.value = group
  notice.value = ''
  error.value = ''
  try {
    const def = settingsGroups.find((item) => item.key === group)
    if (!def) return
    for (const field of def.fields) {
      if (invalid[fieldKey(group, field.path)]) {
        throw new Error(invalid[fieldKey(group, field.path)])
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
    ui.setRestartRequired(result.restart_required ?? [])
    ui.setWarnings(result.warnings ?? [])
    notice.value = `${def.title}设置已保存`
    await load()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    busyGroup.value = ''
  }
}
</script>

<template>
  <div class="stack fade-in">
    <PageHeader title="设置" description="配置保存在数据库中，保存后立即生效，或提示需要重启。">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          重新加载
        </button>
      </template>
    </PageHeader>

    <div v-if="error" class="banner error">{{ error }}</div>
    <div v-if="notice" class="banner success">{{ notice }}</div>
    <div v-if="!loaded" class="loading">正在加载设置…</div>

    <div v-show="loaded" class="settings-layout">
      <nav ref="settingsNav" class="settings-nav" aria-label="设置分组">
        <button
          v-for="group in settingsGroups"
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
          v-for="group in settingsGroups"
          :id="`settings-${group.key}`"
          :key="group.key"
          class="panel settings-group"
          @submit.prevent="save(group.key)"
        >
          <div class="settings-group-head">
            <h2>{{ group.title }}</h2>
            <p>{{ group.description }}</p>
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
              <div v-else class="field">
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
                  @input="setTextField(group.key, field, ($event.target as HTMLTextAreaElement).value)"
                ></textarea>
                <input
                  v-else
                  :id="`${group.key}-${field.path}`"
                  :value="textValue(group.key, field)"
                  :type="field.kind === 'number' ? 'number' : field.kind === 'password' ? 'password' : 'text'"
                  :placeholder="field.placeholder"
                  autocomplete="off"
                  @input="setTextField(group.key, field, ($event.target as HTMLInputElement).value)"
                />
                <span v-if="field.hint" class="hint">{{ field.hint }}</span>
              </div>
            </template>
          </div>
          <div class="modal-actions">
            <button class="btn primary" type="submit" :disabled="busyGroup === group.key">
              {{ busyGroup === group.key ? '保存中…' : '保存' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
