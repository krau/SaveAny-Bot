<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, filesystemApi } from '../api'
import type { FilesystemEntry, FilesystemListing } from '../types'
import { formatBytes } from '../utils'
import AppIcon from './AppIcon.vue'
import LoadingButton from './LoadingButton.vue'

const props = defineProps<{
  open: boolean
  mode: 'file' | 'directory'
  initialPath?: string
  /** Keep navigation and selection inside this absolute directory. */
  rootPath?: string
}>()

const emit = defineEmits<{
  close: []
  select: [path: string]
}>()

const { t } = useI18n()
const listing = ref<FilesystemListing | null>(null)
const pathInput = ref('')
const pathInputEl = ref<HTMLInputElement | null>(null)
const loading = ref(false)
const error = ref('')
const selectedFile = ref('')
const showHidden = ref(false)
let previousHtmlOverflow = ''
let previousBodyOverflow = ''
let previousBodyPaddingRight = ''
let scrollLocked = false

const title = computed(() =>
  props.mode === 'file' ? t('common.pathPicker.titleFile') : t('common.pathPicker.titleDirectory'),
)
const visibleEntries = computed(() =>
  (listing.value?.entries ?? []).filter(
    (entry) =>
      (showHidden.value || !entry.name.startsWith('.')) && (props.mode === 'file' || entry.is_dir),
  ),
)
const atRoot = computed(() => Boolean(props.rootPath && listing.value?.path === props.rootPath))
const canConfirm = computed(() =>
  props.mode === 'file' ? Boolean(selectedFile.value) : Boolean(listing.value?.path),
)
const selectedLabel = computed(() => {
  if (props.mode === 'directory') return listing.value?.path ?? ''
  return selectedFile.value || t('common.pathPicker.noSelection')
})

function isWithinRoot(path: string): boolean {
  const root = props.rootPath?.replace(/[\\/]+$/, '')
  if (!root) return true
  return path === root || path.startsWith(`${root}/`) || path.startsWith(`${root}\\`)
}

async function load(path: string) {
  loading.value = true
  error.value = ''
  pathInput.value = path
  try {
    const data = await filesystemApi.list(path)
    if (!isWithinRoot(data.path)) {
      error.value = t('common.pathPicker.outsideRoot')
      listing.value = null
      return
    }
    listing.value = data
    pathInput.value = data.path
    selectedFile.value = props.mode === 'file' ? data.selected ?? '' : ''
  } catch (err) {
    error.value = errorMessage(err)
    listing.value = null
  } finally {
    loading.value = false
  }
}

function lockScroll() {
  if (scrollLocked) return
  const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth
  previousHtmlOverflow = document.documentElement.style.overflow
  previousBodyOverflow = document.body.style.overflow
  previousBodyPaddingRight = document.body.style.paddingRight
  if (scrollbarWidth > 0) {
    const currentPadding = Number.parseFloat(getComputedStyle(document.body).paddingRight) || 0
    document.body.style.paddingRight = `${currentPadding + scrollbarWidth}px`
  }
  document.documentElement.style.overflow = 'hidden'
  document.body.style.overflow = 'hidden'
  scrollLocked = true
}

function unlockScroll() {
  if (!scrollLocked) return
  document.documentElement.style.overflow = previousHtmlOverflow
  document.body.style.overflow = previousBodyOverflow
  document.body.style.paddingRight = previousBodyPaddingRight
  scrollLocked = false
}

function chooseEntry(entry: FilesystemEntry) {
  if (entry.is_dir) {
    void load(entry.path)
    return
  }
  if (props.mode === 'file') {
    selectedFile.value = entry.path
  }
}

function confirm() {
  if (!canConfirm.value) return
  emit('select', props.mode === 'file' ? selectedFile.value : listing.value?.path ?? '')
}

