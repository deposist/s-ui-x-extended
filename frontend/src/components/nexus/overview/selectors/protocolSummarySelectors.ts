import { isSelectorRecord, plainText, plainTextList } from './selectorUtils'

export interface ProtocolSummary {
  type: string
  activeInbounds: number
  totalInbounds: number
  tags: string[]
}

export interface ProtocolSummarySelectorInput {
  inbounds?: readonly unknown[] | null
  onlines?: {
    inbound?: readonly unknown[] | null
  } | null
}

const compareProtocolSummary = (left: ProtocolSummary, right: ProtocolSummary): number => {
  if (left.type === right.type) return 0
  return left.type < right.type ? -1 : 1
}

export const selectProtocolSummaries = (
  input?: ProtocolSummarySelectorInput | null,
): ProtocolSummary[] => {
  const onlineTags = new Set(plainTextList(input?.onlines?.inbound))
  const summaries = new Map<string, ProtocolSummary>()

  for (const inbound of input?.inbounds ?? []) {
    if (!isSelectorRecord(inbound)) continue

    const type = plainText(inbound.type) ?? 'unknown'
    const tag = plainText(inbound.tag)
    const summary = summaries.get(type) ?? {
      type,
      activeInbounds: 0,
      totalInbounds: 0,
      tags: [],
    }

    summary.totalInbounds++
    if (tag) {
      summary.tags.push(tag)
      if (onlineTags.has(tag)) summary.activeInbounds++
    }
    summaries.set(type, summary)
  }

  return [...summaries.values()].sort(compareProtocolSummary)
}
