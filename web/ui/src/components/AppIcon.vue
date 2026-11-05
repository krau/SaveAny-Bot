<script setup lang="ts">
withDefaults(
  defineProps<{
    name: string
    size?: number
    strokeWidth?: number
  }>(),
  {
    size: 18,
    strokeWidth: 1.7,
  },
)

const paths: Record<string, string[]> = {
  dashboard: ['M4 4h6v7H4z', 'M14 4h6v4h-6z', 'M14 12h6v8h-6z', 'M4 15h6v5H4z'],
  bookOpen: [
    'M12 7v14',
    'M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3Z',
  ],
  tasks: ['m3 17 2 2 4-4', 'm3 7 2 2 4-4', 'M13 6h8', 'M13 12h8', 'M13 18h8'],
  database: [
    'M12 3c4.97 0 9 1.34 9 3s-4.03 3-9 3-9-1.34-9-3 4.03-3 9-3',
    'M21 12c0 1.66-4.03 3-9 3s-9-1.34-9-3',
    'M3 6v12c0 1.66 4.03 3 9 3s9-1.34 9-3V6',
    'M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3',
  ],
  users: [
    'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2',
    'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8',
    'M22 21v-2a4 4 0 0 0-3-3.87',
    'M16 3.13a4 4 0 0 1 0 7.75',
  ],
  send: ['m22 2-7 20-4-9-9-4Z', 'M22 2 11 13'],
  sliders: [
    'M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3',
    'M2 14h4M10 8h4M18 16h4',
  ],
  server: [
    'M4 3h16a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2',
    'M4 14h16a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2',
    'M6 6.5h.01M6 17.5h.01',
  ],
  refresh: ['M21 12a9 9 0 1 1-3-6.7L21 8', 'M21 3v5h-5'],
  plus: ['M12 5v14M5 12h14'],
  key: [
    'M7.5 20.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9',
    'm10.7 13.3 9.8-9.8',
    'm15.5 7.5 3 3L22 7l-3-3',
  ],
  logout: ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4', 'm16 17 5-5-5-5', 'M21 12H9'],
  menu: ['M4 6h16M4 12h16M4 18h16'],
  close: ['M18 6 6 18M6 6l12 12'],
  chevronDown: ['m6 9 6 6 6-6'],
  arrowUp: ['M12 19V5', 'm5 12 7-7 7 7'],
  chevronRight: ['m9 18 6-6-6-6'],
  arrowRight: ['M5 12h14', 'm12 5 7 7-7 7'],
  play: ['M7 4v16l13-8Z'],
  stop: ['M6 6h12v12H6z'],
  trash: [
    'M3 6h18',
    'M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6',
    'M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2',
    'M10 11v6M14 11v6',
  ],
  edit: ['M17 3a2.85 2.85 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z'],
  plug: ['M12 22v-5', 'M9 8V2M15 8V2', 'M18 8v5a4 4 0 0 1-4 4h-4a4 4 0 0 1-4-4V8Z'],
  check: ['m4 12 5 5L20 6'],
  alert: ['M12 9v4M12 17h.01', 'M10.3 3.7 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.7a2 2 0 0 0-3.4 0Z'],
  inbox: [
    'M3 7l3-4h12l3 4v10l-3 4H6l-3-4Z',
    'M3 11h5l1.5 3h5L16 11h5',
  ],
  file: [
    'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z',
    'M14 2v6h6',
  ],
  download: ['M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4', 'm7 10 5 5 5-5', 'M12 15V3'],
  clock: ['M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20', 'M12 6v6l4 2'],
  folder: ['M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z'],
  cloud: ['M17.5 19H6a4 4 0 0 1 0-8 6 6 0 0 1 11.7-1.7A4.5 4.5 0 0 1 17.5 19Z'],
  hardDrive: [
    'M22 12H2',
    'M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11Z',
    'M6 16h.01',
    'M10 16h.01',
  ],
  home: ['m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z', 'M9 22V12h6v10'],
  sun: [
    'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8',
    'M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.66 17.66l1.41 1.41',
    'M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41',
  ],
  moon: ['M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8Z'],
  monitor: ['M3 4h18v12H3z', 'M8 20h8', 'M12 16v4'],
}
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path v-for="(path, index) in paths[name] ?? []" :key="index" :d="path" />
  </svg>
</template>
