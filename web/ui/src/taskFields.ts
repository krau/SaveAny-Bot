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
    label: 'tasks.types.directlinks',
    fields: [
      {
        name: 'urls',
        label: 'tasks.fields.directlinks.urls.label',
        kind: 'list',
        required: true,
        placeholder: 'https://example.com/file.zip\nhttps://example.com/another.zip',
      },
    ],
  },
  {
    type: 'ytdlp',
    label: 'tasks.types.ytdlp',
    fields: [
      {
        name: 'urls',
        label: 'tasks.fields.ytdlp.urls.label',
        kind: 'list',
        required: true,
        placeholder: 'https://www.youtube.com/watch?v=...',
      },
      {
        name: 'flags',
        label: 'tasks.fields.ytdlp.flags.label',
        kind: 'list',
        placeholder: '--cookies-from-browser chrome',
        hint: 'tasks.fields.ytdlp.flags.hint',
      },
    ],
  },
  {
    type: 'aria2',
    label: 'tasks.types.aria2',
    fields: [
      {
        name: 'urls',
        label: 'tasks.fields.aria2.urls.label',
        kind: 'list',
        required: true,
        placeholder: 'https://example.com/file.iso',
      },
      {
        name: 'options',
        label: 'tasks.fields.aria2.options.label',
        kind: 'json',
        placeholder: '{"split": "8", "max-connection-per-server": "8"}',
        hint: 'tasks.fields.aria2.options.hint',
      },
    ],
  },
  {
    type: 'parseditem',
    label: 'tasks.types.parseditem',
    fields: [
      {
        name: 'url',
        label: 'tasks.fields.parseditem.url.label',
        kind: 'text',
        required: true,
        placeholder: 'https://example.com/post/123',
      },
    ],
  },
  {
    type: 'tgfiles',
    label: 'tasks.types.tgfiles',
    fields: [
      {
        name: 'message_links',
        label: 'tasks.fields.tgfiles.message_links.label',
        kind: 'list',
        placeholder: 'https://t.me/c/123456789/42',
      },
      {
        name: 'chat_id',
        label: 'tasks.fields.tgfiles.chat_id.label',
        kind: 'text',
        placeholder: '@channel or -1001234567890',
      },
      {
        name: 'message_range',
        label: 'tasks.fields.tgfiles.message_range.label',
        kind: 'text',
        placeholder: '100-120',
      },
      {
        name: 'filter',
        label: 'tasks.fields.tgfiles.filter.label',
        kind: 'text',
        placeholder: '.*\.mp4$',
      },
    ],
  },
  {
    type: 'tphpics',
    label: 'tasks.types.tphpics',
    fields: [
      {
        name: 'telegraph_url',
        label: 'tasks.fields.tphpics.telegraph_url.label',
        kind: 'text',
        required: true,
        placeholder: 'https://telegra.ph/...',
      },
    ],
  },
  {
    type: 'transfer',
    label: 'tasks.types.transfer',
    fields: [
      {
        name: 'source_storage_id',
        label: 'tasks.fields.transfer.source_storage.label',
        kind: 'text',
        required: true,
        storageSelect: true,
      },
      {
        name: 'source_path',
        label: 'tasks.fields.transfer.source_path.label',
        kind: 'text',
        required: true,
        placeholder: 'folder/subfolder',
      },
      {
        name: 'target_storage_id',
        label: 'tasks.fields.transfer.target_storage.label',
        kind: 'text',
        required: true,
        storageSelect: true,
      },
      {
        name: 'target_path',
        label: 'tasks.fields.transfer.target_path.label',
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
