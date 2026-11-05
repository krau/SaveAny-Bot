import { translate } from './i18n'
import type {
  BotWorker,
  BotWorkerInput,
  BotWorkerTestResult,
  CreateTaskRequest,
  JsonObject,
  CreateUserInput,
  FilesystemListing,
  MeResponse,
  OnboardingStatus,
  SettingsMap,
  SettingsUpdateResult,
  StorageInput,
  StorageRecord,
  StorageTestResult,
  SystemInfo,
  TaskCreated,
  TaskItem,
  TaskListResponse,
  TaskMetadataResponse,
  TaskTypeInfo,
  TelegramStatus,
  UpdateInfo,
  UserDir,
  UserItem,
  UserPatch,
  ParserList,
  UserRule,
  UserWatch,
} from './types'

export interface ApiErrorBody {
  error?: string
  message?: string
  fields?: Record<string, string>
}

/** Sentinel returned by the API for a secret that is already stored. */
export const MASKED_SECRET = '__MASKED__'

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message || code)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

type Method = 'GET' | 'POST' | 'PUT' | 'DELETE'

interface RequestOptions {
  method?: Method
  body?: unknown
  skipAuthRedirect?: boolean
}

let csrfProvider: () => string = () => ''
let unauthorizedHandler: () => void = () => {}

export function configureApi(config: {
  getCsrf?: () => string
  onUnauthorized?: () => void
}): void {
  if (config.getCsrf) {
    csrfProvider = config.getCsrf
  }
  if (config.onUnauthorized) {
    unauthorizedHandler = config.onUnauthorized
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const method = options.method ?? 'GET'
  const headers: Record<string, string> = { Accept: 'application/json' }
  let body: string | undefined
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(options.body)
  }
  if (method !== 'GET') {
    const token = csrfProvider()
    if (token) {
      headers['X-CSRF-Token'] = token
    }
  }

  let response: Response
  try {
    response = await fetch(`/api/v1${path}`, {
      method,
      headers,
      body,
      credentials: 'same-origin',
    })
  } catch (err) {
    throw new ApiError(0, 'network_error', err instanceof Error ? err.message : translate('errors.network'))
  }

  if (response.status === 401 && !options.skipAuthRedirect) {
    unauthorizedHandler()
  }

  if (!response.ok) {
    let payload: ApiErrorBody = {}
    try {
      payload = (await response.json()) as ApiErrorBody
    } catch {
      payload = {}
    }
    throw new ApiError(
      response.status,
      payload.error ?? 'http_error',
      payload.message ?? translate('errors.requestFailed', { status: response.status }),
      payload.fields ?? {},
    )
  }

  if (response.status === 204) {
    return undefined as T
  }
  const text = await response.text()
  if (!text) {
    return undefined as T
  }
  return JSON.parse(text) as T
}

function query(params: Record<string, string | number | undefined | null>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      search.set(key, String(value))
    }
  }
  const encoded = search.toString()
  return encoded ? `?${encoded}` : ''
}

export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    const fieldMessages = Object.entries(err.fields).map(([key, value]) => `${key}: ${value}`)
    if (fieldMessages.length > 0) {
      return `${err.message}（${fieldMessages.join('；')}）`
    }
    return err.message
  }
  if (err instanceof Error) {
    return err.message
  }
  return String(err)
}

export const authApi = {
  login(password: string) {
    return request<{ csrf_token: string }>('/auth/login', {
      method: 'POST',
      body: { password },
      skipAuthRedirect: true,
    })
  },
  logout() {
    return request<void>('/auth/logout', { method: 'POST', skipAuthRedirect: true })
  },
  me() {
    return request<MeResponse>('/auth/me', { skipAuthRedirect: true })
  },
  changePassword(currentPassword: string, newPassword: string) {
    return request<{ ok: boolean }>('/auth/password', {
      method: 'POST',
      body: { current_password: currentPassword, new_password: newPassword },
    })
  },
}

