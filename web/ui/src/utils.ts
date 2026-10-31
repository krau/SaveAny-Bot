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
    return '未命名任务'
  }
  const transfer = inner.match(/^(\d+)\s+files\/([\d.]+)MB$/)
  if (transfer) {
    return `${transfer[1]} 个文件，${transfer[2]} MB`
  }
  const arrow = inner.indexOf('->')
  const display = (arrow >= 0 ? inner.slice(0, arrow) : inner).replace(/\.\.\.$/, '').trim()
  if (display) {
    return display
  }
  return task.type ? taskTypeLabel(task.type) : '未命名任务'
}

export function storageLocation(config: Record<string, unknown>): string {
  const keys = ['url', 'endpoint', 'remote', 'chat_id', 'base_path', 'path', 'bucket_name']
  for (const key of keys) {
    const value = config[key]
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      return `${key} ${String(value)}`
    }
  }
  return '未填写位置'
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
  if (seconds < 60) return '刚刚'
  if (seconds < 3600) return `${Math.floor(seconds / 60)} 分钟前`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} 小时前`
  if (seconds < 172800) return '昨天'
  if (seconds < 604800) return `${Math.floor(seconds / 86400)} 天前`
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
  if (days > 0) parts.push(`${days} 天`)
  if (hours > 0) parts.push(`${hours} 小时`)
  if (minutes > 0) parts.push(`${minutes} 分`)
  if (parts.length === 0 || secs > 0) parts.push(`${secs} 秒`)
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
  queued: '排队中',
  running: '进行中',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
}

export function taskStatusLabel(status: string): string {
  return taskStatusLabels[status] ?? status
}

export function taskStatusTone(status: string): string {
  switch (status) {
    case 'running':
      return 'info'
    case 'completed':
      return 'success'
    case 'failed':
      return 'danger'
    case 'cancelled':
      return 'muted'
    default:
      return 'warning'
  }
}

const telegramStateLabels: Record<string, string> = {
  stopped: '已停止',
  starting: '启动中',
  running: '运行中',
  error: '错误',
  login_required: '需要登录',
}

export function telegramStateLabel(state?: string): string {
  if (!state) {
    return '未知'
  }
  return telegramStateLabels[state] ?? state
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

export function taskTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    directlinks: '直链下载',
    ytdlp: 'yt-dlp',
    aria2: 'Aria2',
    parseditem: '解析链接',
    tgfiles: 'Telegram 文件',
    tphpics: 'Telegraph 图集',
    transfer: '存储迁移',
  }
  return labels[type] ?? type
}
