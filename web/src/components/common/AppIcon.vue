<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import {
  PhFolder,
  PhFolderOpen,
  PhTrash,
  PhChatCircle,
  PhChatCircleDots,
  PhX,
  PhCaretRight,
  PhCaretLeft,
  PhCaretDown,
  PhCaretUp,
  PhPlus,
  PhSun,
  PhMoon,
  PhFire,
  PhHourglass,
  PhCircle,
  PhCheckCircle,
  PhXCircle,
  PhInfo,
  PhWarning,
  PhCopy,
  PhClipboard,
  PhClock,
  PhArrowClockwise,
  PhProhibit,
  PhWrench,
  PhStop,
  PhLightning,
  PhPlug,
  PhBrain,
  PhChartBar,
  PhGear,
  PhUser,
  PhNote,
  PhPencil,
  PhPause,
  PhSpinner,
  PhQuestion,
  PhCheck,
  PhMagnifyingGlass,
  PhSliders,
} from '@phosphor-icons/vue'

const props = withDefaults(defineProps<{
  name: IconName
  size?: number | string
  weight?: 'thin' | 'light' | 'regular' | 'bold' | 'fill' | 'duotone'
  color?: string
}>(), {
  size: 18,
  weight: 'regular',
})

const iconMap: Record<IconName, Component> = {
  folder: PhFolder,
  'folder-open': PhFolderOpen,
  trash: PhTrash,
  chat: PhChatCircle,
  'chat-dots': PhChatCircleDots,
  x: PhX,
  'caret-right': PhCaretRight,
  'caret-left': PhCaretLeft,
  'caret-down': PhCaretDown,
  'caret-up': PhCaretUp,
  plus: PhPlus,
  sun: PhSun,
  moon: PhMoon,
  fire: PhFire,
  hourglass: PhHourglass,
  circle: PhCircle,
  'check-circle': PhCheckCircle,
  'x-circle': PhXCircle,
  info: PhInfo,
  warning: PhWarning,
  copy: PhCopy,
  clipboard: PhClipboard,
  clock: PhClock,
  'arrow-clockwise': PhArrowClockwise,
  prohibit: PhProhibit,
  wrench: PhWrench,
  stop: PhStop,
  lightning: PhLightning,
  plug: PhPlug,
  brain: PhBrain,
  'chart-bar': PhChartBar,
  gear: PhGear,
  user: PhUser,
  note: PhNote,
  pencil: PhPencil,
  pause: PhPause,
  spinner: PhSpinner,
  question: PhQuestion,
  check: PhCheck,
  'magnifying-glass': PhMagnifyingGlass,
  sliders: PhSliders,
}

export type IconName =
  | 'folder'
  | 'folder-open'
  | 'trash'
  | 'chat'
  | 'chat-dots'
  | 'x'
  | 'caret-right'
  | 'caret-left'
  | 'caret-down'
  | 'caret-up'
  | 'plus'
  | 'sun'
  | 'moon'
  | 'fire'
  | 'hourglass'
  | 'circle'
  | 'check-circle'
  | 'x-circle'
  | 'info'
  | 'warning'
  | 'copy'
  | 'clipboard'
  | 'clock'
  | 'arrow-clockwise'
  | 'prohibit'
  | 'wrench'
  | 'stop'
  | 'lightning'
  | 'plug'
  | 'brain'
  | 'chart-bar'
  | 'gear'
  | 'user'
  | 'note'
  | 'pencil'
  | 'pause'
  | 'spinner'
  | 'question'
  | 'check'
  | 'magnifying-glass'
  | 'sliders'

const resolvedIcon = computed(() => iconMap[props.name])

const iconSize = computed(() => {
  if (typeof props.size === 'number') return props.size
  return undefined
})
</script>

<template>
  <component
    :is="resolvedIcon"
    :size="iconSize"
    :weight="weight"
    :color="color"
    :style="typeof size === 'string' ? { width: size, height: size } : undefined"
  />
</template>