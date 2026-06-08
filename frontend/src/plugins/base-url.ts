const defaultBaseUrl = '/app/'

const normalizeBaseUrl = (value: string | null | undefined) => {
  if (!value || value.charAt(0) === '{') return defaultBaseUrl
  return value.endsWith('/') ? value : `${value}/`
}

export const getBaseUrl = () => {
  const legacyWindowValue = typeof window === 'undefined' ? undefined : (window as any).BASE_URL
  if (typeof legacyWindowValue === 'string' && legacyWindowValue.charAt(0) !== '{') {
    return normalizeBaseUrl(legacyWindowValue)
  }
  const metaValue = typeof document === 'undefined'
    ? null
    : document.querySelector('meta[name="s-ui-base-url"]')?.getAttribute('content')
  return normalizeBaseUrl(metaValue)
}
