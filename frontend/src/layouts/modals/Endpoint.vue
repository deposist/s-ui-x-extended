<template>
  <v-dialog transition="dialog-bottom-transition" width="800">
    <v-card class="rounded-lg">
      <v-card-title>
        {{ $t('actions.' + title) + " " + $t('objects.endpoint') }}
      </v-card-title>
      <v-divider></v-divider>
      <v-card-text style="padding: 0 16px; overflow-y: scroll;">
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-select
            hide-details
            :disabled="endpoint.id > 0"
            :label="$t('type')"
            :items="Object.keys(epTypes).map((key,index) => ({title: key, value: Object.values(epTypes)[index]}))"
            v-model="endpoint.type"
            @update:modelValue="changeType">
              <template #append-inner>
                <SettingInfo v-if="fieldHint('type')" :text="fieldHint('type')" />
              </template>
            </v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="endpoint.tag" :label="$t('objects.tag')" hide-details>
              <template #append-inner>
                <SettingInfo v-if="fieldHint('tag')" :text="fieldHint('tag')" />
              </template>
            </v-text-field>
          </v-col>
          <v-col cols="12" v-if="showEndpointRecommendedPreset">
            <v-btn
              color="primary"
              prepend-icon="mdi-star-plus"
              variant="tonal"
              @click="applyCurrentEndpointRecommendations">
              {{ $t('types.endpoint.recommendedPreset') }}
            </v-btn>
          </v-col>
        </v-row>
        <v-card v-if="endpoint.type == epTypes.Wireguard" border density="compact" color="background" class="mb-2">
          <v-card-subtitle style="padding-top: 8px;">
            {{ $t('types.endpoint.awg.title') }}
            <v-switch
              class="d-inline-block"
              style="vertical-align: middle; margin-left: 8px;"
              color="primary"
              hide-details
              :label="$t('types.endpoint.awg.managed')"
              v-model="awgManagedFlag">
            </v-switch>
          </v-card-subtitle>
          <v-card-text v-if="awgManagedFlag">
            <v-alert type="info" variant="tonal" density="compact" class="mb-3">
              {{ $t('types.endpoint.awg.managedHint') }}
            </v-alert>
            <v-alert v-if="amneziaParamsChanged" type="warning" variant="tonal" density="compact" class="mb-3">
              {{ $t('types.endpoint.awg.obfuscationChangedWarning') }}
            </v-alert>
            <v-row>
              <v-col cols="12" sm="6" md="5">
                <v-text-field v-model="awgPublicEndpoint" :label="$t('types.endpoint.awg.publicEndpoint')" placeholder="vpn.example.com:51820" hide-details />
              </v-col>
              <v-col cols="12" sm="6" md="4">
                <v-text-field v-model="awgDns" :label="$t('types.endpoint.awg.dns')" hide-details />
              </v-col>
              <v-col cols="12" sm="6" md="3">
                <v-text-field v-model.number="awgDefaultDeviceLimit" type="number" min="1" max="100" :label="$t('types.endpoint.awg.defaultDeviceLimit')" hide-details />
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
        <Wireguard v-if="endpoint.type == epTypes.Wireguard"
          :data="endpoint"
          :field-hints="currentFieldHints"
          :peers-managed="endpoint.awgManaged === true || awgManagedFlag"
          @getWgPubKey="getWgPubKey"
          @newWgKey="newWgKey"
          @addPeer="addWgPeer"
          @delPeer="delWgPeer"
          @refreshPeerKey="refreshWgPeerKey" />
        <Warp v-if="endpoint.type == epTypes.Warp" :data="endpoint" />
        <TailscaleVue v-if="endpoint.type == epTypes.Tailscale" :data="endpoint" />
        <VpnServer v-if="endpoint.type == epTypes.VpnServer" :data="endpoint" :field-hints="currentFieldHints" />
        <VpnClient v-if="endpoint.type == epTypes.VpnClient" :data="endpoint" :field-hints="currentFieldHints" />
        <Dial v-if="!noDial.includes(endpoint.type)" :dial="endpoint" :field-hints="currentFieldHints" />
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
          :disabled="loading"
          @click="saveChanges"
        >
          {{ $t('actions.save') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import { EpTypes, createEndpoint } from '@/types/endpoints'
import RandomUtil from '@/plugins/randomUtil'
import Dial from '@/components/Dial.vue'
import Wireguard from '@/components/protocols/Wireguard.vue'
import Warp from '@/components/protocols/Warp.vue'
import TailscaleVue from '@/components/protocols/Tailscale.vue'
import VpnServer from '@/components/protocols/VpnServer.vue'
import VpnClient from '@/components/protocols/VpnClient.vue'
import HttpUtils from '@/plugins/httputil'
import { push } from 'notivue'
import { i18n } from '@/locales'
import Data from '@/store/modules/data'
import SettingInfo from '@/components/SettingInfo.vue'
import { applyEndpointRecommendedValues, endpointFieldHintsForType, hasEndpointRecommendedPreset } from '@/utils/defaultRecommendations'
export default {
  props: ['visible', 'data', 'id', 'tags'],
  emits: ['close'],
  data() {
    return {
      endpoint: createEndpoint("wireguard",{ "tag": "" }),
      title: "add",
      tab: "t1",
      loading: false,
      // Snapshot of the amnezia options at modal open; a change means every
      // provisioned device must re-download its config / rescan the QR
      // (configs render live from endpoint options, so nothing regenerates
      // server-side - but already-installed clients keep the old values).
      originalAmnezia: "",
      epTypes: EpTypes,
      noDial: [EpTypes.VpnServer, EpTypes.VpnClient],
    }
  },
  methods: {
    async updateData(id: number) {
      if (id > 0) {
        const newData = JSON.parse(this.$props.data)
        this.endpoint = newData
        this.title = "edit"
        this.originalAmnezia = JSON.stringify(newData.amnezia ?? null)
      }
      else {
        this.endpoint.type = "wireguard"
        this.endpoint.listen_port = RandomUtil.randomIntRange(10000, 60000)
        this.changeType()
        this.title = "add"
        this.originalAmnezia = ""
      }
      this.tab = "t1"
    },
    fieldHint(key: string): string {
      const hintKey = (this.currentFieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    applyCurrentEndpointRecommendations() {
      applyEndpointRecommendedValues(this.endpoint)
    },
    async changeType() {
      // Tag change only in add endpoint
      const tag = this.endpoint.type + "-" + RandomUtil.randomSeq(3)
      
      // Use previous data
      let prevConfig = {}
      switch (this.endpoint.type) {
        case EpTypes.Wireguard:
          const wgKeys = (await this.genWgKey())
          const randomIPoctet = RandomUtil.randomIntRange(1, 255)
          prevConfig = {
            tag: tag,
            listen_port: this.endpoint.listen_port ?? RandomUtil.randomIntRange(10000, 60000),
            address: ['10.0.0.'+ randomIPoctet.toString() +'/32','fe80::'+ randomIPoctet.toString(16) +'/128'],
            private_key: wgKeys.private_key,
            peers: [],
            ext: {
              public_key: wgKeys.public_key,
              keys: []
            }
          }
          break
        case EpTypes.Warp:
          prevConfig = {
            tag: tag,
          }
          break
        case EpTypes.Tailscale:
          prevConfig = { tag: tag }
          break
        case EpTypes.VpnServer:
          prevConfig = {
            tag: tag,
            address: '10.0.0.1',
            users: [{ address: '10.0.0.2', key: RandomUtil.randomUUID() }],
            inbounds: [],
          }
          break
        case EpTypes.VpnClient:
          prevConfig = {
            tag: tag,
            address: '10.0.0.2',
            key: RandomUtil.randomUUID(),
            outbound: {},
          }
          break
      }
      this.endpoint = createEndpoint(this.endpoint.type, prevConfig)
    },
    closeModal() {
      this.updateData(0) // reset
      this.$emit('close')
    },
    async saveChanges() {
      if (!this.$props.visible || this.loading) return
      
      // check duplicate tag
      const isDuplicatedTag = Data().checkTag("endpoint",this.endpoint.id, this.endpoint.tag)
      if (isDuplicatedTag) return

      // save data
      this.loading = true
      try {
        const success = await Data().save("endpoints", this.$props.id == 0 ? "new" : "edit", this.endpoint)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
    async genWgKey(){
      this.loading = true
      const msg = await HttpUtils.get('api/keypairs', { k: "wireguard" })
      this.loading = false
      let result = { private_key: "", public_key: "" }
      if (msg.success) {
        msg.obj.forEach((line:string) => {
          if (line.startsWith("PrivateKey")){
            result.private_key = line.substring(12)
          }
          if (line.startsWith("PublicKey")){
            result.public_key = line.substring(11)
          }
        })
      } else {
        push.error({
          message: i18n.global.t('error') + ": " + msg.obj
        })
      }
      return result
    },
    async newWgKey(){
      this.loading = true
      const newKeys = await this.genWgKey()
      this.endpoint.private_key = newKeys.private_key
      if (!this.endpoint.ext) this.endpoint.ext = {keys: []}
      this.endpoint.ext.public_key = newKeys.public_key
      this.loading = false
    },
    async getWgPubKey(private_key: string) {
      if (!this.endpoint.ext) this.endpoint.ext = {keys: []}
      this.loading = true
      const msg = await HttpUtils.get('api/keypairs', { k: "wireguard", o: private_key })
      if (msg.success) {
        this.endpoint.ext.public_key = msg.obj[0]
      }
      this.loading = false
    },
    async addWgPeer(){
      if (this.endpoint.type != EpTypes.Wireguard) return
      this.loading = true
      const newKeys = await this.genWgKey()
      if (!this.endpoint.ext) this.endpoint.ext = {keys: []}
      this.endpoint.ext.keys.push(newKeys)
      this.endpoint.peers.push({
        public_key: newKeys.public_key,
        allowed_ips: [this.findFreeIP()]
      })
      this.loading = false
    },
    findFreeIP(): string{
      const peerAllowedIPs = this.endpoint.peers.map((peer: any) => peer.allowed_ips).flat()
      for (let i = 2; i < 255; i++) {
        const newIP = '10.0.1.'+ i.toString() +'/32'
        if (!peerAllowedIPs.includes(newIP)) return newIP
      }
      return '0.0.0.0/0'
    },
    delWgPeer(index: number){
      if (this.endpoint.type != EpTypes.Wireguard) return
      this.endpoint.ext.keys = this.endpoint.ext.keys.filter((key: any) => key.public_key != this.endpoint.peers[index].public_key)
      this.endpoint.peers.splice(index, 1)
    },
    async fillRandomAmneziaHeaders() {
      const msg = await HttpUtils.get('api/awg/obfuscation/random')
      if (msg.success && msg.obj && this.endpoint.amnezia) {
        this.endpoint.amnezia.h1 = msg.obj.h1
        this.endpoint.amnezia.h2 = msg.obj.h2
        this.endpoint.amnezia.h3 = msg.obj.h3
        this.endpoint.amnezia.h4 = msg.obj.h4
      }
    },
    async refreshWgPeerKey(index: number) {
      this.loading = true
      const newKeys = await this.genWgKey()
      if (!this.endpoint.ext) this.endpoint.ext = {keys: []}
      const indexKeys = this.endpoint.ext.keys.findIndex((key: any) => key.public_key == this.endpoint.peers[index].public_key)
      this.endpoint.ext.keys[indexKeys == -1 ? this.endpoint.ext.keys.length : indexKeys] = newKeys
      this.endpoint.peers[index].public_key = newKeys.public_key
      this.loading = false
    },
  },
  computed: {
    awgManagedFlag: {
      get(): boolean { return this.endpoint.ext?.managed === true },
      set(v: boolean) {
        if (!this.endpoint.ext) this.endpoint.ext = { keys: [] }
        if (v) {
          this.endpoint.ext.managed = true
          if (!this.endpoint.ext.publicEndpoint) this.endpoint.ext.publicEndpoint = ''
          if (!this.endpoint.ext.dns || this.endpoint.ext.dns.length === 0) this.endpoint.ext.dns = ['1.1.1.1', '1.0.0.1']
          if (!this.endpoint.ext.defaultDeviceLimit) this.endpoint.ext.defaultDeviceLimit = 3
          if (!this.endpoint.amnezia) {
            // Junk/padding defaults stay static; H1-H4 come from the server
            // crypto/rand generator so every managed endpoint gets unique,
            // non-overlapping header ranges.
            this.endpoint.amnezia = { jc: 3, jmin: 10, jmax: 20, s1: 15, s2: 18, s3: 12, s4: 8, i1: '<b 0x01020304><r 8>' }
            this.fillRandomAmneziaHeaders()
          }
          this.endpoint.peers = this.endpoint.peers ?? []
        } else {
          this.endpoint.ext.managed = false
        }
      },
    },
    awgPublicEndpoint: {
      get(): string { return this.endpoint.ext?.publicEndpoint ?? '' },
      set(v: string) { if (this.endpoint.ext) this.endpoint.ext.publicEndpoint = v.trim() },
    },
    awgDns: {
      get(): string { return (this.endpoint.ext?.dns ?? []).join(',') },
      set(v: string) {
        if (!this.endpoint.ext) return
        this.endpoint.ext.dns = v.split(',').map((item: string) => item.trim()).filter((item: string) => item.length > 0)
      },
    },
    awgDefaultDeviceLimit: {
      get(): number { return this.endpoint.ext?.defaultDeviceLimit ?? 3 },
      set(v: number) { if (this.endpoint.ext) this.endpoint.ext.defaultDeviceLimit = v },
    },
    currentFieldHints(): Record<string, string> {
      return endpointFieldHintsForType(this.endpoint.type)
    },
    amneziaParamsChanged(): boolean {
      if (this.title !== 'edit' || this.originalAmnezia === "") return false
      return JSON.stringify(this.endpoint.amnezia ?? null) !== this.originalAmnezia
    },
    showEndpointRecommendedPreset(): boolean {
      return this.$props.id == 0 && hasEndpointRecommendedPreset(this.endpoint.type)
    },
  },
  watch: {
    visible(v) {
      if (v) {
        this.updateData(this.$props.id)
      }
    },
  },
  components: { SettingInfo, Dial, Wireguard, Warp, TailscaleVue, VpnServer, VpnClient }
}
</script>
