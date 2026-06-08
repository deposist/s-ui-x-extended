export interface NexusMenuItem {
  title: string
  icon: string
  path: string
  singBoxSettings?: boolean
}

export const nexusMenu: NexusMenuItem[] = [
  { title: 'pages.home', icon: 'mdi-home', path: '/' },
  { title: 'pages.inbounds', icon: 'mdi-cloud-download', path: '/inbounds', singBoxSettings: true },
  { title: 'pages.clients', icon: 'mdi-account-multiple', path: '/clients' },
  { title: 'pages.outbounds', icon: 'mdi-cloud-upload', path: '/outbounds', singBoxSettings: true },
  { title: 'pages.endpoints', icon: 'mdi-cloud-tags', path: '/endpoints', singBoxSettings: true },
  { title: 'pages.services', icon: 'mdi-server', path: '/services', singBoxSettings: true },
  { title: 'pages.tls', icon: 'mdi-certificate', path: '/tls', singBoxSettings: true },
  { title: 'pages.basics', icon: 'mdi-application-cog', path: '/basics', singBoxSettings: true },
  { title: 'pages.rules', icon: 'mdi-routes', path: '/rules', singBoxSettings: true },
  { title: 'pages.dns', icon: 'mdi-dns', path: '/dns', singBoxSettings: true },
  { title: 'pages.admins', icon: 'mdi-account-tie', path: '/admins' },
  { title: 'pages.telegram', icon: 'mdi-send', path: '/telegram' },
  { title: 'pages.paidSub', icon: 'mdi-cash-multiple', path: '/paid-subscriptions' },
  { title: 'pages.audit', icon: 'mdi-shield-search', path: '/audit' },
  { title: 'pages.settings', icon: 'mdi-cog', path: '/settings' },
]

export const nexusSingBoxSettingsPaths = nexusMenu
  .filter(item => item.singBoxSettings)
  .map(item => item.path)
