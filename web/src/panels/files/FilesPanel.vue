<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'
import { getLanguageFromExtension } from '@/utils/languageMap'
import MonacoEditor from '@/components/editor/MonacoEditor.vue'

const sessionStore = useSessionStore()
const uiStore = useUiStore()

interface TreeNode {
  name: string
  path: string
  type: 'file' | 'dir'
  size: number
  children: TreeNode[]
  loaded: boolean
  loading: boolean
  depth: number
}

const rootNodes = ref<TreeNode[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const expandedPaths = ref<Set<string>>(new Set())
const selectedPath = ref<string | null>(null)
const selectedContent = ref<string | null>(null)
const selectedIsImage = ref(false)
const isLoadingContent = ref(false)
const previewHeight = ref(200)
const isResizingPreview = ref(false)
const panelRef = ref<HTMLElement | null>(null)
let previewRafId = 0

const monacoTheme = computed(() => (uiStore.theme === 'dark' ? 'vs-dark' : 'vs') as 'vs' | 'vs-dark')
const selectedLanguage = computed(() => {
  if (!selectedPath.value) return 'plaintext'
  return getLanguageFromExtension(selectedPath.value)
})
const isPreviewError = computed(() => {
  return selectedContent.value?.startsWith('[') ?? false
})

function startPreviewResize(e: MouseEvent) {
  isResizingPreview.value = true
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
  e.preventDefault()
}

function onPreviewMouseMove(e: MouseEvent) {
  if (!isResizingPreview.value) return
  cancelAnimationFrame(previewRafId)
  previewRafId = requestAnimationFrame(() => {
    if (!panelRef.value) return
    const rect = panelRef.value.getBoundingClientRect()
    const h = rect.bottom - e.clientY
    previewHeight.value = Math.max(80, Math.min(600, h))
  })
}

function onPreviewMouseUp() {
  if (!isResizingPreview.value) return
  isResizingPreview.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

function closePreview() {
  selectedPath.value = null
  selectedContent.value = null
  previewHeight.value = 200
}

const hasSession = computed(() => !!sessionStore.currentSession?.id)

const MAX_PREVIEW_SIZE = 10 * 1024 * 1024
const MAX_TEXT_FALLBACK_SIZE = 1 * 1024 * 1024

const PREVIEWABLE_EXTENSIONS = new Set([
  'png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico',
  'ts', 'tsx', 'js', 'jsx', 'mjs', 'cjs', 'vue', 'svelte', 'astro', 'solid',
  'py', 'pyi', 'pyx', 'go', "mod", "sum", 'rs', 'java', 'kt', 'kts', 'scala', 'cs', 'fs', 'fsx', 'vb',
  'c', 'cpp', 'cxx', 'cc', 'h', 'hpp', 'hxx', 'hh',
  'css', 'scss', 'less', 'sass', 'styl', 'pcss',
  'html', 'htm', 'xml', 'xhtml', 'mjml',
  'json', 'jsonc', 'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf', 'config',
  'md', 'mdx', 'txt', 'log', 'text', 'rst', 'rest', 'adoc', 'asciidoc', 'org', 'pod', 'tex', 'bib',
  'sh', 'bash', 'zsh', 'fish', 'ps1', 'bat', 'cmd', 'psm1', 'psd1',
  'sql', 'graphql', 'gql', 'proto', 'prisma',
  'env', 'envrc', 'gitignore', 'gitattributes', 'gitmodules', 'editorconfig',
  'dockerfile', 'dockerignore', 'makefile', 'justfile', 'earthfile', 'containerfile',
  'rb', 'erb', 'rake', 'gemspec', 'php', 'phtml',
  'swift', 'lua', 'r', 'rmd', 'rnw', 'jl', 'dart',
  'ex', 'exs', 'eex', 'heex', 'leex', 'elm', 'hs', 'lhs', 'ml', 'mli',
  'clj', 'cljs', 'cljc', 'edn', 'zig', 'nim', 'nims', 'v', 'cr',
  'pl', 'pm', 't', 'raku', 'p6', 'nqp',
  'coffee', 'litcoffee', 'pug', 'jade', 'ejs', 'haml', 'slim',
  'tf', 'tfvars', 'hcl', 'nomad',
  'gradle', 'groovy',
  'res', 'resi', 'purs', 'dhall', 'nix', 'cabal',
  's', 'asm', 'nasm', 'wat', 'wast',
  'f', 'f90', 'f95', 'f03', 'f08', 'for',
  'cob', 'cbl', 'cpy',
  'diff', 'patch', 'lock',
  'csv', 'tsv', 'psv',
  'puml', 'plantuml', 'wsd',
  'nginx', 'htaccess', 'apache',
  'cmake', 'meson', 'bazel', 'bzl', 'build',
  'eslintrc', 'prettierrc', 'babelrc', 'stylelintrc', 'npmrc', 'yarnrc',
  'properties', 'prop',
  'xsl', 'xslt', 'xsd', 'dtd', 'wsdl',
  're', 'rei',
  'sas', 'stata', 'do', 'ado',
  'matlab', 'm', 'octave',
  'scheme', 'scm', 'ss', 'rkt',
  'lisp', 'lsp', 'fasl',
  'sc', 'scd', 'supercollider',
  'pde', 'ino', 'odin', 'pony', 'move',
  'smithy', 'cue', 'wgsl', 'slint', 'wit',
])

const IMAGE_EXTENSIONS = new Set([
  'png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico',
])

function isPreviewable(name: string): boolean {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return PREVIEWABLE_EXTENSIONS.has(ext)
}

function isImage(name: string): boolean {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return IMAGE_EXTENSIONS.has(ext)
}

function hasNoExtension(name: string): boolean {
  const dotIndex = name.lastIndexOf('.')
  if (dotIndex <= 0) return true
  const ext = name.slice(dotIndex + 1).toLowerCase()
  return ext === '' || ext === name.toLowerCase()
}

function getPreviewDeniedReason(node: TreeNode): string | null {
  if (node.size > MAX_PREVIEW_SIZE) {
    return `文件过大 (${formatFileSize(node.size)})，无法预览`
  }
  if (isPreviewable(node.name)) {
    return null
  }
  if (hasNoExtension(node.name) && node.size <= MAX_TEXT_FALLBACK_SIZE) {
    return null
  }
  return '此文件类型不支持预览'
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function getFileIcon(type: 'file' | 'dir', name: string): string {
  if (type === 'dir') return '📁'
  const ext = name.split('.').pop()?.toLowerCase() || ''
  const iconMap: Record<string, string> = {
    ts: '🔷', tsx: '⚛️', js: '🟨', jsx: '⚛️', vue: '💚',
    py: '🐍', go: '🔵', rs: '🦀', java: '☕', cs: '🟣',
    css: '🎨', scss: '🎨', less: '🎨',
    html: '🌐', htm: '🌐',
    json: '📋', yaml: '📋', yml: '📋', toml: '📋',
    md: '📝', txt: '📄', log: '📄',
    gitignore: '⚙', dockerfile: '🐳',
    png: '🖼', jpg: '🖼', jpeg: '🖼', gif: '🖼', svg: '🖼',
    sh: '💻', bash: '💻', zsh: '💻',
    lock: '🔒', config: '⚙',
  }
  return iconMap[ext] || '📄'
}

function flattenTree(nodes: TreeNode[], depth: number): TreeNode[] {
  const result: TreeNode[] = []
  for (const node of nodes) {
    node.depth = depth
    result.push(node)
    if (node.type === 'dir' && expandedPaths.value.has(node.path) && node.loaded && node.children.length > 0) {
      result.push(...flattenTree(node.children, depth + 1))
    }
  }
  return result
}

const flatNodes = computed<TreeNode[]>(() => flattenTree(rootNodes.value, 0))

async function fetchDir(path: string, parent: TreeNode[]): Promise<void> {
  const sessionId = sessionStore.currentSession?.id
  if (!sessionId) return

  const url = `${API_BASE}/sessions/${sessionId}/files?path=${encodeURIComponent(path)}`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const data = await res.json()

  if (data.type !== 'dir') throw new Error('Not a directory')

  const entries: { name: string; type: 'file' | 'dir'; size: number }[] = data.entries || []
  const sorted = [...entries].sort((a, b) => {
    if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
    return a.name.localeCompare(b.name)
  })

  parent.length = 0
  for (const entry of sorted) {
    parent.push({
      name: entry.name,
      path: path === '.' ? entry.name : `${path}/${entry.name}`,
      type: entry.type,
      size: entry.size,
      children: [],
      loaded: false,
      loading: false,
      depth: 0,
    })
  }
}

async function fetchRoot() {
  isLoading.value = true
  error.value = null
  try {
    await fetchDir('.', rootNodes.value)
  } catch (e: any) {
    error.value = e.message || '加载文件列表失败'
  } finally {
    isLoading.value = false
  }
}

async function handleNodeClick(node: TreeNode) {
  if (node.type === 'dir') {
    if (expandedPaths.value.has(node.path)) {
      expandedPaths.value.delete(node.path)
      expandedPaths.value = new Set(expandedPaths.value)
    } else {
      expandedPaths.value = new Set([...expandedPaths.value, node.path])
      if (!node.loaded) {
        node.loading = true
        try {
          await fetchDir(node.path, node.children)
          node.loaded = true
        } catch (e: any) {
          error.value = e.message || '加载失败'
        } finally {
          node.loading = false
        }
      }
    }
  } else {
    selectedPath.value = node.path
    selectedContent.value = null
    selectedIsImage.value = false
    const reason = getPreviewDeniedReason(node)
    if (reason) {
      selectedContent.value = `[${reason}]`
      return
    }
    isLoadingContent.value = true
    try {
      const sessionId = sessionStore.currentSession?.id
      if (!sessionId) return
      const url = `${API_BASE}/sessions/${sessionId}/files?path=${encodeURIComponent(node.path)}`
      const res = await fetch(url)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      selectedContent.value = data.content || ''
      selectedIsImage.value = data.is_image || false
    } catch (e: any) {
      selectedContent.value = `[读取失败: ${e.message}]`
    } finally {
      isLoadingContent.value = false
    }
  }
}

watch(() => sessionStore.currentSession?.id, () => {
  if (sessionStore.currentSession?.id) {
    expandedPaths.value = new Set()
    selectedPath.value = null
    selectedContent.value = null
    fetchRoot()
  }
})

onMounted(() => {
  if (hasSession.value) {
    fetchRoot()
  }
  document.addEventListener('mousemove', onPreviewMouseMove)
  document.addEventListener('mouseup', onPreviewMouseUp)
})

onUnmounted(() => {
  document.removeEventListener('mousemove', onPreviewMouseMove)
  document.removeEventListener('mouseup', onPreviewMouseUp)
})
</script>

<template>
  <div ref="panelRef" class="files-panel">
    <div class="files-toolbar">
      <span class="files-title">Files</span>
      <button class="refresh-btn" @click="fetchRoot" :disabled="isLoading" title="刷新">
        {{ isLoading ? '⏳' : '🔄' }}
      </button>
    </div>

    <div v-if="!hasSession" class="files-empty">请先创建或选择一个会话</div>
    <div v-else-if="error" class="files-error">{{ error }}</div>
    <div v-else class="files-body">
      <div class="files-tree">
        <div v-if="isLoading" class="files-loading">加载中...</div>
        <div v-else-if="flatNodes.length === 0" class="files-empty">目录为空</div>
        <div
          v-for="node in flatNodes"
          :key="node.path"
          class="file-row"
          :class="{ selected: selectedPath === node.path }"
          :style="{
            paddingLeft: (8 + node.depth * 22) + 'px',
            '--depth': node.depth,
          }"
          @click="handleNodeClick(node)"
        >
          <span
            v-for="d in node.depth"
            :key="d"
            class="tree-line"
            :style="{ left: (8 + (d - 1) * 22 + 8) + 'px' }"
          ></span>
          <span v-if="node.type === 'dir'" class="file-chevron">{{ expandedPaths.has(node.path) ? '▾' : '▸' }}</span>
          <span class="file-icon">{{ getFileIcon(node.type, node.name) }}</span>
          <span class="file-name">{{ node.name }}</span>
          <span v-if="node.type === 'dir' && !node.loaded && expandedPaths.has(node.path)" class="file-loading-icon">⏳</span>
          <span class="file-size" v-if="node.type === 'file'">{{ formatFileSize(node.size) }}</span>
        </div>
      </div>

      <div
        v-if="selectedPath"
        class="preview-resize-handle"
        @mousedown="startPreviewResize"
      ></div>
      <div v-if="selectedPath" class="file-preview" :style="{ height: previewHeight + 'px' }">
        <div class="preview-header">
          <span class="preview-title">{{ selectedPath }}</span>
          <button class="preview-close" @click="closePreview">✕</button>
        </div>
        <div class="preview-body">
          <div v-if="isLoadingContent" class="preview-loading">加载中...</div>
          <img v-else-if="selectedIsImage" :src="selectedContent ?? ''" class="preview-image" />
          <pre v-else-if="isPreviewError" class="preview-content"><code>{{ selectedContent }}</code></pre>
          <MonacoEditor
            v-else
            :content="selectedContent ?? ''"
            :language="selectedLanguage"
            :theme="monacoTheme"
            :readonly="true"
            :auto-height="true"
            :min-height="80"
            :max-height="600"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.files-panel { display: flex; flex-direction: column; height: 100%; }
.files-toolbar { display: flex; align-items: center; justify-content: space-between; padding: 5px 10px; border-bottom: 1px solid var(--color-border); }
.files-title { font-size: 11px; font-weight: 600; color: var(--color-text-secondary); }
.refresh-btn { background: none; border: none; cursor: pointer; font-size: 13px; padding: 1px 3px; border-radius: 4px; }
.refresh-btn:hover { background: var(--color-bg-hover); }

.files-body { flex: 1; display: flex; flex-direction: column; min-height: 0; overflow: hidden; }
.files-tree { flex: 1; overflow-y: auto; padding: 2px 0; }
.files-loading { padding: 12px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.files-error { padding: 12px; color: var(--color-error); font-size: 12px; }
.files-empty { padding: 12px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }

.file-row { display: flex; align-items: center; gap: 4px; padding: 3px 12px; cursor: pointer; font-size: 12px; position: relative; }
.file-row:hover { background: var(--color-bg-hover); }
.file-row.selected { background: var(--color-accent); color: white; }
.file-row.selected .file-size { color: rgba(255,255,255,0.7); }
.file-row.selected .file-chevron { color: rgba(255,255,255,0.8); }
.tree-line { position: absolute; top: 0; bottom: 0; width: 1px; background: var(--color-border); pointer-events: none; }
.file-chevron { font-size: 9px; width: 12px; flex-shrink: 0; color: var(--color-text-tertiary); text-align: center; }
.file-icon { font-size: 13px; flex-shrink: 0; width: 18px; text-align: center; }
.file-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.file-size { font-size: 10px; color: var(--color-text-tertiary); flex-shrink: 0; margin-left: auto; }
.file-loading-icon { font-size: 10px; flex-shrink: 0; }

.preview-resize-handle { height: 5px; cursor: row-resize; background: var(--color-border); flex-shrink: 0; position: relative; }
.preview-resize-handle::after { content: ''; position: absolute; inset: -3px 0; }
.preview-resize-handle:hover { background: var(--color-accent); }

.file-preview { border-top: none; background: var(--color-bg-primary); display: flex; flex-direction: column; flex-shrink: 0; }
.preview-header { display: flex; align-items: center; justify-content: space-between; padding: 5px 10px; background: var(--color-bg-secondary); border-bottom: 1px solid var(--color-border); }
.preview-title { font-size: 11px; font-weight: 600; color: var(--color-text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.preview-close { background: none; border: none; color: var(--color-text-secondary); cursor: pointer; font-size: 13px; padding: 1px 3px; }
.preview-close:hover { color: var(--color-error); }
.preview-body { flex: 1; overflow: auto; min-height: 0; }
.preview-loading { padding: 12px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.preview-content { padding: 10px; margin: 0; font-family: var(--font-mono); font-size: 12px; line-height: 1.5; color: var(--color-text-primary); white-space: pre-wrap; word-break: break-all; }

.preview-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  display: block;
  margin: 0 auto;
  background: var(--color-bg-secondary);
}
</style>