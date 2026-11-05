import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { configureApi } from './api'
import { useAuthStore } from './stores/auth'
import { useThemeStore } from './stores/theme'
import { i18n, initI18n } from './i18n'
import './assets/main.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(i18n)
initI18n()

const theme = useThemeStore()
theme.init()

const auth = useAuthStore()

configureApi({
  getCsrf: () => auth.csrf,
  onUnauthorized: () => {
    auth.clear()
    const current = router.currentRoute.value
    if (current.name !== 'login') {
      void router.replace({ path: '/login', query: { redirect: current.fullPath } })
    }
  },
})

app.use(router)
app.mount('#app')
