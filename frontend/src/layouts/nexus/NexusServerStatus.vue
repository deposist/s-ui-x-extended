<template>
  <div
    class="nexus-server-status"
    :class="`nexus-server-status--${statusTone}`"
    :title="$t(statusLabel)"
  >
    <v-icon
      class="nexus-server-status__icon"
      :icon="statusIcon"
      size="20"
    />
    <div v-if="!rail" class="nexus-server-status__copy">
      <strong>S-UI</strong>
      <span>{{ $t(statusLabel) }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

import Ws from '@/store/ws'

defineProps<{
  rail: boolean
}>()

const ws = Ws()

const statusLabel = computed(() => {
  if (ws.state === 'connected') {
    return 'nexus.status.online'
  }

  return ws.state === 'reconnecting' ? 'nexus.status.loading' : 'nexus.status.failed'
})

const statusIcon = computed(() => {
  if (ws.state === 'connected') {
    return 'mdi-server-network'
  }

  return ws.state === 'reconnecting' ? 'mdi-sync' : 'mdi-server-off'
})

const statusTone = computed(() => {
  if (ws.state === 'connected') {
    return 'success'
  }

  return ws.state === 'reconnecting' ? 'info' : 'error'
})
</script>

<style scoped>
.nexus-server-status {
  align-items: center;
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  display: flex;
  gap: var(--nexus-gap-2);
  margin-inline: var(--nexus-gap-2);
  min-height: 42px;
  min-width: 0;
  padding: var(--nexus-gap-2);
}

.nexus-server-status__copy {
  display: grid;
  line-height: 1.15;
  min-width: 0;
}

.nexus-server-status__copy strong,
.nexus-server-status__copy span {
  letter-spacing: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nexus-server-status__copy strong {
  font-size: 0.78rem;
}

.nexus-server-status__copy span {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-size: 0.72rem;
}

.nexus-server-status__icon {
  flex: 0 0 auto;
}

.nexus-server-status--success .nexus-server-status__icon {
  color: var(--nexus-status-success);
}

.nexus-server-status--info .nexus-server-status__icon {
  color: var(--nexus-status-info);
}

.nexus-server-status--error .nexus-server-status__icon {
  color: var(--nexus-status-error);
}
</style>
