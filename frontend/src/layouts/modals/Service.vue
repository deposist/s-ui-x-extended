<template>
  <v-dialog transition="dialog-bottom-transition" width="800">
    <v-card class="rounded-lg">
      <v-card-title>
        {{ $t('actions.' + title) + " " + $t('objects.service') }}
      </v-card-title>
      <v-divider></v-divider>
      <v-card-text style="padding: 0 16px; overflow-y: scroll;">
        <v-row>
          <v-col cols="12" sm="6" md="4">
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
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="srv.tag" :label="$t('objects.tag')" hide-details>
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
        <InTLS v-if="HasTls.includes(srv.type)"  :inbound="srv" :tlsConfigs="tlsConfigs" :tls_id="srv.tls_id" :field-hints="currentFieldHints" />
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
import SettingInfo from '@/components/SettingInfo.vue'
import { applyServiceRecommendedValues, hasServiceRecommendedPreset, serviceFieldHintsForType } from '@/utils/defaultRecommendations'
export default {
  props: ['visible', 'data', 'id', 'inTags', 'tsTags', 'ssTags', 'tlsConfigs'],
  emits: ['close'],
  data() {
    return {
      srv: createSrv("derp",{ "tag": "" }),
      title: "add",
      tab: "t1",
      loading: false,
      srvTypes: SrvTypes,
      HasTls: [SrvTypes.DERP, SrvTypes.SSMAPI, SrvTypes.OCM, SrvTypes.CCM],
      NoListen: [SrvTypes.OOMKiller, SrvTypes.Profiler],
    }
  },
  methods: {
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
      this.tab = "t1"
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
      const prevConfig = this.NoListen.includes(this.srv.type)
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
  components: { SettingInfo, Listen, InTLS, Derp, Ocm, Ccm, OomKiller, Profiler, SSMapi },
}
</script>
