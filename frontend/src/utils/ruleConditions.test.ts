import { describe, expect, it } from "vitest";

import {
  hasRuleConditions,
  isLogicalRuleMissingConditions,
} from "./ruleConditions";

import DnsRule from '@/layouts/modals/DnsRule.vue'
import RouteRule from '@/layouts/modals/Rule.vue'

const freshLogicalRule = () => ({
  type: "logical",
  mode: "and",
  rules: [{}],
  invert: false,
  action: "route",
  outbound: "direct",
});

describe("hasRuleConditions", () => {
  it("accepts populated conditions and rejects zero values", () => {
    expect(hasRuleConditions({ domain_suffix: ["example.com"] })).toBe(true);
    expect(hasRuleConditions({ port: [443] })).toBe(true);
    expect(hasRuleConditions({ network_is_expensive: true })).toBe(true);
    expect(hasRuleConditions({})).toBe(false);
    expect(hasRuleConditions({ domain: [] })).toBe(false);
    expect(hasRuleConditions({ domain_keyword: "" })).toBe(false);
    expect(hasRuleConditions({ override_port: 0 })).toBe(false);
    expect(hasRuleConditions({ network_is_constrained: false })).toBe(false);
  });

  it("requires every nested logical branch to be valid", () => {
    expect(
      hasRuleConditions({
        type: "logical",
        rules: [{ domain: ["a.com"] }, { port: [443] }],
      }),
    ).toBe(true);
    expect(
      hasRuleConditions({
        type: "logical",
        rules: [{ domain: ["a.com"] }, {}],
      }),
    ).toBe(false);
    expect(hasRuleConditions({ type: "logical", rules: [] })).toBe(false);
  });
});

type Computed = (this: Record<string, unknown>) => boolean
type ComponentOptions = { computed?: Record<string, Computed> }

function computedResult(component: unknown, name: string, context: Record<string, unknown>): boolean {
  const computed = (component as ComponentOptions).computed?.[name]
  if (!computed) throw new Error(`${name} computed not found`)
  return computed.call(context)
}

describe("isLogicalRuleMissingConditions", () => {
  it("blocks empty logical rules", () => {
    expect(isLogicalRuleMissingConditions(freshLogicalRule())).toBe(true);
    expect(isLogicalRuleMissingConditions({ type: "logical", rules: [] })).toBe(
      true,
    );
  });

  it("allows filled logical and action-only simple rules", () => {
    const rule = freshLogicalRule();
    rule.rules = [{ domain_suffix: ["example.com"] }];
    expect(isLogicalRuleMissingConditions(rule)).toBe(false);
    expect(isLogicalRuleMissingConditions({ action: "sniff" })).toBe(false);
  });
});

describe('route and DNS drawer guards', () => {
  it.each([RouteRule, DnsRule])('blocks Save for an empty logical rule', (component) => {
    expect(computedResult(component, 'saveBlocked', { ruleData: freshLogicalRule() })).toBe(true)
  })

  it.each([RouteRule, DnsRule])('allows Save once the logical rule has conditions', (component) => {
    const ruleData = freshLogicalRule()
    ruleData.rules = [{ domain_suffix: ['example.com'] }]
    expect(computedResult(component, 'saveBlocked', { ruleData })).toBe(false)
  })
})
