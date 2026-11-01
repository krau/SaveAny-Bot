import { watch } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { i18n, translate } from './i18n'
import { useAuthStore } from './stores/auth'
import AppLayout from './components/AppLayout.vue'
import LoginView from './views/LoginView.vue'
import DashboardView from './views/DashboardView.vue'
import GuideView from './views/GuideView.vue'
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
      meta: { title: 'nav.login', public: true },
    },
    {
      path: '/',
      component: AppLayout,
      children: [
        {
          path: '',
          name: 'dashboard',
          component: DashboardView,
          meta: { title: 'nav.overview' },
        },
        {
          path: 'guide',
          name: 'guide',
          component: GuideView,
          meta: { title: 'nav.guide' },
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: TasksView,
          meta: { title: 'nav.tasks' },
        },
        {
          path: 'storages',
          name: 'storages',
          component: StoragesView,
          meta: { title: 'nav.storages' },
        },
        {
          path: 'users',
          name: 'users',
          component: UsersView,
          meta: { title: 'nav.users' },
        },
        {
          path: 'telegram',
          name: 'telegram',
          component: TelegramView,
          meta: { title: 'nav.telegram' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: SettingsView,
          meta: { title: 'nav.settings' },
        },
        {
          path: 'system',
          name: 'system',
          component: SystemView,
          meta: { title: 'nav.system' },
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

function applyTitle(to = router.currentRoute.value) {
  const key = typeof to.meta.title === 'string' ? to.meta.title : ''
  document.title = key ? `${translate(key)} - SaveAny-Bot` : 'SaveAny-Bot'
}

router.afterEach((to) => applyTitle(to))
watch(i18n.global.locale, () => applyTitle())

export default router
