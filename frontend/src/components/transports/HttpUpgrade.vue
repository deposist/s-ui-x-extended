<template>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-combobox
      :label="$t('transport.hosts')"
      hide-details
      :items="sniFrontHosts"
      v-model="transport.host">
      </v-combobox>
    </v-col>
    <v-col cols="12" sm="6" md="4">
      <v-text-field
      :label="$t('transport.path')"
      hide-details
      v-model="transport.path">
      </v-text-field>
    </v-col>
  </v-row>
  <Headers :data="transport" />
</template>

<script lang="ts">
import Headers from '../Headers.vue'
import { sniFrontHosts, RECOMMENDED } from '@/types/recommended'
export default {
  props: ['transport'],
  data() {
    return {
      sniFrontHosts,
    }
  },
  mounted() {
    this.$props.transport.path ??= RECOMMENDED.wsPath
  },
  components: { Headers }
}
</script>