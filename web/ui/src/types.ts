export type JsonValue =
  | string
  | number
  | boolean
  | null
  | JsonValue[]
  | { [key: string]: JsonValue }

export type JsonObject = { [key: string]: JsonValue }

export type TaskStatus = 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface TaskProgress {
  total_bytes?: number
  downloaded_bytes?: number
  total_files?: number
  downloaded_files?: number
  percent?: number
  speed_mbps?: number
}

export interface TaskItem {
  task_id: string
  type: string
  title: string
  status: TaskStatus
  storage: string
  path: string
  source?: string
  error?: string
  created_at: string
  started_at?: string
  finished_at?: string
  updated_at?: string
  progress?: TaskProgress
}

export interface TaskListResponse {
  tasks: TaskItem[]
  total: number
}

export interface CreateTaskRequest {
  type: string
  storage: string
  path: string
  webhook?: string
  params: JsonObject
}

export interface TaskCreated {
  task_id: string
  type: string
  status: TaskStatus
  created_at: string
}

export interface TaskTypeInfo {
  type: string
  label: string
}

export interface StorageRecord {
  id: number
  name: string
  type: string
  enable: boolean
  config: JsonObject
  error?: string
}

export interface StorageInput {
  name: string
  type: string
  enable: boolean
  config: JsonObject
}

export interface StorageTestResult {
  ok: boolean
  error?: string
}

export interface UserStorageFilter {
  storages: string[]
  blacklist: boolean
}

export interface UserItem {
  id: number
  chat_id: number
  enabled: boolean
  silent: boolean
  default_storage: string
  default_dir: number
  apply_rule: boolean
  filename_strategy: string
  filename_template: string
  conflict_strategy: string
  watch_notify: boolean
  storage_filter: UserStorageFilter
}

export interface UserPatch {
  enabled?: boolean
  silent?: boolean
  default_storage?: string
  default_dir?: number
  apply_rule?: boolean
  filename_strategy?: string
  filename_template?: string
  conflict_strategy?: string
  watch_notify?: boolean
  storage_filter?: UserStorageFilter
}

export interface CreateUserInput extends UserPatch {
  chat_id: number
}

export interface BotStatus {
  state: string
  id?: number
  username?: string
  error?: string
  started_at?: string
}

export type LoginFlowState =
  | 'starting'
  | 'code_required'
  | 'password_required'
  | 'verifying'
  | 'failed'
  | 'done'
  | 'cancelled'

export interface LoginFlow {
  flow_id: string
  state: LoginFlowState
  error?: string
}

export interface UserbotStatus {
  enabled: boolean
  state: string
  id?: number
  username?: string
  phone?: string
  error?: string
  login?: LoginFlow
}

export interface TelegramStatus {
  bot: BotStatus
  userbot: UserbotStatus
}

export interface SettingsMap {
  [group: string]: JsonObject
}

export interface SettingsUpdateResult {
  restart_required: string[]
  warnings?: string[]
}

export interface SystemInfo {
  version: string
  commit: string
  build_time: string
  docker: boolean
  uptime_seconds: number
  data_dir: string
  db_path: string
  workers: number
  queued_tasks: number
  running_tasks: number
}

export interface MeResponse {
  authenticated: boolean
  csrf_token?: string
}

export interface FilesystemEntry {
  name: string
  path: string
  is_dir: boolean
  size?: number
}

export interface FilesystemListing {
  path: string
  parent?: string
  root: string
  cwd: string
  selected?: string
  entries: FilesystemEntry[]
  truncated?: boolean
}

export interface OnboardingSteps {
  bot_configured: boolean
  storage_ready: boolean
  user_ready: boolean
  bot_running: boolean
  first_task: boolean
}

export interface OnboardingLegacy {
  imported: boolean
  imported_at?: string
}

export interface OnboardingStatus {
  complete: boolean
  steps: OnboardingSteps
  bot_state: string
  legacy: OnboardingLegacy
}
