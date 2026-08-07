import { computed, ref, onMounted, watch, nextTick } from 'vue'
import hljs from 'highlight.js'

export interface HighlightPreviewProps {
  content: string
  language?: string
  autoHeight?: boolean
  minHeight: number
  maxHeight: number
}

export function useHighlightPreview(props: HighlightPreviewProps) {
  const wrapperRef = ref<HTMLElement | null>(null)
  const contentHeight = ref(200)

function normalizeLang(lang: string): string {
  const map: Record<string, string> = {
    shell: 'bash',
    bat: 'dos',
    powershell: 'powershell',
  }
  return map[lang] || lang
}

const highlightedHtml = computed(() => {
  if (!props.content) return ''
  const lang = normalizeLang(props.language!)
  if (lang && lang !== 'plaintext' && hljs.getLanguage(lang)) {
    return hljs.highlight(props.content, { language: lang }).value
  }
  return hljs.highlightAuto(props.content).value
})

const wrapperStyle = computed(() => {
  if (!props.autoHeight) return {}
  const h = Math.max(props.minHeight, Math.min(contentHeight.value, props.maxHeight))
  return { height: `${h}px` }
})

function updateHeight() {
  if (!props.autoHeight || !wrapperRef.value) return
  const lineCount = Math.max(1, props.content.split('\n').length)
  const lineHeight = 18
  const padding = 20
  contentHeight.value = lineCount * lineHeight + padding
}

watch(() => props.content, () => {
  nextTick(() => updateHeight())
})

onMounted(() => {
  nextTick(() => updateHeight())
})

  return { wrapperRef, highlightedHtml, wrapperStyle }
}