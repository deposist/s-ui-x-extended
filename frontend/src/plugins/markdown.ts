export type MarkdownInline =
  | { type: 'text'; text: string }
  | { type: 'strong'; text: string }
  | { type: 'code'; text: string }

export type MarkdownBlock =
  | { type: 'heading'; level: number; inline: MarkdownInline[] }
  | { type: 'paragraph'; inline: MarkdownInline[] }
  | { type: 'list'; ordered: boolean; items: MarkdownInline[][] }
  | { type: 'code'; text: string }
  | { type: 'rule' }

const headingPattern = /^(#{1,6})\s+(.+)$/
const unorderedItemPattern = /^[-*+]\s+(.+)$/
const orderedItemPattern = /^\d+[.)]\s+(.+)$/

export const parseMarkdownInline = (value: string): MarkdownInline[] => {
  const segments: MarkdownInline[] = []
  let index = 0

  const pushText = (text: string) => {
    if (text) segments.push({ type: 'text', text })
  }

  while (index < value.length) {
    const nextCode = value.indexOf('`', index)
    const nextStrong = value.indexOf('**', index)
    const candidates = [nextCode, nextStrong].filter((pos) => pos >= 0)
    if (candidates.length === 0) {
      pushText(value.slice(index))
      break
    }

    const next = Math.min(...candidates)
    pushText(value.slice(index, next))

    if (next === nextCode) {
      const end = value.indexOf('`', next + 1)
      if (end < 0) {
        pushText(value.slice(next))
        break
      }
      segments.push({ type: 'code', text: value.slice(next + 1, end) })
      index = end + 1
      continue
    }

    const end = value.indexOf('**', next + 2)
    if (end < 0) {
      pushText(value.slice(next))
      break
    }
    segments.push({ type: 'strong', text: value.slice(next + 2, end) })
    index = end + 2
  }

  return segments.length ? segments : [{ type: 'text', text: '' }]
}

export const parseMarkdownBlocks = (markdown: string): MarkdownBlock[] => {
  const blocks: MarkdownBlock[] = []
  const paragraph: string[] = []
  let list: { ordered: boolean; items: MarkdownInline[][] } | undefined
  let inFence = false
  let codeLines: string[] = []

  const flushParagraph = () => {
    if (!paragraph.length) return
    blocks.push({ type: 'paragraph', inline: parseMarkdownInline(paragraph.join(' ')) })
    paragraph.length = 0
  }

  const flushList = () => {
    if (!list) return
    blocks.push({ type: 'list', ordered: list.ordered, items: list.items })
    list = undefined
  }

  const flushCode = () => {
    blocks.push({ type: 'code', text: codeLines.join('\n') })
    codeLines = []
  }

  for (const rawLine of markdown.replace(/\r\n/g, '\n').split('\n')) {
    const line = rawLine.replace(/\s+$/, '')

    if (line.startsWith('```')) {
      if (inFence) {
        flushCode()
        inFence = false
      } else {
        flushParagraph()
        flushList()
        inFence = true
        codeLines = []
      }
      continue
    }

    if (inFence) {
      codeLines.push(rawLine)
      continue
    }

    if (!line.trim()) {
      flushParagraph()
      flushList()
      continue
    }

    const heading = line.match(headingPattern)
    if (heading) {
      flushParagraph()
      flushList()
      blocks.push({
        type: 'heading',
        level: Math.min(heading[1].length, 6),
        inline: parseMarkdownInline(heading[2].trim()),
      })
      continue
    }

    if (/^(-{3,}|_{3,}|\*{3,})$/.test(line.trim())) {
      flushParagraph()
      flushList()
      blocks.push({ type: 'rule' })
      continue
    }

    const unordered = line.match(unorderedItemPattern)
    const ordered = line.match(orderedItemPattern)
    if (unordered || ordered) {
      flushParagraph()
      const orderedList = Boolean(ordered)
      if (!list || list.ordered !== orderedList) flushList()
      if (!list) list = { ordered: orderedList, items: [] }
      list.items.push(parseMarkdownInline((ordered?.[1] ?? unordered?.[1] ?? '').trim()))
      continue
    }

    flushList()
    paragraph.push(line.trim())
  }

  if (inFence) flushCode()
  flushParagraph()
  flushList()
  return blocks
}
