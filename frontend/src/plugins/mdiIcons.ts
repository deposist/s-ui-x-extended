/**
 * plugins/mdiIcons.ts
 *
 * Custom Vuetify icon set that keeps the app's `icon="mdi-cog"` string usage
 * (including dynamic `:icon` bindings from data) but renders each as an SVG path
 * from @mdi/js instead of shipping the ~2.25 MB @mdi/font webfont.
 *
 * How resolution works (see vuetify/lib/composables/icons.js):
 *  - `icon="mdi-account"` has no `set:` prefix, so Vuetify routes it to the
 *    `defaultSet` ("mdi") — this set — with icon = "mdi-account". We map it to a
 *    path via the generated `iconPaths` table and hand it to VSvgIcon.
 *  - Vuetify's internal `$`-aliases (checkboxOn, sortAsc, …) resolve to
 *    "svg:"-prefixed paths handled by the framework's built-in `svg` set, so they
 *    never reach this component. We still export the mdi-svg `aliases` so those
 *    internals get real SVG paths rather than font class names.
 */
import { defineComponent, h } from 'vue'
import { aliases, mdi as mdiSvgBase } from 'vuetify/iconsets/mdi-svg'
import { iconPaths } from './mdiIconPaths'

// The component the official mdi-svg set uses to render <svg><path d="…"/>.
const VSvgIcon = mdiSvgBase.component

/**
 * resolveMdiIcon maps an `mdi-*` icon name to its @mdi/js SVG path. Non-`mdi-`
 * values (raw SVG paths from Vuetify's $-aliases, components, arrays) pass
 * through unchanged. Kept pure and exported so completeness/resolution can be
 * unit-tested without mounting Vuetify in jsdom.
 */
export function resolveMdiIcon(name: unknown): unknown {
  if (typeof name !== 'string' || !name.startsWith('mdi-')) return name
  if (Object.hasOwn(iconPaths, name)) return iconPaths[name]
  if (import.meta.env.DEV) {
    console.warn(
      `[mdiIcons] no SVG path for "${name}" — add the icon and run "node scripts/gen-mdi-icons.cjs".`,
    )
  }
  return name
}

const MdiNamedSvgIcon = defineComponent({
  name: 'MdiNamedSvgIcon',
  props: {
    icon: { type: null },
    tag: { type: String, default: 'i' },
  },
  setup(props) {
    // useIcon only routes plain strings to a set's component (arrays/components
    // are handled upstream), so the resolved value is always a path/name string.
    return () => h(VSvgIcon, { icon: resolveMdiIcon(props.icon) as string, tag: props.tag })
  },
})

// Vuetify's IconSet expects component: IconComponent; reuse the official
// mdi-svg set's component type rather than importing an internal type path.
const mdi = { component: MdiNamedSvgIcon as typeof mdiSvgBase.component }

export { aliases, mdi }
