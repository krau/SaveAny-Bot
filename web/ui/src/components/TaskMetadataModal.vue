<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { lockBodyScroll } from '../composables/useBodyScrollLock'
import { useModalHeightTransition } from '../composables/useModalHeightTransition'
import { formatBytes, formatDateTime } from '../utils'
import type { TaskMetadataResponse } from '../types'
import AppIcon from './AppIcon.vue'

const props = defineProps<{
  open: boolean
  taskId: string
  data: TaskMetadataResponse | null
  loading: boolean
}>()

const emit = defineEmits<{
  close: []
  download: []
}>()

const { t } = useI18n()
const closeButton = ref<HTMLButtonElement | null>(null)
const modal = ref<HTMLElement | null>(null)
const { startResize, endResize } = useModalHeightTransition(modal)
let releaseScroll: (() => void) | undefined

type Row = {
  label: string
  value: string
  href?: string
  mono?: boolean
}

type Section = {
  key: string
  title: string
  rows: Row[]
  text?: string
  tags?: string[]
}

type ItemView = {
  key: string
  index: number
  name: string
  path: string
  version?: number
  sections: Section[]
  raw: string
}

type ObjectValue = Record<string, unknown>

function object(value: unknown): ObjectValue {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as ObjectValue)
    : {}
}

function text(value: unknown): string {
  if (typeof value === 'string') return value
  if (typeof value === 'number' && Number.isFinite(value)) return String(value)
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  return ''
}

