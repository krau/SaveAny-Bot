import { i18n } from './i18n'

export type StorageFieldKind = 'text' | 'number' | 'bool' | 'secret' | 'list'

export interface StorageField {
  name: string
  label: string
  kind: StorageFieldKind
  placeholder?: string
  hint?: string
  required?: boolean
}

export interface StorageTypeDef {
  type: string
  label: string
  fields: StorageField[]
}

export const storageTypeDefs: StorageTypeDef[] = [
  {
    type: 'local',
    label: 'storages.types.local',
    fields: [
      {
        name: 'base_path',
        label: 'storages.fields.basePath',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.localBasePath',
        required: true,
      },
    ],
  },
  {
    type: 'webdav',
    label: 'storages.types.webdav',
    fields: [
      {
        name: 'url',
        label: 'storages.fields.serverUrl',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.webdavUrl',
        required: true,
      },
      { name: 'username', label: 'storages.fields.username', kind: 'text' },
      { name: 'password', label: 'storages.fields.password', kind: 'secret' },
      {
        name: 'base_path',
        label: 'storages.fields.basePath',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.basePath',
      },
    ],
  },
  {
    type: 'alist',
    label: 'storages.types.alist',
    fields: [
      {
        name: 'url',
        label: 'storages.fields.serverUrl',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.alistUrl',
        required: true,
      },
      { name: 'username', label: 'storages.fields.username', kind: 'text' },
      { name: 'password', label: 'storages.fields.password', kind: 'secret' },
      {
        name: 'token',
        label: 'storages.fields.token',
        kind: 'secret',
        hint: 'storages.fields.tokenHint',
      },
      {
        name: 'base_path',
        label: 'storages.fields.basePath',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.basePath',
      },
      {
        name: 'token_exp',
        label: 'storages.fields.tokenExp',
        kind: 'number',
        hint: 'storages.fields.tokenExpHint',
      },
    ],
  },
  {
    type: 's3',
    label: 'storages.types.s3',
    fields: [
      {
        name: 'endpoint',
        label: 'storages.fields.endpoint',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.s3Endpoint',
        required: true,
      },
      { name: 'access_key_id', label: 'storages.fields.accessKeyId', kind: 'secret', required: true },
      { name: 'secret_access_key', label: 'storages.fields.secretAccessKey', kind: 'secret', required: true },
      { name: 'bucket_name', label: 'storages.fields.bucket', kind: 'text', required: true },
      { name: 'base_path', label: 'storages.fields.basePath', kind: 'text' },
      { name: 'use_ssl', label: 'storages.fields.useSsl', kind: 'bool' },
      { name: 'region', label: 'storages.fields.region', kind: 'text', placeholder: 'storages.fields.placeholders.region' },
      { name: 'virtual_host', label: 'storages.fields.virtualHost', kind: 'bool' },
    ],
  },
  {
    type: 'minio',
    label: 'storages.types.minio',
    fields: [
      {
        name: 'endpoint',
        label: 'storages.fields.endpoint',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.minioEndpoint',
        required: true,
      },
      { name: 'access_key_id', label: 'storages.fields.accessKeyId', kind: 'secret', required: true },
      { name: 'secret_access_key', label: 'storages.fields.secretAccessKey', kind: 'secret', required: true },
      { name: 'bucket_name', label: 'storages.fields.bucket', kind: 'text', required: true },
      { name: 'base_path', label: 'storages.fields.basePath', kind: 'text' },
      { name: 'use_ssl', label: 'storages.fields.useSsl', kind: 'bool' },
    ],
  },
  {
    type: 'telegram',
    label: 'storages.types.telegram',
    fields: [
      {
        name: 'chat_id',
        label: 'storages.fields.chatId',
        kind: 'number',
        hint: 'storages.fields.chatIdHint',
        required: true,
      },
      { name: 'force_file', label: 'storages.fields.forceFile', kind: 'bool' },
      {
        name: 'rate_limit',
        label: 'storages.fields.rateLimit',
        kind: 'number',
        hint: 'storages.fields.rateLimitHint',
      },
      { name: 'rate_burst', label: 'storages.fields.rateBurst', kind: 'number' },
      { name: 'skip_large', label: 'storages.fields.skipLarge', kind: 'bool', hint: 'storages.fields.skipLargeHint' },
      { name: 'split_large_video', label: 'storages.fields.splitVideo', kind: 'bool' },
      { name: 'split_size_mb', label: 'storages.fields.splitSize', kind: 'number' },
    ],
  },
  {
    type: 'rclone',
    label: 'storages.types.rclone',
    fields: [
      {
        name: 'remote',
        label: 'storages.fields.remote',
        kind: 'text',
        placeholder: 'storages.fields.placeholders.remote',
        required: true,
      },
      { name: 'base_path', label: 'storages.fields.basePath', kind: 'text' },
      {
        name: 'config_path',
        label: 'storages.fields.configPath',
        kind: 'text',
        hint: 'storages.fields.configPathHint',
      },
      { name: 'flags', label: 'storages.fields.flags', kind: 'list', hint: 'storages.fields.flagsHint' },
    ],
  },
]

export function storageTypeDef(type: string): StorageTypeDef | undefined {
  return storageTypeDefs.find((item) => item.type === type)
}

export function storageTypeLabel(type: string): string {
  const def = storageTypeDef(type)
  return def ? (i18n.global.t(def.label) as string) : type
}

export function storageTypeIcon(type: string): string {
  switch (type) {
    case 'local':
      return 'folder'
    case 'webdav':
    case 'alist':
    case 's3':
      return 'cloud'
    case 'minio':
      return 'server'
    case 'telegram':
      return 'send'
    case 'rclone':
      return 'hardDrive'
    default:
      return 'database'
  }
}

export function storageDefaults(type: string): Record<string, unknown> {
  const def = storageTypeDef(type)
  const config: Record<string, unknown> = {}
  if (!def) {
    return config
  }
  for (const field of def.fields) {
    switch (field.kind) {
      case 'bool':
        config[field.name] = false
        break
      case 'number':
        config[field.name] = 0
        break
      case 'list':
        config[field.name] = []
        break
      default:
        config[field.name] = ''
    }
  }
  if (type === 's3' || type === 'minio') {
    config.use_ssl = true
  }
  return config
}
