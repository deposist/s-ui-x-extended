const STRUCTURAL_KEYS: Record<string, true> = {
  type: true,
  mode: true,
  rules: true,
  invert: true,
};

const isMeaningfulValue = (value: unknown): boolean => {
  if (value == null) return false;
  if (Array.isArray(value)) return value.length > 0;
  if (typeof value === "string") return value.trim() !== "";
  if (typeof value === "number") return value !== 0;
  if (typeof value === "boolean") return value;
  if (typeof value === "object") return Object.keys(value).length > 0;
  return true;
};

export const hasRuleConditions = (rule: unknown): boolean => {
  if (rule == null || typeof rule !== "object") return false;

  const candidate = rule as Record<string, unknown>;
  if (candidate.type === "logical") {
    const subRules = Array.isArray(candidate.rules) ? candidate.rules : [];
    return subRules.length > 0 && subRules.every(hasRuleConditions);
  }

  return Object.entries(candidate).some(
    ([key, value]) => !STRUCTURAL_KEYS[key] && isMeaningfulValue(value),
  );
};

export const isLogicalRuleMissingConditions = (ruleData: unknown): boolean => {
  if (ruleData == null || typeof ruleData !== "object") return false;

  const candidate = ruleData as Record<string, unknown>;
  if (candidate.type !== "logical") return false;

  const subRules = Array.isArray(candidate.rules) ? candidate.rules : [];
  return subRules.length === 0 || !subRules.every(hasRuleConditions);
};
