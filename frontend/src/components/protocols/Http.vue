<template>
  <v-card subtitle="HTTP">
    <template v-if="direction === 'out'">
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
          :label="$t('types.un')"
          hide-details
          v-model="username">
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
          :label="$t('types.pw')"
          hide-details
          v-model="password">
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
          :label="$t('transport.path')"
          hide-details
          v-model="data.path">
          </v-text-field>
        </v-col>
      </v-row>
      <Headers :data="data" />
      <InboundAdvanced :data="data" />
    </template>
    <template v-else>
      <v-row>
        <v-col cols="12">
          <v-alert type="info" variant="tonal" density="compact">
            Other HTTP inbound settings are handled by the core HTTP server.
          </v-alert>
        </v-col>
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
    </template>
  </v-card>
</template>

<script lang="ts">
import Headers from '@/components/Headers.vue'
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'

export default {
  props: {
    data: { type: Object, required: true },
    direction: { type: String, default: 'out' },
  },
  data() {
    return {}
  },
  computed: {
    username: {
      get(): string { return this.data.username?.length > 0 ? this.data.username : '' },
      set(v:string) { this.data.username = v.length > 0 ? v : undefined },
    },
    password: {
      get(): string { return this.data.password?.length > 0 ? this.data.password : '' },
      set(v:string) { this.data.password = v.length > 0 ? v : undefined },
    },
  },
  components: {Headers, InboundAdvanced}
}
</script>