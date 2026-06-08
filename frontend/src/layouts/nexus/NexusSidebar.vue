<template>
  <v-navigation-drawer
    :key="temporary ? 'mobile' : 'desktop'"
    class="nexus-sidebar"
    :expand-on-hover="rail && !temporary"
    :location="rtl ? 'right' : 'left'"
    :model-value="open"
    :permanent="!temporary"
    :rail="rail && !temporary"
    :temporary="temporary"
    @update:model-value="emit('update:open', $event)"
  >
    <v-list-item
      class="nexus-sidebar__brand"
      height="64"
      prepend-avatar="@/assets/logo.svg"
      title="S-UI"
    >
      <template #append v-if="temporary">
        <v-btn
          :aria-label="$t('actions.close')"
          icon="mdi-close"
          variant="text"
          @click="emit('update:open', false)"
        />
      </template>
    </v-list-item>

    <v-divider />

    <v-list class="nexus-sidebar__navigation" nav>
      <v-list-item
        v-for="item in menu"
        :key="item.path"
        link
        :prepend-icon="item.icon"
        :title="$t(item.title)"
        :to="item.path"
        @click="emit('navigate')"
      />
    </v-list>

    <template #append>
      <div class="nexus-sidebar__footer">
        <v-divider />
        <v-list nav>
          <v-list-item
            prepend-icon="mdi-logout"
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
import { logout } from '@/plugins/httputil'
import { nexusMenu as menu } from './nexusMenu'

defineProps<{
  open: boolean
  rail: boolean
  rtl: boolean
  temporary: boolean
}>()

const emit = defineEmits<{
  navigate: []
  'update:open': [value: boolean]
}>()

</script>

<style scoped>
.nexus-sidebar {
  background: var(--nexus-surface-1);
  border-color: var(--nexus-border);
}

.nexus-sidebar__brand {
  letter-spacing: 0;
}

.nexus-sidebar__navigation {
  padding-inline: var(--nexus-gap-2);
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
