import { describe, expect, it } from 'vitest'
import {
  childListPath,
  childNodePath,
  convertDefaultToLogical,
  convertLogicalToDefault,
  discardedDescendantCount,
  discardedMatchKeys,
  firstIssuePath,
  isLogicalNode,
  issuesAtPath,
  nodeKey,
  normalizeModalRule,
  rootNodePath,
} from './ruleTree'
import { actionKeys, isRouteActionKey, routeDefaultMatchKeys, routeDialerActionKeys } from '@/types/rules'
import { actionDnsRuleKeys, dnsDefaultMatchKeys } from '@/types/dns'
import type { RuleConditionIssue } from './ruleValidation'

describe('path helpers', () => {
  it('addresses nodes the way the backend reports them', () => {
    // These strings are compared against issue paths produced by
    // core.RuleConditionIssues, so the shape is a contract, not a preference.
    expect(rootNodePath('route')).toBe('route.rules[0]')
    expect(rootNodePath('dns')).toBe('dns.rules[0]')
    expect(childListPath('route.rules[0]')).toBe('route.rules[0].rules')
    expect(childNodePath('route.rules[0]', 1)).toBe('route.rules[0].rules[1]')
  })

  it('nests to arbitrary depth', () => {
    const depth2 = childNodePath(childNodePath(rootNodePath('dns'), 0), 3)
    expect(depth2).toBe('dns.rules[0].rules[0].rules[3]')
  })
})

describe('node identity', () => {
  it('is stable per object and unique across objects', () => {
    const a = {}
    const b = {}
    expect(nodeKey(a)).toBe(nodeKey(a))
    expect(nodeKey(a)).not.toBe(nodeKey(b))
  })

  it('keeps surviving siblings keyed to the same object after a deletion', () => {
    // The regression this guards: keying children by array index means deleting
    // a middle child shifts every later index, so Vue rebinds the surviving
    // components to different objects and edits appear to jump between branches.
    const children = [{ domain: ['a'] }, { domain: ['b'] }, { domain: ['c'] }]
    const before = children.map((child) => nodeKey(child))

    children.splice(0, 1)

    expect(children.map((child) => nodeKey(child))).toEqual([before[1], before[2]])
  })
})

describe('exhaustive partitioning', () => {
  it('never classifies a key as both action and match', () => {
    const overlap = actionKeys.filter((key) => (routeDefaultMatchKeys as readonly string[]).includes(key))
    expect(overlap).toEqual([])
  })

  it('keeps invert out of the match set so a shape flip preserves it', () => {
    expect(actionKeys).toContain('invert')
    expect(routeDefaultMatchKeys).not.toContain('invert')
    expect(actionDnsRuleKeys).toContain('invert')
    expect(dnsDefaultMatchKeys).not.toContain('invert')
  })

  it('treats network_type as a match field unless the action is direct', () => {
    // The one genuine collision in the fork's schema: network_type is both a
    // DialerOptions field and a RawDefaultRule matcher. A flat lookup has to be
    // wrong in one direction, so the partition is decided by the action.
    expect(routeDefaultMatchKeys).toContain('network_type')
    expect(routeDialerActionKeys).toContain('network_type')
    expect(isRouteActionKey('network_type', 'route')).toBe(false)
    expect(isRouteActionKey('network_type', 'direct')).toBe(true)
  })

  it('classifies direct-only dialer keys as actions only for direct', () => {
    expect(isRouteActionKey('detour', 'direct')).toBe(true)
    expect(isRouteActionKey('detour', 'route')).toBe(false)
    // Shared with route-options, so it is an action regardless of the action.
    expect(isRouteActionKey('fallback_delay', 'route')).toBe(true)
  })

  it('classifies DNS outbound as a match field, not an action', () => {
    // outbound is an action field on a route rule but the legacy *matcher* on a
    // DNS rule. Getting this backwards leaves it behind on a converted node.
    expect(dnsDefaultMatchKeys).toContain('outbound')
    expect(actionDnsRuleKeys).not.toContain('outbound')
  })

  it('includes the deprecated aliases the fork still decodes', () => {
    for (const alias of ['geosite', 'geoip', 'source_geoip', 'rule_set_ipcidr_match_source']) {
      expect(routeDefaultMatchKeys).toContain(alias)
      expect(dnsDefaultMatchKeys).toContain(alias)
    }
  })
})

describe('default to logical conversion', () => {
  it('drops every match key and keeps everything else', () => {
    const node = {
      domain: ['a.example'],
      geosite: ['cn'],
      invert: true,
      action: 'route',
      outbound: 'proxy',
      some_future_passthrough: 'kept',
    }

    const converted = convertDefaultToLogical(node, routeDefaultMatchKeys)

    expect(converted).toEqual({
      invert: true,
      action: 'route',
      outbound: 'proxy',
      some_future_passthrough: 'kept',
      type: 'logical',
      mode: 'and',
      rules: [{}],
    })
  })

  it('preserves a direct action losslessly, including falsy own values', () => {
    // Copying by own-property presence rather than truthiness is the point: a
    // truthy filter would silently erase every one of these.
    const node = {
      action: 'direct',
      detour: '',
      tcp_fast_open: false,
      routing_mark: 0,
      domain: ['a.example'],
      network_type: ['wifi'],
    }

    const converted = convertDefaultToLogical(node, routeDefaultMatchKeys)

    expect(Object.prototype.hasOwnProperty.call(converted, 'detour')).toBe(true)
    expect(converted.detour).toBe('')
    expect(converted.tcp_fast_open).toBe(false)
    expect(converted.routing_mark).toBe(0)
    expect(converted.domain).toBeUndefined()
    // network_type is in the match set, so a shape flip drops it. That is correct
    // for every action except direct, where it is dialer configuration; the modal
    // resolves that via isRouteActionKey before it ever reaches this helper.
    expect(converted.network_type).toBeUndefined()
  })

  it('does not mutate its input', () => {
    const node = { domain: ['a.example'] }
    convertDefaultToLogical(node, routeDefaultMatchKeys)
    expect(node).toEqual({ domain: ['a.example'] })
  })
})

