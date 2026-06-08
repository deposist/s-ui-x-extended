<template>
  <v-card subtitle="Profiler">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.listen" :label="$t('objects.listen')" placeholder="127.0.0.1:8964" hide-details />
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.read_timeout" :label="$t('types.profiler.readTimeout')" placeholder="5s" hide-details />
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.write_timeout" :label="$t('types.profiler.writeTimeout')" placeholder="5s" hide-details />
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data'],
}
</script>
