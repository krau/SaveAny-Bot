import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './stores/auth'
import AppLayout from './components/AppLayout.vue'
import LoginView from './views/LoginView.vue'
import DashboardView from './views/DashboardView.vue'
import TasksView from './views/TasksView.vue'
import StoragesView from './views/StoragesView.vue'
import UsersView from './views/UsersView.vue'
import TelegramView from './views/TelegramView.vue'
import SettingsView from './views/SettingsView.vue'
import SystemView from './views/SystemView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: { title: '登录', public: true },
    },
    {
      path: '/',
      component: AppLayout,
      children: [
        {
          path: '',
          name: 'dashboard',
          component: DashboardView,
          meta: { title: '运行概览' },
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: TasksView,
          meta: { title: '任务' },
        },
        {
          path: 'storages',
          name: 'storages',
          component: StoragesView,
          meta: { title: '存储' },
        },
        {
          path: 'users',
          name: 'users',
          component: UsersView,
          meta: { title: '用户' },
        },
        {
          path: 'telegram',
          name: 'telegram',
          component: TelegramView,
          meta: { title: 'Telegram' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: SettingsView,
          meta: { title: '设置' },
        },
        {
          path: 'system',
          name: 'system',
          component: SystemView,
          meta: { title: '系统' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const isPublic = to.meta.public === true
  if (!auth.checked) {
    await auth.ensure()
  }
  if (isPublic) {
    if (auth.authenticated && to.name === 'login') {
      return { path: '/' }
    }
    return true
  }
  if (!auth.authenticated) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  return true
})

router.afterEach((to) => {
  const title = typeof to.meta.title === 'string' ? to.meta.title : ''
  document.title = title ? `${title} - SaveAny-Bot` : 'SaveAny-Bot'
})

export default router
