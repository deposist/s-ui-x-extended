<template>
  <v-app-bar class="nexus-topbar" flat height="64">
    <template #prepend>
      <v-btn
        v-if="showNavigationToggle"
        :aria-label="$t('menu.navigation')"
        icon="lucide:menu"
        :title="$t('menu.navigation')"
        variant="text"
        @click="emit('toggle-navigation')"
      />
    </template>

    <div class="nexus-topbar__header">
      <template v-if="pageHeader.active">
        <div v-if="!compactActions || !pageHeader.searchable" class="nexus-topbar__titles">
          <span class="nexus-topbar__page">{{ pageHeader.title }}</span>
          <span v-if="pageHeader.subtitle" class="nexus-topbar__sub">{{ pageHeader.subtitle }}</span>
        </div>
        <v-spacer v-if="!compactActions || !pageHeader.searchable" />
        <div
          v-if="pageHeader.searchable"
          class="nexus-topbar__search"
          :class="{ 'nexus-topbar__search--compact': compactActions }"
        >
          <v-text-field
            :aria-label="$t('table.search')"
            clearable
            density="compact"
            hide-details
            :model-value="topbarSearch"
            :placeholder="$t('table.search')"
            prepend-inner-icon="lucide:search"
            variant="outlined"
            @update:model-value="topbarSearch = $event ?? ''"
          />
        </div>
      </template>
      <span v-else class="nexus-topbar__page">{{ $t(<string>route.name) }}</span>
    </div>

    <template #append>
      <v-menu v-if="compactActions">
        <template #activator="{ props }">
          <v-btn
            :aria-label="$t('actions.action')"
            icon="lucide:more-vertical"
            :title="$t('actions.action')"
            variant="text"
            v-bind="props"
          />
        </template>
        <v-list density="compact" min-width="240">
          <v-list-subheader>{{ $t('menu.language') }}</v-list-subheader>
          <v-list-item
            v-for="language in languages"
            :key="language.value"
            :active="isActiveLocale(language.value)"
            prepend-icon="lucide:languages"
            @click="changeLocale(language.value)"
          >
            <v-list-item-title>{{ language.title }}</v-list-item-title>
          </v-list-item>
          <v-divider />
          <v-list-subheader>{{ $t('menu.theme') }}</v-list-subheader>
          <v-list-item
            v-for="item in themes"
            :key="item.value"
            :active="isActiveTheme(item.value)"
            :prepend-icon="item.icon"
            @click="changeTheme(item.value)"
          >
            <v-list-item-title>{{ $t(`theme.${item.value}`) }}</v-list-item-title>
          </v-list-item>
          <v-divider />
          <v-list-subheader>{{ $t('nexus.palette.label') }}</v-list-subheader>
          <v-list-item
            v-for="item in palettes"
            :key="item"
            :active="palette === item"
            prepend-icon="lucide:palette"
            @click="setPalette(item)"
          >
            <v-list-item-title>{{ $t(`nexus.palette.options.${item}`) }}</v-list-item-title>
          </v-list-item>

          <template v-if="uiModeEnabled">
            <v-divider />
            <v-list-item :prepend-icon="quickIcon" :title="quickLabel" @click="toggleMode" />
          </template>
        </v-list>
      </v-menu>

      <template v-else>
        <v-menu>
          <template #activator="{ props }">
            <v-btn
              :aria-label="$t('menu.language')"
              icon
              :title="$t('menu.language')"
              variant="text"
              v-bind="props"
            >
              <v-icon icon="lucide:languages" />
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
            <v-btn
              :aria-label="$t('menu.theme')"
              icon
              :title="$t('menu.theme')"
              variant="text"
              v-bind="props"
            >
              <v-icon icon="lucide:sun-moon" />
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

        <v-menu>
          <template #activator="{ props }">
            <v-btn
              :aria-label="$t('nexus.palette.label')"
              icon
              :title="$t('nexus.palette.label')"
              variant="text"
              v-bind="props"
            >
              <v-icon icon="lucide:palette" />
            </v-btn>
          </template>
          <v-list>
            <v-list-item
              v-for="item in palettes"
              :key="item"
              :active="palette === item"
              @click="setPalette(item)"
            >
              <v-list-item-title>{{ $t(`nexus.palette.options.${item}`) }}</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>

        <ui-mode-control variant="quick" />
      </template>
    </template>
  </v-app-bar>
