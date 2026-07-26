import { beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.fn()

vi.mock('@/plugins/httputil', () => ({
  default: { post: (...args: unknown[]) => post(...args) },
}))

import { validateRuleConditions } from './ruleValidation'

describe('validateRuleConditions', () => {
  beforeEach(() => {
    post.mockReset()
  })

  const ok = (issues: unknown) => ({ success: true, msg: '', obj: { issues } })

  it('sends the rule to the backend as a single JSON data field', async () => {
    post.mockResolvedValue(ok([]))
    const rule = { type: 'logical', mode: 'and', rules: [{ domain: ['a.example'] }] }

    await validateRuleConditions('route', rule)

    expect(post).toHaveBeenCalledWith('api/config/rule-conditions', {
      data: JSON.stringify({ kind: 'route', rule }),
    })
  })

  it('accepts a rule the core reports no issues for', async () => {
    post.mockResolvedValue(ok([]))
    await expect(validateRuleConditions('route', { domain: ['a.example'] })).resolves.toEqual({
      ok: true,
      blocking: [],
      issues: [],
    })
  })

  it('blocks a rule the core would refuse to start on', async () => {
    const issue = {
      kind: 'route',
      path: 'route.rules[0].rules[1].rules',
      code: 'missing-conditions',
      message: 'logical rule has no conditions: its sub-rule list is empty',
    }
    post.mockResolvedValue(ok([issue]))

    const verdict = await validateRuleConditions('route', { type: 'logical' })

    expect(verdict.ok).toBe(false)
    // The exact nested path has to survive: it is the only thing telling the
    // operator which branch to open.
    expect(verdict.blocking).toEqual([issue])
  })

  // A dropped rule is silently discarded by the core, which still starts, so it
  // must not block the editor. The save path only logs it.
  it('does not block on a dropped-rule issue but still surfaces it', async () => {
    const dropped = { kind: 'dns', path: 'dns.rules', code: 'dropped-rule', message: '1 rule discarded' }
    post.mockResolvedValue(ok([dropped]))

    // Not blocking, but the node renderers still need it, so it stays in issues.
    await expect(validateRuleConditions('dns', {})).resolves.toEqual({
      ok: true,
      blocking: [],
      issues: [dropped],
    })
  })

  it('reports only the blocking issues when both kinds come back', async () => {
    const blocking = { kind: 'dns', path: 'dns.rules[0]', code: 'invalid-rule', message: 'no conditions' }
    const dropped = { kind: 'dns', path: 'dns.rules', code: 'dropped-rule', message: 'discarded' }
    post.mockResolvedValue(ok([dropped, blocking]))

    const verdict = await validateRuleConditions('dns', {})

    expect(verdict.ok).toBe(false)
    expect(verdict.blocking).toEqual([blocking])
    // issues keeps both, in the order the core returned them.
    expect(verdict.issues).toEqual([dropped, blocking])
  })

  it('blocks when the authoritative request is unsuccessful', async () => {
    post.mockResolvedValue({ success: false, msg: 'insufficient scope', obj: null })

    await expect(validateRuleConditions('route', { type: 'logical' })).resolves.toEqual({
      ok: false,
      blocking: [{
        kind: 'route',
        path: 'route.rules[0]',
        code: 'validation-unavailable',
        message: 'insufficient scope',
      }],
      issues: [{
        kind: 'route',
        path: 'route.rules[0]',
        code: 'validation-unavailable',
        message: 'insufficient scope',
      }],
    })
  })

  it('tolerates a malformed or absent issues payload', async () => {
    for (const payload of [undefined, null, 'nope', [null, 42, { path: 'x' }]]) {
      post.mockResolvedValue({ success: true, msg: '', obj: { issues: payload } })
      await expect(validateRuleConditions('route', {})).resolves.toEqual({ ok: true, blocking: [], issues: [] })
    }
  })
})
