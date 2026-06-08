<template>
  <v-card subtitle="Health Check" class="mt-2">
    <v-row>
      <v-col cols="12" sm="6" md="3">
        <v-switch
          color="primary"
          hide-details
          :label="$t('enable')"
          v-model="healthCheck.enabled"
          @update:model-value="onEnableChange">
        </v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-combobox
          hide-details
          :label="$t('types.provider.healthUrl')"
          :items="healthCheckUrls"
          v-model="healthCheck.url">
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-combobox
          hide-details
          :label="$t('types.provider.healthInterval')"
          :items="durationPresets"
          v-model="healthCheck.interval">
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-combobox
          hide-details
          :label="$t('types.provider.healthTimeout')"
          :items="durationPresets"
          v-model="healthCheck.timeout">
        </v-combobox>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import { healthCheckUrls, durationPresets, RECOMMENDED } from '@/types/recommended'

export default {
  props: { data: { type: Object, required: true } },
  data() {
    return { healthCheckUrls, durationPresets }
  },
  created() {
    if (!this.$props.data.health_check) {
      this.$props.data.health_check = {}
    }
  },
  computed: {
    healthCheck(): any {
      return this.$props.data.health_check
    },
  },
  methods: {
    onEnableChange(value: any) {
      if (value) {
        this.healthCheck.url ??= RECOMMENDED.healthCheckUrl
        this.healthCheck.interval ??= RECOMMENDED.healthCheckInterval
        this.healthCheck.timeout ??= RECOMMENDED.healthCheckTimeout
      }
    },
  },
}
</script>
