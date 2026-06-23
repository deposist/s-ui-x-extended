<template>
  <entity-drawer
    :dirty="dirty"
    :loading="loading"
    :model-value="visible"
    :save-disabled="!validate"
    :saving="loading"
    :title="$t('actions.' + title) + ' ' + $t('objects.inbound')"
    :width="720"
    @close="closeModal"
    @save="saveChanges"
  >
    <form-section icon="lucide:zap" :title="$t('form.sections.basic')">
      <v-row>
        <v-col cols="12" sm="6">
          <v-select
            hide-details
            :items="Object.keys(inTypes).map((key,index) => ({title: key, value: Object.values(inTypes)[index]}))"
            :label="$t('type')"
            v-model="inbound.type"
            @update:modelValue="changeType">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field v-model="inbound.tag" :label="$t('objects.tag')" hide-details></v-text-field>
        </v-col>
      </v-row>
      <v-card
        v-if="[inTypes.HTTP, inTypes.Mixed].includes(inbound.type)"
        border
        density="compact"
        color="background"
        style="margin-top: 8px;">
        <v-card-text>
          <v-row>
            <v-col cols="12" sm="6">
              <v-switch
                v-model="setSystemProxy"
                color="primary"
                :label="$t('singbox.setSystemProxy')"
                hide-details>
              </v-switch>
            </v-col>
            <v-col cols="12" v-if="setSystemProxy">
              <v-alert type="warning" variant="tonal" density="compact">
                {{ $t('singbox.setSystemProxyWarning') }}
              </v-alert>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
      <DomainResolver
        v-if="[inTypes.SOCKS, inTypes.HTTP, inTypes.Mixed].includes(inbound.type)"
        :data="inbound"
        field="domain_resolver"
        :label="$t('singbox.inboundDomainResolver')" />
    </form-section>

    <form-section icon="lucide:sliders-horizontal" :title="$t('form.sections.configuration')">
      <v-tabs
        v-if="HasInData.includes(inbound.type)"
        v-model="side"
        density="compact"
        fixed-tabs
        align-tabs="center"
      >
        <v-tab value="s">{{ $t('in.sSide') }}</v-tab>
        <v-tab value="c">{{ $t('in.cSide') }}</v-tab>
      </v-tabs>
      <v-window v-model="side" style="margin-top: 10px;">
        <v-window-item value="s">
          <Listen :data="inbound" :inTags="inTags" v-if="inbound.type != inTypes.Tun" />
          <Direct v-if="inbound.type == inTypes.Direct" :data="inbound" />
          <Shadowsocks v-if="inbound.type == inTypes.Shadowsocks" direction="in" :data="inbound" />
          <Hysteria v-if="inbound.type == inTypes.Hysteria" direction="in" :data="inbound" />
          <Hysteria2 v-if="inbound.type == inTypes.Hysteria2" direction="in" :data="inbound" />
          <Naive v-if="inbound.type == inTypes.Naive" direction="in" :data="inbound" />
          <Trojan v-if="inbound.type == inTypes.Trojan" direction="in" :data="inbound" />
          <ShadowTls v-if="inbound.type == inTypes.ShadowTLS" direction="in" :data="inbound" />
          <Tuic v-if="inbound.type == inTypes.TUIC" direction="in" :data="inbound" />
          <Tun v-if="inbound.type == inTypes.Tun" :data="inbound" />
          <AnyTls v-if="inbound.type == inTypes.AnyTls" :data="inbound" direction="in" />
          <TProxy v-if="inbound.type == inTypes.TProxy" :inbound="inbound" />
          <Transport v-if="Object.hasOwn(inbound,'transport')" :data="inbound" />
          <Users v-if="hasUser" :clients="clients" :data="initUsers" />
          <InTls v-if="HasTls.includes(inbound.type)"  :inbound="inbound" :tlsConfigs="tlsConfigs" :tls_id="inbound.tls_id" />
          <Multiplex v-if="MuxAvailable.includes(inbound.type)" direction="in" :data="inbound" />
        </v-window-item>
        <v-window-item value="c">
          <OutJsonVue :inData="inbound" :type="inbound.type" />
          <Multiplex v-if="Object.hasOwn(inbound,'multiplex')" direction="out" :data="inbound.out_json" />
          <Dial v-if="inbound.out_json" :dial="inbound.out_json" mode="client" />
          <v-card>
            <v-card-text>
              <v-card-subtitle>{{ $t('in.multiDomain') }}
                <v-chip color="primary" density="compact" variant="elevated" @click="add_addr"><v-icon icon="mdi-plus" /></v-chip>
              </v-card-subtitle>
              <template v-for="addr,index in inbound.addrs">
                {{ $t('in.addr') }} #{{ (index+1) }} <v-icon icon="mdi-delete" color="error" @click="inbound.addrs?.splice(index,1)" />
                <v-divider></v-divider>
                <AddrVue :addr="addr" :hasTls="HasTls.includes(inbound.type)" />
              </template>
            </v-card-text>
          </v-card>
        </v-window-item>
      </v-window>
    </form-section>
  </entity-drawer>
