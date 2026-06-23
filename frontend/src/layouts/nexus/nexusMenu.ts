// Count keys map a menu entry to the reactive array on the Data() store whose
// length drives its sidebar badge. Only store-backed collections are listed;
// entries without a key simply render no badge.
export type NexusCountKey =
  | 'inbounds'
  | 'clients'
  | 'outbounds'
  | 'endpoints'
  | 'services'
  | 'tlsConfigs'

export interface NexusMenuItem {
  title: string
  icon: string
  path: string
  singBoxSettings?: boolean
  countKey?: NexusCountKey
}

export interface NexusMenuGroup {
  // No labelKey -> the group renders without a subheader (e.g. Dashboard).
  labelKey?: string
  items: NexusMenuItem[]
}

export const nexusMenuGroups: NexusMenuGroup[] = [
  {
    items: [
      { title: 'pages.home', icon: 'lucide:layout-grid', path: '/' },
    ],
  },
  {
    labelKey: 'nav.groups.proxy',
    items: [
      { title: 'pages.inbounds', icon: 'lucide:zap', path: '/inbounds', singBoxSettings: true, countKey: 'inbounds' },
      { title: 'pages.clients', icon: 'lucide:users', path: '/clients', countKey: 'clients' },
      { title: 'pages.outbounds', icon: 'lucide:arrow-up-right', path: '/outbounds', singBoxSettings: true, countKey: 'outbounds' },
      { title: 'pages.endpoints', icon: 'lucide:globe', path: '/endpoints', singBoxSettings: true, countKey: 'endpoints' },
      { title: 'pages.services', icon: 'lucide:server', path: '/services', singBoxSettings: true, countKey: 'services' },
    ],
  },
  {
    labelKey: 'nav.groups.network',
    items: [
      { title: 'pages.tls', icon: 'lucide:lock', path: '/tls', singBoxSettings: true, countKey: 'tlsConfigs' },
      { title: 'pages.rules', icon: 'lucide:list', path: '/rules', singBoxSettings: true },
      { title: 'pages.dns', icon: 'lucide:network', path: '/dns', singBoxSettings: true },
    ],
  },
  {
    labelKey: 'nav.groups.integrations',
    items: [
      { title: 'pages.telegram', icon: 'lucide:send', path: '/telegram' },
      { title: 'pages.paidSub', icon: 'lucide:credit-card', path: '/paid-subscriptions' },
    ],
  },
  {
    labelKey: 'nav.groups.system',
    items: [
      { title: 'pages.admins', icon: 'lucide:user-cog', path: '/admins' },
      { title: 'pages.audit', icon: 'lucide:file-text', path: '/audit' },
      { title: 'pages.basics', icon: 'lucide:sliders-horizontal', path: '/basics', singBoxSettings: true },
      { title: 'pages.settings', icon: 'lucide:settings', path: '/settings' },
    ],
  },
  {
    labelKey: 'nav.groups.support',
    items: [
      { title: 'pages.donations', icon: 'lucide:heart', path: '/donations' },
    ],
  },
]

// Flat projections preserved so existing consumers (and route-parity tests)
// keep working; they are derived, never maintained by hand.
export const nexusMenu: NexusMenuItem[] = nexusMenuGroups.flatMap(group => group.items)

export const nexusSingBoxSettingsPaths = nexusMenu
  .filter(item => item.singBoxSettings)
  .map(item => item.path)
