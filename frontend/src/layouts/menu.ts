// Single source of truth for the navigation menu shared by both UI shells
// (classic Drawer.vue and nexus NexusSidebar.vue). Keeping one list here
// prevents the two environments from drifting apart. The `singBoxSettings`
// flag is nexus-only metadata; the classic shell simply ignores it.
export interface MenuItem {
  title: string
  icon: string
  path: string
  singBoxSettings?: boolean
}

export const appMenu: MenuItem[] = [
  { title: 'pages.home', icon: 'mdi-home', path: '/' },
  { title: 'pages.inbounds', icon: 'mdi-cloud-download', path: '/inbounds', singBoxSettings: true },
  { title: 'pages.clients', icon: 'mdi-account-multiple', path: '/clients' },
  { title: 'pages.outbounds', icon: 'mdi-cloud-upload', path: '/outbounds', singBoxSettings: true },
  { title: 'pages.endpoints', icon: 'mdi-cloud-tags', path: '/endpoints', singBoxSettings: true },
  { title: 'pages.providers', icon: 'mdi-cloud-sync', path: '/providers' },
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

export const singBoxSettingsPaths = appMenu
  .filter(item => item.singBoxSettings)
  .map(item => item.path)
