<template>
  <v-card subtitle="Core Failover">
    <v-row>
      <v-col cols="12">
        <div class="text-caption mb-1">{{ $t('types.coreFailover.outbounds') }}</div>
        <v-combobox
          hide-details
          multiple
          chips
          closable-chips
          :items="tags || []"
          :label="$t('types.coreFailover.outboundsHint')"
          v-model="data.outbounds"
        ></v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :items="strategies"
          :label="$t('types.coreFailover.strategy')"
          v-model="data.strategy"
        ></v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          hide-details
          :label="$t('types.coreFailover.delay')"
          v-model="data.delay"
          placeholder="0s"
        ></v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data', 'tags'],
  data() {
    return {
      strategies: ['', 'sequential', 'cycle'],
    }
  },
}
</script>
