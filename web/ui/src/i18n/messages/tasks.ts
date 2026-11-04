export const zhHans = {
  tasks: {
    count: '共 {total} 条记录',
    empty: '暂无任务',
    createTitle: '新建任务',
    filters: {
      status: '状态',
      type: '类型',
      allStatuses: '全部状态',
      allTypes: '全部类型',
    },
    status: {
      queued: '排队中',
      running: '进行中',
      completed: '已完成',
      partial: '部分完成',
      failed: '失败',
      cancelled: '已取消',
    },
    table: {
      task: '任务',
      status: '状态',
      storagePath: '存储 / 路径',
      progress: '进度',
      createdAt: '创建时间',
      actions: '操作',
      noStorage: '未指定存储',
      rootPath: '根目录',
    },
    progress: {
      files: '{done} / {total} 个文件',
      separator: '，',
    },
    toast: {
      created: '任务已创建',
      retried: '已创建重试任务，仅包含失败的文件',
    },
    failedFiles: '失败文件',
    confirm: {
      cancelTitle: '取消任务',
      cancelMessage: '取消该任务？正在下载的内容会被中断。',
      cancelConfirm: '取消任务',
      retryTitle: '重试失败的文件',
      retryMessage: '将为该任务中失败的文件创建新的下载任务。',
      retryConfirm: '重试',
      removeTitle: '删除任务记录',
      removeMessage: '删除该任务记录？此操作不可撤销。',
      removeConfirm: '删除记录',
    },
    form: {
      taskType: '任务类型',
      storage: '存储',
      storagePlaceholder: '请选择存储',
      targetPath: '目标路径',
      targetPathPlaceholder: '可留空，例如 downloads/anime',
      webhook: 'Webhook（可选）',
      webhookPlaceholder: 'https://example.com/hook',
      required: '请填写{field}',
      invalidJson: '{field} 不是合法的 JSON',
      jsonObject: '{field} 必须是 JSON 对象',
      notNumber: '{field} 必须是数字',
      storageRequired: '请选择存储',
      pathOutsideStorage: '所选目录不在该存储中',
      tgfilesSourceRequired: '请填写消息链接，或填写聊天 ID 和消息范围',
    },
    types: {
      directlinks: '直链下载',
      ytdlp: 'yt-dlp 视频',
      aria2: 'Aria2 下载',
      parseditem: '解析链接',
      tgfiles: 'Telegram 文件',
      tphpics: 'Telegraph 图集',
      transfer: '存储迁移',
    },
    fields: {
      directlinks: {
        urls: {
          label: '下载链接',
          hint: '每行一个链接',
        },
      },
      ytdlp: {
        urls: {
          label: '视频链接',
          hint: '每行一个链接',
        },
        flags: {
          label: '额外参数',
          hint: '每行一个 yt-dlp 参数',
        },
      },
      aria2: {
        urls: {
          label: '下载链接',
          hint: '每行一个链接',
        },
        options: {
          label: 'Aria2 选项 (JSON)',
          hint: 'JSON 对象，键值均为字符串（可选）',
        },
      },
      parseditem: {
        url: {
          label: '页面链接',
        },
      },
      tgfiles: {
        message_links: {
          label: '消息链接',
          hint: '每行一个消息链接；也可以改用下方的聊天 ID + 消息范围',
        },
        chat_id: {
          label: '聊天 ID / 用户名',
          hint: '使用消息范围时填写',
        },
        message_range: {
          label: '消息范围',
          hint: '例如 100-120',
        },
        filter: {
          label: '过滤器',
          hint: '可选的正则表达式，匹配消息文本或文件名',
        },
      },
      tphpics: {
        telegraph_url: {
          label: 'Telegraph 链接',
        },
      },
      transfer: {
        source_storage: {
          label: '源存储',
        },
        source_path: {
          label: '源路径',
        },
        target_storage: {
          label: '目标存储',
        },
        target_path: {
          label: '目标路径',
        },
      },
    },
  },
}

