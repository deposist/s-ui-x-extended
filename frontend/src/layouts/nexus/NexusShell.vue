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
          @toggle-rail="toggleRail"
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

        <confirm-host />
      </v-app>
    </v-defaults-provider>
  </v-theme-provider>
</template>

<script lang="ts" setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useDisplay, useLocale } from 'vuetify'

import ConfirmHost from '@/components/nexus/primitives/ConfirmHost.vue'
import { useNexusTheme } from '@/uiMode/nexusTheme'
import { safeGetItem, safeSetItem } from '@/utils/safeStorage'
import NexusServerStatus from './NexusServerStatus.vue'
import NexusSidebar from './NexusSidebar.vue'
import NexusTopbar from './NexusTopbar.vue'

const { isRtl } = useLocale()
const { lgAndUp, smAndDown } = useDisplay()

const RAIL_STORAGE_KEY = 'nexus.sidebar.rail'

const readStoredRail = () => safeGetItem(RAIL_STORAGE_KEY) === '1'

const isMobile = computed(() => smAndDown.value)
// Auto-collapse only on true tablet widths (<= 1023px). `mdAndDown` covers up to
// 1279px, which collapsed the sidebar on ordinary laptops and left 13 unlabeled
// icons — the nav groups became unrecognisable at a glance.
const isTablet = computed(() => !smAndDown.value && !lgAndUp.value)
// Reference keeps the sidebar expanded on desktop; only tablet auto-collapses to
// a rail. A manual toggle (sidebar brand hamburger) can also collapse it, and
// that preference is remembered across reloads.
const manualRail = ref(readStoredRail())
const sidebarRail = computed(() => !isMobile.value && (isTablet.value || manualRail.value))
const sidebarOpen = ref(true)
const toggleRail = () => {
  manualRail.value = !manualRail.value
  safeSetItem(RAIL_STORAGE_KEY, manualRail.value ? '1' : '0')
}

watch(isMobile, async (mobile) => {
  await nextTick()
  sidebarOpen.value = !mobile
}, { immediate: true })

// Palette→theme resolution (and the data-ui-palette sync that drives the
// pre-mount token blocks) lives in useNexusTheme so the login route, which
// renders outside this shell, can reuse the exact same accent colours.
const nexusThemeName = useNexusTheme()

const nexusDefaults = {
  VBtn: {
    // Default density (36px height) + 4px radius to match the reference buttons.
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
  /* Reference page padding: 24px desktop, 16px tablet/mobile. */
  padding: var(--nexus-gap-5);
}

@media (max-width: 960px) {
  .nexus-shell__view {
    padding: var(--nexus-gap-4);
  }
}
</style>
