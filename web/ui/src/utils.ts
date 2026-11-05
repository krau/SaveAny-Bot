import { translate as t } from './i18n'

export function formatDateTime(value?: string | null): string {
  if (!value) {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

export function taskDisplayTitle(task: { title?: string; type?: string }): string {
  const raw = (task.title ?? '').trim()
  const markdown = raw.match(/^\[[^\]]+\]\((.*)\)$/)
  const inner = (markdown?.[1] ?? raw).trim()
  if (!inner) {
    return t('time.unnamedTask')
  }
  const transfer = inner.match(/^(\d+)\s+files\/([\d.]+)MB$/)
  if (transfer) {
    return t('time.files', { n: transfer[1], size: transfer[2] })
  }
  const arrow = inner.indexOf('->')
  const display = (arrow >= 0 ? inner.slice(0, arrow) : inner).replace(/\.\.\.$/, '').trim()
  if (display) {
    return display
  }
  return task.type ? taskTypeLabel(task.type) : t('time.unnamedTask')
}

export function storageLocation(config: Record<string, unknown>): string {
  const keys = ['url', 'endpoint', 'remote', 'chat_id', 'base_path', 'path', 'bucket_name']
  for (const key of keys) {
    const value = config[key]
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      return `${key} ${String(value)}`
    }
  }
  return t('time.locationUnset')
}

export function formatRelativeTime(value?: string | null): string {
  if (!value) {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  const seconds = Math.max(0, Math.floor((Date.now() - date.getTime()) / 1000))
  if (seconds < 60) return t('time.justNow')
  if (seconds < 3600) return t('time.minutesAgo', { n: Math.floor(seconds / 60) })
  if (seconds < 86400) return t('time.hoursAgo', { n: Math.floor(seconds / 3600) })
  if (seconds < 172800) return t('time.yesterday')
  if (seconds < 604800) return t('time.daysAgo', { n: Math.floor(seconds / 86400) })
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

export function formatUptime(seconds?: number): string {
  if (!seconds || seconds < 0) {
    return '—'
  }
  const total = Math.floor(seconds)
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60
  const parts: string[] = []
  if (days > 0) parts.push(t('time.days', { n: days }))
  if (hours > 0) parts.push(t('time.hours', { n: hours }))
  if (minutes > 0) parts.push(t('time.minutes', { n: minutes }))
  if (parts.length === 0 || secs > 0) parts.push(t('time.seconds', { n: secs }))
  return parts.join(' ')
}

export function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value.toFixed(unit === 0 ? 0 : 2)} ${units[unit]}`
}

const taskStatusLabels: Record<string, string> = {
  queued: 'states.queued',
  running: 'states.running',
  completed: 'states.completed',
  partial: 'states.partial',
  failed: 'states.failed',
  cancelled: 'states.cancelled',
}

export function taskStatusLabel(status: string): string {
  const key = taskStatusLabels[status]
  return key ? t(key) : status
}

export function taskStatusTone(status: string): string {
  switch (status) {
    case 'running':
      return 'info'
    case 'completed':
      return 'success'
    case 'partial':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'cancelled':
      return 'muted'
    default:
      return 'warning'
  }
}

const telegramStateLabels: Record<string, string> = {
  stopped: 'states.stopped',
  starting: 'states.starting',
  running: 'states.running',
  error: 'states.error',
  login_required: 'states.loginRequired',
}

export function telegramStateLabel(state?: string): string {
  if (!state) {
    return t('states.unknown')
  }
  const key = telegramStateLabels[state]
  return key ? t(key) : state
}

export function telegramStateTone(state?: string): string {
  switch (state) {
    case 'running':
      return 'success'
    case 'starting':
      return 'warning'
    case 'error':
      return 'danger'
    case 'login_required':
      return 'warning'
    default:
      return 'muted'
  }
}

const taskTypeKeys: Record<string, string> = {
  directlinks: 'taskTypes.directlinks',
  ytdlp: 'taskTypes.ytdlp',
  aria2: 'taskTypes.aria2',
  parseditem: 'taskTypes.parseditem',
  tgfiles: 'taskTypes.tgfiles',
  tphpics: 'taskTypes.tphpics',
  transfer: 'taskTypes.transfer',
}

export function taskTypeLabel(type: string): string {
  const key = taskTypeKeys[type]
  return key ? t(key) : type
}
