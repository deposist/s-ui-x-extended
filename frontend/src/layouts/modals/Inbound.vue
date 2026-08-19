<template>
  <v-dialog transition="dialog-bottom-transition" width="800" @after-enter="updateData(id)">
    <v-card class="rounded-lg" :loading="loading">
      <v-card-title>
        {{ $t('actions.' + title) + " " + $t('objects.inbound') }}
      </v-card-title>
      <v-divider></v-divider>
      <v-skeleton-loader
          class="mx-auto border"
          width="95%"
          type="card, text, divider, list-item-two-line"
          v-if="loading"
        ></v-skeleton-loader>
      <v-card-text style="padding: 0 16px; overflow-y: scroll;">
        <v-container style="padding: 0;" :hidden="loading">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
              hide-details
              :label="$t('type')"
              :items="inTypeItems"
              :item-props="itemProps"
              v-model="inbound.type"
              @update:modelValue="changeType">
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('type')" :text="fieldHint('type')" />
                </template>
              </v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model="inbound.tag" :label="$t('objects.tag')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('tag')" :text="fieldHint('tag')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" v-if="showInboundRecommendedPreset">
              <v-btn
                color="primary"
                prepend-icon="mdi-star-plus"
                variant="tonal"
                @click="applyCurrentInboundRecommendations">
                {{ $t('types.inbound.recommendedPreset') }}
              </v-btn>
            </v-col>
            <v-col cols="12" v-if="showInboundProtocolNote">
              <v-alert type="info" variant="tonal" density="compact">
                {{ fieldHint('protocol_note') }}
              </v-alert>
            </v-col>
          </v-row>
          <v-card
            v-if="[inTypes.HTTP, inTypes.Mixed].includes(inbound.type)"
            border
            density="compact"
            color="background"
            style="margin-bottom: 8px;">
            <v-card-text>
              <v-row>
                <v-col cols="12" sm="6" md="4">
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
              <Listen :data="inbound" :inTags="inTags" :field-hints="currentFieldHints" v-if="inbound.type != inTypes.Tun && inbound.type != inTypes.Call" />
              <Direct v-if="inbound.type == inTypes.Direct" :data="inbound" :field-hints="currentFieldHints" />
              <Shadowsocks v-if="inbound.type == inTypes.Shadowsocks" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <Hysteria v-if="inbound.type == inTypes.Hysteria" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <Hysteria2 v-if="inbound.type == inTypes.Hysteria2" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <Naive v-if="inbound.type == inTypes.Naive" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <Trojan v-if="inbound.type == inTypes.Trojan" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <ShadowTls v-if="inbound.type == inTypes.ShadowTLS" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <Tuic v-if="inbound.type == inTypes.TUIC" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <Tun v-if="inbound.type == inTypes.Tun" :data="inbound" :field-hints="currentFieldHints" />
              <AnyTls v-if="inbound.type == inTypes.AnyTls" :data="inbound" direction="in" :field-hints="currentFieldHints" />
              <VlessInbound v-if="inbound.type == inTypes.VLESS" :data="inbound" :mode="id == 0 ? 'create' : 'edit'" :field-hints="currentFieldHints" />
              <Mieru v-if="inbound.type == inTypes.Mieru" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <Sudoku v-if="inbound.type == inTypes.Sudoku" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <TrustTunnel v-if="inbound.type == inTypes.TrustTunnel" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <SshInbound v-if="inbound.type == inTypes.SSH" :data="inbound" :field-hints="currentFieldHints" />
              <MTProxy v-if="inbound.type == inTypes.MTProxy" :data="inbound" :field-hints="currentFieldHints" />
              <Call v-if="inbound.type == inTypes.Call" direction="in" :data="inbound" :field-hints="currentFieldHints" />
              <BondInbound v-if="inbound.type == inTypes.Bond" :data="inbound" :inTags="inTags" />
              <CoreFailoverInbound v-if="inbound.type == inTypes.CoreFailover" :data="inbound" :inTags="inTags" />
              <TProxy v-if="inbound.type == inTypes.TProxy" :inbound="inbound" :field-hints="currentFieldHints" />
              <Transport v-if="Object.hasOwn(inbound,'transport')" :data="inbound" :field-hints="currentFieldHints" />
              <Users v-if="hasUser" :clients="clients" :data="initUsers" :field-hints="currentFieldHints" />
              <InTls v-if="HasTls.includes(inbound.type)"  :inbound="inbound" :tlsConfigs="tlsConfigs" :tls_id="inbound.tls_id" :field-hints="currentFieldHints" :allowed-template-kinds="allowedTlsTemplateKinds" />
              <Multiplex v-if="MuxAvailable.includes(inbound.type)" direction="in" :data="inbound" :field-hints="currentFieldHints" />
            </v-window-item>
            <v-window-item value="c">
              <OutJsonVue v-if="inbound.out_json" :inData="inbound" :type="inbound.type" :field-hints="currentFieldHints" />
              <Multiplex v-if="Object.hasOwn(inbound,'multiplex')" direction="out" :data="inbound.out_json" :field-hints="currentFieldHints" />
              <Dial v-if="inbound.out_json" :dial="inbound.out_json" mode="client" :field-hints="currentFieldHints" />
              <v-card>
                <v-card-text>
                  <v-card-subtitle class="d-flex align-center ga-1">{{ $t('in.multiDomain') }}
                    <SettingInfo v-if="fieldHint('multi_domain')" :text="fieldHint('multi_domain')" />
                    <v-chip color="primary" density="compact" variant="elevated" @click="add_addr"><v-icon icon="mdi-plus" /></v-chip>
                  </v-card-subtitle>
                  <template v-for="addr,index in inbound.addrs">
                    {{ $t('in.addr') }} #{{ (index+1) }} <v-icon icon="mdi-delete" color="error" @click="inbound.addrs?.splice(index,1)" />
                    <v-divider></v-divider>
                    <AddrVue :addr="addr" :hasTls="HasTls.includes(inbound.type)" :field-hints="currentFieldHints" />
                  </template>
                </v-card-text>
              </v-card>
            </v-window-item>
          </v-window>
        </v-container>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn
          color="primary"
          variant="outlined"
          @click="closeModal"
        >
          {{ $t('actions.close') }}
        </v-btn>
        <v-btn
          color="primary"
          variant="tonal"
          :loading="loading"
          :disabled="loading || !validate"
          @click="saveChanges"
        >
          {{ $t('actions.save') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import { InTypes, createInbound, Addr, ShadowTLS } from '@/types/inbounds'
import { inboundAssignable, HasInData, HasTls, MuxAvailable, OnlyTLS } from '@/types/capabilities'
import HttpUtils from '@/plugins/httputil'
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
import VlessInbound from '@/components/protocols/VlessInbound.vue'
import Mieru from '@/components/protocols/Mieru.vue'
import Sudoku from '@/components/protocols/Sudoku.vue'
import TrustTunnel from '@/components/protocols/TrustTunnel.vue'
import SshInbound from '@/components/protocols/SshInbound.vue'
import MTProxy from '@/components/protocols/MTProxy.vue'
import Call from '@/components/protocols/Call.vue'
import BondInbound from '@/components/protocols/BondInbound.vue'
import CoreFailoverInbound from '@/components/protocols/CoreFailoverInbound.vue'
import InTls from '@/components/tls/InTLS.vue'
import TProxy from '@/components/protocols/TProxy.vue'
import Multiplex from '@/components/Multiplex.vue'
import Transport from '@/components/Transport.vue'
import AddrVue from '@/components/Addr.vue'
import OutJsonVue from '@/components/OutJson.vue'
import Data from '@/store/modules/data'
import { push } from 'notivue'
import SettingInfo from '@/components/SettingInfo.vue'
import { applyInboundRecommendedValues, hasInboundRecommendedPreset, inboundFieldHintsForType } from '@/utils/defaultRecommendations'
import { inboundAllowedTlsTemplateKinds, isInboundTlsTemplateCompatible } from '@/utils/tlsCompatibility'
export default {
  props: ['visible', 'id', 'inTags', 'tlsConfigs'],
  emits: ['close'],
  data() {
    return {
      inbound: createInbound("direct",{ id:0, "tag": "" }),
      title: "add",
      loading: false,
      side: "s",
      inTypes: InTypes,
      // Capability lists are generated from core/capabilities/protocols.json by
      // scripts/gen-capabilities.cjs (shared source of truth with the Go backend).
      inboundAssignable,
      initUsers: {
        model: 'none',
        values: <any>[],
      },
      HasInData,
      HasTls,
      MuxAvailable,
      OnlyTLS,
      // Inbound types whose build tag is not compiled into the running binary
      // (from /api/capabilities). Such types are shown disabled in the picker.
      unavailableTypes: <string[]>[],
      requiredInitialUsers: [InTypes.Mieru, InTypes.TrustTunnel, InTypes.SSH, InTypes.MTProxy],
      editHadOutJson: true,
    }
  },
  async created() {
    // Best-effort: gate inbound types not compiled into this build. Failure
    // (e.g. older backend without the endpoint) leaves every type available.
    const resp = await HttpUtils.get('api/capabilities')
    const inbounds = resp?.obj?.inbounds
    if (Array.isArray(inbounds)) {
      this.unavailableTypes = inbounds.filter((i: any) => i.available === false).map((i: any) => i.type)
    }
  },
  methods: {
    itemProps(item: any) {
      return item.props ?? {}
    },
    async loadData(id: number) {
      this.loading = true
      const inboundArray = await Data().loadInbounds([id])
      this.inbound = inboundArray[0]
      this.editHadOutJson = this.inbound.out_json != null
      this.loading = false
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
      }
      this.side = "s"
      this.initUsers = {
        model: 'none',
        values: [],
      }
    },
    fieldHint(key: string): string {
      const hintKey = (this.currentFieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    applyCurrentInboundRecommendations() {
      applyInboundRecommendedValues(this.inbound)
    },
    changeType() {
      const listenPort = this.inbound.listen_port || RandomUtil.randomIntRange(10000, 60000)
      // Tag change only in add inbound
      const tag = this.$props.id > 0 ? this.inbound.tag : this.inbound.type + "-" + listenPort
      // Use previous data, except Call inbounds which do not support listen fields.
      const prevConfig: any = { id: this.inbound.id, tag: tag }
      if (this.inbound.type != this.inTypes.Call) {
        prevConfig.listen_port = listenPort
        if (this.inbound.listen != null) prevConfig.listen = this.inbound.listen
        else if (this.$props.id == 0) prevConfig.listen = "::"
      }
      this.inbound = createInbound(this.inbound.type, this.inbound.type != this.inTypes.Tun ? prevConfig : { tag: tag })
      if (this.HasInData.includes(this.inbound.type)){
        this.inbound.addrs = []
        this.inbound.out_json = {}
      } else {
        delete this.inbound.addrs
        delete this.inbound.out_json
      }
      this.side = "s"
      this.resetInitUsersForType()
    },
    resetInitUsersForType() {
      this.initUsers = {
        model: this.hasUser && this.clients.length > 0 ? 'all' : 'none',
        values: [],
      }
    },
    selectedInitialClientIds(): number[] {
      if (!this.hasUser) return []
      switch (this.initUsers.model) {
        case 'all':
          return this.clients.map((c:any) => c.id)
        case 'group':
          return this.clients.filter((c:any) => this.initUsers.values.includes(c.group)).map((c:any) => c.id)
        case 'client':
          return this.initUsers.values
        default:
          return []
      }
    },
    requiresSelectedUsers(): boolean {
      return this.hasUser && this.requiredInitialUsers.includes(this.inbound.type)
    },
    ensureRequiredUsersSelected(): boolean {
      if (!this.requiresSelectedUsers()) return true
      if (this.selectedInitialClientIds().length > 0) return true
      push.error({
        title: this.$t('failed'),
        message: this.$t('error.invalidData') + ': ' + this.$t('pages.clients'),
      })
      return false
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

      const clientIds = this.selectedInitialClientIds()
      if (!this.ensureRequiredUsersSelected()) return

      // save data
      this.loading = true
      try {
        const saveData = JSON.parse(JSON.stringify(this.inbound))
        if (this.$props.id > 0 && !this.editHadOutJson && saveData.out_json && Object.keys(saveData.out_json).length === 0) delete saveData.out_json
        const success = await Data().save("inbounds", this.$props.id == 0 ? "new" : "edit", saveData, clientIds)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
  },
  computed: {
    validate() {
      if (this.inbound == undefined) return false
      if (this.inbound.tag == "") return false
      if (this.inbound.listen_port > 65535 || this.inbound.listen_port < 1) return false
      if (this.OnlyTLS.includes(this.inbound.type) && this.inbound.tls_id == 0) return false
      if (!this.selectedTlsTemplateCompatible) return false
      return true
    },
    clients() {
      return Data().clients?? []
    },
    inTypeItems() {
      const values = Object.values(this.inTypes)
      return Object.keys(this.inTypes).map((key, index) => {
        const value = values[index]
        const unavailable = this.unavailableTypes.includes(value)
        return {
          title: unavailable ? `${key} — not in this build` : key,
          value,
          props: { disabled: unavailable },
        }
      })
    },
    hasUser() {
      if (this.$props.id > 0) return false
      if (!this.inboundAssignable.includes(this.inbound.type)) return false
      if (this.inbound.type == InTypes.ShadowTLS && (<ShadowTLS>this.inbound).version < 3 ) return false
      if ((<any>this.inbound).managed) return false
      return true
    },
    allowedTlsTemplateKinds() {
      return inboundAllowedTlsTemplateKinds(this.inbound.type)
    },
    currentFieldHints(): Record<string, string> {
      return inboundFieldHintsForType(this.inbound.type)
    },
    showInboundRecommendedPreset(): boolean {
      return this.$props.id == 0 && hasInboundRecommendedPreset(this.inbound.type)
    },
    showInboundProtocolNote(): boolean {
      return this.$props.id == 0 && this.inbound.type !== InTypes.VLESS && !hasInboundRecommendedPreset(this.inbound.type) && Boolean(this.fieldHint('protocol_note'))
    },
    selectedTlsTemplateCompatible(): boolean {
      if (!this.inbound?.tls_id) return true
      const selected = (this.$props.tlsConfigs ?? []).find((tlsConfig: any) => tlsConfig.id === this.inbound.tls_id)
      return selected == null || isInboundTlsTemplateCompatible(this.inbound.type, selected)
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
      if (newValue) {
        this.loading = true
      }
    },
  },
  components: {
    Listen, InTls, Hysteria2, Naive, Direct, Shadowsocks,
    Users, Hysteria, ShadowTls, TProxy, Multiplex, Tuic, Tun,
    Trojan, AnyTls, Transport, AddrVue, OutJsonVue, Dial, DomainResolver,
    Mieru, Sudoku, TrustTunnel, SshInbound, MTProxy, Call, BondInbound, CoreFailoverInbound, VlessInbound,
    SettingInfo
  }
}
</script>
