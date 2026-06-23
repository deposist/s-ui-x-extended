/**
 * plugins/lucideIcons.ts
 *
 * Custom Vuetify icon set that renders Lucide icons for the Nexus UI's *visible*
 * icons (sidebar nav, topbar, toolbars, row actions, drawers, empty states),
 * matching the C:\project reference prototype which uses Lucide-style line icons.
 *
 * Vuetify keeps `defaultSet: 'mdi'`, so its internal icons (checkboxes, select
 * carets, expansion chevrons, pagination) stay MDI. Lucide is opt-in per usage
 * via the `lucide:` set prefix: `<v-icon icon="lucide:zap" />`.
 *
 * Resolution (see vuetify/lib/composables/icons.js): a `set:name` prefix routes
 * to `icons.sets[set].component` with `icon = name`. Here that component maps the
 * kebab name to a Lucide component and renders it at 1em so it scales with the
 * surrounding `.v-icon` font-size and inherits `currentColor`.
 */
import { type Component, defineComponent, h, mergeProps } from 'vue'
import { mdi as mdiSvgBase } from 'vuetify/iconsets/mdi-svg'
import {
  Activity,
  AlertCircle,
  AlertTriangle,
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  ArrowUpRight,
  Calendar,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  CloudDownload,
  CloudUpload,
  Copy,
  CreditCard,
  Download,
  ExternalLink,
  Eye,
  EyeOff,
  FileText,
  Filter,
  FilterX,
  Gauge,
  Globe,
  Heart,
  History,
  Inbox,
  Info,
  KeyRound,
  Languages,
  Link2,
  Megaphone,
  LayoutDashboard,
  LayoutGrid,
  LayoutPanelLeft,
  LineChart,
  List,
  Lock,
  LogOut,
  Menu,
  Monitor,
  Moon,
  MoreVertical,
  Network,
  Palette,
  Pencil,
  Plus,
  QrCode,
  Receipt,
  RotateCcw,
  RotateCw,
  Search,
  Send,
  Server,
  Settings,
  SlidersHorizontal,
  Sun,
  SunMoon,
  Tag,
  Trash2,
  Unlink,
  Upload,
  Users,
  UserCheck,
  UserCog,
  UserPlus,
  Wrench,
  X,
  XCircle,
  Zap,
} from 'lucide-vue-next'

// Kebab name → Lucide component. Names mirror the reference prototype glyphs.
// Exported so a unit test can scan source `lucide:*` usages against it (mirrors
// the mdi scan in mdiIcons.test.ts), catching typos/unmapped names that would
// otherwise render a blank icon with only a DEV console.warn.
export const iconMap: Record<string, Component> = {
  activity: Activity,
  // Sidebar navigation
  'layout-grid': LayoutGrid,
  zap: Zap,
  users: Users,
  'arrow-up-right': ArrowUpRight,
  globe: Globe,
  server: Server,
  lock: Lock,
  list: List,
  network: Network,
  send: Send,
  'credit-card': CreditCard,
  'user-cog': UserCog,
  'file-text': FileText,
  'sliders-horizontal': SlidersHorizontal,
  settings: Settings,
  heart: Heart,
  // Topbar / global controls
  menu: Menu,
  languages: Languages,
  'sun-moon': SunMoon,
  sun: Sun,
  moon: Moon,
  monitor: Monitor,
  palette: Palette,
  'log-out': LogOut,
  'layout-dashboard': LayoutDashboard,
  'layout-panel-left': LayoutPanelLeft,
  // Toolbars / table
  search: Search,
  filter: Filter,
  'filter-x': FilterX,
  plus: Plus,
  'rotate-cw': RotateCw,
  x: X,
  'chevron-down': ChevronDown,
  'chevron-right': ChevronRight,
  'chevron-left': ChevronLeft,
  'arrow-up': ArrowUp,
  'arrow-down': ArrowDown,
  calendar: Calendar,
  // Row actions
  pencil: Pencil,
  copy: Copy,
  'external-link': ExternalLink,
  'line-chart': LineChart,
  'trash-2': Trash2,
  'more-vertical': MoreVertical,
  'qr-code': QrCode,
  'key-round': KeyRound,
  history: History,
  gauge: Gauge,
  download: Download,
  upload: Upload,
  eye: Eye,
  'eye-off': EyeOff,
  tag: Tag,
  check: Check,
  wrench: Wrench,
  'user-plus': UserPlus,
  'user-check': UserCheck,
  megaphone: Megaphone,
  link: Link2,
  receipt: Receipt,
  unlink: Unlink,
  'rotate-ccw': RotateCcw,
  // States / dialogs / drawers
  inbox: Inbox,
  'cloud-download': CloudDownload,
  'cloud-upload': CloudUpload,
  'alert-circle': AlertCircle,
  'alert-triangle': AlertTriangle,
  'x-circle': XCircle,
  info: Info,
  'arrow-left': ArrowLeft,
}

const LucideNexusIcon = defineComponent({
  name: 'LucideNexusIcon',
  inheritAttrs: false,
  props: {
    icon: { type: null },
    tag: { type: String, default: 'i' },
  },
  setup(props, { attrs }) {
    return () => {
      const name = typeof props.icon === 'string' ? props.icon : ''
      const Cmp = iconMap[name]

      if (!Cmp) {
        if (import.meta.env.DEV) {
          console.warn(`[lucideIcons] no icon "${name}": add it to iconMap in plugins/lucideIcons.ts.`)
        }

        return h(props.tag, mergeProps(attrs, {}))
      }

      // 1em sizing + currentColor make the Lucide svg behave like an mdi-svg icon
      // inside Vuetify's `.v-icon` (which sets font-size from the `size` prop).
      return h(props.tag, mergeProps(attrs, {}), [
        h(Cmp, { width: '1em', height: '1em', 'stroke-width': 2 }),
      ])
    }
  },
})

// Cast to the official mdi-svg set's component type so the object satisfies
// Vuetify's `IconSet` (same approach as plugins/mdiIcons.ts).
export const lucide = { component: LucideNexusIcon as typeof mdiSvgBase.component }
