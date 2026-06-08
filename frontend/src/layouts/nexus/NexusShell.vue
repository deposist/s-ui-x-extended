<template>
  <v-theme-provider :theme="nexusThemeName">
    <v-defaults-provider :defaults="nexusDefaults">
      <v-app
        class="nexus-shell"
        :class="{ 'nexus-shell--rtl': isRtl }"
        :dir="isRtl ? 'rtl' : 'ltr'"
      >
        <nexus-sidebar
          :open="sidebarOpen"
          :rail="sidebarRail"
          :rtl="isRtl"
          :temporary="isMobile"
          @navigate="closeTemporarySidebar"
          @update:open="sidebarOpen = $event"
        >
          <template #footer>
            <nexus-server-status :rail="sidebarRail" />
          </template>
        </nexus-sidebar>

        <nexus-topbar
          :show-navigation-toggle="isMobile"
          @toggle-navigation="toggleSidebar"
        />

        <v-main class="nexus-shell__main">
          <div class="nexus-shell__view">
            <slot>
              <router-view />
            </slot>
          </div>
        </v-main>
      </v-app>
    </v-defaults-provider>
  </v-theme-provider>
</template>

<script lang="ts" setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useDisplay, useLocale, useTheme } from 'vuetify'

import NexusServerStatus from './NexusServerStatus.vue'
import NexusSidebar from './NexusSidebar.vue'
import NexusTopbar from './NexusTopbar.vue'

const theme = useTheme()
const { isRtl } = useLocale()
const { lgAndDown, mdAndDown, smAndDown } = useDisplay()

const isMobile = computed(() => smAndDown.value)
const isTablet = computed(() => !smAndDown.value && mdAndDown.value)
const isNarrowDesktop = computed(() => !mdAndDown.value && lgAndDown.value)
const sidebarRail = computed(() =>
  !isMobile.value && (isTablet.value || isNarrowDesktop.value),
)
const sidebarOpen = ref(true)

watch(isMobile, async (mobile) => {
  await nextTick()
  sidebarOpen.value = !mobile
}, { immediate: true })

const nexusThemeName = computed(() => {
  const activeThemeName = theme.global.name.value
  const systemIsDark = activeThemeName === 'system' && theme.global.current.value.dark

  return activeThemeName === 'dark' || systemIsDark ? 'nexusDark' : 'nexusLight'
})

const nexusDefaults = {
  VBtn: {
    density: 'compact',
    rounded: 'sm',
  },
  VCard: {
    elevation: 0,
    rounded: 'lg',
  },
  VChip: {
    density: 'compact',
    rounded: 'sm',
  },
  VCombobox: {
    density: 'compact',
    variant: 'outlined',
  },
  VList: {
    density: 'compact',
  },
  VListItem: {
    density: 'compact',
    rounded: 'sm',
  },
  VSelect: {
    density: 'compact',
    variant: 'outlined',
  },
  VTextarea: {
    density: 'compact',
    variant: 'outlined',
  },
  VTextField: {
    density: 'compact',
    variant: 'outlined',
  },
}

const toggleSidebar = () => {
  sidebarOpen.value = !sidebarOpen.value
}

const closeTemporarySidebar = () => {
  if (isMobile.value) {
    sidebarOpen.value = false
  }
}
</script>

<style lang="scss">
@use '@/styles/nexus/tokens';

html[data-ui-mode='nexus'] body {
  background: var(--nexus-surface-0);
}

.nexus-shell {
  background: var(--nexus-surface-0);
  color: rgb(var(--v-theme-on-background));
  min-height: 100vh;
}

.nexus-shell__main {
  background: var(--nexus-surface-0);
  min-width: 0;
}

.nexus-shell__view {
  min-height: 100%;
  min-width: 0;
  padding: var(--nexus-gap-4);
}

@media (max-width: 960px) {
  .nexus-shell__view {
    padding: var(--nexus-gap-3);
  }
}

@media (max-width: 600px) {
  .nexus-shell__view {
    padding: var(--nexus-gap-2);
  }
}
</style>
