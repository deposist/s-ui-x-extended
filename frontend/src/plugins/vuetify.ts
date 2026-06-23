/**
 * plugins/vuetify.ts
 *
 * Framework documentation: https://vuetifyjs.com`
 */

// Styles
import 'vuetify/styles/main.css'

import colors from 'vuetify/util/colors'
import { fa, en, vi, zhHans, zhHant, ru } from 'vuetify/locale'

// Composables
import { createVuetify } from 'vuetify'

// SVG icons (@mdi/js) instead of the full @mdi/font webfont — see ./mdiIcons.
import { aliases, mdi } from './mdiIcons'
// Lucide icon set for the Nexus UI's visible icons (reference prototype) — opt-in
// via the `lucide:` prefix; Vuetify internals keep using the default mdi set.
import { lucide } from './lucideIcons'

// https://vuetifyjs.com/en/introduction/why-vuetify/#feature-guides
export default createVuetify({
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: { mdi, lucide },
  },
  defaults: {
    VRow: { density: 'compact' },
    VTextField: {
      variant: 'solo-filled',
    },
    VSelect: {
      variant: 'solo-filled',
    },
    VCombobox: {
      variant: 'solo-filled',
    },
    VTextarea: {
      variant: 'solo-filled',
    },
  },
  theme: {
    // Default to dark when the user has made no explicit choice, so the Nexus UI
    // opens in the dark "technical" look of the reference even on light-OS hosts.
    // An explicit stored choice (light/dark/system) is always respected.
    defaultTheme: localStorage.getItem('theme') ?? 'dark',
    themes: {
      light: {
        colors: {
          error: '#FF5252',
          background: colors.grey.lighten4,
        },
      },
      dark: {
        colors: {
          primary: colors.blue.darken4,
          error: colors.red.accent3,
        },
      },
      nexusDark: {
        dark: true,
        colors: {
          background: '#0A1226',
          surface: '#0F1830',
          'surface-bright': '#172041',
          'surface-variant': '#1D2752',
          primary: '#22D3EE',
          'on-primary': '#0A1226',
          secondary: '#A78BFA',
          success: '#34D399',
          warning: '#FBBF24',
          error: '#FB7185',
          info: '#38BDF8',
        },
      },
      nexusLight: {
        dark: false,
        colors: {
          background: '#F4FAFB',
          surface: '#FFFFFF',
          'surface-bright': '#FFFFFF',
          'surface-variant': '#D9E9ED',
          primary: '#007D87',
          'on-primary': '#FFFFFF',
          secondary: '#007D74',
          success: '#087F50',
          warning: '#A85700',
          error: '#B92D4B',
          info: '#0069A8',
        },
      },
      technicalDark: {
        dark: true,
        colors: {
          background: '#0a0a0a',
          surface: '#1a1a1a',
          'surface-bright': '#202020',
          'surface-variant': '#252525',
          primary: '#00d4ff',
          'on-primary': '#0a0a0a',
          secondary: '#3399ff',
          success: '#00cc66',
          warning: '#ffaa00',
          error: '#ff4444',
          info: '#3399ff',
          'on-surface': '#ffffff',
        },
        variables: {
          // The reference uses ONE solid border colour everywhere (#2a2a2a), not
          // Vuetify's default white-at-opacity. Pin it (opacity 1) so v-card /
          // v-field / v-list / v-tabs / v-table / v-divider borders are byte-exact
          // to the prototype instead of "practically the same".
          'border-color': '#2a2a2a',
          'border-opacity': 1,
        },
      },
      technicalLight: {
        dark: false,
        colors: {
          background: '#f5f6f7',
          surface: '#ffffff',
          'surface-bright': '#ffffff',
          'surface-variant': '#eceef0',
          primary: '#0091b3',
          'on-primary': '#ffffff',
          secondary: '#2f6fd1',
          success: '#0a8f4d',
          warning: '#b06f00',
          error: '#cc3333',
          info: '#2f6fd1',
          'on-surface': '#0a0a0a',
        },
        variables: {
          'border-color': '#e2e4e7',
          'border-opacity': 1,
        },
      },
    },
  },
  locale: {
    locale: localStorage.getItem("locale") ?? 'en',
    fallback: 'en',
    messages: { en, fa, vi, zhHans, zhHant, ru },
  },
})
