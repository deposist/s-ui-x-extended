<template>
  <v-row>
    <v-col cols="6" sm="4" md="3">
      <v-text-field
        :label="$t('transport.mkcpMtu')"
        type="number" min="576" max="1460"
        hide-details
        :model-value="transport.mtu" @update:model-value="(v:string) => setNum('mtu', v)">
      </v-text-field>
    </v-col>
    <v-col cols="6" sm="4" md="3">
      <v-text-field
        :label="$t('transport.mkcpTti')"
        type="number" min="10" max="100" suffix="ms"
        hide-details
        :model-value="transport.tti" @update:model-value="(v:string) => setNum('tti', v)">
      </v-text-field>
    </v-col>
    <v-col cols="6" sm="4" md="3">
      <v-text-field
        :label="$t('transport.mkcpUplink')"
        type="number" min="0" suffix="MB/s"
        hide-details
        :model-value="transport.uplink_capacity" @update:model-value="(v:string) => setNum('uplink_capacity', v)">
      </v-text-field>
    </v-col>
    <v-col cols="6" sm="4" md="3">
      <v-text-field
        :label="$t('transport.mkcpDownlink')"
        type="number" min="0" suffix="MB/s"
        hide-details
        :model-value="transport.downlink_capacity" @update:model-value="(v:string) => setNum('downlink_capacity', v)">
      </v-text-field>
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="6" sm="4" md="3">
      <v-text-field
        :label="$t('transport.mkcpReadBuffer')"
        type="number" min="0" suffix="MB"
        hide-details
        :model-value="transport.read_buffer_size" @update:model-value="(v:string) => setNum('read_buffer_size', v)">
      </v-text-field>
    </v-col>
    <v-col cols="6" sm="4" md="3">
      <v-text-field
        :label="$t('transport.mkcpWriteBuffer')"
        type="number" min="0" suffix="MB"
        hide-details
        :model-value="transport.write_buffer_size" @update:model-value="(v:string) => setNum('write_buffer_size', v)">
      </v-text-field>
    </v-col>
    <v-col cols="12" sm="4" md="3">
      <v-select
        :label="$t('transport.mkcpHeaderType')"
        :items="['none','srtp','utp','wechat-video','dtls','wireguard']"
        hide-details
        v-model="transport.header_type">
      </v-select>
    </v-col>
    <v-col cols="12" sm="6" md="3" align-self="center">
      <v-switch
        color="primary"
        :label="$t('transport.mkcpCongestion')"
        hide-details
        v-model="transport.congestion">
      </v-switch>
    </v-col>
  </v-row>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-text-field
        :label="$t('transport.mkcpSeed')"
        hide-details
        v-model="transport.seed">
      </v-text-field>
    </v-col>
  </v-row>
</template>

<script lang="ts">
export default {
  props: ['transport'],
  methods: {
    // All mKCP numeric fields are uint32 with omitempty; only persist a key
    // while it holds a value so an empty input never reaches the core as "".
    setNum(key: string, v: string) {
      if (v != null && v !== '') {
        this.$props.transport[key] = Number(v)
      } else {
        delete this.$props.transport[key]
      }
    },
  },
}
</script>
