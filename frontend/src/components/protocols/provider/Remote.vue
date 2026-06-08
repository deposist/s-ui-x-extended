<template>
  <v-card subtitle="Remote">
    <v-row>
      <v-col cols="12">
        <v-text-field
          hide-details
          :label="$t('types.provider.url')"
          v-model="data.url">
        </v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          hide-details
          :label="$t('types.provider.userAgent')"
          v-model="data.user_agent">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-combobox
          hide-details
          clearable
          :label="$t('types.provider.downloadDetour')"
          :items="tags"
          v-model="data.download_detour">
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          hide-details
          :label="$t('types.provider.updateInterval')"
          v-model="data.update_interval">
        </v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6">
        <v-text-field
          hide-details
          :label="$t('types.provider.include')"
          v-model="data.include">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field
          hide-details
          :label="$t('types.provider.exclude')"
          v-model="data.exclude">
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
  props: ['data', 'tags'],
  components: { HealthCheck },
}
</script>
