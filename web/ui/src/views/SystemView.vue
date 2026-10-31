<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ApiError, errorMessage, systemApi } from '../api'
import AppIcon from '../components/AppIcon.vue'
import LoadingButton from '../components/LoadingButton.vue'
import PageHeader from '../components/PageHeader.vue'
import { useConfirmStore } from '../stores/confirm'
import { useToastStore } from '../stores/toasts'
import { formatUptime } from '../utils'
import type { SystemInfo } from '../types'

const toast = useToastStore()
const confirm = useConfirmStore()
const info = ref<SystemInfo | null>(null)
const error = ref('')
const busy = ref(false)
const host = window.location.host

async function load() {
  try {
    info.value = await systemApi.info()
    error.value = ''
  } catch (err) {
    error.value = 'load_failed'
    toast.push(errorMessage(err), 'error', 6000)
  }
}

onMounted(load)

async function restart(force = false) {
  if (
    !force &&
    !(await confirm.ask({
      title: '重启进程',
      message: '重启进程？正在运行的任务会被中断。',
      confirmText: '重启进程',
      danger: true,
    }))
  ) {
    return
  }
  busy.value = true
  try {
    await systemApi.restart(force)
    toast.push('正在重启…', 'info')
    await new Promise((resolve) => setTimeout(resolve, 1500))
    for (let i = 0; i < 20; i += 1) {
      try {
        info.value = await systemApi.info()
        toast.push('重启完成', 'success')
        break
      } catch {
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }
    }
  } catch (err) {
    if (
      !force &&
      err instanceof ApiError &&
      err.status === 409 &&
      (await confirm.ask({
        title: '强制重启',
        message: '仍有任务运行，强制重启会中断它们。继续？',
        confirmText: '强制重启',
        danger: true,
      }))
    ) {
      await restart(true)
      return
    }
    toast.push(errorMessage(err), 'error', 6000)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="stack fade-in">
    <PageHeader title="系统" description="版本、运行路径与进程控制。">
      <template #actions>
        <button class="btn small" type="button" @click="load">
          <AppIcon name="refresh" :size="15" />
          刷新
        </button>
        <LoadingButton class="small danger" type="button" :loading="busy" @click="restart()">
          <AppIcon name="refresh" :size="15" />
          重启进程
        </LoadingButton>
      </template>
    </PageHeader>

    <div v-if="info" class="process-panel">
      <div>
        <span class="badge status success">进程运行中</span>
        <div class="process-version">{{ info.version }}</div>
        <div class="process-meta">{{ info.commit || 'unknown commit' }}</div>
        <div class="process-meta">{{ info.build_time }}</div>
        <p class="muted" style="margin: 18px 0 0; font-size: 14px; line-height: 1.65">
          {{ info.docker ? '运行在 Docker 容器中' : '运行在本地进程中' }}。
          重启会中断正在执行的任务，必要时可使用强制重启。
        </p>
      </div>
      <div class="process-metrics">
        <div>
          <div class="stat-label">运行时长</div>
          <div class="stat-value">{{ formatUptime(info.uptime_seconds) }}</div>
        </div>
        <div>
          <div class="stat-label">任务</div>
          <div class="stat-value">{{ info.running_tasks }}</div>
          <div class="stat-sub">{{ info.queued_tasks }} 个排队</div>
        </div>
        <div>
          <div class="stat-label">Worker</div>
          <div class="stat-value">{{ info.workers }}</div>
          <div class="stat-sub">并发下载数</div>
        </div>
      </div>
    </div>

    <div v-else-if="!error" class="panel loading">正在读取运行信息…</div>

    <div v-if="info" class="panel">
      <div class="panel-head">
        <div>
          <h2 class="panel-title">运行信息</h2>
          <p class="panel-sub">当前进程使用的路径与监听地址</p>
        </div>
        <span class="badge" :class="info.docker ? 'info' : 'muted'">
          {{ info.docker ? 'Docker' : '本地进程' }}
        </span>
      </div>
      <dl class="kv">
        <dt>版本</dt><dd class="mono">{{ info.version }}</dd>
        <dt>Commit</dt><dd class="mono">{{ info.commit || '—' }}</dd>
        <dt>构建时间</dt><dd class="mono">{{ info.build_time }}</dd>
        <dt>数据目录</dt><dd class="mono">{{ info.data_dir }}</dd>
        <dt>数据库</dt><dd class="mono">{{ info.db_path }}</dd>
        <dt>监听地址</dt><dd class="mono">{{ host }}</dd>
      </dl>
    </div>
  </div>
</template>
