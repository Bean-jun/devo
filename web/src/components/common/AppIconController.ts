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
  PhDotsThree,
  PhLightbulb,
  PhArrowUp,
  PhArrowDown,
  PhLockOpen,
  PhPuzzlePiece,
  PhTag,
  PhFile,
  PhFileText,
  PhPalette,
  PhArticle,
  PhAtom,
  PhCode,
  PhTerminal,
  PhImage,
  PhLock,
  PhGlobe,
  PhDatabase,
  PhArrowArcLeft,
  PhArrowsClockwise,
  PhRobot,
  PhCoffee,
  PhDog,
  PhCpu,
  PhVinylRecord,
  PhFileArrowUp,
  PhFileArrowDown,
  PhCircleHalf,
  PhFileCode,
  PhFileCss,
  PhFileJs,
  PhFileTs,
  PhFileVue,
  PhFileRs,
  PhFilePy,
  PhCube,
  PhArrowCircleUp,
  PhArrowSquareOut,
  PhArrowRight,
} from '@phosphor-icons/vue'

export interface AppIconProps {
  name: IconName
  size?: number | string
  weight?: 'thin' | 'light' | 'regular' | 'bold' | 'fill' | 'duotone'
  color?: string
}

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
  'dots-three': PhDotsThree,
  lightbulb: PhLightbulb,
  'arrow-up': PhArrowUp,
  'arrow-down': PhArrowDown,
  'lock-open': PhLockOpen,
  'puzzle-piece': PhPuzzlePiece,
  tag: PhTag,
  file: PhFile,
  'file-text': PhFileText,
  palette: PhPalette,
  article: PhArticle,
  atom: PhAtom,
  code: PhCode,
  terminal: PhTerminal,
  image: PhImage,
  lock: PhLock,
  globe: PhGlobe,
  database: PhDatabase,
  'arrow-arc-left': PhArrowArcLeft,
  'arrows-clockwise': PhArrowsClockwise,
  robot: PhRobot,
  coffee: PhCoffee,
  dog: PhDog,
  cpu: PhCpu,
  'vinyl-record': PhVinylRecord,
  'file-arrow-up': PhFileArrowUp,
  'file-arrow-down': PhFileArrowDown,
  'circle-half': PhCircleHalf,
  'file-code': PhFileCode,
  'file-css': PhFileCss,
  'file-js': PhFileJs,
  'file-ts': PhFileTs,
  'file-vue': PhFileVue,
  'file-rs': PhFileRs,
  'file-py': PhFilePy,
  cube: PhCube,
  'arrow-circle-up': PhArrowCircleUp,
  'arrow-square-out': PhArrowSquareOut,
  'arrow-right': PhArrowRight,
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
  | 'dots-three'
  | 'lightbulb'
  | 'arrow-up'
  | 'arrow-down'
  | 'lock-open'
  | 'puzzle-piece'
  | 'tag'
  | 'file'
  | 'file-text'
  | 'palette'
  | 'article'
  | 'atom'
  | 'code'
  | 'terminal'
  | 'image'
  | 'lock'
  | 'globe'
  | 'database'
  | 'arrow-arc-left'
  | 'arrows-clockwise'
  | 'robot'
  | 'coffee'
  | 'dog'
  | 'cpu'
  | 'vinyl-record'
  | 'file-arrow-up'
  | 'file-arrow-down'
  | 'circle-half'
  | 'file-code'
  | 'file-css'
  | 'file-js'
  | 'file-ts'
  | 'file-vue'
  | 'file-rs'
  | 'file-py'
  | 'cube'
  | 'arrow-circle-up'
  | 'arrow-square-out'
  | 'arrow-right'

export function useAppIcon(props: AppIconProps) {
  const resolvedIcon = computed(() => iconMap[props.name])

  const iconSize = computed(() => {
    if (typeof props.size === 'number') return props.size
    return undefined
  })

  return { resolvedIcon, iconSize }
}