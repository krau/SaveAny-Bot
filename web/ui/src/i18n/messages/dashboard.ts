export const zhHans = {
  dashboard: {
    status: {
      reading: '正在读取连接状态',
      line: 'Bot {bot}，Userbot {userbot}',
      botDisconnected: 'Bot 未连接',
    },
    relay: {
      ariaLabel: '当前中继链路',
      entry: '入口',
      queue: '调度队列',
      queueCount: '{n} 个任务',
      queueBusy: '正在处理或等待 Worker',
      queueIdle: '队列空闲',
      storage: '归档存储',
      storageAvailable: '{n} 个可用',
      storageTotal: '共 {n} 个已配置存储',
    },
    metrics: {
      activeTasks: '活动任务',
      weekTasks: '近 7 天任务',
      completedTasks: '已完成任务',
    },
    activity: {
      viewAll: '查看全部',
      ariaLabel: '最近 7 天共创建 {n} 个任务',
      barTitle: '{label} 创建 {n} 个任务',
      empty: '还没有任务记录',
    },
    storage: {
      title: '存储',
      manage: '管理',
      meta: '{type}，{location}',
      empty: '还没有存储',
    },
    tasks: {
      recentTitle: '最近任务',
      all: '全部任务',
      separator: '，',
      noStorage: '未指定存储',
      empty: '还没有任务',
    },
  },
}

export const en = {
  dashboard: {
    status: {
      reading: 'Reading connection status',
      line: 'Bot {bot}, Userbot {userbot}',
      botDisconnected: 'Bot not connected',
    },
    relay: {
      ariaLabel: 'Current relay chain',
      entry: 'Ingress',
      queue: 'Dispatch queue',
      queueCount: '{n} in queue',
      queueBusy: 'Processing or waiting for a Worker',
      queueIdle: 'Queue idle',
      storage: 'Archive storage',
      storageAvailable: 'Available: {n}',
      storageTotal: 'Configured: {n}',
    },
    metrics: {
      activeTasks: 'Active tasks',
      weekTasks: 'Last 7 days',
      completedTasks: 'Completed tasks',
    },
    activity: {
      viewAll: 'View all',
      ariaLabel: 'Tasks created in the last 7 days: {n}',
      barTitle: '{label}: {n} created',
      empty: 'No task history yet',
    },
    storage: {
      title: 'Storages',
      manage: 'Manage',
      meta: '{type}, {location}',
      empty: 'No storages yet',
    },
    tasks: {
      recentTitle: 'Recent tasks',
      all: 'All tasks',
      separator: ', ',
      noStorage: 'No storage specified',
      empty: 'No tasks yet',
    },
  },
}
