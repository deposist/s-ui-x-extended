import type { RuleConditionIssue, RuleKind } from '@/utils/ruleValidation'

/**
 * Pure tree helpers shared by the recursive route/DNS rule node renderers.
 *
 * Nothing here decides whether a rule is *valid*. That question belongs to
 * core.RuleConditionIssues, reached over the wire by `ruleValidation.ts`. The
 * panel previously kept its own TypeScript copy of the fork's zero/nonzero
 * condition semantics, got it wrong, and disabled Save on rules the core
 * accepts. This module only moves, copies, keys, and addresses nodes.
 */

/** A condition node. Deliberately loose: unknown keys must survive untouched. */
export type RuleNode = Record<string, unknown>

const identities = new WeakMap<object, string>()
let identityCounter = 0

/**
 * A stable per-object key for `v-for`.
 *
 * Array indexes are unusable here: deleting a middle child shifts every later
 * index, so Vue would rebind the surviving components' state to different
 * objects and the operator would see edits jump between branches. Keying on
 * object identity means a node keeps its key for as long as it exists, and a
 * deleted node's key is never reused. The WeakMap holds no strong reference, so
 * discarded nodes stay collectable.
 */
export const nodeKey = (node: object): string => {
  let key = identities.get(node)
  if (key === undefined) {
    identityCounter += 1
    key = `n${identityCounter}`
    identities.set(node, key)
  }
  return key
}

/** The path of the rule the modal is editing, as the backend addresses it. */
export const rootNodePath = (kind: RuleKind): string => `${kind}.rules[0]`

/**
 * The path of a logical node's *child list*, which is what an "empty sub-rules"
 * issue is reported against. This is the inline anchor a logical node owns.
 */
export const childListPath = (nodePath: string): string => `${nodePath}.rules`

/** The path of one child of a logical node. */
export const childNodePath = (nodePath: string, index: number): string =>
  `${childListPath(nodePath)}[${index}]`

export const isLogicalNode = (node: RuleNode): boolean => node.type === 'logical'

/** Own-property copy. Preserves `false`, `0`, `''` and `[]`, unlike a truthy filter. */
const shallowCopy = (node: RuleNode): RuleNode => {
  const copy: RuleNode = {}
  for (const key of Object.keys(node)) copy[key] = node[key]
  return copy
}

/**
 * Whether a value is worth warning the operator about before it is deleted.
 *
 * This is *not* a validity judgment and must never gate Save. It exists only to
 * decide whether a destructive conversion needs a confirmation prompt, so the
 * bar is "would the operator notice this disappearing", not "does the core count
 * this as a condition".
 */
const isWorthWarningAbout = (value: unknown): boolean => {
  if (value === undefined || value === null || value === '') return false
  if (Array.isArray(value)) return value.length > 0
  return true
}

/**
 * Match keys that a default -> logical conversion would drop, restricted to the
 * ones actually carrying a value. A non-empty result means the conversion is
 * destructive and must be confirmed first.
 */
export const discardedMatchKeys = (
  node: RuleNode,
  matchKeys: readonly string[],
): string[] =>
  matchKeys.filter(
    (key) =>
      Object.prototype.hasOwnProperty.call(node, key) && isWorthWarningAbout(node[key]),
  )

/**
 * default -> logical.
 *
 * Deletes exactly the exhaustive match key set and keeps everything else,
 * including `invert`, action fields, and unrendered passthrough options. The key
 * set has to be exhaustive against the fork: a match key missing from it would
 * survive here and then be misread as an action or passthrough field.
 */
export const convertDefaultToLogical = (
  node: RuleNode,
  matchKeys: readonly string[],
): RuleNode => {
  const converted = shallowCopy(node)
  for (const key of matchKeys) delete converted[key]
  converted.type = 'logical'
  converted.mode = 'and'
  // A fresh empty child, so the new logical node has somewhere to edit. The core
  // discards a lone {} on load, which is why the node also renders an Add
  // control rather than relying on this placeholder surviving a round trip.
  converted.rules = [{}]
  return converted
}

/** Whether a logical -> default conversion would discard descendants. */
export const discardedDescendantCount = (node: RuleNode): number =>
  Array.isArray(node.rules) ? node.rules.length : 0

/**
 * logical -> default.
 *
 * Deletes only the three keys that make a node logical and preserves everything
 * else, so action and passthrough data survive a shape flip in either direction.
 */
export const convertLogicalToDefault = (node: RuleNode): RuleNode => {
  const converted = shallowCopy(node)
  delete converted.type
  delete converted.mode
  delete converted.rules
  return converted
}

/** Issues anchored exactly at one path. */
export const issuesAtPath = (
  issues: readonly RuleConditionIssue[],
  path: string,
): RuleConditionIssue[] => issues.filter((issue) => issue.path === path)

/**
 * The path of the issue to scroll to. Shallowest first, so the operator is sent
 * to the outermost broken branch instead of a leaf buried inside it.
 */
export const firstIssuePath = (
  issues: readonly RuleConditionIssue[],
): string | undefined => {
  const depthOf = (path: string) => (path.match(/\.rules/g) ?? []).length
  return [...issues].sort((a, b) => depthOf(a.path) - depthOf(b.path))[0]?.path
}

/**
 * Assembles the tree half of the final rule the modal saves.
 *
 * `actionFields` is whatever the existing root action editors produced; this only
 * decides how the edited children are attached. A `default` root collapses its
 * single child into the rule itself, which is how a non-logical rule is shaped,
 * so action fields are applied last and win any key collision.
 */
export const normalizeModalRule = (
  actionFields: RuleNode,
  shape: 'logical' | 'default',
  mode: string,
  children: readonly RuleNode[],
): RuleNode => {
  if (shape === 'default') {
    return { ...(children[0] ?? {}), ...actionFields }
  }
  return { ...actionFields, type: 'logical', mode, rules: [...children] }
}
