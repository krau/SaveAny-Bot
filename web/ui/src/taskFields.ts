export type TaskFieldKind = 'text' | 'number' | 'bool' | 'list' | 'json'

export interface TaskField {
  name: string
  label: string
  kind: TaskFieldKind
  placeholder?: string
  required?: boolean
  hint?: string
  storageSelect?: boolean
}

export interface TaskTypeDef {
  type: string
  label: string
  fields: TaskField[]
}

export const taskTypeDefs: TaskTypeDef[] = [
  {
    type: 'directlinks',
    label: '直链下载',
    fields: [
      {
        name: 'urls',
        label: '下载链接',
        kind: 'list',
        required: true,
        placeholder: 'https://example.com/file.zip\nhttps://example.com/another.zip',
        hint: '每行一个链接',
      },
    ],
  },
  {
    type: 'ytdlp',
    label: 'yt-dlp 视频',
    fields: [
      {
        name: 'urls',
        label: '视频链接',
        kind: 'list',
        required: true,
        placeholder: 'https://www.youtube.com/watch?v=...',
        hint: '每行一个链接',
      },
      {
        name: 'flags',
        label: '额外参数',
        kind: 'list',
        placeholder: '--cookies-from-browser chrome',
        hint: '每行一个 yt-dlp 参数',
      },
    ],
  },
  {
    type: 'aria2',
    label: 'Aria2 下载',
    fields: [
      {
        name: 'urls',
        label: '下载链接',
        kind: 'list',
        required: true,
        placeholder: 'https://example.com/file.iso',
        hint: '每行一个链接',
      },
      {
        name: 'options',
        label: 'Aria2 选项 (JSON)',
        kind: 'json',
        placeholder: '{"split": "8", "max-connection-per-server": "8"}',
        hint: 'JSON 对象，键值均为字符串（可选）',
      },
    ],
  },
  {
    type: 'parseditem',
    label: '解析链接',
    fields: [
      {
        name: 'url',
        label: '页面链接',
        kind: 'text',
        required: true,
        placeholder: 'https://example.com/post/123',
      },
    ],
  },
  {
    type: 'tgfiles',
    label: 'Telegram 文件',
    fields: [
      {
        name: 'message_links',
        label: '消息链接',
        kind: 'list',
        required: true,
        placeholder: 'https://t.me/c/123456789/42',
        hint: '每行一个消息链接',
      },
    ],
  },
  {
    type: 'tphpics',
    label: 'Telegraph 图集',
    fields: [
      {
        name: 'telegraph_url',
        label: 'Telegraph 链接',
        kind: 'text',
        required: true,
        placeholder: 'https://telegra.ph/...',
      },
    ],
  },
  {
    type: 'transfer',
    label: '存储迁移',
    fields: [
      {
        name: 'source_storage',
        label: '源存储',
        kind: 'text',
        required: true,
        storageSelect: true,
      },
      {
        name: 'source_path',
        label: '源路径',
        kind: 'text',
        required: true,
        placeholder: 'folder/subfolder',
      },
      {
        name: 'target_storage',
        label: '目标存储',
        kind: 'text',
        required: true,
        storageSelect: true,
      },
      {
        name: 'target_path',
        label: '目标路径',
        kind: 'text',
        required: true,
        placeholder: 'backup',
      },
    ],
  },
]

export function taskTypeDef(type: string): TaskTypeDef | undefined {
  return taskTypeDefs.find((item) => item.type === type)
}
