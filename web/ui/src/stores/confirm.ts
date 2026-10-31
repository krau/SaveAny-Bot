import { defineStore } from 'pinia'

export interface ConfirmOptions {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

interface ConfirmState {
  open: boolean
  options: Required<ConfirmOptions>
}

let resolveCurrent: ((confirmed: boolean) => void) | null = null

export const useConfirmStore = defineStore('confirm', {
  state: (): ConfirmState => ({
    open: false,
    options: {
      title: '确认操作',
      message: '',
      confirmText: '确认',
      cancelText: '取消',
      danger: false,
    },
  }),
  actions: {
    ask(options: ConfirmOptions): Promise<boolean> {
      resolveCurrent?.(false)
      this.options = {
        title: options.title,
        message: options.message,
        confirmText: options.confirmText ?? '确认',
        cancelText: options.cancelText ?? '取消',
        danger: options.danger ?? false,
      }
      this.open = true
      const { promise, resolve } = Promise.withResolvers<boolean>()
      resolveCurrent = resolve
      return promise
    },
    respond(confirmed: boolean) {
      this.open = false
      resolveCurrent?.(confirmed)
      resolveCurrent = null
    },
  },
})