export const en = {
  tasks: {
    count: 'Total: {total}',
    empty: 'No tasks',
    createTitle: 'Create task',
    filters: {
      status: 'Status',
      type: 'Type',
      allStatuses: 'All statuses',
      allTypes: 'All types',
    },
    status: {
      queued: 'Queued',
      running: 'Running',
      completed: 'Completed',
      partial: 'Partial',
      failed: 'Failed',
      cancelled: 'Cancelled',
    },
    table: {
      task: 'Task',
      status: 'Status',
      storagePath: 'Storage / path',
      progress: 'Progress',
      createdAt: 'Created',
      actions: 'Actions',
      noStorage: 'No storage',
      rootPath: 'Root',
    },
    progress: {
      files: '{done} / {total} files',
      separator: ', ',
    },
    toast: {
      created: 'Task created',
      retried: 'Retry task created for the failed files',
    },
    failedFiles: 'Failed files',
    confirm: {
      cancelTitle: 'Cancel task',
      cancelMessage: 'Cancel this task? The current download will be interrupted.',
      cancelConfirm: 'Cancel task',
      retryTitle: 'Retry failed files',
      retryMessage: 'A new download task will be created for the failed files of this task.',
      retryConfirm: 'Retry',
      removeTitle: 'Delete task record',
      removeMessage: 'Delete this task record? This cannot be undone.',
      removeConfirm: 'Delete record',
    },
    form: {
      taskType: 'Task type',
      storage: 'Storage',
      storagePlaceholder: 'Select storage',
      targetPath: 'Target path',
      targetPathPlaceholder: 'Optional, e.g. downloads/anime',
      webhook: 'Webhook (optional)',
      webhookPlaceholder: 'https://example.com/hook',
      required: 'Enter {field}',
      invalidJson: '{field} is not valid JSON',
      jsonObject: '{field} must be a JSON object',
      notNumber: '{field} must be a number',
      storageRequired: 'Select storage',
      pathOutsideStorage: 'The selected folder is outside this storage.',
      tgfilesSourceRequired: 'Enter message links, or a chat ID and message range',
    },
    types: {
      directlinks: 'Direct link',
      ytdlp: 'yt-dlp video',
      aria2: 'Aria2 download',
      parseditem: 'Parsed link',
      tgfiles: 'Telegram files',
      tphpics: 'Telegraph gallery',
      transfer: 'Storage transfer',
    },
    fields: {
      directlinks: {
        urls: {
          label: 'Download links',
          hint: 'One link per line',
        },
      },
      ytdlp: {
        urls: {
          label: 'Video links',
          hint: 'One link per line',
        },
        flags: {
          label: 'Extra flags',
          hint: 'One yt-dlp flag per line',
        },
      },
      aria2: {
        urls: {
          label: 'Download links',
          hint: 'One link per line',
        },
        options: {
          label: 'Aria2 options (JSON)',
          hint: 'A JSON object with string keys and values (optional)',
        },
      },
      parseditem: {
        url: {
          label: 'Page link',
        },
      },
      tgfiles: {
        message_links: {
          label: 'Message links',
          hint: 'One message link per line; alternatively use a chat ID and message range',
        },
        chat_id: {
          label: 'Chat ID / username',
          hint: 'Required when using a message range',
        },
        message_range: {
          label: 'Message range',
          hint: 'For example 100-120',
        },
        filter: {
          label: 'Filter',
          hint: 'Optional regular expression matching message text or filename',
        },
      },
      tphpics: {
        telegraph_url: {
          label: 'Telegraph link',
        },
      },
      transfer: {
        source_storage: {
          label: 'Source storage',
        },
        source_path: {
          label: 'Source path',
        },
        target_storage: {
          label: 'Target storage',
        },
        target_path: {
          label: 'Target path',
        },
      },
    },
  },
}