</template>

<script lang="ts" setup>
import UiModeControl from '@/components/UiModeControl.vue'
import { pageHeader, topbarSearch } from '@/components/nexus/primitives/pageHeaderPortal'
import { languages, setI18nLocale } from '@/locales'
import { isNexusEnabled } from '@/uiMode/featureGate'
import { UI_PALETTES, useUiPalette } from '@/uiMode/palette'
import type { UiMode } from '@/uiMode/types'
import { useUiMode } from '@/uiMode/useUiMode'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useDisplay, useLocale, useTheme } from 'vuetify'

defineProps<{
  showNavigationToggle: boolean
}>()

const route = useRoute()

const emit = defineEmits<{
  'toggle-navigation': []
}>()

const theme = useTheme()
const { smAndDown } = useDisplay()
const vuetifyLocale = useLocale()
const { locale: i18nLocale, t } = useI18n()
const { palette, setPalette } = useUiPalette()
const { mode, setMode } = useUiMode()
const palettes = UI_PALETTES
const compactActions = computed(() => smAndDown.value)
const uiModeEnabled = isNexusEnabled()
const nexusMode: UiMode = 'nexus'
const classicMode: UiMode = 'classic'
const nextMode = computed<UiMode>(() => mode.value === nexusMode ? classicMode : nexusMode)
const quickIcon = computed(() =>
  mode.value === nexusMode ? 'lucide:layout-dashboard' : 'lucide:layout-panel-left',
)
const quickLabel = computed(() =>
  t('nexus.mode.switchTo', { mode: t(`nexus.mode.options.${nextMode.value}`) }),
)

const changeLocale = async (nextLocale: string) => {
  const selectedLocale = await setI18nLocale(nextLocale)
  i18nLocale.value = selectedLocale
  vuetifyLocale.current.value = selectedLocale
  window.location.reload()
}

const isActiveLocale = (locale: string) => i18nLocale.value === locale

const themes = [
  { value: 'light', icon: 'lucide:sun' },
  { value: 'dark', icon: 'lucide:moon' },
  { value: 'system', icon: 'lucide:monitor' },
]

const changeTheme = (nextTheme: string) => {
  theme.change(nextTheme)
  localStorage.setItem('theme', nextTheme)
}

const isActiveTheme = (value: string) => {
  // Mirror vuetify.ts defaultTheme: no stored choice → dark.
  const currentTheme = localStorage.getItem('theme') ?? 'dark'

  return currentTheme === value
}

const toggleMode = () => setMode(nextMode.value)
</script>

<style scoped>
.nexus-topbar {
  /* Solid reference surface (#151515) — same as the sidebar header — instead of a
   * translucent shade, so the colour is exact, not "close". */
  background: var(--nexus-surface-1);
  border-block-end: 1px solid var(--nexus-border);
}

.nexus-topbar :deep(.v-toolbar__append) {
  flex: 0 0 auto;
}

/* Renders the active page's section header (title + stats + search) from shared
 * state, between the mobile toggle and the global controls. Title starts at the
 * SAME left offset on every tab (padding-inline-start), so tabs don't jump. */
.nexus-topbar__header {
  align-items: center;
  display: flex;
  flex: 1 1 auto;
  gap: var(--nexus-gap-4);
  min-width: 0;
  padding-inline-start: var(--nexus-gap-3);
}

.nexus-topbar__titles {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
}

.nexus-topbar__page {
  color: var(--nexus-text-primary);
  font-size: 1.05rem;
  font-weight: 600;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nexus-topbar__sub {
  color: var(--nexus-text-secondary);
  font-size: 0.74rem;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nexus-topbar__search {
  flex: 0 1 260px;
  min-width: 160px;
}

.nexus-topbar__search--compact {
  flex: 1 1 auto;
  min-width: 0;
}

/* Reference search input: filled #202020 surface, cyan focus. */
.nexus-topbar__search :deep(.v-field) {
  background: var(--nexus-elevated);
  border-radius: var(--nexus-radius-sm);
}

@media (max-width: 600px) {
  .nexus-topbar__header {
    gap: var(--nexus-gap-2);
    padding-inline-start: 0;
  }

  .nexus-topbar__page {
    font-size: 1rem;
  }

  .nexus-topbar__sub {
    display: none;
  }
}
</style>