export const filesystemApi = {
  list(path?: string) {
    return request<FilesystemListing>(`/filesystem${query({ path })}`)
  },
}

export const onboardingApi = {
  status() {
    return request<OnboardingStatus>('/onboarding')
  },
}

export const settingsApi = {
  getAll() {
    return request<SettingsMap>('/settings')
  },
  updateGroup(group: string, patch: JsonObject) {
    return request<SettingsUpdateResult>(`/settings/${group}`, { method: 'PUT', body: patch })
  },
}

export const storagesApi = {
  async list(): Promise<StorageRecord[]> {
    const data = await request<{ storages: StorageRecord[] }>('/storages')
    return data.storages ?? []
  },
  create(input: StorageInput) {
    return request<StorageRecord>('/storages', { method: 'POST', body: input })
  },
  update(id: number, input: StorageInput) {
    return request<StorageRecord>(`/storages/${id}`, { method: 'PUT', body: input })
  },
  remove(id: number) {
    return request<void>(`/storages/${id}`, { method: 'DELETE' })
  },
  test(id: number) {
    return request<StorageTestResult>(`/storages/${id}/test`, { method: 'POST', body: {} })
  },
}

export const usersApi = {
  async list(): Promise<UserItem[]> {
    const data = await request<{ users: UserItem[] }>('/users')
    return data.users ?? []
  },
  create(input: CreateUserInput) {
    return request<UserItem>('/users', { method: 'POST', body: input })
  },
  update(id: number, patch: UserPatch) {
    return request<UserItem>(`/users/${id}`, { method: 'PUT', body: patch })
  },
  remove(id: number) {
    return request<void>(`/users/${id}`, { method: 'DELETE' })
  },
}

export const dirsApi = {
  async list(userId: number): Promise<UserDir[]> {
    const data = await request<{ dirs: UserDir[] }>(`/users/${userId}/dirs`)
    return data.dirs ?? []
  },
  create(userId: number, input: { storage_id: number; path: string }) {
    return request<UserDir>(`/users/${userId}/dirs`, { method: 'POST', body: input })
  },
  remove(userId: number, dirId: number) {
    return request<void>(`/users/${userId}/dirs/${dirId}`, { method: 'DELETE' })
  },
}

export const rulesApi = {
  async list(userId: number): Promise<UserRule[]> {
    const data = await request<{ rules: UserRule[] }>(`/users/${userId}/rules`)
    return data.rules ?? []
  },
  create(userId: number, input: { type: string; content: string; storage_id: number; dir_path: string }) {
    return request<UserRule>(`/users/${userId}/rules`, { method: 'POST', body: input })
  },
  remove(userId: number, ruleId: number) {
    return request<void>(`/users/${userId}/rules/${ruleId}`, { method: 'DELETE' })
  },
  importPresets(userId: number, input: { storage_id: number; base_path: string }) {
    return request<{ created: number }>(`/users/${userId}/rules/presets`, { method: 'POST', body: input })
  },
}

export const watchesApi = {
  async list(userId: number): Promise<UserWatch[]> {
    const data = await request<{ watches: UserWatch[] }>(`/users/${userId}/watches`)
    return data.watches ?? []
  },
  create(userId: number, input: { chat?: string; chat_id?: number; filter: string }) {
    return request<UserWatch[]>(`/users/${userId}/watches`, { method: 'POST', body: input })
  },
  remove(userId: number, chatId: number) {
    return request<void>(`/users/${userId}/watches/${chatId}`, { method: 'DELETE' })
  },
}

export const parsersApi = {
  list() {
    return request<ParserList>('/parsers')
  },
  create(input: { filename: string; content: string }) {
    return request<{ name: string }>('/parsers', { method: 'POST', body: input })
  },
}

