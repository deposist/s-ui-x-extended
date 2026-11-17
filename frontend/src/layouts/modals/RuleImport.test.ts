import { describe, expect, it, vi } from 'vitest'
import RuleImport from './RuleImport.vue'

describe('rule JSON import', () => {
  it.each([
    { rules: {} },
    { route: { rule_set: 'invalid' } },
    { rules: [null] },
  ])('rejects invalid rule arrays before enabling Save: %j', input => {
    const vm: any = { ...RuleImport.data!.call({} as any, {} as any), rawJson: JSON.stringify(input), $t: (key: string) => key, hasConflicts: false, $emit: vi.fn() }
    for (const [name, method] of Object.entries(RuleImport.methods!)) vm[name] = (method as (...args: any[]) => any).bind(vm)
    vm.parseJson()
    vm.save()
    expect(vm.parsed).toBeNull()
    expect(vm.error).toBe('rule.import.errNoArrays')
    expect(vm.$emit).not.toHaveBeenCalled()
  })

  it('accepts a route block with valid rule and ruleset arrays', () => {
    const route = { rules: [{ domain: ['example.com'], outbound: 'direct' }], rule_set: [] }
    expect(RuleImport.methods!.extractRouteBlock({ route })).toEqual(route)
  })
})
