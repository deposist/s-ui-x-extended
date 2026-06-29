<template>
  <v-card subtitle="TProxy">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <Network :data="inbound" :field-hints="fieldHints" />
      </v-col>
    </v-row>
    <InboundAdvanced :data="inbound" :field-hints="fieldHints" />
  </v-card>
</template>

<script lang="ts">
import Network from '@/components/Network.vue'
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'

export default {
  props: ['inbound', 'fieldHints'],
  data() {
    return {
    }
  },
  components: {Network, InboundAdvanced}
}
</script>