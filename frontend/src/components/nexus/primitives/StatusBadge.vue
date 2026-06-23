<template>
  <span class="nexus-status-badge" :class="toneClasses[tone]">
    <span aria-hidden="true" class="nexus-status-badge__dot" />
    <span class="nexus-status-badge__label">{{ label }}</span>
  </span>
</template>

<script lang="ts" setup>
type StatusBadgeTone = 'info' | 'success' | 'warning' | 'error' | 'neutral'

defineProps<{
  label: string
  tone: StatusBadgeTone
}>()

const toneClasses: Record<StatusBadgeTone, string> = {
  info: 'nexus-status-badge--info',
  success: 'nexus-status-badge--success',
  warning: 'nexus-status-badge--warning',
  error: 'nexus-status-badge--error',
  neutral: 'nexus-status-badge--neutral',
}
</script>

<style scoped>
.nexus-status-badge {
  align-items: center;
  /* Reference .status-badge: rgba(tone, 0.1) tint, no border, 12px/500, dot + label.
   * color-mix(... 10%, transparent) == rgba(tone, 0.1) exactly. */
  background: color-mix(in srgb, var(--nexus-status-badge-tone) 10%, transparent);
  border-radius: var(--nexus-radius-sm);
  color: var(--nexus-status-badge-tone);
  display: inline-flex;
  font-size: 0.75rem;
  font-weight: 500;
  gap: var(--nexus-gap-1);
  letter-spacing: 0;
  line-height: 1.25;
  max-width: 100%;
  min-width: 0;
  padding: 4px var(--nexus-gap-2);
}

.nexus-status-badge__dot {
  background: currentColor;
  border-radius: 50%;
  flex: 0 0 auto;
  height: 6px;
  width: 6px;
}

.nexus-status-badge__label {
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-status-badge--info {
  --nexus-status-badge-tone: var(--nexus-status-info);
}

.nexus-status-badge--success {
  --nexus-status-badge-tone: var(--nexus-status-success);
}

.nexus-status-badge--warning {
  --nexus-status-badge-tone: var(--nexus-status-warn);
}

.nexus-status-badge--error {
  --nexus-status-badge-tone: var(--nexus-status-error);
}

.nexus-status-badge--neutral {
  --nexus-status-badge-tone: var(--nexus-text-muted);
}
</style>
