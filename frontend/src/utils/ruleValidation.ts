import HttpUtils from '@/plugins/httputil'

export type RuleKind = 'route' | 'dns'

export interface RuleConditionIssue {
  kind: string
  path: string
  code: string
  message: string
}

// Mirrors the constants in core/rule_conditions.go. A dropped rule is the one
// issue the core tolerates: it starts fine and silently discards the rule, so it
// must not block a save.
export const DROPPED_RULE_CODE = 'dropped-rule'

export interface RuleValidationResult {
  // False when the rule is invalid or the authoritative backend check could
  // not complete. The modal always remains closable, so failing closed here
  // preserves the operator's edits without admitting an unchecked rule.
  ok: boolean
  blocking: RuleConditionIssue[]
  // Every issue the core returned, including the non-blocking dropped-rule
  // warning. The recursive node renderers anchor each of these to the branch it
  // names, so they need the full list, not just the blocking subset.
  issues: RuleConditionIssue[]
}

const isIssue = (value: unknown): value is RuleConditionIssue => {
  if (value == null || typeof value !== 'object') return false
  const candidate = value as Record<string, unknown>
  return typeof candidate.path === 'string' && typeof candidate.code === 'string'
}

/**
 * Asks the backend whether a single rule carries usable conditions.
 *
 * This deliberately holds no opinion about what a condition is. The panel used to
 * reimplement the fork's zero/nonzero semantics in TypeScript and got them wrong
 * in ways that disabled Save on rules the core accepts, so the only authority
 * here is core.RuleConditionIssues, reached over the wire.
 */
export const validateRuleConditions = async (
  kind: RuleKind,
  rule: unknown,
): Promise<RuleValidationResult> => {
  const msg = await HttpUtils.post('api/config/rule-conditions', {
    data: JSON.stringify({ kind, rule }),
  })

  if (!msg.success) {
    const issue: RuleConditionIssue = {
      kind,
      path: `${kind}.rules[0]`,
      code: 'validation-unavailable',
      message: msg.msg || 'Rule validation is unavailable',
    }
    return { ok: false, blocking: [issue], issues: [issue] }
  }

  const raw = (msg.obj as { issues?: unknown } | null)?.issues
  const issues = Array.isArray(raw) ? raw.filter(isIssue) : []
  const blocking = issues.filter((issue) => issue.code !== DROPPED_RULE_CODE)

  return { ok: blocking.length === 0, blocking, issues }
}
