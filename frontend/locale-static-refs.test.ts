import { readdirSync, readFileSync } from 'node:fs'
import { extname, join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import en from './src/locales/en'
import ru from './src/locales/ru'

const hasLocaleKey = (obj: Record<string, unknown>, key: string): boolean => {
  let current: unknown = obj
  for (const part of key.split('.')) {
    if (!current || typeof current !== 'object' || !(part in current)) return false
    current = (current as Record<string, unknown>)[part]
  }
  return true
}

const sourceRoot = resolve('src')
const staticLocaleRefPattern = /(?:\$t|\bt)\(['"]([^'"]+)['"]\)/g

const walkSourceFiles = (dir: string): string[] => {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'locales') continue
      out.push(...walkSourceFiles(path))
      continue
    }
    if (!['.ts', '.vue'].includes(extname(entry.name))) continue
    if (entry.name.endsWith('.test.ts')) continue
    out.push(path)
  }
  return out
}

const collectStaticLocaleRefs = (): string[] => {
  const refs = new Set<string>()
  for (const path of walkSourceFiles(sourceRoot)) {
    const text = readFileSync(path, 'utf8')
    for (const match of text.matchAll(staticLocaleRefPattern)) {
      refs.add(match[1])
    }
  }
  return [...refs].sort()
}

describe('statically referenced locale keys', () => {
  it('exist in en and ru source-of-truth locales', () => {
    const refs = collectStaticLocaleRefs()
    const enMessages = en as Record<string, unknown>
    const ruMessages = ru as Record<string, unknown>
    const missingEn = refs.filter((k) => !hasLocaleKey(enMessages, k))
    const missingRu = refs.filter((k) => !hasLocaleKey(ruMessages, k))

    expect(
      missingEn,
      `keys referenced in frontend but missing from en: ${missingEn.join(', ')}`,
    ).toEqual([])
    expect(
      missingRu,
      `keys referenced in frontend but missing from ru: ${missingRu.join(', ')}`,
    ).toEqual([])
  })
})
