import { describe, expect, it } from 'vitest'
import changesSource from './Changes.vue?raw'
import logsSource from './Logs.vue?raw'

const modalSource: Record<string, string> = {
  Changes: changesSource,
  Logs: logsSource,
}

describe('audit modal data loading', () => {
  it.each(['Logs', 'Changes'])('%s resets loading in finally, including HTTP, network, and cancellation errors', (name) => {
    const source = modalSource[name]

    expect(source).toMatch(/async loadData\(\)\s*\{[\s\S]*?try\s*\{[\s\S]*?await HttpUtils\.get\([\s\S]*?\}\s*finally\s*\{\s*this\.loading\s*=\s*false\s*\}/)
  })
})
