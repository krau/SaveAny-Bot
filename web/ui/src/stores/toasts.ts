import { defineStore } from 'pinia'

export type ToastTone = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  tone: ToastTone
  message: string
}

let nextId = 1
const timers = new Map<number, number>()

export const useToastStore = defineStore('toasts', {
  state: () => ({
    items: [] as ToastItem[],
  }),
  actions: {
    push(message: string, tone: ToastTone = 'info', duration = 4200): number {
      const id = nextId++
      this.items.push({ id, tone, message })
      if (this.items.length > 4) {
        this.dismiss(this.items[0].id)
      }
      if (duration > 0) {
        timers.set(
          id,
          window.setTimeout(() => this.dismiss(id), duration),
        )
      }
      return id
    },
    dismiss(id: number) {
      const timer = timers.get(id)
      if (timer !== undefined) {
        window.clearTimeout(timer)
        timers.delete(id)
      }
      this.items = this.items.filter((item) => item.id !== id)
    },
    clear() {
      for (const timer of timers.values()) {
        window.clearTimeout(timer)
      }
      timers.clear()
      this.items = []
    },
  },
})
