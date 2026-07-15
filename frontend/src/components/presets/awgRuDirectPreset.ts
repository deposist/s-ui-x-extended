// "RU -> direct for AWG endpoint" preset (plan stage 5).
//
// Server-side split routing already works through Rules: endpoints with a
// listen_port participate as inbounds. This preset is pure UX sugar that
// creates the two pieces in one click:
//   - a remote geoip-ru rule-set (same source as the RU regional preset);
//   - a route rule `inbound=[awg-endpoint] + rule_set=[geoip-ru] -> direct`.
// Traffic from AWG devices to Russian IPs then leaves the server directly
// while everything else follows the regular routing.

import type { Config } from '@/types/config'

export const AWG_RU_RULESET_TAG = 'preset-awg-ru-direct-geoip'

// Same source as routingDnsPresets.ts (runetfreedom geoip-ru, binary .srs).
const AWG_RU_GEOIP_URL = 'https://raw.githubusercontent.com/runetfreedom/russia-v2ray-rules-dat/release/sing-box/rule-set-geoip/geoip-ru.srs'

export interface AWGRuDirectState {
  enabled: boolean
  endpointTag: string
}

export interface AWGRuDirectPreview {
  willAdd: string[]
  willKeep: string[]
  willRemove: string[]
}

const asArray = (value: unknown): string[] => {
  if (Array.isArray(value)) return value.map(String)
  if (typeof value === 'string') return [value]
  return []
}

const ensureShape = (config: Config) => {
  if (!config.route) config.route = { rules: [], rule_set: [] }
  if (!Array.isArray(config.route.rules)) config.route.rules = []
  if (!Array.isArray(config.route.rule_set)) config.route.rule_set = []
}

const isPresetRule = (rule: any): boolean =>
  asArray(rule?.rule_set).includes(AWG_RU_RULESET_TAG)

const presetRuleFor = (rule: any, endpointTag: string): boolean =>
  isPresetRule(rule) && asArray(rule?.inbound).includes(endpointTag)

// Returns the endpoint tags that already have the preset rule.
export function detectAWGRuDirect(config: Config): string[] {
  ensureShape(config)
  const tags = new Set<string>()
  for (const rule of config.route.rules as any[]) {
    if (!isPresetRule(rule)) continue
    for (const tag of asArray(rule?.inbound)) tags.add(tag)
  }
  return [...tags].sort()
}

// Adds the geoip-ru rule-set (once) and the endpoint rule (idempotent).
export function applyAWGRuDirect(config: Config, endpointTag: string, directOutbound: string): string[] {
  if (!endpointTag) throw new Error('endpointTag is required')
  if (!directOutbound) throw new Error('directOutbound is required')
  ensureShape(config)
  const changes: string[] = []
  const ruleSets = config.route.rule_set as any[]
  if (!ruleSets.some(item => String(item?.tag ?? '') === AWG_RU_RULESET_TAG)) {
    ruleSets.push({
      type: 'remote',
      tag: AWG_RU_RULESET_TAG,
      format: 'binary',
      url: AWG_RU_GEOIP_URL,
      download_detour: directOutbound,
      update_interval: '24h',
    })
    changes.push(`add rule-set ${AWG_RU_RULESET_TAG}`)
  }
  const rules = config.route.rules as any[]
  if (!rules.some(rule => presetRuleFor(rule, endpointTag))) {
    // Placed first: more specific (inbound-scoped) rules must win over the
    // broader regional preset rules that match by rule_set only.
    rules.unshift({
      inbound: [endpointTag],
      rule_set: [AWG_RU_RULESET_TAG],
      outbound: directOutbound,
    })
    changes.push(`add route rule ${endpointTag} -> ${directOutbound}`)
  }
  return changes
}

// Removes the endpoint rule; the rule-set is dropped when no rule uses it.
export function removeAWGRuDirect(config: Config, endpointTag: string): string[] {
  ensureShape(config)
  const changes: string[] = []
  const rules = config.route.rules as any[]
  const remaining = rules.filter(rule => !presetRuleFor(rule, endpointTag))
  if (remaining.length !== rules.length) {
    config.route.rules = remaining as any
    changes.push(`remove route rule ${endpointTag}`)
  }
  const stillUsed = (config.route.rules as any[]).some(isPresetRule)
  if (!stillUsed) {
    const ruleSets = config.route.rule_set as any[]
    const kept = ruleSets.filter(item => String(item?.tag ?? '') !== AWG_RU_RULESET_TAG)
    if (kept.length !== ruleSets.length) {
      config.route.rule_set = kept as any
      changes.push(`remove rule-set ${AWG_RU_RULESET_TAG}`)
    }
  }
  return changes
}

export function computeAWGRuDirectPreview(config: Config, state: AWGRuDirectState, directOutbound: string): AWGRuDirectPreview {
  const preview: AWGRuDirectPreview = { willAdd: [], willKeep: [], willRemove: [] }
  const existing = detectAWGRuDirect(config)
  if (state.enabled && state.endpointTag) {
    if (existing.includes(state.endpointTag)) {
      preview.willKeep.push(`route rule ${state.endpointTag} -> ${directOutbound}`)
    } else {
      preview.willAdd.push(`route rule ${state.endpointTag} -> ${directOutbound}`)
      if (!(config.route?.rule_set as any[] ?? []).some((item: any) => String(item?.tag ?? '') === AWG_RU_RULESET_TAG)) {
        preview.willAdd.push(`rule-set ${AWG_RU_RULESET_TAG}`)
      }
    }
  }
  for (const tag of existing) {
    if (!state.enabled || tag !== state.endpointTag) {
      preview.willRemove.push(`route rule ${tag}`)
    }
  }
  return preview
}

// Applies the desired state: exactly one endpoint (or none) carries the rule.
export function applyAWGRuDirectState(config: Config, state: AWGRuDirectState, directOutbound: string): string[] {
  const changes: string[] = []
  for (const tag of detectAWGRuDirect(config)) {
    if (!state.enabled || tag !== state.endpointTag) {
      changes.push(...removeAWGRuDirect(config, tag))
    }
  }
  if (state.enabled && state.endpointTag) {
    changes.push(...applyAWGRuDirect(config, state.endpointTag, directOutbound))
  }
  return changes
}
