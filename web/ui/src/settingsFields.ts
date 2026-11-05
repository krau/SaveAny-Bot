export type SettingsFieldKind = 'text' | 'number' | 'bool' | 'password' | 'select' | 'list' | 'json'

export interface SettingsField {
  path: string
  /** Translation key under `settings.*`. */
  label: string
  kind: SettingsFieldKind
  placeholder?: string
  /** Translation key under `settings.*`. */
  hint?: string
  min?: number
  /** Option labels are translation keys under `settings.*`. */
  options?: { value: string; label: string }[]
  /** Span both columns in the settings grid. */
  wide?: boolean
  /** Local path fields can use the server-side picker. */
  pathKind?: 'file' | 'directory'
}

export interface SettingsGroupDef {
  key: string
  /** Translation key under `settings.groups.*`. */
  title: string
  fields: SettingsField[]
}

export const settingsGroups: SettingsGroupDef[] = [
  {
    key: 'core',
    title: 'settings.groups.core',
    fields: [
      {
        path: 'lang',
        label: 'settings.core.botLanguage',
        kind: 'select',
        options: [
          { value: 'zh-Hans', label: 'settings.core.langOptions.zhHans' },
          { value: 'en', label: 'settings.core.langOptions.en' },
        ],
      },
      {
        path: 'workers',
        label: 'settings.core.workers',
        kind: 'number',
        hint: 'settings.core.workersHint',
        min: 1,
      },
      { path: 'retry', label: 'settings.core.retry', kind: 'number', min: 0 },
      {
        path: 'threads',
        label: 'settings.core.threads',
        kind: 'number',
        hint: 'settings.core.threadsHint',
        min: 1,
      },
      {
        path: 'download_threads',
        label: 'settings.core.downloadThreads',
        kind: 'number',
        hint: 'settings.core.downloadThreadsHint',
        min: 0,
      },
      {
        path: 'max_download_threads',
        label: 'settings.core.maxDownloadThreads',
        kind: 'number',
        hint: 'settings.core.maxDownloadThreadsHint',
        min: 1,
      },
      {
        path: 'transfer_budget',
        label: 'settings.core.transferBudget',
        kind: 'number',
        hint: 'settings.core.transferBudgetHint',
        min: 0,
      },
      {
        path: 'relay_min_size_mb',
        label: 'settings.core.relayMinSizeMb',
        kind: 'number',
        hint: 'settings.core.relayMinSizeMbHint',
        min: 1,
      },
      {
        path: 'chunk_retries',
        label: 'settings.core.chunkRetries',
        kind: 'number',
        hint: 'settings.core.chunkRetriesHint',
        min: 1,
      },
      { path: 'stream', label: 'settings.core.stream', kind: 'bool' },
      { path: 'no_clean_cache', label: 'settings.core.noCleanCache', kind: 'bool' },
      {
        path: 'proxy',
        label: 'settings.core.proxy',
        kind: 'text',
        placeholder: 'socks5://127.0.0.1:1080',
        hint: 'settings.core.proxyHint',
        wide: true,
      },
    ],
  },
  {
    key: 'metadata',
    title: 'settings.groups.metadata',
    fields: [
      {
        path: 'enabled',
        label: 'settings.metadata.enabled',
        kind: 'bool',
        hint: 'settings.metadata.enabledHint',
      },
      {
        path: 'include_text',
        label: 'settings.metadata.includeText',
        kind: 'bool',
        hint: 'settings.metadata.includeTextHint',
      },
      {
        path: 'sidecar',
        label: 'settings.metadata.sidecar',
        kind: 'bool',
        hint: 'settings.metadata.sidecarHint',
      },
    ],
  },
  {
    key: 'media',
    title: 'settings.groups.media',
    fields: [
      {
        path: 'allowed_media_types',
        label: 'settings.media.allowedMediaTypes',
        kind: 'list',
        placeholder: 'image\nvideo\naudio\ndocument',
        hint: 'settings.media.allowedMediaTypesHint',
        wide: true,
      },
    ],
  },
  {
    key: 'log',
    title: 'settings.groups.log',
    fields: [
      {
        path: 'level',
        label: 'settings.log.level',
        kind: 'select',
        options: [
          { value: 'debug', label: 'settings.log.levels.debug' },
          { value: 'info', label: 'settings.log.levels.info' },
          { value: 'warn', label: 'settings.log.levels.warn' },
          { value: 'error', label: 'settings.log.levels.error' },
        ],
      },
    ],
  },
  {
    key: 'aria2',
    title: 'settings.groups.aria2',
    fields: [
      { path: 'enable', label: 'settings.aria2.enable', kind: 'bool' },
      {
        path: 'url',
        label: 'settings.aria2.url',
        kind: 'text',
        placeholder: 'http://127.0.0.1:6800/jsonrpc',
      },
      { path: 'secret', label: 'settings.aria2.secret', kind: 'password' },
      { path: 'keep_file', label: 'settings.aria2.keepFile', kind: 'bool' },
    ],
  },
  {
    key: 'ytdlp',
    title: 'settings.groups.ytdlp',
    fields: [
      {
        path: 'max_height',
        label: 'settings.ytdlp.maxHeight',
        kind: 'number',
        hint: 'settings.ytdlp.maxHeightHint',
        min: 0,
      },
      {
        path: 'format',
        label: 'settings.ytdlp.format',
        kind: 'text',
        placeholder: 'bestvideo+bestaudio/best',
      },
      { path: 'recode', label: 'settings.ytdlp.recode', kind: 'text', placeholder: 'mp4' },
      {
        path: 'filename_template',
        label: 'settings.ytdlp.filenameTemplate',
        kind: 'text',
        placeholder: '%(title)s.%(ext)s',
      },
      {
        path: 'restrict_filenames',
        label: 'settings.ytdlp.restrictFilenames',
        kind: 'bool',
      },
    ],
  },
  {
    key: 'parser',
    title: 'settings.groups.parser',
    fields: [
      {
        path: 'plugin_enable',
        label: 'settings.parser.pluginEnable',
        kind: 'bool',
        hint: 'settings.parser.restartHint',
      },
      {
        path: 'plugin_dirs',
        label: 'settings.parser.pluginDirs',
        kind: 'list',
        hint: 'settings.parser.pluginDirsHint',
        placeholder: 'plugins',
        wide: true,
        pathKind: 'directory',
      },
      {
        path: 'proxy',
        label: 'settings.parser.proxy',
        kind: 'text',
        placeholder: 'http://127.0.0.1:7890',
        wide: true,
      },
      {
        path: 'configs',
        label: 'settings.parser.configs',
        kind: 'json',
        placeholder: '{"example": {"key": "value"}}',
        hint: 'settings.parser.configsHint',
        wide: true,
      },
    ],
  },
  {
    key: 'cache',
    title: 'settings.groups.cache',
    fields: [
      { path: 'ttl', label: 'settings.cache.ttl', kind: 'number', min: 0 },
      { path: 'num_counters', label: 'settings.cache.numCounters', kind: 'number', min: 1 },
      { path: 'max_cost', label: 'settings.cache.maxCost', kind: 'number', min: 0, wide: true },
    ],
  },
  {
    key: 'db',
    title: 'settings.groups.db',
    fields: [
      {
        path: 'session',
        label: 'settings.db.session',
        kind: 'text',
        placeholder: 'data/bot_session.db',
        pathKind: 'file',
      },
    ],
  },
  {
    key: 'temp',
    title: 'settings.groups.temp',
    fields: [
      {
        path: 'base_path',
        label: 'settings.temp.basePath',
        kind: 'text',
        placeholder: 'cache/',
        pathKind: 'directory',
      },
    ],
  },
  {
    key: 'hook',
    title: 'settings.groups.hook',
    fields: [
      {
        path: 'exec.task_before_start',
        label: 'settings.hook.taskBeforeStart',
        kind: 'text',
        placeholder: '/path/to/hook.sh start',
      },
      { path: 'exec.task_success', label: 'settings.hook.taskSuccess', kind: 'text' },
      { path: 'exec.task_fail', label: 'settings.hook.taskFail', kind: 'text' },
      { path: 'exec.task_cancel', label: 'settings.hook.taskCancel', kind: 'text' },
    ],
  },
  {
    key: 'web',
    title: 'settings.groups.web',
    fields: [
      {
        path: 'enable',
        label: 'settings.web.enable',
        kind: 'bool',
        hint: 'settings.web.restartHint',
      },
      {
        path: 'listen',
        label: 'settings.web.listen',
        kind: 'text',
        placeholder: '127.0.0.1:28080',
        hint: 'settings.web.listenHint',
      },
      {
        path: 'api_token',
        label: 'settings.web.apiToken',
        kind: 'password',
        hint: 'settings.web.apiTokenHint',
      },
      {
        path: 'secure_cookie',
        label: 'settings.web.secureCookie',
        kind: 'bool',
        hint: 'settings.web.secureCookieHint',
      },
    ],
  },
]