export const telegramApi = {
  status() {
    return request<TelegramStatus>('/telegram/status')
  },
  startBot() {
    return request<{ state: string }>('/telegram/bot/start', { method: 'POST', body: {} })
  },
  stopBot(force = false) {
    return request<void>('/telegram/bot/stop', { method: 'POST', body: { force } })
  },
  restartBot(force = false) {
    return request<{ state: string }>('/telegram/bot/restart', { method: 'POST', body: { force } })
  },
  login(phone: string) {
    return request<{ flow_id: string }>('/telegram/userbot/login', {
      method: 'POST',
      body: { phone },
    })
  },
  submitCode(flowId: string, code: string) {
    return request<{ ok: boolean }>(`/telegram/userbot/login/${flowId}/code`, {
      method: 'POST',
      body: { code },
    })
  },
  submitPassword(flowId: string, password: string) {
    return request<{ ok: boolean }>(`/telegram/userbot/login/${flowId}/password`, {
      method: 'POST',
      body: { password },
    })
  },
  cancelLogin(flowId: string) {
    return request<void>(`/telegram/userbot/login/${flowId}`, { method: 'DELETE' })
  },
  logout() {
    return request<void>('/telegram/userbot/logout', { method: 'POST', body: {} })
  },
  syncPeers() {
    return request<{ synced: number }>('/telegram/userbot/sync-peers', { method: 'POST', body: {} })
  },
}

export const workersApi = {
  async list(): Promise<BotWorker[]> {
    const data = await request<{ workers: BotWorker[] }>('/telegram/workers')
    return data.workers ?? []
  },
  create(input: BotWorkerInput) {
    return request<BotWorker>('/telegram/workers', { method: 'POST', body: input })
  },
  update(id: number, input: BotWorkerInput) {
    return request<BotWorker>(`/telegram/workers/${id}`, { method: 'PUT', body: input })
  },
  remove(id: number) {
    return request<void>(`/telegram/workers/${id}`, { method: 'DELETE' })
  },
  start(id: number) {
    return request<{ state: string }>(`/telegram/workers/${id}/start`, { method: 'POST', body: {} })
  },
  stop(id: number) {
    return request<{ state: string }>(`/telegram/workers/${id}/stop`, { method: 'POST', body: {} })
  },
  test(id: number) {
    return request<BotWorkerTestResult>(`/telegram/workers/${id}/test`, { method: 'POST', body: {} })
  },
}

export const tasksApi = {
  list(params: { status?: string; type?: string; limit?: number; offset?: number }) {
    return request<TaskListResponse>(`/tasks${query(params)}`)
  },
  get(taskId: string) {
    return request<TaskItem>(`/tasks/${encodeURIComponent(taskId)}`)
  },
  metadata(taskId: string) {
    return request<TaskMetadataResponse>(`/tasks/${encodeURIComponent(taskId)}/metadata`)
  },
  create(input: CreateTaskRequest) {
    return request<TaskCreated>('/tasks', { method: 'POST', body: input })
  },
  retry(taskId: string) {
    return request<{ task_id: string; status: string; file_count: number }>(
      `/tasks/${encodeURIComponent(taskId)}/retry`,
      { method: 'POST', body: {} },
    )
  },
  cancel(taskId: string) {
    return request<{ task_id: string; status: string }>(
      `/tasks/${encodeURIComponent(taskId)}/cancel`,
      { method: 'POST', body: {} },
    )
  },
  remove(taskId: string) {
    return request<void>(`/tasks/${encodeURIComponent(taskId)}`, { method: 'DELETE' })
  },
  async types(): Promise<TaskTypeInfo[]> {
    const data = await request<{ types: TaskTypeInfo[] }>('/task-types')
    return data.types ?? []
  },
}

export const systemApi = {
  info() {
    return request<SystemInfo>('/system/info')
  },
  restart(force = false) {
    return request<{ restarting: boolean }>('/system/restart', {
      method: 'POST',
      body: { force },
    })
  },
  checkUpdate() {
    return request<UpdateInfo>('/system/update')
  },
  applyUpdate() {
    return request<{ version: string; restarting: boolean }>('/system/update', {
      method: 'POST',
      body: {},
    })
  },
}
