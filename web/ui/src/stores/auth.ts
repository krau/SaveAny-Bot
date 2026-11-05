import { defineStore } from 'pinia'
import { ApiError, authApi } from '../api'

interface AuthState {
  csrf: string
  authenticated: boolean
  checked: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    csrf: '',
    authenticated: false,
    checked: false,
  }),
  actions: {
    clear() {
      this.csrf = ''
      this.authenticated = false
      this.checked = true
    },
    async login(password: string) {
      try {
        const result = await authApi.login(password)
        this.csrf = result.csrf_token
        this.authenticated = true
        this.checked = true
      } catch (err) {
        this.clear()
        throw err
      }
    },
    async logout() {
      try {
        await authApi.logout()
      } finally {
        this.clear()
      }
    },
    async changePassword(currentPassword: string, newPassword: string) {
      await authApi.changePassword(currentPassword, newPassword)
    },
    async ensure(): Promise<boolean> {
      if (this.checked) {
        return this.authenticated
      }
      try {
        const me = await authApi.me()
        this.authenticated = me.authenticated === true
        this.csrf = this.authenticated ? (me.csrf_token ?? '') : ''
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          this.authenticated = false
          this.csrf = ''
        } else {
          this.authenticated = false
          this.csrf = ''
        }
      } finally {
        this.checked = true
      }
      return this.authenticated
    },
  },
})
