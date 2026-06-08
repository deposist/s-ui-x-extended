import './uiMode/bootstrap'

/**
 * main.ts
 *
 * Bootstraps Vuetify and other plugins then mounts the App`
 */

// Composables
import { createApp, ref } from 'vue'

// Components
import App from './App.vue'

// Use router
import router from './router'

// Store
import store from './store'

// Plugins
import { registerPlugins } from '@/plugins'

// Locale
import { i18n, loadInitialLocaleMessages } from '@/locales'

// Notivue
import { createNotivue } from 'notivue'
import 'notivue/notification.css'
import 'notivue/animations.css'
const notivue = createNotivue({
  position: 'bottom-center',
  limit: 4,
  enqueue: false,
  avoidDuplicates: true,
  notifications: {
    global: {
      duration: 3000
    }
  },
})

const bootstrap = async () => {
  await loadInitialLocaleMessages()

  const loading = ref(false)
  const app = createApp(App)
  app.provide('loading', loading)

  registerPlugins(app)

  app
    .use(router)
    .use(store)
    .use(i18n)
    .use(notivue)
    .mount('#app')
}

void bootstrap()
