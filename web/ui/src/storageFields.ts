export type StorageFieldKind = 'text' | 'number' | 'bool' | 'secret' | 'list'

export interface StorageField {
  name: string
  label: string
  kind: StorageFieldKind
  placeholder?: string
  hint?: string
}

export interface StorageTypeDef {
  type: string
  label: string
  fields: StorageField[]
}

export const storageTypeDefs: StorageTypeDef[] = [
  {
    type: 'local',
    label: '本地磁盘',
    fields: [
      { name: 'base_path', label: '基础路径', kind: 'text', placeholder: '/data/downloads' },
    ],
  },
  {
    type: 'webdav',
    label: 'WebDAV',
    fields: [
      { name: 'url', label: '服务器地址', kind: 'text', placeholder: 'https://dav.example.com' },
      { name: 'username', label: '用户名', kind: 'text' },
      { name: 'password', label: '密码', kind: 'secret' },
      { name: 'base_path', label: '基础路径', kind: 'text', placeholder: '/saveany' },
    ],
  },
  {
    type: 'alist',
    label: 'AList',
    fields: [
      { name: 'url', label: '服务器地址', kind: 'text', placeholder: 'https://alist.example.com' },
      { name: 'username', label: '用户名', kind: 'text' },
      { name: 'password', label: '密码', kind: 'secret' },
      { name: 'token', label: 'Token', kind: 'secret', hint: '可留空，登录后自动获取' },
      { name: 'base_path', label: '基础路径', kind: 'text', placeholder: '/saveany' },
      {
        name: 'token_exp',
        label: 'Token 过期时间',
        kind: 'number',
        hint: 'Unix 时间戳，可留空由服务端刷新',
      },
    ],
  },
  {
    type: 's3',
    label: 'S3 兼容存储',
    fields: [
      { name: 'endpoint', label: 'Endpoint', kind: 'text', placeholder: 's3.example.com' },
      { name: 'access_key_id', label: 'Access Key ID', kind: 'secret' },
      { name: 'secret_access_key', label: 'Secret Access Key', kind: 'secret' },
      { name: 'bucket_name', label: 'Bucket', kind: 'text' },
      { name: 'base_path', label: '基础路径', kind: 'text' },
      { name: 'use_ssl', label: '使用 SSL', kind: 'bool' },
      { name: 'region', label: 'Region', kind: 'text', placeholder: 'us-east-1' },
      { name: 'virtual_host', label: 'Virtual Host 模式', kind: 'bool' },
    ],
  },
  {
    type: 'minio',
    label: 'MinIO',
    fields: [
      { name: 'endpoint', label: 'Endpoint', kind: 'text', placeholder: 'minio.example.com:9000' },
      { name: 'access_key_id', label: 'Access Key ID', kind: 'secret' },
      { name: 'secret_access_key', label: 'Secret Access Key', kind: 'secret' },
      { name: 'bucket_name', label: 'Bucket', kind: 'text' },
      { name: 'base_path', label: '基础路径', kind: 'text' },
      { name: 'use_ssl', label: '使用 SSL', kind: 'bool' },
    ],
  },
  {
    type: 'telegram',
    label: 'Telegram',
    fields: [
      { name: 'chat_id', label: 'Chat ID', kind: 'number', hint: '目标频道或群组的数字 ID' },
      { name: 'force_file', label: '强制以文件发送', kind: 'bool' },
      { name: 'rate_limit', label: '速率限制', kind: 'number', hint: '每秒请求数，0 表示默认' },
      { name: 'rate_burst', label: '突发数量', kind: 'number' },
      { name: 'skip_large', label: '跳过超限文件', kind: 'bool', hint: '跳过超过 2GB 的文件' },
      { name: 'split_large_video', label: '分割大视频', kind: 'bool' },
      { name: 'split_size_mb', label: '分割大小 (MB)', kind: 'number' },
    ],
  },
  {
    type: 'rclone',
    label: 'rclone',
    fields: [
      { name: 'remote', label: 'Remote 名称', kind: 'text', placeholder: 'myremote' },
      { name: 'base_path', label: '基础路径', kind: 'text' },
      {
        name: 'config_path',
        label: '配置文件路径',
        kind: 'text',
        hint: '留空使用默认 rclone 配置',
      },
      { name: 'flags', label: '额外参数', kind: 'list', hint: '每行一个参数，例如 --transfers 4' },
    ],
  },
]

export function storageTypeDef(type: string): StorageTypeDef | undefined {
  return storageTypeDefs.find((item) => item.type === type)
}

export function storageTypeLabel(type: string): string {
  return storageTypeDef(type)?.label ?? type
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
