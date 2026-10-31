import { defineStore } from 'pinia'

export type ThemePreference = 'system' | 'light' | 'dark'
export type ResolvedTheme = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'saveany-theme'

const systemMedia = window.matchMedia('(prefers-color-scheme: dark)')
const themeColors: Record<ResolvedTheme, string> = {
  light: '#f5f6f8',
  dark: '#0e1014',
}

let systemListenerAttached = false

function readPreference(): ThemePreference {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY)
    if (value === 'light' || value === 'dark' || value === 'system') return value
  } catch {
    // Storage may be unavailable in privacy-restricted contexts.
  }
  return 'system'
}

export const useThemeStore = defineStore('theme', {
  state: () => ({
    preference: 'system' as ThemePreference,
    systemDark: systemMedia.matches,
  }),
  getters: {
    resolved(state): ResolvedTheme {
      if (state.preference === 'system') return state.systemDark ? 'dark' : 'light'
      return state.preference
    },
  },
  actions: {
    init() {
      this.preference = readPreference()
      this.systemDark = systemMedia.matches
      this.apply()

      if (!systemListenerAttached) {
        systemMedia.addEventListener('change', (event) => {
          this.systemDark = event.matches
          if (this.preference === 'system') this.apply()
        })
        systemListenerAttached = true
      }
    },
    setPreference(preference: ThemePreference) {
      this.preference = preference
      try {
        localStorage.setItem(THEME_STORAGE_KEY, preference)
      } catch {
        // The in-memory preference still applies for this session.
      }
      this.apply()
    },
    apply() {
      const theme = this.resolved
      const root = document.documentElement
      root.dataset.theme = theme
      root.style.colorScheme = theme
      document
        .querySelector('meta[name="theme-color"]')
        ?.setAttribute('content', themeColors[theme])
    },
  },
})
