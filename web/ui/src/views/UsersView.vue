<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, storagesApi, usersApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import AppSelect from '../components/AppSelect.vue'
import PageHeader from '../components/PageHeader.vue'
import { useFormFeedback } from '../composables/useFormFeedback'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import type { StorageRecord, UserItem, UserPatch, UserStorageFilter } from '../types'

const { t } = useI18n()
const toast = useToastStore()
const confirm = useConfirmStore()
const { invalidField, fail, clear, success } = useFormFeedback()
const users = ref<UserItem[]>([])
const storages = ref<StorageRecord[]>([])
const busy = ref(false)
const formOpen = ref(false)
const editing = ref<UserItem | null>(null)
const enabledStorages = computed(() => storages.value.filter((storage) => storage.enable))
const enabledUsers = computed(() => users.value.filter((user) => user.enabled).length)
const defaultStorageOptions = computed(() => [
  { value: '', label: t('users.unset') },
  ...enabledStorages.value.map((storage) => ({ value: storage.name, label: storage.name })),
])
const filenameStrategyOptions = computed(() => [
  { value: 'default', label: t('users.filenameStrategy.default') },
  { value: 'message', label: t('users.filenameStrategy.message') },
  { value: 'template', label: t('users.filenameStrategy.template') },
])
const conflictStrategyOptions = computed(() => [
  { value: '', label: t('users.conflictStrategy.default') },
  { value: 'rename', label: t('users.conflictStrategy.rename') },
  { value: 'ask', label: t('users.conflictStrategy.ask') },
  { value: 'overwrite', label: t('users.conflictStrategy.overwrite') },
  { value: 'skip', label: t('users.conflictStrategy.skip') },
])

const form = reactive({
  chat_id: '',
  enabled: true,
  silent: false,
  default_storage: '',
  apply_rule: false,
  watch_notify: false,
  filename_strategy: '',
  filename_template: '',
  conflict_strategy: '',
  blacklist: false,
  storages: [] as string[],
})

