<template>
  <header class="nexus-panel-header">
    <h2 class="nexus-panel-header__title">{{ title }}</h2>

    <div class="nexus-panel-header__tools">
      <div class="nexus-panel-header__range-tabs">
        <slot name="range-tabs" />
      </div>
      <div class="nexus-panel-header__action">
        <slot name="action" />
      </div>
    </div>
  </header>
</template>

<script lang="ts" setup>
defineProps<{
  title: string
}>()
</script>

<style scoped>
.nexus-panel-header {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2) var(--nexus-gap-3);
  justify-content: space-between;
  min-width: 0;
}

.nexus-panel-header__title {
  font-size: 1rem;
  font-weight: 650;
  letter-spacing: 0;
  line-height: 1.25;
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.nexus-panel-header__tools {
  align-items: center;
  display: flex;
  flex: 0 1 auto;
  flex-wrap: wrap;
  gap: var(--nexus-gap-2);
  justify-content: flex-end;
  min-width: 0;
}

.nexus-panel-header__action,
.nexus-panel-header__range-tabs {
  align-items: center;
  display: flex;
  min-width: 0;
}
</style>
