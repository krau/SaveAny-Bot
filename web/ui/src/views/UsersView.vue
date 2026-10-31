<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { errorMessage, storagesApi, usersApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import AppSelect from '../components/AppSelect.vue'
import PageHeader from '../components/PageHeader.vue'
import { useFormFeedback } from '../composables/useFormFeedback'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import type { StorageRecord, UserItem, UserPatch, UserStorageFilter } from '../types'

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
  { value: '', label: '未设置' },
  ...enabledStorages.value.map((storage) => ({ value: storage.name, label: storage.name })),
])
const filenameStrategyOptions = [
  { value: 'default', label: '默认' },
  { value: 'message', label: '从消息生成' },
  { value: 'template', label: '自定义模板' },
]
const conflictStrategyOptions = [
  { value: '', label: '默认' },
  { value: 'rename', label: '始终重命名' },
  { value: 'ask', label: '每次询问' },
  { value: 'overwrite', label: '始终覆盖' },
  { value: 'skip', label: '始终跳过' },
]

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
  busy.value = true
  clear()
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
      success('用户已更新')
    } else {
      const chatID = Number(form.chat_id)
      if (!Number.isFinite(chatID) || chatID === 0) {
        fail('请填写有效的 Telegram Chat ID', 'user-chat-id')
        return
      }
      await usersApi.create({ chat_id: chatID, ...patch })
      success('用户已创建')
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
    title: '删除用户',
    message: `删除用户 ${user.chat_id} 及其目录、规则？此操作不可撤销。`,
    confirmText: '删除用户',
    danger: true,
  })
  if (!confirmed) return
  try {
    await usersApi.remove(user.id)
    success(`${user.chat_id} 已删除`)
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  }
}
</script>

<template>
  <div class="stack">
    <PageHeader
      title="用户"
      description="允许使用机器人的 Telegram 账号，以及它们能访问的存储范围。"
    >
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          刷新
        </button>
        <button class="btn primary small" type="button" @click="openCreate">
          <AppIcon name="plus" :size="15" />
          添加用户
        </button>
      </template>
    </PageHeader>

    <div class="filter-bar">
      <span class="count">{{ users.length }} 个用户，{{ enabledUsers }} 个启用</span>
      <div class="filter-spacer"></div>
      <span class="count">访问范围以白名单或黑名单控制</span>
    </div>

    <div class="users-table">
      <div class="table-wrap users-table-wrap">
        <table>
        <thead><tr><th>Chat ID</th><th>状态</th><th>默认存储</th><th>存储过滤</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td class="mono">{{ user.chat_id }}</td>
            <td>
              <span class="badge status" :class="user.enabled ? 'success' : 'muted'">
                {{ user.enabled ? '启用' : '禁用' }}
              </span>
              <span v-if="user.silent" class="badge info" style="margin-left: 6px">静默</span>
            </td>
            <td>{{ user.default_storage || '—' }}</td>
            <td class="muted">
              {{ user.storage_filter.blacklist ? '黑名单' : '白名单' }}：
              {{ user.storage_filter.storages.length ? user.storage_filter.storages.join('、') : (user.storage_filter.blacklist ? '全部' : '无') }}
            </td>
            <td>
              <div class="actions">
                <button class="btn small" type="button" @click="openEdit(user)">编辑</button>
                <button class="btn small danger" type="button" @click="remove(user)">删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="users.length === 0">
            <td colspan="5" class="empty">
              <span class="empty-icon"><AppIcon name="users" :size="17" /></span>
              <strong>还没有用户</strong>
              未添加到列表的账号会被机器人拒绝
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
                {{ user.enabled ? '启用' : '禁用' }}
              </span>
              <span v-if="user.silent" class="badge info">静默</span>
            </div>
          </div>
          <dl class="user-mobile-details">
            <div>
              <dt>默认存储</dt>
              <dd>{{ user.default_storage || '未设置' }}</dd>
            </div>
            <div>
              <dt>存储过滤</dt>
              <dd>
                {{ user.storage_filter.blacklist ? '黑名单' : '白名单' }}：{{
                  user.storage_filter.storages.length
                    ? user.storage_filter.storages.join('、')
                    : user.storage_filter.blacklist
                      ? '全部'
                      : '无'
                }}
              </dd>
            </div>
          </dl>
          <div class="user-mobile-actions">
            <button class="btn small" type="button" @click="openEdit(user)">编辑</button>
            <button class="btn small danger" type="button" @click="remove(user)">删除</button>
          </div>
        </article>
        <div v-if="users.length === 0" class="empty">
          <span class="empty-icon"><AppIcon name="users" :size="17" /></span>
          <strong>还没有用户</strong>
          未添加到列表的账号会被机器人拒绝
        </div>
      </div>
    </div>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="formOpen" class="modal-overlay" @click.self="formOpen = false">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="user-dialog-title">
          <div class="modal-head">
            <h3 id="user-dialog-title" class="modal-title">{{ editing ? '编辑用户' : '添加用户' }}</h3>
            <button class="btn ghost icon small" type="button" aria-label="关闭" @click="formOpen = false">
              <AppIcon name="close" :size="17" />
            </button>
          </div>
          <form class="stack" @submit.prevent="save" novalidate>
            <div class="form-grid">
              <div class="field" :class="{ 'is-invalid': invalidField === 'user-chat-id' }">
                <label for="user-chat-id">Telegram Chat ID</label>
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
                <label for="user-default-storage">默认存储</label>
                <AppSelect
                  id="user-default-storage"
                  v-model="form.default_storage"
                  :options="defaultStorageOptions"
                />
              </div>
            </div>
            <div class="row">
              <label class="checkbox"><input v-model="form.enabled" type="checkbox" /><span>启用用户</span></label>
              <label class="checkbox"><input v-model="form.silent" type="checkbox" /><span>静默模式</span></label>
              <label class="checkbox"><input v-model="form.apply_rule" type="checkbox" /><span>应用规则</span></label>
              <label class="checkbox"><input v-model="form.watch_notify" type="checkbox" /><span>监听通知</span></label>
            </div>
            <div class="field">
              <label>存储过滤（{{ form.blacklist ? '黑名单' : '白名单' }}）</label>
              <label class="checkbox"><input v-model="form.blacklist" type="checkbox" /><span>黑名单模式</span></label>
              <div class="checkbox-list">
                <label v-for="storage in enabledStorages" :key="storage.name">
                  <input type="checkbox" :checked="form.storages.includes(storage.name)" @change="toggleStorage(storage.name)" />
                  <span>{{ storage.name }}</span>
                </label>
              </div>
            </div>
            <div class="form-grid">
              <div class="field">
                <label for="user-fname">文件名策略</label>
                <AppSelect
                  id="user-fname"
                  v-model="form.filename_strategy"
                  :options="filenameStrategyOptions"
                />
              </div>
              <div class="field">
                <label for="user-conflict">重名策略</label>
                <AppSelect
                  id="user-conflict"
                  v-model="form.conflict_strategy"
                  :options="conflictStrategyOptions"
                />
              </div>
            </div>
            <div class="field">
              <label for="user-template">文件名模板</label>
              <input id="user-template" v-model="form.filename_template" :placeholder="'{{.origname}}'" />
            </div>
            <div class="modal-actions">
              <button class="btn ghost" type="button" @click="formOpen = false">取消</button>
              <LoadingButton class="primary" type="submit" :loading="busy">保存</LoadingButton>
            </div>
          </form>
        </div>
      </div>
      </Transition>
    </Teleport>
  </div>
</template>
