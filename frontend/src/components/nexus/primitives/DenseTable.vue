<template>
  <div class="nexus-dense-table">
    <table class="nexus-dense-table__table">
      <slot />
    </table>
  </div>
</template>

<style scoped>
.nexus-dense-table {
  /* Reference: panel/table container = surface-2 (#1a1a1a); the thead row is the
   * darker surface-1 (#151515) — set in NexusTableHeader. */
  background: var(--nexus-surface-2);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-md);
  min-width: 0;
  scrollbar-width: thin;
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
  font-size: 0.875rem;
  height: 44px;
  letter-spacing: 0;
  line-height: 1.4;
  min-width: 0;
  overflow-wrap: anywhere;
  padding: var(--nexus-gap-3);
  text-align: start;
  vertical-align: middle;
}

.nexus-dense-table__table :deep(th) {
  color: rgb(var(--v-theme-on-surface) / 68%);
  font-weight: 650;
}

/* Reference: table body text is secondary grey (#a0a0a0); only <strong>/tags and
 * explicit values are white. Cells inherit this unless they set their own colour
 * (tag → text-primary, badges → tone), matching the prototype. */
.nexus-dense-table__table :deep(td) {
  color: var(--nexus-text-secondary);
}

.nexus-dense-table__table :deep(tbody tr:last-child td) {
  border-block-end: 0;
}

@media (max-width: 700px) {
  .nexus-dense-table {
    -webkit-overflow-scrolling: touch;
    box-shadow: inset -14px 0 18px -18px var(--nexus-text-muted);
    overscroll-behavior-x: contain;
  }

  .nexus-dense-table__table {
    min-width: max(720px, 100%);
    table-layout: auto;
  }

  .nexus-dense-table__table :deep(th),
  .nexus-dense-table__table :deep(td) {
    min-width: 96px;
    overflow-wrap: normal;
    white-space: nowrap;
    word-break: normal;
  }

  .nexus-dense-table__table :deep(td) {
    max-width: 240px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .nexus-dense-table__table :deep([data-column-key="actor"]),
  .nexus-dense-table__table :deep([data-column-key="dateTime"]),
  .nexus-dense-table__table :deep([data-column-key="download_detour"]),
  .nexus-dense-table__table :deep([data-column-key="event"]),
  .nexus-dense-table__table :deep([data-column-key="resource"]),
  .nexus-dense-table__table :deep([data-column-key="server"]),
  .nexus-dense-table__table :deep([data-column-key="tag"]) {
    min-width: 144px;
  }

  .nexus-dense-table__table :deep([data-column-key="actions"]) {
    background: var(--nexus-surface-2);
    box-shadow: -1px 0 0 var(--nexus-border);
    min-width: 104px;
    position: sticky;
    right: 0;
    z-index: 2;
  }

  .nexus-dense-table__table :deep(.nexus-thead__actions) {
    background: var(--nexus-surface-1);
    z-index: 3;
  }
}
</style>
