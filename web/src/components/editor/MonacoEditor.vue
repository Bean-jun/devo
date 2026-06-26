<template>
  <div ref="containerRef" class="monaco-editor-wrapper" :style="{ height: computedHeight }"></div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick, computed, shallowRef } from 'vue'
import type { editor } from 'monaco-editor'

const props = withDefaults(
  defineProps<{
    content: string
    language?: string
    readonly?: boolean
    theme?: 'vs' | 'vs-dark'
    autoHeight?: boolean
    minHeight?: number
    maxHeight?: number
  }>(),
  {
    language: 'plaintext',
    readonly: true,
    theme: 'vs',
    autoHeight: true,
    minHeight: 100,
    maxHeight: 600,
  }
)

const containerRef = ref<HTMLElement | null>(null)
const editorInstance = shallowRef<editor.IStandaloneCodeEditor | null>(null)
const contentHeight = ref(200)
let monacoModule: typeof import('monaco-editor') | null = null

const computedHeight = computed(() => {
  if (!props.autoHeight) return '100%'
  const h = Math.max(props.minHeight, Math.min(contentHeight.value, props.maxHeight))
  return `${h}px`
})

function setupMonacoEnvironment() {
  ;(self as any).MonacoEnvironment = {
    getWorker(_workerId: string, label: string) {
      const getWorkerModule = (moduleUrl: string, label: string) => {
        return new Worker(moduleUrl, {
          name: label,
          type: 'module',
        })
      }

      switch (label) {
        case 'typescript':
        case 'javascript':
          return getWorkerModule(
            new URL('monaco-editor/esm/vs/language/typescript/ts.worker.js', import.meta.url).href,
            label
          )
        case 'json':
          return getWorkerModule(
            new URL('monaco-editor/esm/vs/language/json/json.worker.js', import.meta.url).href,
            label
          )
        case 'css':
        case 'scss':
        case 'less':
          return getWorkerModule(
            new URL('monaco-editor/esm/vs/language/css/css.worker.js', import.meta.url).href,
            label
          )
        case 'html':
        case 'handlebars':
        case 'razor':
          return getWorkerModule(
            new URL('monaco-editor/esm/vs/language/html/html.worker.js', import.meta.url).href,
            label
          )
        case 'editorWorkerService':
          return getWorkerModule(
            new URL('monaco-editor/esm/vs/editor/editor.worker.js', import.meta.url).href,
            label
          )
        default:
          return getWorkerModule(
            new URL('monaco-editor/esm/vs/editor/editor.worker.js', import.meta.url).href,
            label
          )
      }
    },
  }
}

async function initEditor() {
  if (!containerRef.value) return

  setupMonacoEnvironment()

  const monaco = await import('monaco-editor')
  monacoModule = monaco

  const model = monaco.editor.createModel(props.content, props.language)

  const ed = monaco.editor.create(containerRef.value, {
    model,
    readOnly: props.readonly,
    theme: props.theme,
    automaticLayout: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    lineNumbers: 'on',
    renderWhitespace: 'selection',
    tabSize: 2,
    fontSize: 13,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
    padding: { top: 8, bottom: 8 },
    overviewRulerBorder: false,
    hideCursorInOverviewRuler: true,
    scrollbar: {
      vertical: 'auto',
      horizontal: 'auto',
      verticalScrollbarSize: 8,
      horizontalScrollbarSize: 8,
    },
    contextmenu: false,
    quickSuggestions: false,
    parameterHints: { enabled: false },
    hover: { enabled: false },
    codeLens: false,
    glyphMargin: false,
    folding: true,
    lineDecorationsWidth: 0,
    lineNumbersMinChars: 3,
  })

  editorInstance.value = ed

  updateContentHeight()

  ed.onDidContentSizeChange(() => {
    updateContentHeight()
  })
}

function updateContentHeight() {
  const ed = editorInstance.value
  if (!ed || !props.autoHeight) return
  const model = ed.getModel()
  if (!model) return
  const lineCount = model.getLineCount()
  const lineHeight = ed.getOption(monacoModule!.editor.EditorOption.lineHeight) as number
  const padding = 16
  contentHeight.value = lineCount * lineHeight + padding
}

watch(
  () => props.content,
  (newContent) => {
    const ed = editorInstance.value
    if (ed) {
      const model = ed.getModel()
      if (model && model.getValue() !== newContent) {
        model.setValue(newContent)
      }
    }
  }
)

watch(
  () => props.language,
  (newLang) => {
    const ed = editorInstance.value
    if (ed && monacoModule) {
      const model = ed.getModel()
      if (model) {
        monacoModule.editor.setModelLanguage(model, newLang)
      }
    }
  }
)

watch(
  () => props.theme,
  (newTheme) => {
    if (monacoModule) {
      monacoModule.editor.setTheme(newTheme)
    }
  }
)

onMounted(() => {
  nextTick(() => {
    initEditor()
  })
})

onBeforeUnmount(() => {
  const ed = editorInstance.value
  if (ed) {
    ed.dispose()
    editorInstance.value = null
  }
})
</script>

<style scoped>
.monaco-editor-wrapper {
  width: 100%;
  border-radius: 4px;
  overflow: hidden;
}
</style>