</template>

<script lang="ts">
import { InTypes, createInbound, Addr, ShadowTLS } from '@/types/inbounds'
import RandomUtil from '@/plugins/randomUtil'
import Dial from '@/components/Dial.vue'
import DomainResolver from '@/components/DomainResolver.vue'
import Listen from '@/components/Listen.vue'
import Direct from '@/components/protocols/Direct.vue'
import Users from '@/components/Users.vue'
import Shadowsocks from '@/components/protocols/Shadowsocks.vue'
import Hysteria from '@/components/protocols/Hysteria.vue'
import Hysteria2 from '@/components/protocols/Hysteria2.vue'
import Naive from '@/components/protocols/Naive.vue'
import ShadowTls from '@/components/protocols/ShadowTls.vue'
import Tuic from '@/components/protocols/Tuic.vue'
import Tun from '@/components/protocols/Tun.vue'
import Trojan from '@/components/protocols/Trojan.vue'
import AnyTls from '@/components/protocols/AnyTls.vue'
import InTls from '@/components/tls/InTLS.vue'
import TProxy from '@/components/protocols/TProxy.vue'
import Multiplex from '@/components/Multiplex.vue'
import Transport from '@/components/Transport.vue'
import AddrVue from '@/components/Addr.vue'
import OutJsonVue from '@/components/OutJson.vue'
import Data from '@/store/modules/data'
import EntityDrawer from './EntityDrawer.vue'
import FormSection from './FormSection.vue'
export default {
  // The drawer is driven explicitly by the `visible` prop (the parent passes it
  // alongside v-model); inheritAttrs:false stops the v-model's modelValue from
  // also falling through onto the EntityDrawer root and double-binding it.
  inheritAttrs: false,
  props: ['visible', 'id', 'inTags', 'tlsConfigs'],
  emits: ['close'],
  data() {
    return {
      inbound: createInbound("direct",{ id:0, "tag": "" }),
      title: "add",
      loading: false,
      side: "s",
      snapshot: "",
      inTypes: InTypes,
      inboundWithUsers: ['mixed', 'socks', 'http', 'shadowsocks', 'vmess', 'trojan', 'naive', 'hysteria', 'shadowtls', 'tuic', 'hysteria2', 'vless', 'anytls'],
      initUsers: {
        model: 'none',
        values: <any>[],
      },
      HasInData: [
        InTypes.SOCKS,
        InTypes.HTTP,
        InTypes.Mixed,
        InTypes.Shadowsocks,
        InTypes.VMess,
        InTypes.ShadowTLS,
        InTypes.Trojan,
        InTypes.Hysteria,
        InTypes.VLESS,
        InTypes.AnyTls,
        InTypes.TUIC,
        InTypes.Hysteria2,
        InTypes.Naive,
      ],
      HasTls: [
        InTypes.HTTP,
        InTypes.VMess,
        InTypes.Trojan,
        InTypes.Naive,
        InTypes.Hysteria,
        InTypes.TUIC,
        InTypes.Hysteria2,
        InTypes.VLESS,
        InTypes.AnyTls,
      ],
      MuxAvailable: [
        InTypes.VLESS,
        InTypes.VMess,
        InTypes.Trojan,
        InTypes.Shadowsocks,
      ],
      OnlyTLS: [InTypes.Hysteria, InTypes.Hysteria2, InTypes.TUIC, InTypes.Naive, InTypes.AnyTls ],
    }
  },
  methods: {
    async loadData(id: number) {
      this.loading = true
      const inboundArray = await Data().loadInbounds([id])
      this.inbound = inboundArray[0]
      if (this.HasInData.includes(this.inbound.type) && this.inbound.out_json == null) {
        this.inbound.out_json = {}
      }
      this.loading = false
      this.snapshot = JSON.stringify(this.inbound)
    },
    updateData(id: number) {
      if (id > 0) {
        this.loadData(id)
        this.title = "edit"
      }
      else {
        const port = RandomUtil.randomIntRange(10000, 60000)
        this.inbound = createInbound("direct",{ id: 0, tag: "direct-"+port ,listen: "::", listen_port: port })
        if (this.HasInData.includes(this.inbound.type)){
          this.inbound.addrs = []
          this.inbound.out_json = {}
        } else {
          delete this.inbound.addrs
          delete this.inbound.out_json
        }
        this.title = "add"
        this.loading = false
        this.snapshot = JSON.stringify(this.inbound)
      }
      this.side = "s"
      this.initUsers = {
        model: 'none',
        values: [],
      }
    },
    changeType() {
      if (!this.inbound.listen_port) this.inbound.listen_port = RandomUtil.randomIntRange(10000, 60000)
      // Tag change only in add inbound
      const tag = this.$props.id > 0 ? this.inbound.tag : this.inbound.type + "-" + this.inbound.listen_port
      // Use previous data
      const prevConfig = { id: this.inbound.id, tag: tag, listen: this.inbound.listen?? "::", listen_port: this.inbound.listen_port }
      this.inbound = createInbound(this.inbound.type, this.inbound.type != this.inTypes.Tun ? prevConfig : { tag: tag })
      if (this.HasInData.includes(this.inbound.type)){
        this.inbound.addrs = []
        this.inbound.out_json = {}
      } else {
        delete this.inbound.addrs
        delete this.inbound.out_json
      }
      this.side = "s"
    },
    add_addr() {
      this.inbound.addrs?.push(<Addr>{ server: location.hostname, server_port: this.inbound.listen_port })
    },
    closeModal() {
      this.updateData(0) // reset
      this.$emit('close')
    },
    async saveChanges() {
      // Guard against double-submit (button is also :disabled while loading).
      if (!this.$props.visible || this.loading) return
      // check duplicate tag
      const isDuplicatedTag = Data().checkTag("inbound", this.inbound.id, this.inbound.tag)
      if (isDuplicatedTag) return

      // save data
      this.loading = true
      try {
        let clientIds = []
        if (this.hasUser) {
          switch (this.initUsers.model) {
            case 'all':
              clientIds = this.clients.map((c:any) => c.id)
              break
            case 'group':
              clientIds = this.clients.filter((c:any) => this.initUsers.values.includes(c.group)).map((c:any) => c.id)
              break
            case 'client':
              clientIds = this.initUsers.values
          }
        }
        const success = await Data().save("inbounds", this.$props.id == 0 ? "new" : "edit", this.inbound, clientIds)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
  },
  computed: {
    dirty(): boolean {
      return this.snapshot !== "" && JSON.stringify(this.inbound) !== this.snapshot
    },
    validate() {
      if (this.inbound == undefined) return false
      if (this.inbound.tag == "") return false
      if (this.inbound.listen_port > 65535 || this.inbound.listen_port < 1) return false
      if (this.OnlyTLS.includes(this.inbound.type) && this.inbound.tls_id == 0) return false
      return true
    },
    clients() {
      return Data().clients?? []
    },
    hasUser() {
      if (this.$props.id > 0) return false
      if (!this.inboundWithUsers.includes(this.inbound.type)) return false
      if (this.inbound.type == InTypes.ShadowTLS && (<ShadowTLS>this.inbound).version < 3 ) return false
      if ((<any>this.inbound).managed) return false
      return true
    },
    setSystemProxy: {
      get(): boolean {
        return (<any>this.inbound).set_system_proxy === true
      },
      set(v:boolean) {
        if (v) {
          (<any>this.inbound).set_system_proxy = true
        } else {
          delete (<any>this.inbound).set_system_proxy
        }
      }
    },
  },
  watch: {
    visible(newValue) {
      // Mirror the modal's @after-enter: load/seed the form when the drawer opens.
      if (newValue) {
        this.loading = true
        this.updateData(this.$props.id)
      }
    },
  },
  components: {
    EntityDrawer, FormSection,
    Listen, InTls, Hysteria2, Naive, Direct, Shadowsocks,
    Users, Hysteria, ShadowTls, TProxy, Multiplex, Tuic, Tun,
    Trojan, AnyTls, Transport, AddrVue, OutJsonVue, Dial, DomainResolver
  }
}
</script>
