import { describe, it, expect } from 'vitest'
import {
  useInputSegments,
  createPasteSegment,
  parseDOMToSegments,
  renderSegmentsToDOM,
  shouldFoldPaste,
  chipLabel,
  PASTE_CHIP_THRESHOLD_LINES,
  PASTE_CHIP_THRESHOLD_CHARS,
} from './useInputSegments'

describe('useInputSegments', () => {
  it('starts empty', () => {
    const { segments, isEmpty } = useInputSegments()
    expect(segments.value).toEqual([])
    expect(isEmpty()).toBe(true)
  })

  it('setText replaces segments with a single text segment', () => {
    const { segments, setText } = useInputSegments()
    setText('hello world')
    expect(segments.value).toHaveLength(1)
    expect(segments.value[0]).toEqual({ kind: 'text', value: 'hello world' })
  })

  it('setText with empty string clears segments', () => {
    const { segments, setText } = useInputSegments()
    setText('hello')
    setText('')
    expect(segments.value).toEqual([])
  })

  it('reset clears segments and paste counter', () => {
    const { segments, nextPasteId, reset } = useInputSegments()
    nextPasteId()
    nextPasteId()
    segments.value.push(createPasteSegment(1, 'a\nb'))
    reset()
    expect(segments.value).toEqual([])
    expect(nextPasteId()).toBe(1)
  })

  it('nextPasteId increments monotonically', () => {
    const { nextPasteId } = useInputSegments()
    expect(nextPasteId()).toBe(1)
    expect(nextPasteId()).toBe(2)
    expect(nextPasteId()).toBe(3)
  })

  it('serialize concatenates text and paste content in order', () => {
    const { segments, serialize } = useInputSegments()
    segments.value = [
      { kind: 'text', value: 'before ' },
      createPasteSegment(1, 'line1\nline2\nline3'),
      { kind: 'text', value: ' after' },
    ]
    expect(serialize()).toBe('before line1\nline2\nline3 after')
  })

  it('serialize handles only-paste segments', () => {
    const { segments, serialize } = useInputSegments()
    segments.value = [createPasteSegment(1, 'a\nb\nc')]
    expect(serialize()).toBe('a\nb\nc')
  })

  it('totalLength sums text length and paste content length', () => {
    const { segments, totalLength } = useInputSegments()
    segments.value = [
      { kind: 'text', value: 'abc' },
      createPasteSegment(1, '12345'),
    ]
    expect(totalLength()).toBe(8)
  })

  it('isEmpty treats whitespace-only text as empty', () => {
    const { segments, isEmpty } = useInputSegments()
    segments.value = [{ kind: 'text', value: '   \n\t  ' }]
    expect(isEmpty()).toBe(true)
  })

  it('isEmpty treats paste segment as non-empty', () => {
    const { segments, isEmpty } = useInputSegments()
    segments.value = [createPasteSegment(1, 'a\nb')]
    expect(isEmpty()).toBe(false)
  })

  it('trimEdges strips whitespace from text segments but preserves paste segments', () => {
    const { segments, trimEdges } = useInputSegments()
    segments.value = [
      { kind: 'text', value: '  hello  ' },
      createPasteSegment(1, 'a\nb'),
      { kind: 'text', value: '  world  ' },
    ]
    trimEdges()
    expect(segments.value).toEqual([
      { kind: 'text', value: 'hello  ' },
      { kind: 'paste', id: 1, content: 'a\nb', lineCount: 2 },
      { kind: 'text', value: '  world' },
    ])
  })

  it('trimEdges drops whitespace-only leading/trailing text segments', () => {
    const { segments, trimEdges } = useInputSegments()
    segments.value = [
      { kind: 'text', value: '  \n  ' },
      createPasteSegment(1, 'content'),
      { kind: 'text', value: '  ' },
    ]
    trimEdges()
    expect(segments.value).toEqual([
      { kind: 'paste', id: 1, content: 'content', lineCount: 1 },
    ])
  })
})

describe('createPasteSegment', () => {
  it('computes line count from content', () => {
    expect(createPasteSegment(1, 'one line')).toEqual({
      kind: 'paste',
      id: 1,
      content: 'one line',
      lineCount: 1,
    })
    expect(createPasteSegment(2, 'a\nb\nc\n').lineCount).toBe(4)
  })

  it('preserves exact content', () => {
    const content = 'line1\nline2\r\nline3\ttab'
    const seg = createPasteSegment(1, content)
    expect(seg.content).toBe(content)
  })
})