function close() {
  emit('close')
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && props.open) {
    close()
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      showHidden.value = false
      lockScroll()
      void load(props.initialPath ?? '')
      void nextTick(() => pathInputEl.value?.focus())
      return
    }
    unlockScroll()
  },
  { immediate: true },
)

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  unlockScroll()
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-overlay path-picker-overlay" @click.self="close">
        <div class="modal path-picker-modal" role="dialog" aria-modal="true" aria-labelledby="path-picker-title">
          <div class="modal-head">
            <h3 id="path-picker-title" class="modal-title">{{ title }}</h3>
            <button class="btn ghost icon small" type="button" :aria-label="t('common.close')" @click="close">
              <AppIcon name="close" :size="17" />
            </button>
          </div>

          <div class="path-picker-toolbar">
            <button
              class="btn small"
              type="button"
              :disabled="!listing?.parent || loading || atRoot"
              @click="load(listing?.parent ?? '')"
            >
              <AppIcon name="arrowUp" :size="14" />
              {{ t('common.pathPicker.parent') }}
            </button>
            <div class="path-picker-location">
              <input
                ref="pathInputEl"
                v-model="pathInput"
                :aria-label="t('common.pathPicker.open')"
                spellcheck="false"
                @keydown.enter.prevent="load(pathInput)"
              />
              <LoadingButton class="small" :loading="loading" @click="load(pathInput)">
                {{ t('common.pathPicker.open') }}
              </LoadingButton>
            </div>
          </div>

          <div class="path-picker-options">
            <div v-if="!rootPath" class="path-picker-shortcuts">
              <button class="btn ghost small" type="button" @click="load(listing?.root ?? '/')">
                <AppIcon name="hardDrive" :size="14" />
                {{ t('common.pathPicker.root') }}
              </button>
              <button class="btn ghost small" type="button" @click="load(listing?.cwd ?? '')">
                <AppIcon name="home" :size="14" />
                {{ t('common.pathPicker.cwd') }}
              </button>
            </div>
            <label class="checkbox path-picker-hidden-toggle">
              <input v-model="showHidden" type="checkbox" />
              <span>{{ t('common.pathPicker.showHidden') }}</span>
            </label>
          </div>

          <div class="path-picker-list">
            <div v-if="loading" class="path-picker-state">{{ t('common.pathPicker.loading') }}</div>
            <div v-else-if="error" class="path-picker-state path-picker-error">
              <span>{{ error }}</span>
              <button class="btn small" type="button" @click="load(pathInput)">
                <AppIcon name="refresh" :size="14" />
                {{ t('common.actions.refresh') }}
              </button>
            </div>
            <template v-else>
              <button
                v-for="entry in visibleEntries"
                :key="entry.path"
                class="path-picker-entry"
                :class="{ selected: selectedFile === entry.path }"
                type="button"
                @click="chooseEntry(entry)"
              >
                <AppIcon :name="entry.is_dir ? 'folder' : 'file'" :size="16" />
                <span class="path-picker-entry-name">{{ entry.name }}</span>
                <span v-if="!entry.is_dir" class="path-picker-entry-size">{{ formatBytes(entry.size) }}</span>
                <AppIcon v-else name="chevronRight" :size="14" class="path-picker-entry-arrow" />
              </button>
              <div v-if="visibleEntries.length === 0" class="path-picker-state">
                {{ mode === 'directory' ? t('common.pathPicker.emptyFolders') : t('common.pathPicker.empty') }}
              </div>
            </template>
          </div>

          <div v-if="listing?.truncated" class="path-picker-truncated">
            {{ t('common.pathPicker.truncated') }}
          </div>

          <div class="path-picker-footer">
            <span class="path-picker-selected" :title="selectedLabel">{{ selectedLabel }}</span>
            <div class="actions">
              <button class="btn ghost" type="button" @click="close">{{ t('common.pathPicker.cancel') }}</button>
              <button class="btn primary" type="button" :disabled="!canConfirm" @click="confirm">
                {{ mode === 'file' ? t('common.pathPicker.selectFile') : t('common.pathPicker.selectDirectory') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
