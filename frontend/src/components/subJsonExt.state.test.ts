import { afterEach, describe, expect, it } from 'vitest'
import { createRenderer, ssrContextKey, type App } from 'vue'
import SubJsonExt from './SubJsonExt.vue'

const renderer = createRenderer({
  insert: () => {}, remove: () => {},
  createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
  setText: () => {}, setElementText: () => {}, patchProp: () => {},
  parentNode: () => null, nextSibling: () => null,
})
const apps: App[] = []

afterEach(() => {
  for (const app of apps.splice(0)) app.unmount()
})

const mountEditor = (config: Record<string, unknown> = {}) => {
  const app = renderer.createApp({ ...SubJsonExt, render: () => null }, {
    settings: { subJsonExt: JSON.stringify(config) },
  })
  app.provide(ssrContextKey, { modules: new Set() })
  apps.push(app)
  return app.mount({}) as any
}

describe('JSON subscription rules', () => {
  for (const field of ['ruleToDirect', 'ruleToBlock']) {
    it(`clears ${field} when routing rules are absent`, () => {
      const editor = mountEditor()
      expect(() => { editor[field] = [] }).not.toThrow()
      expect(editor.subJsonExt).toEqual({})
    })
  }

  it('creates DNS rules without requiring routing rules', () => {
    const editor = mountEditor({ dns: { servers: [{ tag: 'direct-dns', type: 'local' }] } })
    expect(() => { editor.dnsToDirect = ['geosite-private'] }).not.toThrow()
    expect(editor.subJsonExt.dns.rules).toEqual([
      { rule_set: ['geosite-private'], action: 'route', server: 'direct-dns' },
    ])
    expect(editor.subJsonExt.rule_set.map((rule: any) => rule.tag)).toEqual(['geosite-private'])
    expect(editor.subJsonExt.rules).toBeUndefined()
  })

  it('clears DNS categories when the DNS rules section is absent', () => {
    const editor = mountEditor({ dns: { servers: [] } })
    expect(() => { editor.dnsToDirect = [] }).not.toThrow()
    expect(editor.subJsonExt).toEqual({ dns: { servers: [] } })
  })

  it('updates DNS categories when routing rules are absent', () => {
    const editor = mountEditor({
      dns: { rules: [{ rule_set: ['geosite-private'], action: 'route', server: 'direct-dns' }] },
    })
    expect(() => { editor.dnsToDirect = ['geosite-cn'] }).not.toThrow()
    expect(editor.subJsonExt.dns.rules[0].rule_set).toEqual(['geosite-cn'])
    expect(editor.subJsonExt.rule_set.map((rule: any) => rule.tag)).toEqual(['geosite-cn'])
  })

  it('retains DNS rule sets when the last routing category is removed', () => {
    const editor = mountEditor({
      dns: { rules: [{ rule_set: ['geosite-private'], action: 'route', server: 'direct-dns' }] },
      rules: [{ rule_set: ['geoip-cn'], action: 'route', outbound: 'direct' }],
    })
    editor.ruleToDirect = []
    expect(editor.subJsonExt.rules).toBeUndefined()
    expect(editor.subJsonExt.rule_set.map((rule: any) => rule.tag)).toEqual(['geosite-private'])
  })
})
