<template>
  <span class="nexus-badge" :class="`nexus-badge--${variant}`">{{ label }}</span>
</template>

<script lang="ts" setup>
// Reference `.badge`: a small UPPERCASE pill WITHOUT a status dot — used for
// boolean/metadata chips like TLS On/Off and client tags. Distinct from
// StatusBadge (which renders online/offline state with a coloured dot).
type BadgeVariant = 'success' | 'secondary' | 'warning'

withDefaults(defineProps<{
  label: string
  variant?: BadgeVariant
}>(), {
  variant: 'secondary',
})
</script>

<style scoped>
.nexus-badge {
  align-items: center;
  border-radius: var(--nexus-radius-sm);
  display: inline-flex;
  font-size: 0.6875rem;
  font-weight: 500;
  letter-spacing: 0.3px;
  line-height: 1.4;
  padding: 2px var(--nexus-gap-2);
  text-transform: uppercase;
  white-space: nowrap;
}

.nexus-badge--success {
  background: color-mix(in srgb, var(--nexus-status-success) 15%, transparent);
  color: var(--nexus-status-success);
}

.nexus-badge--warning {
  background: color-mix(in srgb, var(--nexus-status-warn) 15%, transparent);
  color: var(--nexus-status-warn);
}

.nexus-badge--secondary {
  background: var(--nexus-elevated);
  color: var(--nexus-text-secondary);
}
</style>
