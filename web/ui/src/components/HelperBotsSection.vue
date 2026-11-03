<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage, workersApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageLoading from '../components/PageLoading.vue'
import { useFormFeedback } from '../composables/useFormFeedback'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import type { BotWorker, BotWorkerInput } from '../types'

const { t } = useI18n()
const toast = useToastStore()
const confirm = useConfirmStore()
const workers = ref<BotWorker[]>([])
const loaded = ref(false)
const formOpen = ref(false)
const editing = ref<BotWorker | null>(null)
const busy = ref(false)
const actionId = ref(0)
const action = ref<'start' | 'stop' | 'test' | ''>('')
const form = ref<BotWorkerInput>({ name: '', token: '', enabled: true })
const { invalidField, fail, clear } = useFormFeedback()
let poll: number | undefined
let loadSeq = 0

const hasStarting = computed(() => workers.value.some((worker) => worker.state === 'starting'))

async function load() {
  const seq = ++loadSeq
  try {
    const next = await workersApi.list()
    if (seq !== loadSeq) return
    workers.value = next
  } catch (err) {
    if (seq !== loadSeq) return
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    if (seq === loadSeq) loaded.value = true
  }
}

onMounted(() => {
  void load()
  poll = window.setInterval(() => {
    if (hasStarting.value) void load()
  }, 2000)
})

onUnmounted(() => {
  loadSeq++
  if (poll !== undefined) window.clearInterval(poll)
})

function openCreate() {
  clear()
  editing.value = null
  form.value = { name: '', token: '', enabled: true }
  formOpen.value = true
}

function openEdit(worker: BotWorker) {
  clear()
  editing.value = worker
  form.value = { name: worker.name, token: '', enabled: worker.enabled }
  formOpen.value = true
}

async function save() {
  clear()
  if (!form.value.name.trim()) {
    fail(t('validation.required', { field: t('workers.name') }), 'worker-name')
    return
  }
  if (!editing.value && !form.value.token.trim()) {
    fail(t('validation.required', { field: t('workers.token') }), 'worker-token')
    return
  }
  busy.value = true
  try {
    if (editing.value) {
      await workersApi.update(editing.value.id, form.value)
      toast.push(t('workers.updated'), 'success')
    } else {
      await workersApi.create(form.value)
      toast.push(t('workers.created'), 'success')
    }
    formOpen.value = false
    await load()
  } catch (err) {
    fail(errorMessage(err))
  } finally {
    busy.value = false
  }
}

async function act(worker: BotWorker, next: 'start' | 'stop' | 'test') {
  actionId.value = worker.id
  action.value = next
  try {
    if (next === 'start') {
      await workersApi.start(worker.id)
    } else if (next === 'stop') {
      await workersApi.stop(worker.id)
    } else {
      const result = await workersApi.test(worker.id)
      if (result.ok) {
        toast.push(t('workers.testOk', { name: worker.name }), 'success')
      } else {
        toast.push(t('workers.testFailed', { name: worker.name, error: result.error ?? t('states.unknown') }), 'error', 6000)
      }
    }
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    actionId.value = 0
    action.value = ''
  }
}

async function remove(worker: BotWorker) {
  const confirmed = await confirm.ask({
    title: t('workers.deleteTitle'),
    message: t('workers.deleteMessage', { name: worker.name }),
    confirmText: t('workers.deleteTitle'),
    danger: true,
  })
  if (!confirmed) return
  actionId.value = worker.id
  try {
    await workersApi.remove(worker.id)
    toast.push(t('workers.deleted'), 'success')
    await load()
  } catch (err) {
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    actionId.value = 0
  }
}

function stateLabel(state: string): string {
  const key = `states.${state}`
  const label = t(key)
  return label === key ? state : label
}

function stateTone(state: string): string {
  if (state === 'running') return 'success'
  if (state === 'error') return 'danger'
  if (state === 'starting') return 'warning'
  return 'muted'
}
</script>