describe('shouldFoldPaste', () => {
  it('does not fold short single-line paste', () => {
    expect(shouldFoldPaste('hello')).toBe(false)
  })

  it('does not fold paste just under line threshold', () => {
    const text = Array.from({ length: PASTE_CHIP_THRESHOLD_LINES }, (_, i) => `line ${i + 1}`).join('\n')
    expect(shouldFoldPaste(text)).toBe(false)
  })

  it('folds paste exceeding line threshold', () => {
    const text = Array.from({ length: PASTE_CHIP_THRESHOLD_LINES + 1 }, (_, i) => `line ${i + 1}`).join('\n')
    expect(shouldFoldPaste(text)).toBe(true)
  })

  it('does not fold paste just under char threshold', () => {
    expect(shouldFoldPaste('a'.repeat(PASTE_CHIP_THRESHOLD_CHARS))).toBe(false)
  })

  it('folds paste exceeding char threshold', () => {
    expect(shouldFoldPaste('a'.repeat(PASTE_CHIP_THRESHOLD_CHARS + 1))).toBe(true)
  })

  it('folds multi-line paste that also exceeds char threshold', () => {
    const text = Array.from({ length: 100 }, (_, i) => `line ${i + 1} with content`).join('\n')
    expect(shouldFoldPaste(text)).toBe(true)
  })
})

describe('chipLabel', () => {
  it('formats label with id and line count', () => {
    const seg = createPasteSegment(3, 'a\nb\nc\nd\ne')
    expect(chipLabel(seg)).toBe('[Pasted text #3 +5 lines]')
  })
})

describe('parseDOMToSegments / renderSegmentsToDOM', () => {
  it('round-trips text-only segments', () => {
    const root = document.createElement('div')
    renderSegmentsToDOM(root, [{ kind: 'text', value: 'hello world' }])
    const parsed = parseDOMToSegments(root, new Map())
    expect(parsed).toEqual([{ kind: 'text', value: 'hello world' }])
  })

  it('round-trips mixed text and paste segments', () => {
    const root = document.createElement('div')
    const pasteMap = new Map([
      [1, createPasteSegment(1, 'line1\nline2\nline3')],
    ])
    const segs = [
      { kind: 'text' as const, value: 'before ' },
      createPasteSegment(1, 'line1\nline2\nline3'),
      { kind: 'text' as const, value: ' after' },
    ]
    renderSegmentsToDOM(root, segs)
    const parsed = parseDOMToSegments(root, pasteMap)
    expect(parsed).toEqual(segs)
  })

  it('merges adjacent text nodes into one segment', () => {
    const root = document.createElement('div')
    root.appendChild(document.createTextNode('hello '))
    root.appendChild(document.createTextNode('world'))
    const parsed = parseDOMToSegments(root, new Map())
    expect(parsed).toEqual([{ kind: 'text', value: 'hello world' }])
  })

  it('restores paste segment from data-paste-id attribute', () => {
    const root = document.createElement('div')
    const pasteSeg = createPasteSegment(2, 'multi\nline\ncontent')
    const pasteMap = new Map([[2, pasteSeg]])
    root.appendChild(document.createTextNode('prefix '))
    const span = document.createElement('span')
    span.contentEditable = 'false'
    span.dataset.pasteId = '2'
    span.textContent = chipLabel(pasteSeg)
    root.appendChild(span)
    const parsed = parseDOMToSegments(root, pasteMap)
    expect(parsed).toEqual([
      { kind: 'text', value: 'prefix ' },
      pasteSeg,
    ])
  })

  it('falls back to text for unknown paste id', () => {
    const root = document.createElement('div')
    const span = document.createElement('span')
    span.dataset.pasteId = '999'
    span.textContent = '[Pasted text #999 +5 lines]'
    root.appendChild(span)
    const parsed = parseDOMToSegments(root, new Map())
    expect(parsed).toEqual([{ kind: 'text', value: '[Pasted text #999 +5 lines]' }])
  })

  it('renders paste chip with contenteditable=false and data-paste-id', () => {
    const root = document.createElement('div')
    const pasteSeg = createPasteSegment(1, 'a\nb\nc')
    renderSegmentsToDOM(root, [pasteSeg])
    const chip = root.querySelector('.paste-chip') as HTMLElement
    expect(chip).toBeTruthy()
    expect(chip.contentEditable).toBe('false')
    expect(chip.dataset.pasteId).toBe('1')
    expect(chip.textContent).toBe('[Pasted text #1 +3 lines]')
  })

  it('clears root before rendering', () => {
    const root = document.createElement('div')
    root.appendChild(document.createTextNode('stale'))
    renderSegmentsToDOM(root, [{ kind: 'text', value: 'fresh' }])
    expect(root.childNodes).toHaveLength(1)
    expect(root.textContent).toBe('fresh')
  })

  it('treats <br> as a newline', () => {
    const root = document.createElement('div')
    root.appendChild(document.createTextNode('line1'))
    root.appendChild(document.createElement('br'))
    root.appendChild(document.createTextNode('line2'))
    const parsed = parseDOMToSegments(root, new Map())
    expect(parsed).toEqual([{ kind: 'text', value: 'line1\nline2' }])
  })

  it('treats <div> after content as a newline + text', () => {
    const root = document.createElement('div')
    root.appendChild(document.createTextNode('first'))
    const div = document.createElement('div')
    div.textContent = 'second'
    root.appendChild(div)
    const parsed = parseDOMToSegments(root, new Map())
    expect(parsed).toEqual([{ kind: 'text', value: 'first\nsecond' }])
  })
})
