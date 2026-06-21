import { marked } from 'marked'
import { markedHighlight } from 'marked-highlight'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'

let initialized = false

function ensureInitialized(): void {
  if (initialized) return
  initialized = true

  marked.use(
    markedHighlight({
      langPrefix: 'hljs language-',
      highlight(code: string, lang: string) {
        if (lang && hljs.getLanguage(lang)) {
          return hljs.highlight(code, { language: lang }).value
        }
        return hljs.highlightAuto(code).value
      },
    })
  )
}

/**
 * 渲染 Markdown 为 HTML
 * 将所有 <pre><code> 块包裹在带有语言标签的容器中
 */
export function renderMarkdown(content: string): string {
  ensureInitialized()

  let html = marked.parse(content, { breaks: true }) as string

  html = html.replace(
    /<pre><code class="hljs language-(\w+)">/g,
    '<pre data-lang="$1"><code class="hljs language-$1">'
  )

  return html
}