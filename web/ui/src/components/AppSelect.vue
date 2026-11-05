<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppIcon from './AppIcon.vue'

export interface SelectOption {
  value: string
  label: string
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue: string
    options: SelectOption[]
    placeholder?: string
    disabled?: boolean
    id?: string
    ariaLabel?: string
  }>(),
  {
    placeholder: '',
    disabled: false,
    id: '',
    ariaLabel: '',
  },
)

const { t } = useI18n()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()

let nextSelectId = 0
const uid = `app-select-${++nextSelectId}`
const triggerId = computed(() => props.id || `${uid}-trigger`)
const listboxId = `${uid}-listbox`
const trigger = ref<HTMLButtonElement | null>(null)
const menu = ref<HTMLElement | null>(null)
const open = ref(false)
const activeIndex = ref(-1)
const menuStyle = ref<Record<string, string>>({})
const menuPlacement = ref<'top' | 'bottom'>('bottom')

const selectedIndex = computed(() => props.options.findIndex((option) => option.value === props.modelValue))
const selected = computed(() => (selectedIndex.value >= 0 ? props.options[selectedIndex.value] : null))
const display = computed(() => selected.value?.label || (props.placeholder || t('common.select')))
const enabledIndices = computed(() =>
  props.options.map((option, index) => ({ option, index })).filter((entry) => !entry.option.disabled).map((entry) => entry.index),
)

function optionId(index: number): string {
  return `${uid}-option-${index}`
}

function updatePosition() {
  if (!trigger.value) return
  const rect = trigger.value.getBoundingClientRect()
  const estimatedHeight = Math.min(props.options.length * 34 + 10, 260)
  const spaceBelow = window.innerHeight - rect.bottom
  const spaceAbove = rect.top
  const width = Math.max(rect.width, 160)
  const left = Math.min(Math.max(8, rect.left), window.innerWidth - width - 8)
  const available = spaceBelow < estimatedHeight && spaceAbove > spaceBelow ? spaceAbove : spaceBelow
  const style: Record<string, string> = {
    left: `${left}px`,
    width: `${width}px`,
    maxHeight: `${Math.max(120, Math.min(260, available - 12))}px`,
  }
  if (spaceBelow < estimatedHeight && spaceAbove > spaceBelow) {
    style.bottom = `${window.innerHeight - rect.top + 6}px`
    menuPlacement.value = 'top'
  } else {
    style.top = `${rect.bottom + 6}px`
    menuPlacement.value = 'bottom'
  }
  menuStyle.value = style
}

function openMenu() {
  if (props.disabled) return
  updatePosition()
  activeIndex.value = selectedIndex.value >= 0 ? selectedIndex.value : (enabledIndices.value[0] ?? -1)
  open.value = true
  void nextTick(updatePosition)
}

function closeMenu(focusTrigger = false) {
  open.value = false
  if (focusTrigger) trigger.value?.focus()
}

function toggle() {
  if (open.value) closeMenu()
  else openMenu()
}

function choose(index: number) {
  const option = props.options[index]
  if (!option || option.disabled) return
  if (option.value !== props.modelValue) {
    emit('update:modelValue', option.value)
    emit('change', option.value)
  }
  closeMenu(true)
}

function move(step: 1 | -1) {
  if (enabledIndices.value.length === 0) return
  const current = enabledIndices.value.indexOf(activeIndex.value)
  const next =
    current === -1
      ? step > 0
        ? 0
        : enabledIndices.value.length - 1
      : (current + step + enabledIndices.value.length) % enabledIndices.value.length
  activeIndex.value = enabledIndices.value[next]
}

function onKeydown(event: KeyboardEvent) {
  if (props.disabled) return
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      if (open.value) move(1)
      else openMenu()
      break
    case 'ArrowUp':
      event.preventDefault()
      if (open.value) move(-1)
      else openMenu()
      break
    case 'Home':
      if (open.value) {
        event.preventDefault()
        activeIndex.value = enabledIndices.value[0] ?? -1
      }
      break
    case 'End':
      if (open.value) {
        event.preventDefault()
        activeIndex.value = enabledIndices.value[enabledIndices.value.length - 1] ?? -1
      }
      break
    case 'Enter':
    case ' ':
      event.preventDefault()
      if (open.value) choose(activeIndex.value)
      else openMenu()
      break
    case 'Escape':
      if (open.value) {
        event.preventDefault()
        closeMenu(true)
      }
      break
    case 'Tab':
      if (open.value) closeMenu()
      break
  }
}

function onOutsidePointer(event: PointerEvent) {
  if (!open.value) return
  const target = event.target as Node
  if (menu.value?.contains(target) || trigger.value?.contains(target)) return
  closeMenu()
}

function closeOnViewportChange(event?: Event) {
  if (!open.value) return
  if (event?.target instanceof Node && menu.value?.contains(event.target)) return
  closeMenu()
}

watch(
  () => props.modelValue,
  () => {
    if (!open.value) return
    activeIndex.value = selectedIndex.value >= 0 ? selectedIndex.value : (enabledIndices.value[0] ?? -1)
    updatePosition()
  },
)

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) closeMenu()
  },
)

onMounted(() => {
  document.addEventListener('pointerdown', onOutsidePointer)
  window.addEventListener('resize', closeOnViewportChange)
  window.addEventListener('scroll', closeOnViewportChange, true)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onOutsidePointer)
  window.removeEventListener('resize', closeOnViewportChange)
  window.removeEventListener('scroll', closeOnViewportChange, true)
})
</script>

<template>
  <div class="select">
    <button
      ref="trigger"
      :id="triggerId"
      type="button"
      class="select-trigger"
      :disabled="disabled"
      :aria-expanded="open"
      :aria-label="ariaLabel || undefined"
      aria-haspopup="listbox"
      :aria-controls="open ? listboxId : undefined"
      :aria-activedescendant="open && activeIndex >= 0 ? optionId(activeIndex) : undefined"
      @click="toggle"
      @keydown="onKeydown"
    >
      <span class="select-value" :class="{ placeholder: !selected }">{{ display }}</span>
      <AppIcon class="select-chevron" name="chevronDown" :size="15" />
    </button>

    <Teleport to="body">
      <Transition name="select-menu">
        <ul
          v-if="open"
          ref="menu"
          :id="listboxId"
          class="select-menu"
          role="listbox"
          :aria-labelledby="triggerId"
          :data-placement="menuPlacement"
          :style="menuStyle"
          @mousedown.prevent
        >
          <li
            v-for="(option, index) in options"
            :id="optionId(index)"
            :key="option.value"
            role="option"
            class="select-option"
            :class="{ active: activeIndex === index, selected: selectedIndex === index, disabled: option.disabled }"
            :aria-selected="selectedIndex === index"
            :aria-disabled="option.disabled || undefined"
            @mouseenter="!option.disabled && (activeIndex = index)"
            @click="choose(index)"
          >
            <span>{{ option.label }}</span>
            <AppIcon v-if="selectedIndex === index" name="check" :size="14" />
          </li>
        </ul>
      </Transition>
    </Teleport>
  </div>
</template>
