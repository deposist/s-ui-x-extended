<template>
  <v-navigation-drawer
    :key="temporary ? 'mobile' : 'desktop'"
    class="nexus-sidebar"
    :location="rtl ? 'right' : 'left'"
    :model-value="open"
    :permanent="!temporary"
    :rail="rail && !temporary"
    :rail-width="60"
    :temporary="temporary"
    :width="240"
    @update:model-value="emit('update:open', $event)"
  >
    <div class="nexus-sidebar__brand">
      <button
        v-if="collapsed"
        class="nexus-sidebar__logo nexus-sidebar__logo--button"
        type="button"
        :aria-label="$t('menu.navigation')"
        @click="emit('toggle-rail')"
      >X</button>
      <div v-else aria-hidden="true" class="nexus-sidebar__logo">X</div>

      <span v-if="!collapsed" class="nexus-sidebar__brand-text">S-UI-X Extended</span>
      <v-spacer v-if="!collapsed" />

      <v-btn
        v-if="!collapsed && !temporary"
        :aria-label="$t('menu.navigation')"
        icon="lucide:menu"
        size="small"
        variant="text"
        @click="emit('toggle-rail')"
      />
      <v-btn
        v-if="!collapsed && temporary"
        :aria-label="$t('actions.close')"
        icon="lucide:x"
        variant="text"
        @click="emit('update:open', false)"
      />
    </div>

    <v-divider />

    <v-list
      :aria-label="$t('menu.navigation')"
      class="nexus-sidebar__navigation"
      nav
      role="navigation"
    >
      <template v-for="group in groups" :key="group.labelKey ?? 'dashboard'">
        <v-list-subheader
          v-if="group.labelKey && !collapsed"
          class="nexus-sidebar__group"
        >
          {{ $t(group.labelKey) }}
        </v-list-subheader>

        <v-list-item
          v-for="item in group.items"
          :key="item.path"
          :active="itemActive(item.path)"
          active-class="nexus-sidebar__item--active"
          :aria-label="$t(item.title)"
          class="nexus-sidebar__item"
          link
          :to="item.path"
          @click="emit('navigate')"
        >
          <template #prepend>
            <v-badge
              :content="badgeCount(item)"
              color="primary"
              :model-value="collapsed && badgeCount(item) > 0"
              offset-x="-2"
              offset-y="-2"
            >
              <v-icon :icon="item.icon" />
            </v-badge>
          </template>

          <v-list-item-title>{{ $t(item.title) }}</v-list-item-title>

          <template #append v-if="!collapsed && badgeCount(item) > 0">
            <span class="nexus-sidebar__badge">{{ badgeCount(item) }}</span>
          </template>

          <v-tooltip
            v-if="collapsed"
            activator="parent"
            :location="rtl ? 'start' : 'end'"
          >
            {{ $t(item.title) }}
          </v-tooltip>
        </v-list-item>
      </template>
    </v-list>

    <template #append>
      <div class="nexus-sidebar__footer">
        <v-divider />
        <v-list nav>
          <v-list-item
            :aria-label="$t('menu.logout')"
            prepend-icon="lucide:log-out"
            :title="$t('menu.logout')"
            @click="logout"
          />
        </v-list>
        <slot name="footer" />
      </div>
    </template>
  </v-navigation-drawer>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { logout } from '@/plugins/httputil'
import Data from '@/store/modules/data'
import { nexusMenuGroups as groups, type NexusMenuItem } from './nexusMenu'

const props = defineProps<{
  open: boolean
  rail: boolean
  rtl: boolean
  temporary: boolean
}>()

const emit = defineEmits<{
  navigate: []
  'toggle-rail': []
  'update:open': [value: boolean]
}>()

const data = Data()
const route = useRoute()

const collapsed = computed(() => props.rail && !props.temporary)

// Exact-match the dashboard ('/') so it isn't flagged active on every route
// (Vue Router treats '/' as a prefix of all paths); other items match their
// own path or sub-paths.
const itemActive = (path: string): boolean =>
  path === '/' ? route.path === '/' : route.path === path || route.path.startsWith(`${path}/`)

const badgeCount = (item: NexusMenuItem): number => {
  if (!item.countKey) return 0

  const collection = data[item.countKey]

  return Array.isArray(collection) ? collection.length : 0
}
</script>

