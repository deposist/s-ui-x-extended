import { describe, expect, it } from 'vitest'
import routeNodeSource from './RouteRuleNode.vue?raw'
import dnsNodeSource from './DnsRuleNode.vue?raw'
import routeModalSource from '@/layouts/modals/Rule.vue?raw'
import dnsModalSource from '@/layouts/modals/DnsRule.vue?raw'

// These are source-level assertions, matching the project's existing .spec.ts
// style (the test environment is node, with no DOM or @vue/test-utils). The
// behavioural core is covered by ruleTree.test.ts; here we lock the wiring that
// connects those pure helpers to the templates, since a mismatch there is silent
// at type-check time but breaks issue anchoring and the recursion at runtime.

describe.each([
  ['RouteRuleNode', routeNodeSource],
  ['DnsRuleNode', dnsNodeSource],
])('%s recursion and anchoring', (name, source) => {
  it('recurses on itself so nested logical rules render at any depth', () => {
    // A <script setup> component references itself by its own file name.
    expect(source).toContain(`<${name}`)
  })

  it('anchors its own issues and its child-list issues to distinct paths', () => {
    expect(source).toContain('issuesAtPath(props.issues, props.path)')
    expect(source).toContain('issuesAtPath(props.issues, childListPathOf(props.path))')
    // Both anchors expose a data attribute so a scroll-to-issue can find them.
    expect(source).toContain(':data-rule-issue="path"')
    expect(source).toContain(':data-rule-issue="childListPathOf(path)"')
  })

  it('derives every child path from its own path rather than a fixed root', () => {
    // Regression guard: a hard-coded "route.rules[0]" here would give every
    // node at the same index an identical path and collapse issue anchoring.
    expect(source).toContain('childNodePathOf(path, index)')
    expect(source).not.toMatch(/rules\[0\]/)
  })

  it('keys children by node identity, not array index', () => {
    // Index keys would re-bind editor state to the wrong node after a delete.
    expect(source).toContain('nodeKey(')
  })

  it('confirms both destructive shape conversions before discarding data', () => {
    expect(source).toContain('convert.toLogicalTitle')
    expect(source).toContain('convert.toDefaultTitle')
  })
})

describe('route modal action-aware partition', () => {
  it('routes match/action classification through the context-sensitive helper', () => {
    // network_type is both a DialerOptions and a match field; a flat lookup
    // would misclassify it. The modal splits the loaded rule into action fields
    // and the match half via isRouteActionKey, so the classification lives here
    // rather than in the node (the node renders the already-split match half).
    expect(routeModalSource).toContain('isRouteActionKey')
  })
})

describe('DnsRuleNode match keys', () => {
  it('uses the fork-derived DNS match key set, not the route one', () => {
    expect(dnsNodeSource).toContain('dnsDefaultMatchKeys')
    expect(dnsNodeSource).not.toContain('routeDefaultMatchKeys')
  })
})

describe.each([
  ['route', routeModalSource, 'route.rules[0]'],
  ['dns', dnsModalSource, 'dns.rules[0]'],
])('%s modal issue wiring', (_kind, source, rootPath) => {
  it('roots the edited rule where the backend addresses it', () => {
    expect(source).toContain(`const ROOT_PATH = '${rootPath}'`)
  })

  it('owns the root and root-child-list anchors the recursive nodes cannot reach', () => {
    // The root rule and its empty child-list are above the recursion, so the
    // modal renders their alerts itself.
    expect(source).toContain('rootOwnIssues')
    expect(source).toContain('rootChildListIssues')
    expect(source).toContain(':data-rule-issue="rootPath"')
    expect(source).toContain(':data-rule-issue="rootChildListPath"')
  })

  it('keeps every issue, so non-blocking warnings still reach their node', () => {
    expect(source).toContain('this.conditionIssues = verdict.issues')
  })

  it('passes the full issue list down to the recursive node', () => {
    expect(source).toContain(':issues="conditionIssues"')
  })

  it('hands each top-level child a path built from the root', () => {
    expect(source).toContain('topChildPath(Number(index))')
    expect(source).toContain('childNodePath(ROOT_PATH, index)')
  })
})
