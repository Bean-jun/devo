import { ref, type Ref } from 'vue'

export type TextSegment = { kind: 'text'; value: string }
export type PasteSegment = { kind: 'paste'; id: number; content: string; lineCount: number }
export type Segment = TextSegment | PasteSegment

export const PASTE_CHIP_THRESHOLD_LINES = 4
export const PASTE_CHIP_THRESHOLD_CHARS = 200

export function shouldFoldPaste(pasted: string): boolean {
  if (pasted.length > PASTE_CHIP_THRESHOLD_CHARS) return true
  if (pasted.split('\n').length > PASTE_CHIP_THRESHOLD_LINES) return true
  return false
}

export function useInputSegments() {
  const segments: Ref<Segment[]> = ref([])
  let pasteCounter = 0

  function nextPasteId(): number {
    return ++pasteCounter
  }

  function reset(): void {
    segments.value = []
    pasteCounter = 0
  }

  function setText(text: string): void {
    segments.value = text ? [{ kind: 'text', value: text }] : []
  }

  function serialize(): string {
    return segments.value
      .map((s) => (s.kind === 'text' ? s.value : s.content))
      .join('')
  }

  function totalLength(): number {
    return segments.value.reduce(
      (sum, s) => sum + (s.kind === 'text' ? s.value.length : s.content.length),
      0,
    )
  }

  function isEmpty(): boolean {
    if (segments.value.length === 0) return true
    return segments.value.every(
      (s) => s.kind === 'text' && s.value.trim() === '',
    )
  }

  function trimEdges(): void {
    const segs = segments.value
    while (segs.length > 0) {
      const first = segs[0]
      if (first.kind === 'paste') break
      const trimmed = first.value.replace(/^\s+/, '')
      if (trimmed === first.value) break
      if (trimmed === '') {
        segs.shift()
      } else {
        segs[0] = { kind: 'text', value: trimmed }
      }
    }
    while (segs.length > 0) {
      const last = segs[segs.length - 1]
      if (last.kind === 'paste') break
      const trimmed = last.value.replace(/\s+$/, '')
      if (trimmed === last.value) break
      if (trimmed === '') {
        segs.pop()
      } else {
        segs[segs.length - 1] = { kind: 'text', value: trimmed }
      }
    }
    segments.value = [...segs]
  }

  return {
    segments,
    nextPasteId,
    reset,
    setText,
    serialize,
    totalLength,
    isEmpty,
    trimEdges,
  }
}

export function createPasteSegment(id: number, content: string): PasteSegment {
  return {
    kind: 'paste',
    id,
    content,
    lineCount: content.split('\n').length,
  }
}

export function parseDOMToSegments(
  root: HTMLElement,
  pasteMap: Map<number, PasteSegment>,
): Segment[] {
  const result: Segment[] = []
  const appendText = (text: string): void => {
    if (!text) return
    const last = result[result.length - 1]
    if (last && last.kind === 'text') {
      last.value += text
    } else {
      result.push({ kind: 'text', value: text })
    }
  }
  root.childNodes.forEach((node, index) => {
    if (node.nodeType === Node.TEXT_NODE) {
      appendText(node.textContent || '')
    } else if (node.nodeType === Node.ELEMENT_NODE) {
      const el = node as HTMLElement
      const id = Number(el.dataset.pasteId)
      if (id && pasteMap.has(id)) {
        result.push(pasteMap.get(id)!)
        return
      }
      if (el.tagName === 'BR') {
        appendText('\n')
        return
      }
      if (el.tagName === 'DIV' || el.tagName === 'P') {
        if (index > 0) appendText('\n')
        appendText(el.textContent || '')
        return
      }
      appendText(el.textContent || '')
    }
  })
  return result
}

export function renderSegmentsToDOM(
  root: HTMLElement,
  segs: Segment[],
): void {
  root.textContent = ''
  for (const seg of segs) {
    if (seg.kind === 'text') {
      root.appendChild(document.createTextNode(seg.value))
    } else {
      const span = document.createElement('span')
      span.contentEditable = 'false'
      span.dataset.pasteId = String(seg.id)
      span.className = 'paste-chip'
      span.textContent = `[Pasted text #${seg.id} +${seg.lineCount} lines]`
      root.appendChild(span)
    }
  }
}

export function chipLabel(seg: PasteSegment): string {
  return `[Pasted text #${seg.id} +${seg.lineCount} lines]`
}
