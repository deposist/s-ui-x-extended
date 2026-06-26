<template>
  <v-card subtitle="Block">
    <v-alert type="info" variant="tonal" density="compact">
      {{ $t('types.block.info') }}
    </v-alert>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data'],
}
</script>