export const telegramGroup: SettingsGroupDef = {
  key: 'telegram',
  title: 'settings.groups.telegram',
  fields: [
    {
      path: 'token',
      label: 'settings.telegram.token',
      kind: 'password',
      placeholder: '123456:ABC-DEF...',
    },
    { path: 'app_id', label: 'settings.telegram.appId', kind: 'number' },
    { path: 'app_hash', label: 'settings.telegram.appHash', kind: 'password' },
    { path: 'rpc_retry', label: 'settings.telegram.rpcRetry', kind: 'number' },
    { path: 'media_group_timeout', label: 'settings.telegram.mediaGroupTimeout', kind: 'number' },
    { path: 'proxy.enable', label: 'settings.telegram.proxyEnable', kind: 'bool' },
    {
      path: 'proxy.url',
      label: 'settings.telegram.proxyUrl',
      kind: 'text',
      placeholder: 'socks5://127.0.0.1:1080',
    },
    { path: 'userbot.enable', label: 'settings.telegram.userbotEnable', kind: 'bool' },
    { path: 'userbot.session', label: 'settings.telegram.userbotSession', kind: 'text' },
    {
      path: 'userbot.phone',
      label: 'settings.telegram.userbotPhone',
      kind: 'text',
      placeholder: '+8613800000000',
    },
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
