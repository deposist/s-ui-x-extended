import { describe, expect, it } from "vitest";

import { hasRuleConditions } from "./ruleConditions";


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
