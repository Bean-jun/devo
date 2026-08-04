import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'

let initialized = false

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function highlightCode(code: string, lang: string): string {
  if (lang && hljs.getLanguage(lang)) {
    return hljs.highlight(code, { language: lang }).value
  }
  return hljs.highlightAuto(code).value
}

function ensureInitialized(): void {
  if (initialized) return
  initialized = true

  const renderer = {
    code({ text, lang }: { text: string; lang?: string }): string {
      const language = (lang || '').trim()
      const langLabel = language ? escapeHtml(language) : 'code'
      const langClass = language ? `hljs language-${language}` : 'hljs'
      // `text` is the raw source; highlight it ourselves
      const highlighted = highlightCode(text, language)
      const dataCode = escapeHtml(text)
      return `<div class="code-block">` +
        `<div class="code-block-header">` +
        `<span class="code-block-lang">${langLabel}</span>` +
        `<button class="code-block-copy" type="button" data-code="${dataCode}" title="复制代码">复制</button>` +
        `</div>` +
        `<pre data-lang="${langLabel}"><code class="${langClass}">${highlighted}</code></pre>` +
        `</div>`
    },
  }

  marked.use({ renderer })
}

const MAX_CACHE_SIZE = 300
const markdownCache = new Map<string, string>()

/**
 * 渲染 Markdown 为 HTML（带缓存）
 */
export function renderMarkdown(content: string): string {
  const cached = markdownCache.get(content)
  if (cached !== undefined) return cached

  ensureInitialized()

  let html = marked.parse(content, { breaks: true }) as string

  // Wrap tables in a scroll container
  html = html.replace(/<table>/g, '<div class="table-wrap"><table>')
  html = html.replace(/<\/table>/g, '</table></div>')

  if (markdownCache.size >= MAX_CACHE_SIZE) {
    const firstKey = markdownCache.keys().next().value
    if (firstKey !== undefined) {
      markdownCache.delete(firstKey)
    }
  }
  markdownCache.set(content, html)

  return html
}

export function clearMarkdownCache(): void {
  markdownCache.clear()
}
