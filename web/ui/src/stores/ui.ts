import { defineStore } from 'pinia'

interface UiState {
  restartRequired: string[]
  warnings: string[]
}

export const useUiStore = defineStore('ui', {
  state: (): UiState => ({
    restartRequired: [],
    warnings: [],
  }),
  actions: {
    setRestartRequired(fields: string[]) {
      this.restartRequired = [...fields]
    },
    setWarnings(fields: string[]) {
      this.warnings = [...fields]
    },
    clearRestartRequired() {
      this.restartRequired = []
    },
  },
})
