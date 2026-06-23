export type RowActionTone = 'default' | 'error'

export interface RowAction {
  key: string
  // i18n key resolved for the tooltip / menu label and the icon button aria-label.
  labelKey: string
  icon: string
  // Always-visible icon button (true) vs collapsed into the overflow menu (false).
  inline?: boolean
  tone?: RowActionTone
  // Render a divider above this entry in the overflow menu (e.g. before Delete).
  divider?: boolean
  // Caller-computed per-row visibility (e.g. Stats only when traffic is enabled).
  hidden?: boolean
}
