// Client-side mirror of the server-side Amnezia obfuscation validator
// (service/awg_obfuscation.go). The server remains authoritative; this module
// only provides instant feedback in the endpoint form.

// Reserved header values 0-4 are the vanilla WireGuard message types
// (amneziawg-go device/noise-protocol.go). Using them defeats obfuscation and
// breaks mixed AWG deployments, so they are rejected like the server does.
export const AWG_HEADER_RESERVED_MAX = 4
export const AWG_HEADER_MAX = 4294967295 // uint32
export const AWG_MAX_JUNK_COUNT = 128
export const AWG_MAX_JUNK_SIZE = 1280
export const AWG_MAX_PADDING = 1280

// Base wire sizes from amneziawg-go device/noise-protocol.go; padded packets
// must stay pairwise distinguishable by size.
const MESSAGE_BASE_SIZES: Record<string, number> = {
  s1: 148, // handshake initiation
  s2: 92, // handshake response
  s3: 64, // cookie reply
  s4: 32, // transport
}

export interface AmneziaHeaderRange {
  from: number
  to: number
}

// Parses one H1-H4 field: a number, a numeric string, or an "a-b" range
// (sing-box badoption.Range syntax). Returns null for an absent/empty value
// and undefined for a malformed one.
export function parseAmneziaHeader(value: unknown): AmneziaHeaderRange | null | undefined {
  if (value === undefined || value === null) return null
  if (typeof value === 'number') {
    if (!Number.isInteger(value) || value < 0 || value > AWG_HEADER_MAX) return undefined
    return { from: value, to: value }
  }
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  if (trimmed.length === 0) return null
  const parts = trimmed.split('-')
  if (parts.length > 2) return undefined
  if (!/^\d+$/.test(parts[0])) return undefined
  const from = Number(parts[0])
  let to = from
  if (parts.length === 2) {
    if (!/^\d+$/.test(parts[1])) return undefined
    to = Number(parts[1])
  }
  if (from > AWG_HEADER_MAX || to > AWG_HEADER_MAX) return undefined
  if (to < from) return undefined
  return { from, to }
}

export function headerRangesOverlap(a: AmneziaHeaderRange, b: AmneziaHeaderRange): boolean {
  return a.from <= b.to && b.from <= a.to
}

// Field-keyed error map; values are locale keys under types.amnezia.errors.
export type AmneziaErrors = Record<string, string>

// Validates the amnezia options object from the endpoint form. Returns a map
// of field -> locale key (empty object = valid). Keys: jc, jmin, jmax,
// s1..s4, h1..h4.
export function validateAmnezia(amnezia: Record<string, unknown> | undefined | null): AmneziaErrors {
  const errors: AmneziaErrors = {}
  if (!amnezia) return errors

  const num = (key: string): number => {
    const v = amnezia[key]
    return typeof v === 'number' && Number.isFinite(v) ? v : 0
  }

  const jc = num('jc')
  const jmin = num('jmin')
  const jmax = num('jmax')
  if (jc < 0 || jc > AWG_MAX_JUNK_COUNT) errors.jc = 'jcRange'
  if (jmin < 0 || jmin > AWG_MAX_JUNK_SIZE) errors.jmin = 'junkSizeRange'
  if (jmax < 0 || jmax > AWG_MAX_JUNK_SIZE) errors.jmax = 'junkSizeRange'
  if (!errors.jmin && !errors.jmax && jmin > jmax) {
    errors.jmin = 'jminAboveJmax'
    errors.jmax = 'jminAboveJmax'
  }

  const paddings: Record<string, number> = {}
  for (const key of ['s1', 's2', 's3', 's4']) {
    const value = num(key)
    if (value < 0 || value > AWG_MAX_PADDING) {
      errors[key] = 'paddingRange'
    } else {
      paddings[key] = value
    }
  }
  const keys = Object.keys(paddings)
  for (let i = 0; i < keys.length; i++) {
    for (let j = i + 1; j < keys.length; j++) {
      const a = keys[i]
      const b = keys[j]
      if (MESSAGE_BASE_SIZES[a] + paddings[a] === MESSAGE_BASE_SIZES[b] + paddings[b]) {
        errors[a] = 'equalPacketSizes'
        errors[b] = 'equalPacketSizes'
      }
    }
  }

  const headers: Record<string, AmneziaHeaderRange> = {}
  for (const key of ['h1', 'h2', 'h3', 'h4']) {
    const parsed = parseAmneziaHeader(amnezia[key])
    if (parsed === undefined) {
      errors[key] = 'headerFormat'
    } else if (parsed !== null) {
      if (parsed.from <= AWG_HEADER_RESERVED_MAX) {
        errors[key] = 'headerReserved'
      } else {
        headers[key] = parsed
      }
    }
  }
  const headerKeys = Object.keys(headers)
  for (let i = 0; i < headerKeys.length; i++) {
    for (let j = i + 1; j < headerKeys.length; j++) {
      const a = headerKeys[i]
      const b = headerKeys[j]
      if (headerRangesOverlap(headers[a], headers[b])) {
        errors[a] = 'headerOverlap'
        errors[b] = 'headerOverlap'
      }
    }
  }
  return errors
}
