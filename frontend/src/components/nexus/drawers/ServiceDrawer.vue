<template>
  <entity-drawer
    :dirty="dirty"
    :loading="loading"
    :model-value="visible"
    :save-disabled="saveBlockedReason !== ''"
    :save-disabled-reason="saveBlockedReason"
    :saving="loading"
    :title="$t('actions.' + title) + ' ' + $t('objects.service')"
    :width="720"
    @close="closeModal"
    @save="saveChanges"
  >
    <form-section icon="lucide:sliders-horizontal" :title="$t('form.sections.configuration')">
      <v-row>
        <v-col cols="12" sm="6">
          <v-select
            hide-details
            :label="$t('type')"
            :items="Object.keys(srvTypes).map((key,index) => ({title: key, value: Object.values(srvTypes)[index]}))"
            v-model="srv.type"
            @update:modelValue="changeType">
            <template #append-inner>
              <SettingInfo v-if="fieldHint('type')" :text="fieldHint('type')" />
            </template>
          </v-select>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field v-model="srv.tag" :label="$t('objects.tag')" hide-details :error="isBlankIdentity(srv.tag)">
            <template #append-inner>
              <SettingInfo v-if="fieldHint('tag')" :text="fieldHint('tag')" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" v-if="showServiceRecommendedPreset">
          <v-btn
            color="primary"
            prepend-icon="mdi-star-plus"
            variant="tonal"
            @click="applyCurrentServiceRecommendations">
            {{ $t('types.service.recommendedPreset') }}
          </v-btn>
        </v-col>
      </v-row>

      <Listen v-if="!NoListen.includes(srv.type)" :data="srv" :inTags="inTags" :field-hints="currentFieldHints" />
      <Derp v-if="srv.type == srvTypes.DERP" :data="srv" :inTags="inTags" :tsTags="tsTags" :field-hints="currentFieldHints" />
      <SSMapi v-if="srv.type == srvTypes.SSMAPI" :data="srv" :ssTags="ssTags" :field-hints="currentFieldHints" />
      <Ocm v-if="srv.type == srvTypes.OCM" :data="srv" :field-hints="currentFieldHints" />
      <Ccm v-if="srv.type == srvTypes.CCM" :data="srv" :field-hints="currentFieldHints" />
      <OomKiller v-if="srv.type == srvTypes.OOMKiller" :data="srv" :field-hints="currentFieldHints" />
      <Profiler v-if="srv.type == srvTypes.Profiler" :data="srv" :field-hints="currentFieldHints" />
      <div v-if="srv.type == srvTypes.Resolved" style="margin-top: 8px;">
        <v-alert type="info" variant="tonal" density="compact">
          Resolved DNS service has no type-specific settings beyond Listen.
        </v-alert>
      </div>
      <InTLS v-if="HasTls.includes(srv.type)"  :inbound="srv" :tlsConfigs="tlsConfigs" :tls_id="srv.tls_id" :field-hints="currentFieldHints" />
    </form-section>
  </entity-drawer>
</template>

