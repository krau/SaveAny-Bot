<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, parsersApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import PageLoading from '../components/PageLoading.vue'
import { useToastStore } from '../stores/toasts'
import type { ParserList } from '../types'

const { t } = useI18n()
const toast = useToastStore()
const data = ref<ParserList | null>(null)
const loaded = ref(false)
const busy = ref(false)
const form = reactive({ filename: '', content: '' })

async function load() {
  try {
    data.value = await parsersApi.list()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    loaded.value = true
  }
}

onMounted(load)

async function selectFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  form.filename = file.name
  form.content = await file.text()
}

async function install() {
  const filename = form.filename.trim()
  const content = form.content.trim()
  if (!filename || !content) {
    toast.push(t('parsers.required'), 'error')
    return
  }
  busy.value = true
  try {
    await parsersApi.create({ filename, content })
    form.filename = ''
    form.content = ''
    await load()
    toast.push(t('parsers.installed'), 'success')
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('nav.parsers')">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </button>
      </template>
    </PageHeader>

    <PageLoading v-if="!loaded" :rows="4" />

    <template v-else-if="data">
      <div class="panel">
        <div class="panel-head">
          <h2 class="panel-title">{{ t('parsers.title') }}</h2>
          <span class="badge" :class="data.enabled ? 'success' : 'muted'">
            {{ data.enabled ? t('parsers.enabled') : t('parsers.disabled') }}
          </span>
        </div>
        <p class="muted">{{ t('parsers.hint') }}</p>
        <div class="stack">
          <div class="field">
            <label for="parser-file">{{ t('parsers.file') }}</label>
            <div class="file-picker">
              <label class="btn small file-picker-button" for="parser-file">
                <AppIcon name="file" :size="15" />
                {{ t('parsers.chooseFile') }}
              </label>
              <input id="parser-file" class="file-picker-input" type="file" accept=".js,text/javascript" @change="selectFile" />
              <span class="file-picker-name" :class="{ muted: !form.filename }">{{ form.filename || t('parsers.noFile') }}</span>
            </div>
          </div>
          <div class="field">
            <label for="parser-filename">{{ t('parsers.filename') }}</label>
            <input id="parser-filename" v-model="form.filename" placeholder="parser.js" />
          </div>
          <div class="field">
            <label for="parser-content">{{ t('parsers.content') }}</label>
            <textarea id="parser-content" v-model="form.content" rows="10" :placeholder="t('parsers.contentPlaceholder')"></textarea>
          </div>
          <div>
            <button class="btn primary small" type="button" :disabled="busy || !data.enabled" @click="install">
              <AppIcon name="plug" :size="15" />
              {{ t('parsers.install') }}
            </button>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <h2 class="panel-title">{{ t('parsers.installedTitle') }}</h2>
          <span class="badge muted">{{ t('parsers.count', { count: data.files.length }) }}</span>
        </div>
        <div v-if="data.files.length === 0" class="empty"><strong>{{ t('parsers.empty') }}</strong></div>
        <div v-else class="table-wrap">
          <table>
            <thead><tr><th>{{ t('parsers.filename') }}</th><th>{{ t('parsers.directory') }}</th><th>{{ t('parsers.size') }}</th></tr></thead>
            <tbody>
              <tr v-for="file in data.files" :key="`${file.directory}/${file.name}`">
                <td class="mono">{{ file.name }}</td>
                <td class="mono">{{ file.directory }}</td>
                <td>{{ file.size }} B</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.file-picker {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.file-picker-input {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.file-picker-button {
  cursor: pointer;
}

.file-picker-name {
  min-width: 0;
  overflow: hidden;
  font-family: var(--font-mono);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
