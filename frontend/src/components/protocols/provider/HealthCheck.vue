<template>
  <v-card subtitle="Health Check" class="mt-2">
    <v-row>
      <v-col cols="12" sm="6" md="3">
        <v-switch
          color="primary"
          hide-details
          :label="$t('enable')"
          v-model="healthCheck.enabled">
        </v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-text-field
          hide-details
          :label="$t('types.provider.healthUrl')"
          v-model="healthCheck.url">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-text-field
          hide-details
          :label="$t('types.provider.healthInterval')"
          v-model="healthCheck.interval">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-text-field
          hide-details
          :label="$t('types.provider.healthTimeout')"
          v-model="healthCheck.timeout">
        </v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: { data: { type: Object, required: true } },
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
}
</script>
