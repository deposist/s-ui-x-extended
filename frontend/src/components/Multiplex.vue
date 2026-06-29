<template>
  <v-card :subtitle="$t('objects.multiplex')">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <div class="d-flex align-center ga-1">
          <v-switch color="primary" :label="$t('mux.enable')" v-model="muxEnable" hide-details></v-switch>
          <SettingInfo v-if="hint('enable')" :text="hint('enable')" />
        </div>
      </v-col>
      <template v-if="muxEnable">
        <template v-if="direction=='out'">
          <v-col cols="12" sm="6" md="4">
            <v-select
              hide-details
              :items="[ 'smux', 'yamux', 'h2mux']"
              :label="$t('protocol')"
              clearable
              @click:clear="delete mux?.protocol"
              v-model="mux.protocol">
              <template #append-inner>
                <SettingInfo v-if="hint('protocol')" :text="hint('protocol')" />
              </template>
            </v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
            :label="$t('mux.maxConn')"
            hide-details
            type="number"
            min=0
            v-model.number="max_connections">
              <template #append-inner>
                <SettingInfo v-if="hint('max_connections')" :text="hint('max_connections')" />
              </template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
            :label="$t('mux.minStr')"
            hide-details
            type="number"
            min=0
            v-model.number="min_streams">
              <template #append-inner>
                <SettingInfo v-if="hint('min_streams')" :text="hint('min_streams')" />
              </template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
            :label="$t('mux.maxStr')"
            hide-details
            type="number"
            :min="min_streams"
            v-model.number="max_streams">
              <template #append-inner>
                <SettingInfo v-if="hint('max_streams')" :text="hint('max_streams')" />
              </template>
            </v-text-field>
          </v-col>
        </template>
        <v-col cols="12" sm="6" md="4">
          <div class="d-flex align-center ga-1">
            <v-switch color="primary" :label="$t('mux.padding')" v-model="padding" hide-details></v-switch>
            <SettingInfo v-if="hint('padding')" :text="hint('padding')" />
          </div>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <div class="d-flex align-center ga-1">
            <v-switch color="primary" :label="$t('mux.enableBrutal')" v-model="burtalEnable" hide-details></v-switch>
            <SettingInfo v-if="hint('brutal')" :text="hint('brutal')" />
          </div>
        </v-col>
      </template>
    </v-row>
    <v-row v-if="mux?.brutal?.enabled">
      <v-col cols="12" sm="6" md="4">
        <v-text-field
        :label="$t('stats.upload')"
        hide-details
        type="number"
        :suffix="$t('stats.Mbps')"
        v-model.number="up_mbps">
          <template #append-inner>
            <SettingInfo v-if="hint('brutal_up_mbps')" :text="hint('brutal_up_mbps')" />
          </template>
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
          <template #append-inner>
            <SettingInfo v-if="hint('brutal_down_mbps')" :text="hint('brutal_down_mbps')" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import { oMultiplex } from '@/types/multiplex'
import SettingInfo from '@/components/SettingInfo.vue'
export default {
  props: {
    data: { type: Object, required: true },
    direction: { type: String, default: 'out' },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {}
  },
  methods: {
    hint(field: string): string {
      const prefix = this.$props.direction == 'in' ? 'inbound_multiplex' : 'out_multiplex'
      const hintKey = (this.$props.fieldHints as Record<string, string>)[`${prefix}_${field}`]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
  },
  computed: {
    mux(): oMultiplex {
      return <oMultiplex> this.$props.data.multiplex ?? null
    },
    muxEnable: {
      get(): boolean { return this.mux ? this.mux.enabled : false },
      set(newValue:boolean) { this.$props.data.multiplex = newValue ? { enabled: newValue } : undefined }
    },
    max_connections: {
      get(): number { return this.mux?.max_connections ? this.mux.max_connections : 0 },
      set(newValue:number) { this.mux.max_connections = newValue > 0 ? newValue : undefined }
    },
    min_streams: {
      get(): number { return this.mux?.min_streams ? this.mux.min_streams : 0 },
      set(newValue:number) { this.mux.min_streams = newValue > 0 ? newValue : undefined }
    },
    max_streams: {
      get(): number { return this.mux?.max_streams ? this.mux.max_streams : 0 },
      set(newValue:number) { this.mux.max_streams = newValue > 0 ? newValue : undefined }
    },
    padding: {
      get(): boolean { return this.mux?.padding ? this.mux.padding : false },
      set(newValue:boolean) { this.mux.padding = newValue ? true : undefined }
    },
    burtalEnable: {
      get(): boolean { return this.mux?.brutal ? this.mux.brutal.enabled : false },
      set(newValue:boolean) { this.mux.brutal = newValue ? { enabled: newValue, up_mbps: 100, down_mbps: 100 } : undefined }
    },
    down_mbps: {
      get() { return this.mux?.brutal && this.mux.brutal.down_mbps ? this.mux.brutal.down_mbps : 0 },
      set(newValue:any) {
        if (this.mux.brutal){
          this.mux.brutal.down_mbps = newValue.length != 0 ? newValue : 0
        }
      }
    },
    up_mbps: {
      get() { return this.mux?.brutal && this.mux.brutal.up_mbps ? this.mux.brutal.up_mbps : 0 },
      set(newValue:any) {
        if (this.mux.brutal){
          this.mux.brutal.up_mbps = newValue.length != 0 ? newValue : 0
        }
      }
    },
  },
  components: { SettingInfo }
}
</script>