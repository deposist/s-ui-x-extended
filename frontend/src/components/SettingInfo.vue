<template>
  <v-tooltip :text="text" location="top" max-width="320" open-on-click>
    <template v-slot:activator="{ props }">
      <v-icon
        v-bind="props"
        :size="size"
        icon="mdi-information-outline"
        class="setting-info-icon"
        tabindex="0"
        role="img"
        :aria-label="text"
      />
    </template>
  </v-tooltip>
</template>

<script lang="ts" setup>
// Small, reusable “(i)” affordance that explains a field and its default value on
// hover/focus/tap. Drop it into a field's append-inner slot or next to a switch.
withDefaults(defineProps<{ text: string; size?: string | number }>(), {
  size: 'small',
})
</script>

<style scoped>
.setting-info-icon {
  opacity: 0.55;
  cursor: help;
}
.setting-info-icon:hover,
.setting-info-icon:focus-visible {
  opacity: 1;
}
</style>
