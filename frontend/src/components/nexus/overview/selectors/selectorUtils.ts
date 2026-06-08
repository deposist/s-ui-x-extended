export type SelectorRecord = Record<string, unknown>

export const isSelectorRecord = (value: unknown): value is SelectorRecord => {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export const finiteNumber = (value: unknown): number | undefined => {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

export const nonNegativeNumber = (value: unknown): number | undefined => {
  const number = finiteNumber(value)
  return number === undefined ? undefined : Math.max(number, 0)
}

export const plainText = (value: unknown): string | undefined => {
  if (typeof value !== 'string') return undefined

  const withoutControlCharacters = Array.from(value, (character) => {
    const charCode = character.charCodeAt(0)
    return charCode <= 31 || charCode === 127 ? ' ' : character
  }).join('')
  const text = withoutControlCharacters
    .replace(/[<>]/g, '')
    .replace(/\s+/g, ' ')
    .trim()

  return text.length > 0 ? text : undefined
}

export const plainTextList = (value: unknown): string[] => {
  if (!Array.isArray(value)) return []

  return value.reduce<string[]>((items, item) => {
    const text = plainText(item)
    if (text) items.push(text)
    return items
  }, [])
}
