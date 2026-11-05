<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  dirsApi,
  errorMessage,
  rulesApi,
  storagesApi,
  telegramApi,
  usersApi,
  watchesApi,
} from '../api'
import AppIcon from '../components/AppIcon.vue'
import AppSelect from '../components/AppSelect.vue'
import PageHeader from '../components/PageHeader.vue'
import PageLoading from '../components/PageLoading.vue'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import type { StorageRecord, UserDir, UserItem, UserRule, UserWatch } from '../types'

const { t } = useI18n()
const toast = useToastStore()
const confirm = useConfirmStore()
const users = ref<UserItem[]>([])
const storages = ref<StorageRecord[]>([])
const selectedId = ref<number | null>(null)
const dirs = ref<UserDir[]>([])
const rules = ref<UserRule[]>([])
const watches = ref<UserWatch[]>([])
const loaded = ref(false)
const refreshing = ref(false)
const syncing = ref(false)

const selectedUser = computed(() => users.value.find((user) => user.id === selectedId.value) ?? null)
const userOptions = computed(() =>
  users.value.map((user) => ({
    value: String(user.id),
    label: `${user.chat_id}${user.enabled ? '' : ` (${t('states.disabled')})`}`,
  })),
)
const enabledStorages = computed(() => storages.value.filter((storage) => storage.enable))
const storageOptions = computed(() => [
  { value: '', label: t('automation.selectStorage') },
  ...enabledStorages.value.map((storage) => ({ value: String(storage.storage_id), label: storage.name })),
])
const ruleStorageOptions = computed(() => [
  { value: '0', label: t('automation.rules.chosenStorage') },
  ...enabledStorages.value.map((storage) => ({ value: String(storage.storage_id), label: storage.name })),
])
const ruleTypeOptions = computed(() => [
  { value: 'FILENAME-REGEX', label: t('automation.rules.types.filename') },
  { value: 'MESSAGE-REGEX', label: t('automation.rules.types.message') },
  { value: 'IS-ALBUM', label: t('automation.rules.types.album') },
])

const dirForm = reactive({ storage_id: '', path: '' })
const ruleForm = reactive({ type: 'FILENAME-REGEX', content: '', storage_id: '0', dir_path: '' })
const presetForm = reactive({ storage_id: '0', base_path: '' })
const watchForm = reactive({ chat_id: '', filter: '' })

function storageName(id: number): string {
  return storages.value.find((storage) => storage.storage_id === id)?.name ?? `#${id}`
}

function ruleTypeLabel(type: string): string {
  switch (type) {
    case 'FILENAME-REGEX':
      return t('automation.rules.types.filename')
    case 'MESSAGE-REGEX':
      return t('automation.rules.types.message')
    case 'IS-ALBUM':
      return t('automation.rules.types.album')
    default:
      return type
  }
}

async function loadUsers() {
  const [userList, storageList] = await Promise.all([usersApi.list(), storagesApi.list()])
  users.value = userList
  storages.value = storageList
  if (selectedId.value === null || !userList.some((user) => user.id === selectedId.value)) {
    selectedId.value = userList[0]?.id ?? null
  }
}

async function loadAutomation() {
  if (!selectedUser.value) {
    dirs.value = []
    rules.value = []
    watches.value = []
    return
  }
  const id = selectedUser.value.id
  const [dirList, ruleList, watchList] = await Promise.all([
    dirsApi.list(id),
    rulesApi.list(id),
    watchesApi.list(id),
  ])
  dirs.value = dirList
  rules.value = ruleList
  watches.value = watchList
}

async function refresh() {
  refreshing.value = true
  try {
    await loadUsers()
    await loadAutomation()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    refreshing.value = false
    loaded.value = true
  }
}