async function load() {
  try {
    const [userList, storageList] = await Promise.all([usersApi.list(), storagesApi.list()])
    users.value = userList
    storages.value = storageList
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}

onMounted(load)

function openCreate() {
  clear()
  editing.value = null
  Object.assign(form, {
    chat_id: '', enabled: true, silent: false, default_storage: '', apply_rule: false,
    watch_notify: false, filename_strategy: '', filename_template: '', conflict_strategy: '',
    blacklist: false, storages: [],
  })
  formOpen.value = true
}

function openEdit(user: UserItem) {
  clear()
  editing.value = user
  Object.assign(form, {
    chat_id: String(user.chat_id),
    enabled: user.enabled,
    silent: user.silent,
    default_storage: user.default_storage,
    apply_rule: user.apply_rule,
    watch_notify: user.watch_notify,
    filename_strategy: user.filename_strategy,
    filename_template: user.filename_template,
    conflict_strategy: user.conflict_strategy,
    blacklist: user.storage_filter.blacklist,
    storages: [...user.storage_filter.storages],
  })
  formOpen.value = true
}

function toggleStorage(name: string) {
  const index = form.storages.indexOf(name)
  if (index >= 0) form.storages.splice(index, 1)
  else form.storages.push(name)
}

async function save() {
  clear()
  if (form.filename_strategy === 'template' && !form.filename_template.trim()) {
    fail(t('users.toast.templateRequired'), 'user-template')
    return
  }
  busy.value = true
  const filter: UserStorageFilter = { storages: form.storages, blacklist: form.blacklist }
  const patch: UserPatch = {
    enabled: form.enabled,
    silent: form.silent,
    default_storage: form.default_storage,
    apply_rule: form.apply_rule,
    watch_notify: form.watch_notify,
    filename_strategy: form.filename_strategy,
    filename_template: form.filename_template,
    conflict_strategy: form.conflict_strategy,
    storage_filter: filter,
  }
  try {
    if (editing.value) {
      await usersApi.update(editing.value.id, patch)
      success(t('users.toast.updated'))
    } else {
      const chatID = Number(form.chat_id)
      if (!Number.isFinite(chatID) || chatID === 0) {
        fail(t('users.toast.chatIdInvalid'), 'user-chat-id')
        return
      }
      await usersApi.create({ chat_id: chatID, ...patch })
      success(t('users.toast.created'))
    }
    formOpen.value = false
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busy.value = false
  }
}

async function remove(user: UserItem) {
  const confirmed = await confirm.ask({
    title: t('users.confirm.deleteTitle'),
    message: t('users.confirm.deleteMessage', { chatId: user.chat_id }),
    confirmText: t('users.confirm.deleteTitle'),
    danger: true,
  })
  if (!confirmed) return
  try {
    await usersApi.remove(user.id)
    success(t('users.toast.deleted', { chatId: user.chat_id }))
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader :title="t('nav.users')">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </button>
        <button class="btn primary small" type="button" @click="openCreate">
          <AppIcon name="plus" :size="15" />
          {{ t('users.add') }}
        </button>
      </template>
    </PageHeader>

    <div class="filter-bar">
      <span class="count">{{ t('users.count', { count: users.length, enabled: enabledUsers }) }}</span>
      <div class="filter-spacer"></div>
    </div>

    <div class="users-table">
      <div class="table-wrap users-table-wrap">
        <table>
        <thead><tr><th>{{ t('users.columns.chatId') }}</th><th>{{ t('users.columns.status') }}</th><th>{{ t('users.columns.defaultStorage') }}</th><th>{{ t('users.columns.storageFilter') }}</th><th>{{ t('users.columns.actions') }}</th></tr></thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td class="mono">{{ user.chat_id }}</td>
            <td>
              <span class="badge status" :class="user.enabled ? 'success' : 'muted'">
                {{ user.enabled ? t('users.status.enabled') : t('users.status.disabled') }}
              </span>
              <span v-if="user.silent" class="badge info" style="margin-left: 6px">{{ t('users.status.silent') }}</span>
            </td>
            <td>{{ user.default_storage || '—' }}</td>
            <td class="muted">
              {{ user.storage_filter.blacklist ? t('users.filter.blacklist') : t('users.filter.whitelist') }}{{ t('users.filter.separator') }}
              {{ user.storage_filter.storages.length ? user.storage_filter.storages.join(t('users.storageSeparator')) : (user.storage_filter.blacklist ? t('users.filter.all') : t('users.filter.none')) }}
            </td>
            <td>
              <div class="actions">
                <button class="btn small" type="button" @click="openEdit(user)">{{ t('common.actions.edit') }}</button>
                <button class="btn small danger" type="button" @click="remove(user)">{{ t('common.actions.delete') }}</button>
              </div>
            </td>
          </tr>
          <tr v-if="users.length === 0">
            <td colspan="5" class="empty">
              <span class="empty-icon"><AppIcon name="users" :size="17" /></span>
              <strong>{{ t('users.empty') }}</strong>
            </td>
          </tr>
        </tbody>
      </table>
      </div>

      <div class="user-mobile-list">
        <article v-for="user in users" :key="user.id" class="user-mobile-card">
          <div class="user-mobile-head">
            <span class="user-chat mono">{{ user.chat_id }}</span>
            <div class="actions">
              <span class="badge status" :class="user.enabled ? 'success' : 'muted'">
                {{ user.enabled ? t('users.status.enabled') : t('users.status.disabled') }}
              </span>
              <span v-if="user.silent" class="badge info">{{ t('users.status.silent') }}</span>
            </div>
          </div>
          <dl class="user-mobile-details">
            <div>
              <dt>{{ t('users.columns.defaultStorage') }}</dt>
              <dd>{{ user.default_storage || t('users.unset') }}</dd>
            </div>
            <div>
              <dt>{{ t('users.columns.storageFilter') }}</dt>
              <dd>
                {{ user.storage_filter.blacklist ? t('users.filter.blacklist') : t('users.filter.whitelist') }}{{ t('users.filter.separator') }}{{
                  user.storage_filter.storages.length
                    ? user.storage_filter.storages.join(t('users.storageSeparator'))
                    : user.storage_filter.blacklist
                      ? t('users.filter.all')
                      : t('users.filter.none')
                }}
              </dd>
            </div>
          </dl>
          <div class="user-mobile-actions">
            <button class="btn small" type="button" @click="openEdit(user)">{{ t('common.actions.edit') }}</button>
            <button class="btn small danger" type="button" @click="remove(user)">{{ t('common.actions.delete') }}</button>
          </div>
        </article>
        <div v-if="users.length === 0" class="empty">
          <span class="empty-icon"><AppIcon name="users" :size="17" /></span>
          <strong>{{ t('users.empty') }}</strong>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="formOpen" class="modal-overlay" @click.self="formOpen = false">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="user-dialog-title">
          <div class="modal-head">
            <h3 id="user-dialog-title" class="modal-title">{{ editing ? t('users.edit') : t('users.add') }}</h3>
            <button class="btn ghost icon small" type="button" :aria-label="t('common.close')" @click="formOpen = false">
              <AppIcon name="close" :size="17" />
            </button>
          </div>
          <form class="stack" @submit.prevent="save" novalidate>
            <div class="form-grid">
              <div class="field" :class="{ 'is-invalid': invalidField === 'user-chat-id' }">
                <label for="user-chat-id">{{ t('users.form.chatId') }} *</label>
                <input
                  id="user-chat-id"
                  v-model="form.chat_id"
                  type="number"
                  :disabled="!!editing"
                  :aria-invalid="invalidField === 'user-chat-id' ? 'true' : undefined"
                  required
                  @input="clear('user-chat-id')"
                />
              </div>
              <div class="field">
                <label for="user-default-storage">{{ t('users.form.defaultStorage') }}</label>
                <AppSelect
                  id="user-default-storage"
                  v-model="form.default_storage"
                  :options="defaultStorageOptions"
                />
              </div>
            </div>
            <div class="row">
              <label class="checkbox"><input v-model="form.enabled" type="checkbox" /><span>{{ t('users.form.enabled') }}</span></label>
              <label class="checkbox"><input v-model="form.silent" type="checkbox" /><span>{{ t('users.form.silent') }}</span></label>
              <label class="checkbox"><input v-model="form.apply_rule" type="checkbox" /><span>{{ t('users.form.applyRule') }}</span></label>
              <label class="checkbox"><input v-model="form.watch_notify" type="checkbox" /><span>{{ t('users.form.watchNotify') }}</span></label>
            </div>
            <div class="field">
              <label>{{ t('users.form.storageFilter', { mode: form.blacklist ? t('users.filter.blacklist') : t('users.filter.whitelist') }) }}</label>
              <label class="checkbox"><input v-model="form.blacklist" type="checkbox" /><span>{{ t('users.form.blacklistMode') }}</span></label>
              <div class="checkbox-list">
                <label v-for="storage in enabledStorages" :key="storage.name">
                  <input type="checkbox" :checked="form.storages.includes(storage.name)" @change="toggleStorage(storage.name)" />
                  <span>{{ storage.name }}</span>
                </label>
              </div>
            </div>
            <div class="form-grid">
              <div class="field">
                <label for="user-fname">{{ t('users.form.filenameStrategy') }}</label>
                <AppSelect
                  id="user-fname"
                  v-model="form.filename_strategy"
                  :options="filenameStrategyOptions"
                  @update:model-value="clear('user-template')"
                />
              </div>
              <div class="field">
                <label for="user-conflict">{{ t('users.form.conflictStrategy') }}</label>
                <AppSelect
                  id="user-conflict"
                  v-model="form.conflict_strategy"
                  :options="conflictStrategyOptions"
                />
              </div>
            </div>
            <div class="field" :class="{ 'is-invalid': invalidField === 'user-template' }">
              <label for="user-template">{{ t('users.form.filenameTemplate') }}</label>
              <input
                id="user-template"
                v-model="form.filename_template"
                :placeholder="'{{.origname}}'"
                :aria-invalid="invalidField === 'user-template' ? 'true' : undefined"
                @input="clear('user-template')"
              />
            </div>
            <div class="modal-actions">
              <button class="btn ghost" type="button" @click="formOpen = false">{{ t('common.actions.cancel') }}</button>
              <LoadingButton class="primary" type="submit" :loading="busy">{{ t('common.actions.save') }}</LoadingButton>
            </div>
          </form>
        </div>
      </div>
      </Transition>
    </Teleport>
  </div>
</template>
