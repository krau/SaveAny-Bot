export type SettingsFieldKind = 'text' | 'number' | 'bool' | 'password' | 'select' | 'list' | 'json'

export interface SettingsField {
  path: string
  label: string
  kind: SettingsFieldKind
  placeholder?: string
  hint?: string
  min?: number
  options?: { value: string; label: string }[]
}

export interface SettingsGroupDef {
  key: string
  title: string
  fields: SettingsField[]
}

export const settingsGroups: SettingsGroupDef[] = [
  {
    key: 'core',
    title: '核心',
    fields: [
      {
        path: 'lang',
        label: '界面语言',
        kind: 'select',
        options: [
          { value: 'zh-Hans', label: '简体中文' },
          { value: 'en', label: 'English' },
        ],
      },
      { path: 'workers', label: 'Worker 数量', kind: 'number', hint: '重启后生效', min: 1 },
      { path: 'retry', label: '重试次数', kind: 'number', min: 0 },
      { path: 'threads', label: '下载线程数', kind: 'number', min: 1 },
      { path: 'stream', label: '流式下载', kind: 'bool' },
      { path: 'no_clean_cache', label: '保留缓存', kind: 'bool' },
      {
        path: 'proxy',
        label: '全局代理',
        kind: 'text',
        placeholder: 'socks5://127.0.0.1:1080',
        hint: '重启后生效',
      },
    ],
  },
  {
    key: 'log',
    title: '日志',
    fields: [
      {
        path: 'level',
        label: '日志级别',
        kind: 'select',
        options: [
          { value: 'debug', label: 'debug' },
          { value: 'info', label: 'info' },
          { value: 'warn', label: 'warn' },
          { value: 'error', label: 'error' },
        ],
      },
    ],
  },
  {
    key: 'aria2',
    title: 'Aria2',
    fields: [
      { path: 'enable', label: '启用 Aria2', kind: 'bool' },
      {
        path: 'url',
        label: 'RPC 地址',
        kind: 'text',
        placeholder: 'http://127.0.0.1:6800/jsonrpc',
      },
      { path: 'secret', label: 'RPC 密钥', kind: 'password' },
      { path: 'keep_file', label: '保留临时文件', kind: 'bool' },
    ],
  },
  {
    key: 'ytdlp',
    title: 'yt-dlp',
    fields: [
      { path: 'max_height', label: '最大高度', kind: 'number', hint: '0 表示不限制', min: 0 },
      { path: 'format', label: '格式选择', kind: 'text', placeholder: 'bestvideo+bestaudio/best' },
      { path: 'recode', label: '转码容器', kind: 'text', placeholder: 'mp4' },
      {
        path: 'filename_template',
        label: '文件名模板',
        kind: 'text',
        placeholder: '%(title)s.%(ext)s',
      },
      { path: 'restrict_filenames', label: '限制文件名（仅 ASCII）', kind: 'bool' },
    ],
  },
  {
    key: 'parser',
    title: '解析器',
    fields: [
      { path: 'plugin_enable', label: '启用插件', kind: 'bool', hint: '重启后生效' },
      {
        path: 'plugin_dirs',
        label: '插件目录',
        kind: 'list',
        hint: '每行一个目录',
        placeholder: 'plugins',
      },
      { path: 'proxy', label: '解析代理', kind: 'text', placeholder: 'http://127.0.0.1:7890' },
      {
        path: 'configs',
        label: '插件配置 (JSON)',
        kind: 'json',
        placeholder: '{"example": {"key": "value"}}',
        hint: '按插件名组织的 JSON 对象',
      },
    ],
  },
  {
    key: 'cache',
    title: '缓存',
    fields: [
      { path: 'ttl', label: 'TTL（秒）', kind: 'number', min: 0 },
      { path: 'num_counters', label: '计数器数量', kind: 'number', min: 1 },
      { path: 'max_cost', label: '最大开销', kind: 'number', min: 0 },
    ],
  },
  {
    key: 'db',
    title: '数据库',
    fields: [
      { path: 'session', label: '会话数据库', kind: 'text', placeholder: 'data/session.db' },
    ],
  },
  {
    key: 'temp',
    title: '临时目录',
    fields: [{ path: 'base_path', label: '基础路径', kind: 'text', placeholder: 'cache/' }],
  },
  {
    key: 'hook',
    title: '钩子',
    fields: [
      {
        path: 'exec.task_before_start',
        label: '任务开始前',
        kind: 'text',
        placeholder: '/path/to/hook.sh start',
      },
      { path: 'exec.task_success', label: '任务成功', kind: 'text' },
      { path: 'exec.task_fail', label: '任务失败', kind: 'text' },
      { path: 'exec.task_cancel', label: '任务取消', kind: 'text' },
    ],
  },
  {
    key: 'web',
    title: 'Web 服务',
    fields: [
      { path: 'enable', label: '启用 Web 服务', kind: 'bool', hint: '重启后生效' },
      {
        path: 'listen',
        label: '监听地址',
        kind: 'text',
        placeholder: '0.0.0.0:8080',
        hint: '重启后生效',
      },
      {
        path: 'api_token',
        label: 'API Token',
        kind: 'password',
        hint: '用于 Bearer 认证，留空禁用',
      },
      { path: 'secure_cookie', label: 'Secure Cookie', kind: 'bool', hint: '仅在 HTTPS 下启用' },
    ],
  },
]

export const telegramGroup: SettingsGroupDef = {
  key: 'telegram',
  title: 'Telegram 凭据',
  fields: [
    { path: 'token', label: 'Bot Token', kind: 'password', placeholder: '123456:ABC-DEF...' },
    { path: 'app_id', label: 'API ID', kind: 'number' },
    { path: 'app_hash', label: 'API Hash', kind: 'password' },
    { path: 'rpc_retry', label: 'RPC 重试次数', kind: 'number' },
    { path: 'media_group_timeout', label: '媒体组超时（秒）', kind: 'number' },
    { path: 'proxy.enable', label: '启用代理', kind: 'bool' },
    { path: 'proxy.url', label: '代理地址', kind: 'text', placeholder: 'socks5://127.0.0.1:1080' },
    { path: 'userbot.enable', label: '启用 Userbot', kind: 'bool' },
    { path: 'userbot.session', label: 'Userbot 会话文件', kind: 'text' },
    { path: 'userbot.phone', label: 'Userbot 手机号', kind: 'text', placeholder: '+8613800000000' },
  ],
}

export function getByPath(source: Record<string, unknown>, path: string): unknown {
  let current: unknown = source
  for (const key of path.split('.')) {
    if (current === null || typeof current !== 'object') {
      return undefined
    }
    current = (current as Record<string, unknown>)[key]
  }
  return current
}

export function setByPath(target: Record<string, unknown>, path: string, value: unknown): void {
  const keys = path.split('.')
  let current: Record<string, unknown> = target
  for (let i = 0; i < keys.length - 1; i += 1) {
    const key = keys[i]
    const next = current[key]
    if (next === null || typeof next !== 'object' || Array.isArray(next)) {
      const created: Record<string, unknown> = {}
      current[key] = created
      current = created
    } else {
      current = next as Record<string, unknown>
    }
  }
  current[keys[keys.length - 1]] = value
}

export function cloneGroup(source: Record<string, unknown> | undefined): Record<string, unknown> {
  if (!source) {
    return {}
  }
  // Values originate from the server's JSON document, so the round-trip shape is preserved.
  return JSON.parse(JSON.stringify(source)) as Record<string, unknown>
}