<template>
  <section class="panel telegram-panel helper-bots-panel">
    <div class="panel-head">
      <h2 class="panel-title">{{ t('workers.title') }}</h2>
      <div class="actions">
        <span v-if="loaded" class="badge muted">{{ t('workers.count', { count: workers.length }) }}</span>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          {{ t('common.actions.refresh') }}
        </button>
        <button class="btn primary small" type="button" @click="openCreate">
          <AppIcon name="plus" :size="15" />
          {{ t('workers.create') }}
        </button>
      </div>
    </div>
    <p class="hint">{{ t('workers.intro') }}</p>
    <p class="hint">{{ t('workers.note') }}</p>

    <PageLoading v-if="!loaded" :rows="3" />

    <template v-else>
    <div class="table-wrap helper-bots-table-wrap">
        <table class="users-table">
          <thead>
            <tr>
              <th>{{ t('workers.name') }}</th>
              <th>{{ t('workers.state') }}</th>
              <th>{{ t('workers.username') }}</th>
              <th>{{ t('workers.enabled') }}</th>
              <th>{{ t('workers.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="worker in workers" :key="worker.id">
              <td>
                <strong>{{ worker.name }}</strong>
                <div v-if="worker.error" class="field-error">{{ worker.error }}</div>
              </td>
              <td>
                <span class="badge status" :class="stateTone(worker.state)">{{ stateLabel(worker.state) }}</span>
              </td>
              <td class="mono">{{ worker.username ? `@${worker.username}` : '—' }}</td>
              <td>
                <span class="badge" :class="worker.enabled ? 'success' : 'muted'">
                  {{ worker.enabled ? t('states.enabled') : t('states.disabled') }}
                </span>
              </td>
              <td>
                <div class="actions">
                  <LoadingButton
                    v-if="worker.state === 'running' || worker.state === 'starting'"
                    class="small"
                    type="button"
                    :loading="actionId === worker.id && action === 'stop'"
                    @click="act(worker, 'stop')"
                  >
                    {{ t('workers.stop') }}
                  </LoadingButton>
                  <LoadingButton
                    v-else
                    class="small"
                    type="button"
                    :loading="actionId === worker.id && action === 'start'"
                    :disabled="!worker.enabled"
                    @click="act(worker, 'start')"
                  >
                    {{ t('workers.start') }}
                  </LoadingButton>
                  <button class="btn small" type="button" @click="openEdit(worker)">
                    {{ t('common.actions.edit') }}
                  </button>
                  <button
                    class="btn small"
                    type="button"
                    :disabled="worker.state !== 'running' || (actionId === worker.id && action !== '')"
                    @click="act(worker, 'test')"
                  >
                    {{ t('workers.test') }}
                  </button>
                  <button class="btn small danger" type="button" :disabled="actionId === worker.id" @click="remove(worker)">
                    {{ t('common.actions.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="workers.length === 0">
              <td colspan="5" class="empty">
                <strong>{{ t('workers.empty') }}</strong>
                <div class="hint">{{ t('workers.emptyHint') }}</div>
              </td>
            </tr>
          </tbody>
        </table>
    </div>

    <div class="helper-bots-mobile">
      <article v-for="worker in workers" :key="worker.id" class="helper-bot-card">
        <div class="helper-bot-head">
          <div class="helper-bot-identity">
            <strong>{{ worker.name }}</strong>
            <span class="mono">{{ worker.username ? `@${worker.username}` : '—' }}</span>
          </div>
          <span class="badge status" :class="stateTone(worker.state)">{{ stateLabel(worker.state) }}</span>
        </div>
        <div class="helper-bot-meta">
          <span class="badge" :class="worker.enabled ? 'success' : 'muted'">
            {{ worker.enabled ? t('states.enabled') : t('states.disabled') }}
          </span>
        </div>
        <div v-if="worker.error" class="field-error">{{ worker.error }}</div>
        <div class="helper-bot-actions">
          <LoadingButton
            v-if="worker.state === 'running' || worker.state === 'starting'"
            class="small"
            type="button"
            :loading="actionId === worker.id && action === 'stop'"
            @click="act(worker, 'stop')"
          >
            {{ t('workers.stop') }}
          </LoadingButton>
          <LoadingButton
            v-else
            class="small"
            type="button"
            :loading="actionId === worker.id && action === 'start'"
            :disabled="!worker.enabled"
            @click="act(worker, 'start')"
          >
            {{ t('workers.start') }}
          </LoadingButton>
          <button class="btn small" type="button" @click="openEdit(worker)">
            {{ t('common.actions.edit') }}
          </button>
          <button
            class="btn small"
            type="button"
            :disabled="worker.state !== 'running' || (actionId === worker.id && action !== '')"
            @click="act(worker, 'test')"
          >
            {{ t('workers.test') }}
          </button>
          <button class="btn small danger" type="button" :disabled="actionId === worker.id" @click="remove(worker)">
            {{ t('common.actions.delete') }}
          </button>
        </div>
      </article>
      <div v-if="workers.length === 0" class="empty helper-bots-empty">
        <strong>{{ t('workers.empty') }}</strong>
        <div class="hint">{{ t('workers.emptyHint') }}</div>
      </div>
    </div>
    </template>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="formOpen" class="modal-overlay" @click.self="formOpen = false">
          <div class="modal" role="dialog" aria-modal="true" aria-labelledby="worker-dialog-title">
            <div class="modal-head">
              <h3 id="worker-dialog-title" class="modal-title">
                {{ editing ? t('workers.edit') : t('workers.create') }}
              </h3>
              <button class="btn ghost icon small" type="button" :aria-label="t('common.close')" @click="formOpen = false">
                <AppIcon name="close" :size="17" />
              </button>
            </div>
            <form class="stack" @submit.prevent="save" novalidate>
              <div class="field" :class="{ 'is-invalid': invalidField === 'worker-name' }">
                <label for="worker-name">{{ t('workers.name') }} *</label>
                <input
                  id="worker-name"
                  v-model="form.name"
                  maxlength="64"
                  required
                  :aria-invalid="invalidField === 'worker-name' ? 'true' : undefined"
                  @input="clear('worker-name')"
                />
              </div>
              <div class="field" :class="{ 'is-invalid': invalidField === 'worker-token' }">
                <label for="worker-token">{{ t('workers.token') }}{{ editing ? '' : ' *' }}</label>
                <input
                  id="worker-token"
                  v-model="form.token"
                  type="password"
                  autocomplete="off"
                  :placeholder="editing ? t('workers.tokenHint') : t('workers.tokenPlaceholder')"
                  :aria-invalid="invalidField === 'worker-token' ? 'true' : undefined"
                  @input="clear('worker-token')"
                />
                <span class="hint">{{ editing ? t('workers.tokenHint') : t('workers.tokenCreateHint') }}</span>
              </div>
              <label class="checkbox">
                <input v-model="form.enabled" type="checkbox" />
                <span>{{ t('workers.enabledHint') }}</span>
              </label>
              <div class="modal-actions">
                <button class="btn" type="button" @click="formOpen = false">{{ t('common.actions.cancel') }}</button>
                <LoadingButton class="primary" type="submit" :loading="busy">{{ t('common.actions.save') }}</LoadingButton>
              </div>
            </form>
          </div>
        </div>
      </Transition>
    </Teleport>
  </section>
</template>
