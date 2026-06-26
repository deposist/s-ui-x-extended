<template>
  <v-card subtitle="Core Failover">
    <v-row>
      <v-col cols="12">
        <div class="text-caption mb-1">{{ $t('types.coreFailoverInbound.inbounds') }}</div>
        <v-combobox
          hide-details
          multiple
          chips
          closable-chips
          :items="inTags || []"
          :label="$t('types.coreFailoverInbound.inboundsHint')"
          v-model="data.inbounds"
        ></v-combobox>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data', 'inTags'],
}
</script>