describe('destructive conversion prompts', () => {
  it('reports only match keys that actually carry a value', () => {
    const node = { domain: ['a.example'], domain_suffix: [], port: 0, invert: true }
    // Empty array: nothing to lose. port: 0 is a real own value and is reported.
    // Sorted so this asserts the membership, not the key-list ordering.
    expect([...discardedMatchKeys(node, routeDefaultMatchKeys)].sort()).toEqual(['domain', 'port'])
  })

  it('says nothing is lost when no match key is set', () => {
    expect(discardedMatchKeys({ action: 'route', invert: false }, routeDefaultMatchKeys)).toEqual([])
  })

  it('counts descendants a logical to default flip would discard', () => {
    expect(discardedDescendantCount({ type: 'logical', rules: [{}, {}] })).toBe(2)
    expect(discardedDescendantCount({ domain: ['a'] })).toBe(0)
  })

  it('leaves the original object and its identity untouched when not converted', () => {
    // Standing in for a cancelled confirmation: the helpers are pure, so
    // declining simply means never calling them.
    const node = { type: 'logical', mode: 'or', rules: [{ domain: ['a'] }] }
    const key = nodeKey(node)
    expect(discardedDescendantCount(node)).toBe(1)
    expect(node).toEqual({ type: 'logical', mode: 'or', rules: [{ domain: ['a'] }] })
    expect(nodeKey(node)).toBe(key)
  })
})

describe('logical to default conversion', () => {
  it('removes only the three logical keys', () => {
    const node = {
      type: 'logical',
      mode: 'or',
      rules: [{ domain: ['a'] }],
      invert: true,
      action: 'reject',
      no_drop: false,
    }

    expect(convertLogicalToDefault(node)).toEqual({
      invert: true,
      action: 'reject',
      no_drop: false,
    })
  })
})

describe('shape detection', () => {
  it('treats only an explicit logical type as logical', () => {
    expect(isLogicalNode({ type: 'logical' })).toBe(true)
    // An explicit default and an implicit one are both non-logical.
    expect(isLogicalNode({ type: 'default' })).toBe(false)
    expect(isLogicalNode({})).toBe(false)
  })
})

describe('issue to node mapping', () => {
  const issues: RuleConditionIssue[] = [
    { kind: 'route', path: 'route.rules[0].rules', code: 'missing-conditions', message: 'outer' },
    { kind: 'route', path: 'route.rules[0].rules[1].rules', code: 'missing-conditions', message: 'inner' },
  ]

  it('anchors an issue to the exact node that owns it', () => {
    expect(issuesAtPath(issues, 'route.rules[0].rules').map((i) => i.message)).toEqual(['outer'])
    expect(issuesAtPath(issues, 'route.rules[0].rules[1].rules').map((i) => i.message)).toEqual(['inner'])
  })

  it('does not leak an issue onto a prefix of its path', () => {
    expect(issuesAtPath(issues, 'route.rules[0].rules[1]')).toEqual([])
  })

  it('scrolls to the shallowest issue first', () => {
    expect(firstIssuePath(issues)).toBe('route.rules[0].rules')
    expect(firstIssuePath([issues[1]])).toBe('route.rules[0].rules[1].rules')
    expect(firstIssuePath([])).toBeUndefined()
  })
})

describe('final modal normalization', () => {
  it('collapses a default root into the rule itself', () => {
    const result = normalizeModalRule({ action: 'route', outbound: 'proxy' }, 'default', 'and', [
      { domain: ['a.example'] },
    ])
    expect(result).toEqual({ domain: ['a.example'], action: 'route', outbound: 'proxy' })
    expect(result.type).toBeUndefined()
  })

  it('lets action fields win a collision with the collapsed child', () => {
    // A DNS rule can carry `outbound` as a matcher; if the action also sets it,
    // the action is what the operator just edited.
    const result = normalizeModalRule({ outbound: 'action-wins' }, 'default', 'and', [
      { outbound: 'match-loses' },
    ])
    expect(result.outbound).toBe('action-wins')
  })

  it('keeps a logical root with its mode and children', () => {
    const children = [{ domain: ['a'] }, { type: 'logical', mode: 'or', rules: [{}] }]
    expect(normalizeModalRule({ action: 'route' }, 'logical', 'or', children)).toEqual({
      action: 'route',
      type: 'logical',
      mode: 'or',
      rules: children,
    })
  })

  it('tolerates a default root with no child at all', () => {
    expect(normalizeModalRule({ action: 'route' }, 'default', 'and', [])).toEqual({ action: 'route' })
  })
})
