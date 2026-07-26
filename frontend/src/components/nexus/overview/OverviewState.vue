<template>
  <div class="nexus-overview-panel__state">
    <p class="nexus-overview-panel__state-text">
      <slot />
    </p>
    <!-- Optional call to action so an empty panel points at the next step
         instead of being a dead end ("No data" with nowhere to go). -->
    <div v-if="$slots.action" class="nexus-overview-panel__state-action">
      <slot name="action" />
    </div>
  </div>
</template>

<style scoped>
.nexus-overview-panel__state {
  align-items: center;
  background: var(--nexus-surface-2);
  border: 1px dashed var(--nexus-border-strong);
  border-radius: var(--nexus-radius-md);
  color: rgb(var(--v-theme-on-surface) / 68%);
  display: flex;
  flex-direction: column;
  gap: var(--nexus-gap-2);
  font-size: 0.86rem;
  justify-content: center;
  letter-spacing: 0;
  line-height: 1.4;
  min-height: auto;
  overflow-wrap: anywhere;
  padding: var(--nexus-gap-3);
  text-align: center;
}

.nexus-overview-panel__state-text {
  margin: 0;
  max-width: 42ch;
  /* These messages are centred, so an unbalanced last line ("...are set up
   * / yet.") reads as a mistake. Balancing evens the lines out without
   * touching the 42ch readability cap. */
  text-wrap: balance;
}

.nexus-overview-panel__state-action {
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2);
  justify-content: center;
}
</style>