onMounted(refresh)
watch(selectedId, async () => {
  try {
    await loadAutomation()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
})

async function addDir() {
  if (!selectedUser.value) return
  const storageId = Number(dirForm.storage_id)
  const path = dirForm.path.trim()
  if (!storageId || !path) {
    toast.push(t('automation.directories.required'), 'error')
    return
  }
  try {
    await dirsApi.create(selectedUser.value.id, { storage_id: storageId, path })
    dirForm.path = ''
    await loadAutomation()
    toast.push(t('automation.directories.created'), 'success')
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function removeDir(dir: UserDir) {
  if (!selectedUser.value) return
  const confirmed = await confirm.ask({
    title: t('automation.directories.deleteTitle'),
    message: t('automation.directories.deleteMessage', { path: dir.path }),
    confirmText: t('common.actions.delete'),
    danger: true,
  })
  if (!confirmed) return
  try {
    await dirsApi.remove(selectedUser.value.id, dir.id)
    await loadAutomation()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function setDefaultDir(dir: UserDir) {
  if (!selectedUser.value) return
  try {
    await usersApi.update(selectedUser.value.id, {
      default_storage_id: dir.storage_id,
      default_dir_id: dir.id,
    })
    await loadUsers()
    toast.push(t('automation.directories.defaultSet'), 'success')
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function toggleRules() {
  if (!selectedUser.value) return
  try {
    await usersApi.update(selectedUser.value.id, { apply_rule: !selectedUser.value.apply_rule })
    await loadUsers()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function addRule() {
  if (!selectedUser.value) return
  const content = ruleForm.content.trim()
  if (!content) {
    toast.push(t('automation.rules.contentRequired'), 'error')
    return
  }
  try {
    await rulesApi.create(selectedUser.value.id, {
      type: ruleForm.type,
      content,
      storage_id: Number(ruleForm.storage_id),
      dir_path: ruleForm.dir_path.trim(),
    })
    ruleForm.content = ''
    ruleForm.dir_path = ''
    await loadAutomation()
    toast.push(t('automation.rules.created'), 'success')
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function removeRule(rule: UserRule) {
  if (!selectedUser.value) return
  const confirmed = await confirm.ask({
    title: t('automation.rules.deleteTitle'),
    message: t('automation.rules.deleteMessage', { type: ruleTypeLabel(rule.type) }),
    confirmText: t('common.actions.delete'),
    danger: true,
  })
  if (!confirmed) return
  try {
    await rulesApi.remove(selectedUser.value.id, rule.id)
    await loadAutomation()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function importPresets() {
  if (!selectedUser.value) return
  try {
    const result = await rulesApi.importPresets(selectedUser.value.id, {
      storage_id: Number(presetForm.storage_id),
      base_path: presetForm.base_path.trim(),
    })
    await loadAutomation()
    toast.push(t('automation.rules.presetsImported', { count: result.created }), 'success')
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function toggleWatchNotify() {
  if (!selectedUser.value) return
  try {
    await usersApi.update(selectedUser.value.id, { watch_notify: !selectedUser.value.watch_notify })
    await loadUsers()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function addWatch() {
  if (!selectedUser.value) return
  const chat = watchForm.chat_id.trim()
  if (!chat) {
    toast.push(t('automation.watches.chatRequired'), 'error')
    return
  }
  try {
    await watchesApi.create(selectedUser.value.id, {
      chat,
      filter: watchForm.filter.trim(),
    })
    watchForm.chat_id = ''
    watchForm.filter = ''
    await loadAutomation()
    toast.push(t('automation.watches.created'), 'success')
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function removeWatch(watch: UserWatch) {
  if (!selectedUser.value) return
  const confirmed = await confirm.ask({
    title: t('automation.watches.deleteTitle'),
    message: t('automation.watches.deleteMessage', { chat: watch.chat_id }),
    confirmText: t('common.actions.delete'),
    danger: true,
  })
  if (!confirmed) return
  try {
    await watchesApi.remove(selectedUser.value.id, watch.chat_id)
    await loadAutomation()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

async function syncPeers() {
  if (syncing.value) return
  const confirmed = await confirm.ask({
    title: t('automation.watches.syncConfirmTitle'),
    message: t('automation.watches.syncConfirmMessage'),
    confirmText: t('automation.watches.syncConfirmText'),
  })
  if (!confirmed) return
  syncing.value = true
  toast.push(t('automation.watches.syncStarted'), 'info', 6000)
  try {
    const result = await telegramApi.syncPeers()
    toast.push(t('automation.watches.synced', { count: result.synced }), 'success')
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    syncing.value = false
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('nav.automation')">
      <template #actions>
        <button class="btn small" type="button" :disabled="refreshing" @click="refresh">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </button>
      </template>
    </PageHeader>

    <PageLoading v-if="!loaded" :rows="5" />

    <template v-else>
      <div v-if="users.length === 0" class="panel empty">
        <span class="empty-icon"><AppIcon name="users" :size="17" /></span>
        <strong>{{ t('automation.noUsers') }}</strong>
      </div>

      <template v-else>
        <div class="panel">
          <div class="panel-head">
            <h2 class="panel-title">{{ t('automation.userTitle') }}</h2>
          </div>
          <div class="field">
            <label for="automation-user">{{ t('automation.selectUser') }}</label>
            <AppSelect id="automation-user" :model-value="selectedId === null ? '' : String(selectedId)" :options="userOptions" @update:model-value="selectedId = Number($event)" />
          </div>
        </div>

        <div class="panel">
          <div class="panel-head">
            <h2 class="panel-title">{{ t('automation.directories.title') }}</h2>
            <span class="badge muted">{{ t('automation.directories.count', { count: dirs.length }) }}</span>
          </div>
          <form class="form-section" @submit.prevent="addDir">
            <div class="form-grid">
              <div class="field">
                <label for="dir-storage">{{ t('automation.directories.storage') }}</label>
                <AppSelect id="dir-storage" v-model="dirForm.storage_id" :options="storageOptions" />
              </div>
              <div class="field">
                <label for="dir-path">{{ t('automation.directories.path') }}</label>
                <input id="dir-path" v-model="dirForm.path" :placeholder="t('automation.directories.pathPlaceholder')" />
              </div>
            </div>
            <div class="form-actions">
              <button class="btn primary" type="submit">
                <AppIcon name="plus" :size="15" />
                {{ t('automation.directories.add') }}
              </button>
            </div>
          </form>

          <div v-if="dirs.length === 0" class="empty"><strong>{{ t('automation.directories.empty') }}</strong></div>
          <div v-else class="table-wrap">
            <table>
              <thead><tr><th>{{ t('automation.directories.storage') }}</th><th>{{ t('automation.directories.path') }}</th><th>{{ t('automation.actions') }}</th></tr></thead>
              <tbody>
                <tr v-for="dir in dirs" :key="dir.id">
                  <td>{{ storageName(dir.storage_id) }}</td>
                  <td class="mono">{{ dir.path }}</td>
                  <td>
                    <div class="actions">
                      <button class="btn small" type="button" @click="setDefaultDir(dir)">{{ t('automation.directories.makeDefault') }}</button>
                      <button class="btn small danger" type="button" @click="removeDir(dir)">{{ t('common.actions.delete') }}</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="panel">
          <div class="panel-head">
            <h2 class="panel-title">{{ t('automation.rules.title') }}</h2>
            <div class="panel-toolbar">
              <span class="badge" :class="selectedUser?.apply_rule ? 'success' : 'muted'">
                {{ selectedUser?.apply_rule ? t('automation.rules.enabled') : t('automation.rules.disabled') }}
              </span>
              <button class="btn small" type="button" @click="toggleRules">
                {{ selectedUser?.apply_rule ? t('automation.rules.disable') : t('automation.rules.enable') }}
              </button>
            </div>
          </div>

          <form class="form-section" @submit.prevent="addRule">
            <div class="form-grid">
              <div class="field">
                <label for="rule-type">{{ t('automation.rules.type') }}</label>
                <AppSelect id="rule-type" v-model="ruleForm.type" :options="ruleTypeOptions" />
              </div>
              <div class="field">
                <label for="rule-content">{{ t('automation.rules.content') }}</label>
                <input id="rule-content" v-model="ruleForm.content" :placeholder="t('automation.rules.contentPlaceholder')" />
              </div>
              <div class="field">
                <label for="rule-storage">{{ t('automation.rules.storage') }}</label>
                <AppSelect id="rule-storage" v-model="ruleForm.storage_id" :options="ruleStorageOptions" />
              </div>
              <div class="field">
                <label for="rule-dir">{{ t('automation.rules.dirPath') }}</label>
                <input id="rule-dir" v-model="ruleForm.dir_path" :placeholder="t('automation.rules.dirPathPlaceholder')" />
              </div>
            </div>
            <div class="form-actions">
              <button class="btn primary" type="submit">
                <AppIcon name="plus" :size="15" />
                {{ t('automation.rules.add') }}
              </button>
            </div>
          </form>

          <form class="form-section" @submit.prevent="importPresets">
            <h3 class="form-section-title">{{ t('automation.rules.presetsTitle') }}</h3>
            <div class="form-grid">
              <div class="field">
                <label for="preset-storage">{{ t('automation.rules.presetStorage') }}</label>
                <AppSelect id="preset-storage" v-model="presetForm.storage_id" :options="ruleStorageOptions" />
              </div>
              <div class="field">
                <label for="preset-path">{{ t('automation.rules.presetBasePath') }}</label>
                <input id="preset-path" v-model="presetForm.base_path" :placeholder="t('automation.rules.presetBasePathPlaceholder')" />
              </div>
            </div>
            <div class="form-actions">
              <button class="btn" type="submit">
                <AppIcon name="inbox" :size="15" />
                {{ t('automation.rules.importPresets') }}
              </button>
            </div>
          </form>

          <div v-if="rules.length === 0" class="empty"><strong>{{ t('automation.rules.empty') }}</strong></div>
          <div v-else class="table-wrap">
            <table>
              <thead><tr><th>{{ t('automation.rules.type') }}</th><th>{{ t('automation.rules.content') }}</th><th>{{ t('automation.rules.storage') }}</th><th>{{ t('automation.rules.dirPath') }}</th><th>{{ t('automation.actions') }}</th></tr></thead>
              <tbody>
                <tr v-for="rule in rules" :key="rule.id">
                  <td>{{ ruleTypeLabel(rule.type) }}</td>
                  <td class="mono">{{ rule.content }}</td>
                  <td>{{ rule.storage_id === 0 ? t('automation.rules.chosenStorage') : storageName(rule.storage_id) }}</td>
                  <td class="mono">{{ rule.dir_path || '—' }}</td>
                  <td><button class="btn small danger" type="button" @click="removeRule(rule)">{{ t('common.actions.delete') }}</button></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="panel">
          <div class="panel-head">
            <h2 class="panel-title">{{ t('automation.watches.title') }}</h2>
            <div class="panel-toolbar">
              <label class="checkbox">
                <input type="checkbox" :checked="selectedUser?.watch_notify" @change="toggleWatchNotify" />
                <span>{{ t('automation.watches.notify') }}</span>
              </label>
              <button class="btn small" type="button" :disabled="syncing" :aria-busy="syncing || undefined" @click="syncPeers">
                <span v-if="syncing" class="sync-spinner" aria-hidden="true"></span>
                <AppIcon v-else name="refresh" :size="15" />
                {{ syncing ? t('automation.watches.syncing') : t('automation.watches.syncPeers') }}
              </button>
            </div>
          </div>

          <form class="form-section" @submit.prevent="addWatch">
            <div class="form-grid">
              <div class="field">
                <label for="watch-chat">{{ t('automation.watches.chatId') }}</label>
                <input id="watch-chat" v-model="watchForm.chat_id" :placeholder="t('automation.watches.chatPlaceholder')" />
              </div>
              <div class="field">
                <label for="watch-filter">{{ t('automation.watches.filter') }}</label>
                <input id="watch-filter" v-model="watchForm.filter" :placeholder="t('automation.watches.filterPlaceholder')" />
              </div>
            </div>
            <div class="form-actions">
              <button class="btn primary" type="submit">
                <AppIcon name="plus" :size="15" />
                {{ t('automation.watches.add') }}
              </button>
            </div>
          </form>

          <div v-if="watches.length === 0" class="empty"><strong>{{ t('automation.watches.empty') }}</strong></div>
          <div v-else class="table-wrap">
            <table>
              <thead><tr><th>{{ t('automation.watches.chatId') }}</th><th>{{ t('automation.watches.filter') }}</th><th>{{ t('automation.actions') }}</th></tr></thead>
              <tbody>
                <tr v-for="watch in watches" :key="watch.id">
                  <td class="mono">{{ watch.chat_id }}</td>
                  <td class="mono">{{ watch.message_filter || '—' }}</td>
                  <td><button class="btn small danger" type="button" @click="removeWatch(watch)">{{ t('common.actions.delete') }}</button></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.panel-head {
  align-items: center;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.panel-toolbar .badge {
  min-height: 32px;
  padding: 0 10px;
}

.panel-toolbar .checkbox {
  min-height: 32px;
  padding-top: 0;
}

.form-section + .form-section {
  padding-top: 24px;
  margin-top: 24px;
  border-top: 1px solid var(--line);
}

.form-section-title {
  margin: 0 0 16px;
  color: var(--text);
  font-size: 14px;
  font-weight: 500;
}

.form-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
}

.form-actions .btn {
  min-width: 120px;
  height: 40px;
}

@media (max-width: 640px) {
  .form-actions {
    justify-content: stretch;
  }

  .form-actions .btn {
    width: 100%;
  }
}

.sync-spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 1.5px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 700ms linear infinite;
}
</style>
