import { ref } from 'vue'
import { useToastStore } from '../stores/toasts'

export function useFormFeedback() {
  const toast = useToastStore()
  const invalidField = ref('')

  function fail(message: string, fieldId = '') {
    invalidField.value = fieldId
    toast.push(message, 'error', 5000)
    if (!fieldId) return
    window.requestAnimationFrame(() => {
      const element = document.getElementById(fieldId)
      element?.scrollIntoView({ block: 'nearest' })
      element?.focus()
    })
  }

  function clear(fieldId = '') {
    if (!fieldId || invalidField.value === fieldId) {
      invalidField.value = ''
    }
  }

  function success(message: string) {
    toast.push(message, 'success')
  }

  return { invalidField, fail, clear, success }
}