<script lang="ts">
import { SrvTypes, createSrv } from '@/types/services'
import RandomUtil from '@/plugins/randomUtil'
import Listen from '@/components/Listen.vue'
import Derp from '@/components/services/Derp.vue'
import Ocm from '@/components/services/Ocm.vue'
import Ccm from '@/components/services/Ccm.vue'
import OomKiller from '@/components/services/OomKiller.vue'
import Profiler from '@/components/services/Profiler.vue'
import InTLS from '@/components/tls/InTLS.vue'
import SSMapi from '@/components/services/SSMAPI.vue'
import Data from '@/store/modules/data'
import EntityDrawer from './EntityDrawer.vue'
import FormSection from './FormSection.vue'
import SettingInfo from '@/components/SettingInfo.vue'
import { applyServiceRecommendedValues, hasServiceRecommendedPreset, serviceFieldHintsForType } from '@/utils/defaultRecommendations'
import { isBlankIdentity } from '@/utils/entityIdentity'
export default {
  inheritAttrs: false,
  props: ['visible', 'data', 'id', 'inTags', 'tsTags', 'ssTags', 'tlsConfigs'],
  emits: ['close'],
  data() {
    return {
      srv: createSrv("derp",{ "tag": "" }),
      title: "add",
      loading: false,
      snapshot: "",
      srvTypes: SrvTypes,
      HasTls: [SrvTypes.DERP, SrvTypes.SSMAPI, SrvTypes.OCM, SrvTypes.CCM],
      NoListen: [SrvTypes.OOMKiller, SrvTypes.Profiler],
    }
  },
  methods: {
    // Exposed so the tag field's error state uses the same blank rule as Save.
    isBlankIdentity,
    async updateData(id: number) {
      if (id > 0) {
        const newData = JSON.parse(this.$props.data)
        this.srv = newData
        this.title = "edit"
      }
      else {
        const port = RandomUtil.randomIntRange(10000, 60000)
        this.srv = createSrv("derp", {
          tag: "derp-" + RandomUtil.randomSeq(3),
          listen: '::',
          listen_port: port,
        })
        this.title = "add"
      }
      this.snapshot = JSON.stringify(this.srv)
    },
    fieldHint(key: string): string {
      const hintKey = (this.currentFieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    applyCurrentServiceRecommendations() {
      applyServiceRecommendedValues(this.srv)
    },
    changeType() {
      // Tag change only in add service
      const tag = this.$props.id > 0 ? this.srv.tag : this.srv.type + "-" + RandomUtil.randomSeq(3)
      // Use previous data
      const prevConfig = this.srv.type == SrvTypes.OOMKiller
        ? { id: this.srv.id, tag: tag }
        : { id: this.srv.id, tag: tag, listen: this.srv.listen, listen_port: this.srv.listen_port }
      this.srv = createSrv(this.srv.type, prevConfig)
    },
    closeModal() {
      this.updateData(0) // reset
      this.$emit('close')
    },
    async saveChanges() {
      // Guard against double-submit (button is also :disabled while loading).
      if (!this.$props.visible || this.loading) return

      // check duplicate tag
      const isDuplicatedTag = Data().checkTag("service",this.srv.id, this.srv.tag)
      if (isDuplicatedTag) return

      // save data
      this.loading = true
      try {
        const success = await Data().save("services", this.$props.id == 0 ? "new" : "edit", this.srv)
        if (success) this.closeModal()
      } finally {
        this.loading = false
      }
    },
  },
  computed: {
    dirty(): boolean {
      return this.snapshot !== "" && JSON.stringify(this.srv) !== this.snapshot
    },
    // A service saved without a tag cannot be referenced or told apart in the
    // list, so block it up front instead of failing later.
    saveBlockedReason(): string {
      if (this.srv == undefined) return this.$t('error.invalidData')
      if (isBlankIdentity(this.srv.tag)) return this.$t('form.cannotSave.tagRequired')
      // OOMKiller/Profiler have no listen_port; only check when present.
      if (this.srv.listen_port != null && (this.srv.listen_port > 65535 || this.srv.listen_port < 1)) {
        return this.$t('form.cannotSave.portRange')
      }
      return ''
    },
    currentFieldHints(): Record<string, string> {
      return serviceFieldHintsForType(this.srv.type)
    },
    showServiceRecommendedPreset(): boolean {
      return this.$props.id == 0 && hasServiceRecommendedPreset(this.srv.type)
    },
  },
  watch: {
    visible(v) {
      if (v) {
        this.updateData(this.$props.id)
      }
    },
  },
  components: { EntityDrawer, FormSection, SettingInfo, Listen, InTLS, Derp, Ocm, Ccm, OomKiller, Profiler, SSMapi },
}
</script>
