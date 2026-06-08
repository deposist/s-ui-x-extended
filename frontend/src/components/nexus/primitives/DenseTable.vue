<template>
  <div class="nexus-dense-table">
    <table class="nexus-dense-table__table">
      <slot />
    </table>
  </div>
</template>

<style scoped>
.nexus-dense-table {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  min-width: 0;
  overflow-x: auto;
}

.nexus-dense-table__table {
  border-collapse: collapse;
  min-width: 100%;
  table-layout: fixed;
}

.nexus-dense-table__table :deep(th),
.nexus-dense-table__table :deep(td) {
  border-block-end: 1px solid var(--nexus-border);
  font-size: 0.8rem;
  height: 38px;
  letter-spacing: 0;
  line-height: 1.3;
  min-width: 0;
  overflow-wrap: anywhere;
  padding: var(--nexus-gap-2) var(--nexus-gap-3);
  text-align: start;
  vertical-align: middle;
}

.nexus-dense-table__table :deep(th) {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-weight: 650;
}

.nexus-dense-table__table :deep(tbody tr:last-child td) {
  border-block-end: 0;
}
</style>
