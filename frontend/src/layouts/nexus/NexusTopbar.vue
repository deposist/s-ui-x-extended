<template>
  <v-app-bar class="nexus-topbar" flat height="64">
    <template #prepend>
      <v-btn
        v-if="showNavigationToggle"
        icon="mdi-menu"
        variant="text"
        @click="emit('toggle-navigation')"
      />
    </template>

    <v-app-bar-title class="nexus-topbar__title">
      <span class="nexus-topbar__page">{{ $t(<string>route.name) }}</span>
      <span class="nexus-topbar__host" dir="ltr">{{ panelHost }}</span>
    </v-app-bar-title>

    <template #append>
      <v-menu>
        <template #activator="{ props }">
          <v-btn icon variant="text" v-bind="props">
            <v-icon icon="mdi-translate" />
          </v-btn>
        </template>
        <v-list>
          <v-list-item
            v-for="language in languages"
            :key="language.value"
            :active="isActiveLocale(language.value)"
            @click="changeLocale(language.value)"
          >
            <v-list-item-title>{{ language.title }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>

      <v-menu>
        <template #activator="{ props }">
          <v-btn icon variant="text" v-bind="props">
            <v-icon icon="mdi-theme-light-dark" />
          </v-btn>
        </template>
        <v-list>
          <v-list-item
            v-for="item in themes"
            :key="item.value"
            :active="isActiveTheme(item.value)"
            :prepend-icon="item.icon"
            @click="changeTheme(item.value)"
          >
            <v-list-item-title>{{ $t(`theme.${item.value}`) }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>

      <ui-mode-control variant="quick" />
    </template>
  </v-app-bar>
</template>

<script lang="ts" setup>
import UiModeControl from '@/components/UiModeControl.vue'
import { languages, setI18nLocale } from '@/locales'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useLocale, useTheme } from 'vuetify'

defineProps<{
  showNavigationToggle: boolean
}>()

const emit = defineEmits<{
  'toggle-navigation': []
}>()

const route = useRoute()
const theme = useTheme()
const vuetifyLocale = useLocale()
const { locale: i18nLocale } = useI18n()
const panelHost = window.location.hostname

const changeLocale = async (nextLocale: string) => {
  const selectedLocale = await setI18nLocale(nextLocale)
  i18nLocale.value = selectedLocale
  vuetifyLocale.current.value = selectedLocale
  window.location.reload()
}

const isActiveLocale = (locale: string) => i18nLocale.value === locale

const themes = [
  { value: 'light', icon: 'mdi-white-balance-sunny' },
  { value: 'dark', icon: 'mdi-moon-waning-crescent' },
  { value: 'system', icon: 'mdi-laptop' },
]

const changeTheme = (nextTheme: string) => {
  theme.change(nextTheme)
  localStorage.setItem('theme', nextTheme)
}

const isActiveTheme = (value: string) => {
  const currentTheme = localStorage.getItem('theme') ?? 'system'

  return currentTheme === value
}
</script>

<style scoped>
.nexus-topbar {
  background: color-mix(in srgb, var(--nexus-surface-1) 92%, transparent);
  border-block-end: 1px solid var(--nexus-border);
  backdrop-filter: blur(12px);
}

.nexus-topbar__title {
  min-width: 0;
}

.nexus-topbar__page {
  display: block;
  font-size: 0.95rem;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nexus-topbar__host {
  color: rgb(var(--v-theme-on-surface) / 62%);
  display: block;
  font-size: 0.74rem;
  letter-spacing: 0;
  line-height: 1.1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
