<template>
  <v-card subtitle="Hysteria">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
        :label="$t('stats.upload')"
        hide-details
        type="number"
        :suffix="$t('stats.Mbps')"
        v-model.number="up_mbps">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
        :label="$t('stats.download')"
        hide-details
        type="number"
        :suffix="$t('stats.Mbps')"
        min="0"
        v-model.number="down_mbps">
        </v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
       <v-text-field
       :label="$t('types.hy.obfs')"
        hide-details
        v-model="data.obfs">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction=='out'">
        <v-text-field
        :label="$t('types.hy.auth')"
        hide-details
        v-model="data.auth_str">
        </v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4" v-if="direction=='out'">
        <Network :data="data" />
      </v-col>
      <v-col cols="12" sm="8" v-if="direction=='out' && optionMPort">
        <v-text-field
          :label="$t('rule.portRange') + ' ' + $t('commaSeparated')"
          v-model="server_ports"
          hide-details>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction=='out' && optionMPort">
        <v-text-field
          :label="$t('ruleset.interval')"
          type="number"
          min="0"
          :suffix="$t('date.s')"
          v-model.number="hop_interval"
          hide-details>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="data.disable_mtu_discovery" color="primary" label="Disable MTU discovery" hide-details></v-switch>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4" v-if="data.recv_window_conn != undefined">
        <v-text-field
        label="Recv window conn"
        hide-details
        type="number"
        min="0"
        v-model.number="data.recv_window_conn">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="data.recv_window != undefined">
        <v-text-field
        label="Recv window"
        hide-details
        type="number"
        min="0"
        v-model.number="data.recv_window">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="data.recv_window_client != undefined">
        <v-text-field
        label="Recv window client"
        hide-details
        type="number"
        min="0"
        v-model.number="data.recv_window_client">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="data.max_conn_client != undefined">
        <v-text-field
        label="Max conn client"
        hide-details
        type="number"
        min="0"
        v-model.number="data.max_conn_client">
        </v-text-field>
      </v-col>
    </v-row>
    <v-card-actions>
      <v-spacer></v-spacer>
      <v-menu v-model="menu" :close-on-content-click="false" location="start">
        <template v-slot:activator="{ props }">
          <v-btn v-bind="props" hide-details variant="tonal">{{ $t('types.hy.hyOptions') }}</v-btn>
        </template>
        <v-card>
          <v-list>
            <v-list-item>
              <v-switch v-model="optionRsvConn" color="primary" label="Recv window conn" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="direction=='out'">
              <v-switch v-model="optionRsvWin" color="primary" label="Recv window" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="direction=='in'">
              <v-switch v-model="optionRsvClnt" color="primary" label="Recv window client" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="direction=='in'">
              <v-switch v-model="optionMaxConn" color="primary" label="Max conn client" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="direction=='out'">
              <v-switch v-model="optionMPort" color="primary" :label="$t('rule.portRange')" hide-details></v-switch>
            </v-list-item>
          </v-list>
          <v-divider v-if="direction=='out'"></v-divider>
          <v-card-text v-if="direction=='out'" style="min-width: 360px;">
            <v-text-field
              v-model="upBandwidth"
              :label="$t('singbox.uploadBandwidth')"
              placeholder="100 Mbps"
              hide-details
              style="margin-bottom: 8px;">
            </v-text-field>
            <v-text-field
              v-model="downBandwidth"
              :label="$t('singbox.downloadBandwidth')"
              placeholder="100 Mbps"
              hide-details>
            </v-text-field>
          </v-card-text>
        </v-card>
      </v-menu>
    </v-card-actions>
  </v-card>
</template>

<script lang="ts">
import Network from '@/components/Network.vue'

export default {
  props: ['direction','data'],
  data() {
    return {
      menu: false,
    }
  },
  computed: {
    optionRsvConn: {
      get(): boolean { return this.$props.data.recv_window_conn != undefined },
      set(v:boolean) { this.$props.data.recv_window_conn = v ? 15728640 : undefined }
    },
    optionRsvWin: {
      get(): boolean { return this.$props.data.recv_window != undefined },
      set(v:boolean) { this.$props.data.recv_window = v ? 67108864 : undefined }
    },
    optionRsvClnt: {
      get(): boolean { return this.$props.data.recv_window_client != undefined },
      set(v:boolean) { this.$props.data.recv_window_client = v ? 67108864 : undefined }
    },
    optionMaxConn: {
      get(): boolean { return this.$props.data.max_conn_client != undefined },
      set(v:boolean) { this.$props.data.max_conn_client = v ? 1024 : undefined }
    },
    optionMPort: {
      get(): boolean { return this.$props.data.server_ports != undefined },
      set(v:boolean) { this.$props.data.server_ports = v ? [] : undefined }
    },
    server_ports: {
      get(): string { return this.$props.data.server_ports?.join(',') ?? '' },
      set(v:string) { this.$props.data.server_ports = v.length > 0 ? v.split(',').map((item:string) => item.trim()).filter((item:string) => item.length > 0) : undefined }
    },
    hop_interval: {
      get(): number { return this.$props.data.hop_interval ? parseInt(this.$props.data.hop_interval.replace('s','')) : 0 },
      set(v:number) { this.$props.data.hop_interval = v>0 ? v + 's' : undefined }
    },
    upBandwidth: {
      get(): string { return this.$props.data.up ?? '' },
      set(v:string) {
        if (v.length > 0) {
          this.$props.data.up = v
        } else {
          delete this.$props.data.up
        }
      }
    },
    downBandwidth: {
      get(): string { return this.$props.data.down ?? '' },
      set(v:string) {
        if (v.length > 0) {
          this.$props.data.down = v
        } else {
          delete this.$props.data.down
        }
      }
    },
    down_mbps: {
      get() { return this.$props.data.down_mbps ? this.$props.data.down_mbps : 0 },
      set(newValue:any) {
        if (newValue.length != 0 ){
          this.$props.data.down_mbps = newValue
        } else {
          this.$props.data.down_mbps = 0
        }
      }
    },
    up_mbps: {
      get() { return this.$props.data.up_mbps ? this.$props.data.up_mbps : 0 },
      set(newValue:number) { this.$props.data.up_mbps = newValue > 0 ? newValue : 0 }
    },
  },
  components: { Network }
}
</script>
