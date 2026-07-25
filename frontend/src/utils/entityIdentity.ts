// The backend applies the same trimmed check before saving referenced entities.
export function isBlankIdentity(value: string | null | undefined): boolean {
  return value == null || value.trim() === ''
}
