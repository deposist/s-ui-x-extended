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

// https://vuetifyjs.com/en/introduction/why-vuetify/#feature-guides
export default createVuetify({
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: { mdi },
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
    defaultTheme: localStorage.getItem('theme') ?? 'system',
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
          secondary: '#007D74',
          success: '#087F50',
          warning: '#A85700',
          error: '#B92D4B',
          info: '#0069A8',
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
