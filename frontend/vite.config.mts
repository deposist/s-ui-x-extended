// Plugins
import vue from '@vitejs/plugin-vue'
import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'

// Utilities
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'

const isE2E = process.env.SUI_E2E === '1'

export default defineConfig({
  base: '',
  plugins: [
    vue({
      template: { transformAssetUrls },
    }),
    vuetify({
      autoImport: true,
      styles: {
        configFile: 'src/styles/settings.scss',
      },
    })
  ],
  build: {
    manifest: false,
    outDir: 'dist',
    chunkSizeWarningLimit: 2000,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[hash].js',
        chunkFileNames: 'assets/[hash].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.names.some(name => name.endsWith('.css')))
            return 'assets/[hash].css'
          return 'assets/[name][extname]'
        },
      },
    }
  },
  define: {
    'process.env': {},
    // vue-i18n / @intlify compile-time feature flags. Required because vue-i18n is
    // excluded from dep pre-bundling below (its esm-bundler build reads these guards
    // directly in the browser); also silences the flag warnings in the production build.
    __VUE_I18N_FULL_INSTALL__: true,
    __VUE_I18N_LEGACY_API__: false,
    __INTLIFY_PROD_DEVTOOLS__: false,
    __INTLIFY_JIT_COMPILATION__: false,
    __INTLIFY_DROP_MESSAGE_COMPILER__: false,
  },
  optimizeDeps: {
    // vue-i18n is excluded because Rolldown's dep optimizer mis-bundles it on this
    // toolchain (rolldown-vite 8 + Node 25): the optimized chunk references
    // `init_runtime_dom_esm_bundler` (the @vue/runtime-dom esm-bundler init) without
    // defining it, so `app.use(i18n)` throws `ReferenceError: init_runtime_dom_esm_bundler
    // is not defined` and the whole SPA fails to mount (blank login page). Serving it as
    // native ESM via the flags above avoids the broken pre-bundle.
    exclude: ['vuetify', 'vuetify/components', 'vuetify/directives', 'vue-i18n'],
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
    extensions: ['.js', '.json', '.jsx', '.mjs', '.ts', '.tsx', '.vue'],
  },
  server: {
    hmr: isE2E ? false : undefined,
    port: 3000,
    proxy: {
      '/app/api': {
        target: 'http://localhost:2095',
        changeOrigin: true,
      },
    },
  }
})
