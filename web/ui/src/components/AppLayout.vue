<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { errorMessage } from '../api'
import { useFormFeedback } from '../composables/useFormFeedback'
import { useAuthStore } from '../stores/auth'
import AppIcon from './AppIcon.vue'
import LoadingButton from './LoadingButton.vue'
import BrandMark from './BrandMark.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const navigation = [
  {
    label: '工作台',
    items: [
      { path: '/', label: '运行概览', icon: 'dashboard' },
      { path: '/tasks', label: '任务', icon: 'tasks' },
    ],
  },
  {
    label: '资源',
    items: [
      { path: '/storages', label: '存储', icon: 'database' },
      { path: '/users', label: '用户', icon: 'users' },
    ],
  },
  {
    label: '连接',
    items: [{ path: '/telegram', label: 'Telegram', icon: 'send' }],
  },
  {
    label: '系统',
    items: [
      { path: '/settings', label: '设置', icon: 'sliders' },
      { path: '/system', label: '系统', icon: 'server' },
    ],
  },
]

const sidebarOpen = ref(false)
const passwordOpen = ref(false)
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordBusy = ref(false)
const { invalidField, fail, clear, success } = useFormFeedback()

watch(
  () => route.fullPath,
  () => {
    sidebarOpen.value = false
  },
)

function openPasswordDialog() {
  currentPassword.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
  clear()
  passwordOpen.value = true
}

async function submitPassword() {
  clear()
  if (newPassword.value.length < 8) {
    fail('新密码至少需要 8 个字符', 'new-password')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    fail('两次输入的新密码不一致', 'confirm-password')
    return
  }
  passwordBusy.value = true
  try {
    await auth.changePassword(currentPassword.value, newPassword.value)
    success('密码已更新')
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (err) {
    fail(errorMessage(err))
  } finally {
    passwordBusy.value = false
  }
}

async function handleLogout() {
  try {
    await auth.logout()
  } catch {
    // The local session is cleared regardless of the server response.
  }
  await router.replace({ path: '/login' })
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  if (passwordOpen.value) {
    passwordOpen.value = false
  } else if (sidebarOpen.value) {
    sidebarOpen.value = false
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">跳到主内容</a>
    <div v-if="sidebarOpen" class="sidebar-backdrop" @click="sidebarOpen = false"></div>

    <div class="layout">
      <aside id="primary-navigation" class="sidebar" :class="{ open: sidebarOpen }">
        <div class="brand">
          <span class="brand-mark"><BrandMark :size="22" /></span>
          <div class="brand-copy">
            <strong>SaveAny-Bot</strong>
            <span>文件中继控制台</span>
          </div>
          <button class="sidebar-close" type="button" aria-label="关闭导航" @click="sidebarOpen = false">
            <AppIcon name="close" :size="18" />
          </button>
        </div>

        <nav class="nav" aria-label="主导航">
          <div v-for="group in navigation" :key="group.label" class="nav-group">
            <div class="nav-label">{{ group.label }}</div>
            <RouterLink
              v-for="item in group.items"
              :key="item.path"
              :to="item.path"
              active-class="nav-prefix"
              :class="{ 'is-active': route.path === item.path }"
            >
              <AppIcon :name="item.icon" :size="17" />
              <span>{{ item.label }}</span>
            </RouterLink>
          </div>
        </nav>

        <div class="sidebar-foot">
          <div class="sidebar-status">
            <span class="live-dot"></span>
            <span>管理界面已连接</span>
          </div>
          <div class="sidebar-account">
            <button class="sidebar-action" type="button" @click="openPasswordDialog">
              <AppIcon name="key" :size="14" />
              修改密码
            </button>
            <button class="sidebar-action" type="button" @click="handleLogout">
              <AppIcon name="logout" :size="14" />
              退出登录
            </button>
          </div>
          <div class="sidebar-meta">配置与会话保存在本机 SQLite</div>
        </div>
      </aside>

      <div class="main">
        <header class="topbar">
          <button
            class="btn ghost icon menu-button"
            type="button"
            aria-label="打开导航"
            aria-controls="primary-navigation"
            :aria-expanded="sidebarOpen"
            @click="sidebarOpen = true"
          >
            <AppIcon name="menu" :size="19" />
          </button>
          <span class="topbar-mobile-title">SaveAny-Bot</span>
        </header>

        <main id="main-content" class="content" tabindex="-1">
          <RouterView />
        </main>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="passwordOpen" class="modal-overlay" @click.self="passwordOpen = false">
        <div class="modal" role="dialog" aria-modal="true" aria-labelledby="password-dialog-title">
          <div class="modal-head">
            <h3 id="password-dialog-title" class="modal-title">修改管理员密码</h3>
            <button class="btn ghost icon small" type="button" aria-label="关闭" @click="passwordOpen = false">
              <AppIcon name="close" :size="17" />
            </button>
          </div>
          <form class="stack" @submit.prevent="submitPassword" novalidate>
            <div class="field">
              <label for="current-password">当前密码</label>
              <input id="current-password" v-model="currentPassword" type="password" autocomplete="current-password" required />
            </div>
            <div class="field" :class="{ 'is-invalid': invalidField === 'new-password' }">
              <label for="new-password">新密码</label>
              <input
                id="new-password"
                v-model="newPassword"
                type="password"
                autocomplete="new-password"
                :aria-invalid="invalidField === 'new-password' ? 'true' : undefined"
                required
                @input="clear('new-password')"
              />
              <span class="hint">至少 8 个字符</span>
            </div>
            <div class="field" :class="{ 'is-invalid': invalidField === 'confirm-password' }">
              <label for="confirm-password">确认新密码</label>
              <input
                id="confirm-password"
                v-model="confirmPassword"
                type="password"
                autocomplete="new-password"
                :aria-invalid="invalidField === 'confirm-password' ? 'true' : undefined"
                required
                @input="clear('confirm-password')"
              />
            </div>
            <div class="modal-actions">
              <button class="btn ghost" type="button" @click="passwordOpen = false">取消</button>
              <LoadingButton type="submit" class="primary" :loading="passwordBusy">
                保存新密码
              </LoadingButton>
            </div>
          </form>
        </div>
      </div>
    </Teleport>
  </div>
</template>
