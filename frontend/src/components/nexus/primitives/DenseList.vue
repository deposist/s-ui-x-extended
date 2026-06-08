<template>
  <ul class="nexus-dense-list">
    <slot />
  </ul>
</template>

<style scoped>
.nexus-dense-list {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  list-style: none;
  margin: 0;
  min-width: 0;
  overflow: hidden;
  padding: 0;
}

.nexus-dense-list :slotted(li) {
  align-items: center;
  border-block-end: 1px solid var(--nexus-border);
  display: flex;
  gap: var(--nexus-gap-2);
  letter-spacing: 0;
  min-height: 38px;
  min-width: 0;
  padding: var(--nexus-gap-2) var(--nexus-gap-3);
}

.nexus-dense-list :slotted(li:last-child) {
  border-block-end: 0;
}
</style>
