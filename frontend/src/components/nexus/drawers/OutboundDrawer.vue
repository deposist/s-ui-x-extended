<template>
  <entity-drawer
    :dirty="dirty"
    :loading="loading"
    :model-value="visible"
    :saving="loading"
    :title="$t('actions.' + title) + ' ' + $t('objects.outbound')"
    :width="720"
    @close="closeModal"
    @save="saveChanges"
  >
    <form-section icon="lucide:sliders-horizontal" :title="$t('form.sections.configuration')">
      <v-row>
        <v-col cols="12" sm="6">
          <v-select
            hide-details
            :items="Object.keys(outTypes).map((key,index) => {
              const value = Object.values(outTypes)[index] as string
              const unavailable = unavailableOutboundTypes.includes(value)
              return { title: unavailable ? key + ' \u2014 not in this build' : key, value, props: { disabled: unavailable } }
            })"
            :label="$t('type')"
            v-model="outbound.type"
            @update:modelValue="changeType">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field v-model="outbound.tag" :label="$t('objects.tag')" hide-details></v-text-field>
        </v-col>
      </v-row>
      <RecommendedValues
        :model="outbound"
        :specs="recommendationSpecs"
        :context="recommendationContext"
        class="mb-3"
        @apply="applyRecommended"
      />
      <v-row v-if="!NoServer.includes(outbound.type)">
        <v-col cols="12" sm="6">
          <v-text-field :label="$t('out.addr')" hide-details v-model="outbound.server"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field :label="$t('out.port')" type="number" min="0" hide-details v-model.number="outbound.server_port"></v-text-field>
        </v-col>
      </v-row>
      <Socks v-if="outbound.type == outTypes.SOCKS" :data="outbound" />
      <Http v-if="outbound.type == outTypes.HTTP" :data="outbound" />
      <Shadowsocks v-if="outbound.type == outTypes.Shadowsocks" direction="out" :data="outbound" />
      <Vmess v-if="outbound.type == outTypes.VMess" :data="outbound" />
      <Trojan v-if="outbound.type == outTypes.Trojan" direction="out" :data="outbound" />
      <Hysteria v-if="outbound.type == outTypes.Hysteria" direction="out" :data="outbound" />
      <Naive v-if="outbound.type == outTypes.Naive" direction="out" :data="outbound" />
      <ShadowTls v-if="outbound.type == outTypes.ShadowTLS" :data="outbound" />
      <Vless v-if="outbound.type == outTypes.VLESS" :data="outbound" />
      <Tuic v-if="outbound.type == outTypes.TUIC" direction="out" :data="outbound" />
      <Hysteria2 v-if="outbound.type == outTypes.Hysteria2" direction="out" :data="outbound" />
      <AnyTls v-if="outbound.type == outTypes.AnyTls" :data="outbound" direction="out" />
      <Tor v-if="outbound.type == outTypes.Tor" :data="outbound" />
      <Ssh v-if="outbound.type == outTypes.SSH" :data="outbound" />
      <Selector v-if="outbound.type == outTypes.Selector" :data="outbound" :tags="tags" />
      <UrlTest v-if="outbound.type == outTypes.URLTest" :data="outbound" :tags="tags" />
      <Failover v-if="outbound.type == outTypes.Failover" :data="outbound" :tags="tags" />
      <Block v-if="outbound.type == outTypes.Block" :data="outbound" />
      <CoreFailover v-if="outbound.type == outTypes.CoreFailover" :data="outbound" :tags="tags" />
      <Mieru v-if="outbound.type == outTypes.Mieru" direction="out" :data="outbound" />
      <Sudoku v-if="outbound.type == outTypes.Sudoku" direction="out" :data="outbound" />
      <TrustTunnel v-if="outbound.type == outTypes.TrustTunnel" direction="out" :data="outbound" />
      <Masque v-if="outbound.type == outTypes.MASQUE" :data="outbound" />
      <OpenVPN v-if="outbound.type == outTypes.OpenVPN" :data="outbound" />
      <Bond v-if="outbound.type == outTypes.Bond" :data="outbound" :tags="tags" />
      <Parser v-if="outbound.type == outTypes.Parser" :data="outbound" :tags="tags" />
      <BandwidthLimiter v-if="outbound.type == outTypes.BandwidthLimiter" :data="outbound" :tags="tags" />
      <ConnectionLimiter v-if="outbound.type == outTypes.ConnectionLimiter" :data="outbound" :tags="tags" />
      <TrafficLimiter v-if="outbound.type == outTypes.TrafficLimiter" :data="outbound" :tags="tags" />
      <RateLimiter v-if="outbound.type == outTypes.RateLimiter" :data="outbound" :tags="tags" />
      <Fallback v-if="outbound.type == outTypes.Fallback" :data="outbound" :tags="tags" />
      <Transport v-if="Object.hasOwn(outbound,'transport')" :data="outbound" />
      <OutTLS v-if="Object.hasOwn(outbound,'tls')" :outbound="outbound" />
      <Multiplex v-if="Object.hasOwn(outbound,'multiplex')" direction="out" :data="outbound" />
      <Dial v-if="!NoDial.includes(outbound.type)" :dial="outbound" />
    </form-section>

    <form-section icon="lucide:globe" :title="$t('client.external')" :default-open="false">
      <v-row>
        <v-col cols="12">
          <v-text-field v-model="link" :label="$t('client.external')" hide-details />
        </v-col>
        <v-col cols="12" align="center">
          <v-btn variant="tonal" :loading="loading" @click="linkConvert">{{ $t('submit') }}</v-btn>
        </v-col>
      </v-row>
    </form-section>
  </entity-drawer>
