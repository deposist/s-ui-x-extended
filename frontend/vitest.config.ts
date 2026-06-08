import { defineConfig } from 'vitest/config'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts', 'src/**/*.spec.ts'],
    css: false,
    // Transform Vuetify through Vite (not Node's ESM loader) so its side-effect
    // .css imports are neutralised by `css: false` instead of crashing on the
    // unknown ".css" extension. Lets tests render real Vuetify components.
    server: {
      deps: {
        inline: ['vuetify'],
      },
    },
  },
})
