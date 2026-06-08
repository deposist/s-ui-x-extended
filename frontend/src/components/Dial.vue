<template>
  <v-card :subtitle="$t('objects.dial')" style="background-color: inherit;">
    <v-row>
      <v-col cols="12" sm="6" md="4" v-if="optionDetour">
        <v-select
          hide-details
          :label="$t('dial.detourText')"
          :items="outTags"
          v-model="dial.detour">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionBind">
        <v-text-field
        :label="$t('dial.bindIf')"
        hide-details
        v-model="dial.bind_interface"></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4" v-if="optionIPV4">
        <v-text-field
        :label="$t('dial.bindIp4')"
        hide-details
        v-model="dial.inet4_bind_address"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionIPV6">
        <v-text-field
        :label="$t('dial.bindIp6')"
        hide-details
        v-model="dial.inet6_bind_address"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionBindNoPort">
        <v-switch v-model="dial.bind_address_no_port" color="primary" :label="$t('dial.bindNoPort')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionProtect">
        <v-text-field
        :label="$t('singbox.protectPath')"
        hide-details
        v-model="dial.protect_path"></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4" v-if="optionRM">
        <v-text-field
        :label="$t('singbox.linuxRoutingMark')"
        hide-details
        placeholder="0x2024"
        v-model="routingMark"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionRA">
        <v-switch v-model="dial.reuse_addr" color="primary" :label="$t('dial.reuseAddr')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionNetns">
        <v-text-field
        :label="$t('singbox.networkNamespace')"
        hide-details
        v-model="dial.netns"></v-text-field>
      </v-col>
    </v-row>
    <v-row v-if="optionTCP">
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="dial.tcp_fast_open" color="primary" label="TCP Fast Open" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="dial.tcp_multi_path" color="primary" label="TCP Multi Path" hide-details></v-switch>
      </v-col>
    </v-row>
    <v-row v-if="optionTcpKeepAlive">
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="dial.disable_tcp_keep_alive" color="primary" :label="$t('dial.disableTcpKeepAlive')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="dial.tcp_keep_alive" :label="$t('dial.tcpKeepAlive')" hide-details></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="dial.tcp_keep_alive_interval" :label="$t('dial.tcpKeepAliveInterval')" hide-details></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4" v-if="optionUDP">
        <v-switch v-model="dial.udp_fragment" color="primary" label="UDP Fragment" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionCT">
        <v-text-field
        :label="$t('dial.connTimeout')"
        hide-details
        type="number"
        min="1"
        :suffix="$t('date.s')"
        v-model.number="connectTimeout"></v-text-field>
      </v-col>
    </v-row>
    <DomainResolver v-if="optionDR" :data="dial" field="domain_resolver" />
    <v-row v-if="optionNetworkStrategy">
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('singbox.networkStrategy')"
          clearable
          @click:clear="networkStrategy = undefined"
          :items="networkStrategies"
          v-model="networkStrategy">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="dial.network_strategy != undefined">
        <v-select
          hide-details multiple chips closable-chips
          :label="$t('singbox.networkType')"
          :items="networkTypes"
          v-model="dial.network_type">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="dial.network_strategy == 'fallback'">
        <v-select
          hide-details multiple chips closable-chips
          :label="$t('singbox.fallbackNetworkType')"
          :items="networkTypes"
          v-model="dial.fallback_network_type">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="dial.network_strategy && dial.network_strategy != 'default'">
        <v-text-field
          v-model="fallbackDelayMs"
          hide-details
          type="number"
          min="1"
          suffix="ms"
          :label="$t('rule.fallbackDelay')">
        </v-text-field>
      </v-col>
      <v-col cols="12" v-if="networkConflict">
        <v-alert density="compact" type="warning" variant="tonal">
          {{ $t('singbox.networkStrategyConflict') }}
        </v-alert>
      </v-col>
    </v-row>
    <v-card-actions class="pt-0">
      <v-spacer></v-spacer>
      <v-menu v-model="menu" :close-on-content-click="false" location="start">
        <template v-slot:activator="{ props }">
          <v-btn v-bind="props" hide-details variant="tonal">{{ $t('dial.options') }}</v-btn>
        </template>
        <v-card>
          <v-list>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionDetour" color="primary" :label="$t('listen.detour')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionBind" color="primary" :label="$t('dial.bindIf')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionIPV4" color="primary" :label="$t('dial.bindIp4')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionIPV6" color="primary" :label="$t('dial.bindIp6')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionBindNoPort" color="primary" :label="$t('dial.bindNoPort')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionProtect" color="primary" :label="$t('singbox.protectPath')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionRM" color="primary" :label="$t('singbox.routingMark')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionRA" color="primary" :label="$t('dial.reuseAddr')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionNetns" color="primary" :label="$t('singbox.networkNamespace')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionTCP" color="primary" :label="$t('listen.tcpOptions')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionUDP" color="primary" :label="$t('listen.udpOptions')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionCT" color="primary" :label="$t('dial.connTimeout')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionTcpKeepAlive" color="primary" :label="$t('dial.tcpKeepAlive')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionDR" color="primary" :label="$t('dial.domainResolver')" hide-details></v-switch>
            </v-list-item>
            <v-list-item v-if="mode != 'client'">
              <v-switch v-model="optionNetworkStrategy" color="primary" :label="$t('singbox.networkStrategy')" hide-details></v-switch>
            </v-list-item>
          </v-list>
        </v-card>
      </v-menu>
    </v-card-actions>
  </v-card>