<style scoped>
.nexus-sidebar {
  background: var(--nexus-surface-1);
  border-color: var(--nexus-border);
}

.nexus-sidebar__brand {
  align-items: center;
  display: flex;
  gap: var(--nexus-gap-3);
  height: 60px;
  letter-spacing: 0;
  padding-inline: var(--nexus-gap-4);
}

.nexus-sidebar__logo {
  align-items: center;
  background: var(--nexus-accent-primary);
  border-radius: var(--nexus-radius-md);
  color: var(--nexus-surface-0);
  display: inline-flex;
  flex: 0 0 auto;
  font-size: 18px;
  font-weight: 700;
  height: 32px;
  justify-content: center;
  line-height: 1;
  width: 32px;
}

.nexus-sidebar__logo--button {
  border: 0;
  cursor: pointer;
}

.nexus-sidebar__brand-text {
  color: var(--nexus-text-primary);
  font-size: 16px;
  font-weight: 600;
}

/* Slim scrollbar to match the reference sidebar nav (6px). */
.nexus-sidebar :deep(.v-navigation-drawer__content)::-webkit-scrollbar {
  width: 6px;
}

.nexus-sidebar :deep(.v-navigation-drawer__content)::-webkit-scrollbar-thumb {
  background: var(--nexus-border);
  border-radius: 3px;
}

/* Full-width flat rows (no rounded inset) like the reference .sidebar-nav. */
.nexus-sidebar__navigation {
  padding-block: var(--nexus-gap-1);
  padding-inline: 0;
}

/* Reference .nav-group-label: 11px/600, tertiary grey (#666). Tight vertical
 * footprint (Vuetify subheaders are ~40px by default) so the whole sidebar is
 * as compact as the reference. */
.nexus-sidebar__group {
  color: var(--nexus-text-muted);
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.5px;
  line-height: 1.2;
  min-height: 0;
  padding-block: var(--nexus-gap-3) var(--nexus-gap-1);
  padding-inline: var(--nexus-gap-4);
  text-transform: uppercase;
}

.nexus-sidebar__group :deep(.v-list-subheader__text) {
  color: var(--nexus-text-muted);
}

/* Reference .nav-item: 40px tall, full-width, 16px inline padding, 12px icon→label
 * gap; secondary grey, active = subtle #252525 band + 3px cyan left bar + cyan
 * text. No Vuetify rounding/tonal overlay. */
.nexus-sidebar__item {
  border-radius: 0;
  color: var(--nexus-text-secondary);
  margin-block: 0;
  margin-inline: 0;
  min-height: 40px;
  padding-block: 0;
  padding-inline: var(--nexus-gap-4);
  transition: background var(--nexus-transition-fast), color var(--nexus-transition-fast), opacity var(--nexus-transition-fast);
  opacity: 0.72;
}

/* Compact rows (reference 40px) — drop Vuetify's extra content padding. */
.nexus-sidebar__item :deep(.v-list-item__content) {
  padding-block: 0;
}

/* Drop Vuetify's tonal hover/active overlay so the flat reference band shows. */
.nexus-sidebar__item :deep(.v-list-item__overlay) {
  display: none;
}

/* Tighten the icon→label gap to the reference 12px. */
.nexus-sidebar__item :deep(.v-list-item__spacer) {
  width: var(--nexus-gap-3);
}

.nexus-sidebar__item :deep(.v-list-item-title) {
  color: inherit;
  font-size: 14px;
}

.nexus-sidebar__item:hover {
  background: var(--nexus-surface-hover);
  color: var(--nexus-text-primary);
  opacity: 1;
}

.nexus-sidebar__item.nexus-sidebar__item--active {
  background: var(--nexus-surface-hover);
  box-shadow: inset 3px 0 0 var(--nexus-accent-primary);
  color: var(--nexus-accent-primary);
  opacity: 1;
}

.nexus-sidebar__badge {
  background: var(--nexus-elevated);
  border-radius: var(--nexus-radius-sm);
  color: var(--nexus-text-secondary);
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1.4;
  min-width: 20px;
  padding: 2px 6px;
  text-align: center;
}

.nexus-sidebar__footer {
  background: var(--nexus-surface-1);
  border-block-start: 1px solid var(--nexus-border);
  padding-block-end: var(--nexus-gap-2);
}

.nexus-sidebar__footer .v-divider {
  margin-block-end: var(--nexus-gap-1);
}
</style>