function number(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function tags(value: unknown): string[] {
  return Array.isArray(value) ? value.map(text).filter(Boolean) : []
}

function chatLabel(chat: ObjectValue): string {
  const title = text(chat.title)
  const username = text(chat.username)
  const id = number(chat.id)
  const parts: string[] = []
  if (title) parts.push(title)
  if (username) parts.push(`@${username}`)
  if (id) parts.push(String(id))
  return parts.join(' · ')
}

function senderLabel(sender: ObjectValue): string {
  const name = text(sender.name)
  const username = text(sender.username)
  const id = number(sender.id)
  const parts: string[] = []
  if (name) parts.push(name)
  if (username) parts.push(`@${username}`)
  if (id) parts.push(String(id))
  return parts.join(' · ')
}

function formatDuration(value: number): string {
  const total = Math.max(0, Math.round(value))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = total % 60
  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

function makeRow(label: string, value: string, options: { href?: string; mono?: boolean } = {}): Row | null {
  if (!value.trim()) return null
  return { label, value, ...options }
}

function addRows(target: Row[], rows: Array<Row | null>): void {
  for (const row of rows) {
    if (row) target.push(row)
  }
}

function buildItem(item: TaskMetadataResponse['items'][number]): ItemView {
  const meta = object(item.metadata)
  const message = object(meta.message)
  const file = object(meta.file)
  const chat = object(message.chat)
  const sender = object(message.sender)
  const reply = object(message.reply_to)
  const forward = object(message.forward_from)

  const messageRows: Row[] = []
  addRows(messageRows, [
    makeRow(t('tasks.metadataDialog.fields.chat'), chatLabel(chat)),
    makeRow(t('tasks.metadataDialog.fields.messageId'), text(message.id), { mono: true }),
    makeRow(t('tasks.metadataDialog.fields.date'), message.date ? formatDateTime(text(message.date)) : ''),
    makeRow(t('tasks.metadataDialog.fields.edited'), message.edit_date ? formatDateTime(text(message.edit_date)) : ''),
    makeRow(t('tasks.metadataDialog.fields.sender'), senderLabel(sender)),
    makeRow(t('tasks.metadataDialog.fields.topic'), text(message.topic_id), { mono: true }),
    makeRow(t('tasks.metadataDialog.fields.groupedId'), text(message.grouped_id), { mono: true }),
    makeRow(t('tasks.metadataDialog.fields.link'), text(message.link), { href: text(message.link) }),
  ])
  const messageText = text(message.text)
  const messageTags = tags(meta.tags)
  const sections: Section[] = []
  if (messageRows.length > 0 || messageText || messageTags.length > 0) {
    sections.push({
      key: 'message',
      title: t('tasks.metadataDialog.sections.message'),
      rows: messageRows,
      text: messageText,
      tags: messageTags,
    })
  }

  const replyRows: Row[] = []
  addRows(replyRows, [
    makeRow(t('tasks.metadataDialog.fields.chat'), text(reply.chat_id), { mono: true }),
    makeRow(t('tasks.metadataDialog.fields.messageId'), text(reply.message_id), { mono: true }),
    makeRow(t('tasks.metadataDialog.fields.topic'), text(reply.topic_id), { mono: true }),
  ])
  if (replyRows.length > 0) {
    sections.push({ key: 'reply', title: t('tasks.metadataDialog.sections.reply'), rows: replyRows })
  }

  const forwardRows: Row[] = []
  addRows(forwardRows, [
    makeRow(
      t('tasks.metadataDialog.fields.chat'),
      chatLabel({ title: forward.chat_title, username: forward.chat_username, id: forward.chat_id }),
    ),
    makeRow(t('tasks.metadataDialog.fields.date'), forward.date ? formatDateTime(text(forward.date)) : ''),
    makeRow(t('tasks.metadataDialog.fields.messageId'), text(forward.message_id), { mono: true }),
    makeRow(t('tasks.metadataDialog.fields.author'), text(forward.author)),
  ])
  if (forwardRows.length > 0) {
    sections.push({ key: 'forward', title: t('tasks.metadataDialog.sections.forward'), rows: forwardRows })
  }

  const fileRows: Row[] = []
  const width = number(file.width)
  const height = number(file.height)
  const size = number(file.size)
  const duration = number(file.duration)
  addRows(fileRows, [
    makeRow(t('tasks.metadataDialog.fields.name'), text(file.name) || item.name || ''),
    makeRow(t('tasks.metadataDialog.fields.originalName'), text(file.original_name)),
    makeRow(t('tasks.metadataDialog.fields.mediaType'), text(file.media_type)),
    makeRow(t('tasks.metadataDialog.fields.mimeType'), text(file.mime_type), { mono: true }),
    makeRow(t('tasks.metadataDialog.fields.size'), size && size > 0 ? formatBytes(size) : ''),
    makeRow(
      t('tasks.metadataDialog.fields.dimensions'),
      width && height ? `${width} × ${height}` : '',
    ),
    makeRow(t('tasks.metadataDialog.fields.duration'), duration && duration > 0 ? formatDuration(duration) : ''),
    makeRow(t('tasks.metadataDialog.fields.title'), text(file.title)),
    makeRow(t('tasks.metadataDialog.fields.performer'), text(file.performer)),
    makeRow(t('tasks.metadataDialog.fields.sticker'), text(file.sticker_emoji)),
  ])
  if (fileRows.length > 0) {
    sections.push({ key: 'file', title: t('tasks.metadataDialog.sections.file'), rows: fileRows })
  }

  const version = number(meta.version)
  return {
    key: `${item.index}-${item.name ?? ''}`,
    index: item.index,
    name: item.name || text(file.name) || t('tasks.metadataDialog.unnamed'),
    path: item.path ?? '',
    version,
    sections,
    raw: JSON.stringify(meta, null, 2),
  }
}

const items = computed(() => (props.data?.items ?? []).map(buildItem))
const fileCount = computed(() => items.value.length)

watch(
  () => props.open,
  (open) => {
    if (open) {
      releaseScroll?.()
      releaseScroll = lockBodyScroll()
      void nextTick(() => closeButton.value?.focus())
      return
    }
    releaseScroll?.()
    releaseScroll = undefined
  },
  { immediate: true },
)

watch(
  () => props.loading,
  (loading, previous) => {
    if (!previous || loading) return
    startResize()
    void nextTick(() => endResize())
  },
)

function onKeydown(event: KeyboardEvent) {
  if (!props.open || event.key !== 'Escape') return
  event.preventDefault()
  emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  releaseScroll?.()
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-overlay" @click.self="emit('close')">
        <div
          ref="modal"
          class="modal wide auto-height-modal metadata-modal"
          role="dialog"
          aria-modal="true"
          aria-labelledby="task-metadata-title"
        >
          <div class="modal-head">
            <div class="metadata-heading">
              <h3 id="task-metadata-title" class="modal-title">{{ t('tasks.metadataDialog.title') }}</h3>
              <p class="metadata-subtitle">
                <span class="mono">{{ taskId }}</span>
                <span v-if="fileCount"> · {{ t('tasks.metadataDialog.fileCount', { count: fileCount }) }}</span>
              </p>
            </div>
            <button
              ref="closeButton"
              class="btn ghost icon small"
              type="button"
              :aria-label="t('common.close')"
              @click="emit('close')"
            >
              <AppIcon name="close" :size="17" />
            </button>
          </div>

          <div class="metadata-body">
            <div v-if="loading" class="metadata-state">
              <span class="metadata-spinner" aria-hidden="true"></span>
              <span>{{ t('tasks.metadataDialog.loading') }}</span>
            </div>
            <div v-else-if="items.length === 0" class="metadata-state">
              {{ t('tasks.metadataDialog.empty') }}
            </div>
            <template v-else>
              <article v-for="item in items" :key="item.key" class="metadata-card">
                <div class="metadata-card-head">
                  <div>
                    <h4 class="metadata-file-title">{{ item.name }}</h4>
                    <p v-if="item.path" class="metadata-file-path mono">{{ item.path }}</p>
                    <p v-if="items.length > 1" class="metadata-item-label">{{ t('tasks.metadataDialog.item', { index: item.index + 1 }) }}</p>
                  </div>
                  <span v-if="item.version !== undefined" class="badge">
                    {{ t('tasks.metadataDialog.version', { version: item.version }) }}
                  </span>
                </div>
                <section v-for="section in item.sections" :key="section.key" class="metadata-section">
                  <h5 class="metadata-section-title">{{ section.title }}</h5>
                  <dl v-if="section.rows.length" class="metadata-grid">
                    <template v-for="row in section.rows" :key="row.label">
                      <dt>{{ row.label }}</dt>
                      <dd :class="{ mono: row.mono }">
                        <a v-if="row.href" :href="row.href" target="_blank" rel="noopener noreferrer">{{ row.value }}</a>
                        <span v-else>{{ row.value }}</span>
                      </dd>
                  </template>
                </dl>
                <div v-if="section.text" class="metadata-text-block">
                  <span class="metadata-field-label">{{ t('tasks.metadataDialog.fields.text') }}</span>
                  <p class="metadata-text">{{ section.text }}</p>
                </div>
                <div v-if="section.tags?.length" class="metadata-tags-block">
                  <span class="metadata-field-label">{{ t('tasks.metadataDialog.fields.tags') }}</span>
                  <div class="metadata-tags">
                    <span v-for="tag in section.tags" :key="tag" class="metadata-tag">{{ tag }}</span>
                  </div>
                </div>
              </section>
              <pre v-if="item.sections.length === 0" class="metadata-raw">{{ item.raw }}</pre>
            </article>
            </template>
          </div>

          <div class="modal-actions">
            <button class="btn ghost" type="button" @click="emit('close')">{{ t('common.close') }}</button>
            <button class="btn primary" type="button" :disabled="loading || items.length === 0" @click="emit('download')">
              <AppIcon name="download" :size="15" />
              {{ t('tasks.metadataDialog.download') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
