<template>
  <v-card subtitle="Mixed">
    <v-row>
      <v-col cols="12" sm="6">
        <v-switch
          v-model="data.set_system_proxy"
          color="primary"
          :label="$t('singbox.setSystemProxy')"
          hide-details>
        </v-switch>
      </v-col>
      <v-col cols="12" v-if="data.set_system_proxy">
        <v-alert type="warning" variant="tonal" density="compact">
          {{ $t('singbox.setSystemProxyWarning') }}
        </v-alert>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data'],
}
</script>
