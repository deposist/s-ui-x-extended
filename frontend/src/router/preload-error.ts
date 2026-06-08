export const isPreloadError = (err: unknown): boolean => {
  if (!err) return false
  const candidate = err as { message?: unknown, name?: unknown }
  const msg = typeof candidate.message === 'string' ? candidate.message : String(err)
  return /Failed to fetch dynamically imported module/i.test(msg) ||
    /Importing a module script failed/i.test(msg) ||
    /Failed to load module script/i.test(msg) ||
    /error loading dynamically imported module/i.test(msg) ||
    candidate.name === 'ChunkLoadError'
}
