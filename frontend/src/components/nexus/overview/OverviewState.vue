<template>
  <div class="nexus-overview-panel__state">
    <slot />
  </div>
</template>

<style scoped>
.nexus-overview-panel__state {
  align-items: center;
  background: var(--nexus-surface-2);
  border: 1px dashed var(--nexus-border-strong);
  border-radius: var(--nexus-radius-md);
  color: rgb(var(--v-theme-on-surface) / 68%);
  display: grid;
  font-size: 0.86rem;
  letter-spacing: 0;
  line-height: 1.4;
  min-height: auto;
  overflow-wrap: anywhere;
  padding: var(--nexus-gap-3);
  text-align: center;
}
</style>
