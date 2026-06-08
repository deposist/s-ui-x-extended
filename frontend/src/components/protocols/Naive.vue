<template>
  <v-card>
    <v-card-subtitle v-if="direction != 'out_json'">Naive</v-card-subtitle>
    <!-- Inbound -->
    <template v-if="direction === 'in'">
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <Network :data="data" />
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-select
            hide-details
            :label="$t('types.naive.quicCongestion')"
            :items="naiveCongestion"
            v-model="data.quic_congestion_control"
            @click:clear="delete data.quic_congestion_control"
            clearable>
          </v-select>
        </v-col>
      </v-row>
    </template>
    <!-- Outbound -->
    <template v-if="['out', 'out_json'].includes(direction)">
      <v-row v-if="direction === 'out'">
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.un')"
            hide-details
            v-model="data.username">
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.pw')"
            hide-details
            type="password"
            v-model="data.password">
          </v-text-field>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-text-field
            :label="$t('types.naive.insecureConcurrency')"
            type="number"
            min="0"
            hide-details
            v-model.number="insecure_concurrency">
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <UoT :data="data" />
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-combobox
            :label="$t('types.naive.streamReceiveWindow')"
            hide-details
            :items="sizePresets"
            v-model="data.stream_receive_window">
          </v-combobox>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-combobox
            :label="$t('types.naive.quicSessionReceiveWindow')"
            hide-details
            :items="sizePresets"
            v-model="data.quic_session_receive_window">
          </v-combobox>
        </v-col>
      </v-row>
      <v-row v-if="direction === 'out'">
        <v-col cols="12" sm="6" md="4">
          <v-switch v-model="data.quic" color="primary" :label="$t('types.naive.quic')" hide-details></v-switch>
        </v-col>
        <v-col cols="12" sm="6" md="4" v-if="data.quic">
          <v-select
            hide-details
            :label="$t('types.naive.quicCongestion')"
            :items="naiveCongestion"
            @click:clear="delete data.quic_congestion_control"
            clearable
            v-model="data.quic_congestion_control">
          </v-select>
        </v-col>
      </v-row>
      <Headers :data="extra_headers" />
    </template>
  </v-card>
</template>

<script lang="ts">
import Network from '@/components/Network.vue'
import Headers from '@/components/Headers.vue'
import UoT from '@/components/UoT.vue'
import { naiveCongestion, sizePresets, RECOMMENDED } from '@/types/recommended'

export default {
  props: ['data', 'direction'],
  data() {
    return {
      naiveCongestion,
      sizePresets,
    }
  },
  mounted() {
    this.data.quic_congestion_control ??= RECOMMENDED.naiveCongestion
  },
  computed: {
    insecure_concurrency: {
      get(): number { return this.$props.data?.insecure_concurrency ?? 0 },
      set(v: number) {
        this.$props.data.insecure_concurrency = v > 0 ? v : undefined
      }
    },
    extra_headers(): any {
      const d = this.$props.data
      return new Proxy({}, {
        get(_, prop) {
          if (prop === 'headers') return d?.extra_headers ?? {}
          return undefined
        },
        set(_, prop, value) {
          if (prop === 'headers') {
            d.extra_headers = value
            return true
          }
          return false
        }
      })
    },
  },
  components: { Network, Headers, UoT }
}
</script>
