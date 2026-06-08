<template>
  <v-card subtitle="Local">
    <v-row>
      <v-col cols="12">
        <v-text-field
          hide-details
          :label="$t('types.provider.path')"
          v-model="data.path">
        </v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-switch
          color="primary"
          hide-details
          :label="$t('types.provider.removeEmojis')"
          v-model="data.remove_emojis">
        </v-switch>
      </v-col>
    </v-row>
    <HealthCheck :data="data" />
  </v-card>
</template>

<script lang="ts">
import HealthCheck from './HealthCheck.vue'

export default {
  props: ['data'],
  components: { HealthCheck },
}
</script>
