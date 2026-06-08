<template>
  <v-card subtitle="Trojan">
    <v-row>
      <v-col cols="12" sm="6" md="4" v-if="direction != 'in'">
        <v-text-field v-model="data.password" :label="$t('types.pw')" hide-details></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <Network :data="data" />
      </v-col>
    </v-row>
    <template v-if="direction == 'in'">
      <v-row>
        <v-col cols="12">
          <v-alert type="info" variant="tonal" density="compact">
            {{ $t('singbox.fallbackInfo') }}
          </v-alert>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            v-model="fallbackServer"
            :label="$t('singbox.fallbackServer')"
            placeholder="127.0.0.1"
            hide-details>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            v-model.number="fallbackPort"
            :label="$t('singbox.fallbackPort')"
            type="number"
            min="0"
            hide-details>
          </v-text-field>
        </v-col>
      </v-row>
      <v-card border density="compact" color="background" style="margin-top: 8px;">
        <v-card-subtitle>
          {{ $t('singbox.fallbackForAlpn') }}
          <v-chip color="primary" density="compact" variant="elevated" @click="addFallbackAlpn">
            <v-icon icon="mdi-plus" />
          </v-chip>
        </v-card-subtitle>
        <v-row v-for="(value, alpn) in data.fallback_for_alpn" :key="alpn">
          <v-col cols="auto" align-self="center">
            <v-icon icon="mdi-delete" color="error" @click="deleteFallbackAlpn(String(alpn))" />
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field
              :model-value="alpn"
              label="ALPN"
              hide-details
              @update:model-value="renameFallbackAlpn(String(alpn), String($event))">
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="4">
            <v-text-field
              v-model="value.server"
              label="Server"
              placeholder="127.0.0.1"
              hide-details>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="3">
            <v-text-field
              v-model.number="value.server_port"
              label="Port"
              type="number"
              min="0"
              hide-details>
            </v-text-field>
          </v-col>
        </v-row>
      </v-card>
    </template>
  </v-card>
</template>

<script lang="ts">
import Network from '@/components/Network.vue'

export default {
  props: ['direction', 'data'],
  data() {
    return {
      fallbackAlpnCounter: 0
    }
  },
  computed: {
    fallbackServer: {
      get(): string {
        return this.$props.data.fallback?.server ?? ''
      },
      set(v:string) {
        this.updateFallback(v, this.fallbackPort)
      }
    },
    fallbackPort: {
      get(): number | undefined {
        return this.$props.data.fallback?.server_port
      },
      set(v:number) {
        this.updateFallback(this.fallbackServer, v)
      }
    }
  },
  methods: {
    updateFallback(server:string, port:number | undefined) {
      if (server.length == 0 && (!port || port == 0)) {
        delete this.$props.data.fallback
        return
      }
      this.$props.data.fallback = {
        server,
        server_port: port && port > 0 ? port : 0
      }
    },
    addFallbackAlpn() {
      if (!this.$props.data.fallback_for_alpn) {
        this.$props.data.fallback_for_alpn = {}
      }
      let key = this.fallbackAlpnCounter == 0 ? 'h2' : 'alpn-' + this.fallbackAlpnCounter
      while (this.$props.data.fallback_for_alpn[key]) {
        this.fallbackAlpnCounter++
        key = 'alpn-' + this.fallbackAlpnCounter
      }
      this.fallbackAlpnCounter++
      this.$props.data.fallback_for_alpn[key] = { server: '', server_port: 0 }
    },
    deleteFallbackAlpn(alpn:string) {
      if (!this.$props.data.fallback_for_alpn) return
      delete this.$props.data.fallback_for_alpn[alpn]
      if (Object.keys(this.$props.data.fallback_for_alpn).length == 0) {
        delete this.$props.data.fallback_for_alpn
      }
    },
    renameFallbackAlpn(oldKey:string, newKey:string) {
      if (!this.$props.data.fallback_for_alpn || newKey.length == 0 || oldKey == newKey) return
      this.$props.data.fallback_for_alpn[newKey] = this.$props.data.fallback_for_alpn[oldKey]
      delete this.$props.data.fallback_for_alpn[oldKey]
    }
  },
  components: { Network }
}
</script>
