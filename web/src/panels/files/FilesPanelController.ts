import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useSessionStore } from '@/stores/session'
import { API_BASE } from '@/utils/constants'
import { getLanguageFromExtension } from '@/utils/languageMap'
import HighlightPreview from '@/components/editor/HighlightPreview.vue'
import AppIcon from '@/components/common/AppIcon.vue'
import type { IconName } from '@/components/common/AppIconController'

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

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function getFileIcon(type: 'file' | 'dir', name: string): IconName {
  if (type === 'dir') return 'folder'
  const ext = name.split('.').pop()?.toLowerCase() || ''
  const iconMap: Record<string, IconName> = {
    ts: 'file-ts', tsx: 'atom', js: 'file-js', jsx: 'atom', vue: 'file-vue',
    py: 'file-py', go: 'cpu', rs: 'file-rs', java: 'coffee', cs: 'cube',
    css: 'file-css', scss: 'file-css', less: 'file-css',
    html: 'globe', htm: 'globe',
    json: 'article', yaml: 'article', yml: 'article', toml: 'article',
    md: 'note', txt: 'file-text', log: 'file-text',
    gitignore: 'gear', dockerfile: 'cube',
    png: 'image', jpg: 'image', jpeg: 'image', gif: 'image', svg: 'image',
    sh: 'terminal', bash: 'terminal', zsh: 'terminal',
    lock: 'lock', config: 'gear',
  }
  return iconMap[ext] || 'file'
}

export function useFilesPanel() {
  const sessionStore = useSessionStore()

  const rootNodes = ref<TreeNode[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const expandedPaths = ref<Set<string>>(new Set())
  const selectedPath = ref<string | null>(null)
  const selectedContent = ref<string | null>(null)
  const selectedIsImage = ref(false)
  const isLoadingContent = ref(false)
  let rootAbortController: AbortController | null = null
  const previewHeight = ref(200)
  const isResizingPreview = ref(false)
  const panelRef = ref<HTMLElement | null>(null)
  let previewRafId = 0

  const selectedLanguage = computed(() => {
    if (!selectedPath.value) return 'plaintext'
    return getLanguageFromExtension(selectedPath.value)
  })
  const isPreviewError = computed(() => {
    return selectedContent.value?.startsWith('[') ?? false
  })
  const hasWorkspace = computed(() => !!sessionStore.workingDirectory)

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

  async function fetchDir(path: string, parent: TreeNode[], signal?: AbortSignal): Promise<void> {
    const url = `${API_BASE}/files?path=${encodeURIComponent(path)}`
    const res = await fetch(url, { signal })
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
    if (rootAbortController) {
      rootAbortController.abort()
    }
    rootAbortController = new AbortController()
    const signal = rootAbortController.signal

    isLoading.value = true
    error.value = null
    try {
      await fetchDir('.', rootNodes.value, signal)
    } catch (e: any) {
      if (e.name === 'AbortError') return
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
        const url = `${API_BASE}/files?path=${encodeURIComponent(node.path)}`
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

  watch(() => sessionStore.workingDirectory, () => {
    if (sessionStore.workingDirectory) {
      expandedPaths.value = new Set()
      selectedPath.value = null
      selectedContent.value = null
      fetchRoot()
    }
  })

  onMounted(() => {
    if (hasWorkspace.value) {
      fetchRoot()
    }
    document.addEventListener('mousemove', onPreviewMouseMove)
    document.addEventListener('mouseup', onPreviewMouseUp)
  })

  onUnmounted(() => {
    document.removeEventListener('mousemove', onPreviewMouseMove)
    document.removeEventListener('mouseup', onPreviewMouseUp)
  })

  return {
    panelRef,
    isLoading,
    fetchRoot,
    hasWorkspace,
    error,
    flatNodes,
    selectedPath,
    handleNodeClick,
    expandedPaths,
    getFileIcon,
    formatFileSize,
    startPreviewResize,
    previewHeight,
    closePreview,
    isLoadingContent,
    selectedIsImage,
    selectedContent,
    isPreviewError,
    selectedLanguage,
  }
}