</template>

<script lang="ts">
import Data from '@/store/modules/data'
import DomainResolver from '@/components/DomainResolver.vue'

export default {
  props: ['dial', 'mode'],
  data() {
    return {
      menu: false
    }
  },
  computed: {
    outTags() { return [...Data().outbounds?.map((o:any) => o.tag), ...Data().endpoints?.map((e:any) => e.tag)] },
    networkTypes() { return ['wifi', 'cellular', 'ethernet', 'other'] },
    networkStrategies() { return ['fallback', 'hybrid'] },
    networkConflict(): boolean {
      return this.$props.dial.network_strategy != undefined &&
        (this.$props.dial.bind_interface != undefined ||
         this.$props.dial.inet4_bind_address != undefined ||
         this.$props.dial.inet6_bind_address != undefined ||
         this.$props.dial.tcp_fast_open === true)
    },
    networkStrategy: {
      get(): string | undefined {
        return this.$props.dial.network_strategy
      },
      set(v:string | undefined) {
        if (v == undefined || v.length == 0) {
          delete this.$props.dial.network_strategy
          delete this.$props.dial.network_type
          delete this.$props.dial.fallback_network_type
          delete this.$props.dial.fallback_delay
          return
        }
        this.$props.dial.network_strategy = v
        if (v != 'fallback') {
          delete this.$props.dial.fallback_network_type
        }
      }
    },
    connectTimeout: {
      get() { return this.$props.dial.connect_timeout ? parseInt(this.$props.dial.connect_timeout.replace('s','')) : 5 },
      set(newValue:number) { this.$props.dial.connect_timeout = newValue > 0 ? newValue + 's' : '5s' }
    },
    routingMark: {
      get() { return this.$props.dial.routing_mark?.toString() ?? '' },
      set(newValue:string) {
        const trimmed = (newValue ?? '').toString().trim()
        if (trimmed.length == 0 || trimmed == '0' || trimmed == '0x0') delete this.$props.dial.routing_mark
        else if (trimmed.startsWith('0x')) this.$props.dial.routing_mark = trimmed
        else {
          const parsed = Number(trimmed)
          if (Number.isFinite(parsed) && parsed > 0) this.$props.dial.routing_mark = parsed
          else delete this.$props.dial.routing_mark
        }
      }
    },
    fallbackDelayMs: {
      get() { return this.$props.dial.fallback_delay ? parseInt(this.$props.dial.fallback_delay.replace('ms','')) : undefined },
      set(newValue:number | undefined) {
        if (typeof newValue == 'number' && !isNaN(newValue) && newValue > 0 && newValue != 300) this.$props.dial.fallback_delay = `${newValue}ms`
        else delete this.$props.dial.fallback_delay
      }
    },
    optionDetour: {
      get(): boolean { return this.$props.dial.detour != undefined },
      set(v:boolean) { v ? this.$props.dial.detour = this.outTags[0]?? '' : delete this.$props.dial.detour }
    },
    optionBind: {
      get(): boolean { return this.$props.dial.bind_interface != undefined },
      set(v:boolean) { v ? this.$props.dial.bind_interface = '' : delete this.$props.dial.bind_interface }
    },
    optionIPV4: {
      get(): boolean { return this.$props.dial.inet4_bind_address != undefined },
      set(v:boolean) { v ? this.$props.dial.inet4_bind_address = '' : delete this.$props.dial.inet4_bind_address }
    },
    optionIPV6: {
      get(): boolean { return this.$props.dial.inet6_bind_address != undefined },
      set(v:boolean) { v ? this.$props.dial.inet6_bind_address = '' : delete this.$props.dial.inet6_bind_address }
    },
    optionBindNoPort: {
      get(): boolean { return this.$props.dial.bind_address_no_port != undefined },
      set(v:boolean) { v ? this.$props.dial.bind_address_no_port = true : delete this.$props.dial.bind_address_no_port }
    },
    optionProtect: {
      get(): boolean { return this.$props.dial.protect_path != undefined },
      set(v:boolean) { v ? this.$props.dial.protect_path = '' : delete this.$props.dial.protect_path }
    },
    optionTcpKeepAlive: {
      get(): boolean {
        return this.$props.dial.disable_tcp_keep_alive != undefined ||
               this.$props.dial.tcp_keep_alive != undefined ||
               this.$props.dial.tcp_keep_alive_interval != undefined
      },
      set(v:boolean) {
        if (v) {
          this.$props.dial.tcp_keep_alive = '5m'
          this.$props.dial.tcp_keep_alive_interval = '75s'
        } else {
          delete this.$props.dial.disable_tcp_keep_alive
          delete this.$props.dial.tcp_keep_alive
          delete this.$props.dial.tcp_keep_alive_interval
        }
      }
    },
    optionRM: {
      get(): boolean { return this.$props.dial.routing_mark != undefined },
      set(v:boolean) { v ? this.$props.dial.routing_mark = '' : delete this.$props.dial.routing_mark }
    },
    optionRA: {
      get(): boolean { return this.$props.dial.reuse_addr != undefined },
      set(v:boolean) { v ? this.$props.dial.reuse_addr = true : delete this.$props.dial.reuse_addr }
    },
    optionNetns: {
      get(): boolean { return this.$props.dial.netns != undefined },
      set(v:boolean) { v ? this.$props.dial.netns = '' : delete this.$props.dial.netns }
    },
    optionTCP: {
      get(): boolean { 
        return this.$props.dial.tcp_fast_open != undefined && 
               this.$props.dial.tcp_multi_path != undefined
      },
      set(v:boolean) {
        if (v) {
          this.$props.dial.tcp_fast_open = false
          this.$props.dial.tcp_multi_path = false
        } else {
          delete this.$props.dial.tcp_fast_open
          delete this.$props.dial.tcp_multi_path
        }
      }
    },
    optionUDP: {
      get(): boolean { return this.$props.dial.udp_fragment != undefined },
      set(v:boolean) { v ? this.$props.dial.udp_fragment = true : delete this.$props.dial.udp_fragment }
    },
    optionCT: {
      get(): boolean { return this.$props.dial.connect_timeout != undefined },
      set(v:boolean) { v ? this.$props.dial.connect_timeout = '5s' : delete this.$props.dial.connect_timeout }
    },
    optionDR: {
      get(): boolean { return this.$props.dial.domain_resolver != undefined },
      set(v:boolean) { v ? (this.dnsTags.length > 0 ? this.$props.dial.domain_resolver = this.dnsTags[0] : delete this.$props.dial.domain_resolver) : delete this.$props.dial.domain_resolver }
    },
    optionNetworkStrategy: {
      get(): boolean {
        return this.$props.dial.network_strategy != undefined ||
               this.$props.dial.network_type != undefined ||
               this.$props.dial.fallback_network_type != undefined ||
               this.$props.dial.fallback_delay != undefined
      },
      set(v:boolean) {
        if (v) {
          this.$props.dial.network_strategy = 'fallback'
        } else {
          delete this.$props.dial.network_strategy
          delete this.$props.dial.network_type
          delete this.$props.dial.fallback_network_type
          delete this.$props.dial.fallback_delay
        }
      }
    },
    dnsTags() {return Data().config.dns?.servers?.map((d:any) => d.tag) ?? []}
  },
  components: { DomainResolver }
}
</script>
