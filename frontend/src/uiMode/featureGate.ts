export const isNexusEnabled = (): boolean => {
  const raw = import.meta.env.VITE_ENABLE_NEXUS

  if (raw === undefined) return true

  return String(raw).toLowerCase() !== 'false' && raw !== '0'
}
