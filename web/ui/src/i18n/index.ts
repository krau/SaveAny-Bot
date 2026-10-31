import { createI18n } from 'vue-i18n'
import type { MessageModule } from './messages/types'

export type Locale = 'zh-Hans' | 'en'

export const LOCALE_STORAGE_KEY = 'saveany-locale'

const modules = import.meta.glob<MessageModule>('./messages/*.ts', { eager: true })

const messages: Record<Locale, Record<string, unknown>> = {
  en: {},
  'zh-Hans': {},
}

for (const module of Object.values(modules)) {
  if (!module.en || !module.zhHans) continue
  Object.assign(messages.en, module.en)
  Object.assign(messages['zh-Hans'], module.zhHans)
}

function detectLocale(): Locale {
  try {
    const stored = localStorage.getItem(LOCALE_STORAGE_KEY)
    if (stored === 'zh-Hans' || stored === 'en') return stored
  } catch {
    // Fall back to the browser language when storage is unavailable.
  }
  return navigator.languages.some((language) => language.toLowerCase().startsWith('zh'))
    ? 'zh-Hans'
    : 'en'
}

export const i18n = createI18n<false>({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'zh-Hans',
  messages: messages as never,
  missingWarn: false,
  fallbackWarn: false,
})

export function setLocale(next: Locale): void {
  i18n.global.locale.value = next
  document.documentElement.lang = next
  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, next)
  } catch {
    // The in-memory locale still applies for this session.
  }
}

export function initI18n(): void {
  document.documentElement.lang = i18n.global.locale.value
}

export function translate(key: string, params?: Record<string, unknown>): string {
  return i18n.global.t(key, params ?? {}) as string
}

export { useI18n } from 'vue-i18n'