</template>

<script lang="ts">
import { OutTypes, createOutbound } from '@/types/outbounds'
import RandomUtil from '@/plugins/randomUtil'
import Dial from '@/components/Dial.vue'
import Multiplex from '@/components/Multiplex.vue'
import Transport from '@/components/Transport.vue'
import OutTLS from '@/components/tls/OutTLS.vue'
import Direct from '@/components/protocols/Direct.vue'
import Socks from '@/components/protocols/Socks.vue'
import Http from '@/components/protocols/Http.vue'
import Shadowsocks from '@/components/protocols/Shadowsocks.vue'
import Vmess from '@/components/protocols/Vmess.vue'
import Trojan from '@/components/protocols/Trojan.vue'
import Wireguard from '@/components/protocols/Wireguard.vue'
import Hysteria from '@/components/protocols/Hysteria.vue'
import Naive from '@/components/protocols/Naive.vue'
import ShadowTls from '@/components/protocols/OutShadowTls.vue'
import Vless from '@/components/protocols/Vless.vue'
import Tuic from '@/components/protocols/Tuic.vue'
import Hysteria2 from '@/components/protocols/Hysteria2.vue'
import Tor from '@/components/protocols/Tor.vue'
import Ssh from '@/components/protocols/Ssh.vue'
import Selector from '@/components/protocols/Selector.vue'
import UrlTest from '@/components/protocols/UrlTest.vue'
import Failover from '@/components/protocols/Failover.vue'
import Block from '@/components/protocols/Block.vue'
import CoreFailover from '@/components/protocols/CoreFailover.vue'
import Mieru from '@/components/protocols/Mieru.vue'
import Sudoku from '@/components/protocols/Sudoku.vue'
import TrustTunnel from '@/components/protocols/TrustTunnel.vue'
import Masque from '@/components/protocols/Masque.vue'
import OpenVPN from '@/components/protocols/OpenVPN.vue'
import Bond from '@/components/protocols/Bond.vue'
import Parser from '@/components/protocols/Parser.vue'
import BandwidthLimiter from '@/components/protocols/BandwidthLimiter.vue'
import ConnectionLimiter from '@/components/protocols/ConnectionLimiter.vue'
import TrafficLimiter from '@/components/protocols/TrafficLimiter.vue'
import RateLimiter from '@/components/protocols/RateLimiter.vue'
import Fallback from '@/components/protocols/Fallback.vue'
import HttpUtils from '@/plugins/httputil'
import AnyTls from '@/components/protocols/AnyTls.vue'
import Data from '@/store/modules/data'
import EntityDrawer from './EntityDrawer.vue'
import FormSection from './FormSection.vue'
import RecommendedValues from '@/components/recommendations/RecommendedValues.vue'
import { applyRecommendation } from '@/utils/recommendations'
import { outboundRecommendationSpecs } from '@/utils/defaultRecommendations'
export default {
  inheritAttrs: false,
  props: ['visible', 'data', 'id', 'tags'],
  emits: ['close'],
  data() {
    return {
      outbound: createOutbound("direct",{ "tag": "" }),
      title: "add",
      link: "",
      loading: false,
      snapshot: "",
      outTypes: OutTypes,
      unavailableOutboundTypes: <string[]>[],
      NoDial: [OutTypes.Selector, OutTypes.URLTest, OutTypes.Failover, OutTypes.Block, OutTypes.CoreFailover, OutTypes.Bond, OutTypes.BandwidthLimiter, OutTypes.ConnectionLimiter, OutTypes.TrafficLimiter, OutTypes.RateLimiter, OutTypes.Parser, OutTypes.Fallback],
      NoServer: [OutTypes.Direct, OutTypes.Selector, OutTypes.URLTest, OutTypes.Tor, OutTypes.Failover, OutTypes.Block, OutTypes.CoreFailover, OutTypes.Bond, OutTypes.BandwidthLimiter, OutTypes.ConnectionLimiter, OutTypes.TrafficLimiter, OutTypes.RateLimiter, OutTypes.Parser, OutTypes.Fallback],
      recommendationSpecs: outboundRecommendationSpecs,
    }
  },
  async mounted() {
    try {
      const resp = await HttpUtils.get('api/capabilities')
      if (resp.success && resp.obj?.outbounds) {
        this.unavailableOutboundTypes = resp.obj.outbounds
          .filter((o: any) => o.available === false)
          .map((o: any) => o.type)
      }
    } catch { /* capabilities endpoint optional */ }
  },
  methods: {
    updateData(id: number) {
      if (id > 0) {
        const newData = JSON.parse(this.$props.data)
        this.outbound = newData
        this.title = "edit"
      }
      else {
        this.outbound = createOutbound("direct",{ tag: "direct-" + RandomUtil.randomSeq(3) })
        this.title = "add"
      }
      this.snapshot = JSON.stringify(this.outbound)
    },
    applyRecommended(spec: any) {
      applyRecommendation(this.outbound, spec, this.recommendationContext, { force: true })
    },
    changeType() {
      // Tag change only in add outbound
      const tag = this.$props.id > 0 ? this.outbound.tag : this.outbound.type + "-" + RandomUtil.randomSeq(3)
      // Use previous data
      const prevConfig = { id: this.outbound.id, tag: tag, listen: this.outbound.listen, listen_port: this.outbound.listen_port }
      this.outbound = createOutbound(this.outbound.type, prevConfig)
    },
    closeModal() {
      this.updateData(0) // reset
      this.$emit('close')
    },
    async saveChanges() {
      // Guard against double-submit (button is also :disabled while loading).
      if (!this.$props.visible || this.loading) return
      // check duplicate tag
      const isDuplicatedTag = Data().checkTag("outbound",this.$props.id, this.outbound.tag)
      if (isDuplicatedTag) return

      // save data
      this.loading = true
      try {
        const success = await Data().save("outbounds", this.$props.id == 0 ? "new" : "edit", this.outbound)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
    async linkConvert() {
      if (this.link.length>0){
        this.loading = true
        const msg = await HttpUtils.post('api/linkConvert', { link: this.link })
        this.loading = false
        if (msg.success) {
          this.outbound = msg.obj
          if (this.$props.id > 0) this.outbound.id = this.$props.id
          this.link = ""
        }
      }
    }
  },
  computed: {
    dirty(): boolean {
      return this.snapshot !== "" && JSON.stringify(this.outbound) !== this.snapshot
    },
    recommendationContext() {
      return { model: this.outbound, mode: this.$props.id > 0 ? 'edit' : 'create', type: this.outbound.type, unavailableTypes: this.unavailableOutboundTypes }
    },
  },
  watch: {
    visible(newValue) {
      if (newValue) {
        this.updateData(this.$props.id)
      }
    },
  },
  components: { EntityDrawer, FormSection, RecommendedValues, Dial, Multiplex, Transport, OutTLS,
    Direct, Socks, Http, Shadowsocks, Vmess, Trojan,
    Wireguard, Hysteria, Naive, ShadowTls, Vless, Tuic,
    Hysteria2, AnyTls, Tor, Ssh, Selector, UrlTest, Failover, Block, CoreFailover,
    Mieru, Sudoku, TrustTunnel, Masque, OpenVPN, Bond, Parser,
    BandwidthLimiter, ConnectionLimiter, TrafficLimiter, RateLimiter, Fallback }
}
</script>
