<template>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-select
        v-model="mode"
        hide-details
        :label="label || $t('dial.domainResolver')"
        :items="modes">
      </v-select>
    </v-col>
    <template v-if="mode == 'tag'">
      <v-col cols="12" sm="6" md="4" v-if="dnsTags.length > 0">
        <v-select
          v-model="tagValue"
          hide-details
          :label="$t('dns.server')"
          :items="dnsTags">
        </v-select>
      </v-col>
      <v-col cols="12" sm="8" v-else>
        <v-alert density="compact" type="warning" variant="tonal">
          {{ $t('singbox.noDnsTag') }}
        </v-alert>
      </v-col>
    </template>
    <template v-if="mode == 'advanced'">
      <v-col cols="12" sm="6" md="4">
        <v-combobox
          v-model="advanced.server"
          hide-details
          :label="$t('dns.server')"
          :items="dnsTags">
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          v-model="advanced.strategy"
          hide-details
          clearable
          @click:clear="delete advanced.strategy"
          :label="$t('rule.strategy')"
          :items="strategies">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="advanced.disable_cache" color="primary" :label="$t('dns.disableCache')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          v-model.number="rewriteTtl"
          type="number"
          min="0"
          hide-details
          :label="$t('singbox.rewriteTtl')">
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          v-model="advanced.client_subnet"
          hide-details
          clearable
          @click:clear="delete advanced.client_subnet"
          :label="$t('dns.rule.action.clientSubnet')">
        </v-text-field>
      </v-col>
    </template>
  </v-row>
</template>

<script lang="ts">
import Data from '@/store/modules/data'
import { DomainResolveOptions } from '@/types/dial'

type ResolverObject = Exclude<DomainResolveOptions, string>

export default {
  props: ['data', 'field', 'label'],
  data() {
    return {
      strategies: ['', 'prefer_ipv4', 'prefer_ipv6', 'ipv4_only', 'ipv6_only']
    }
  },
  computed: {
    modes() {
      return [
        { title: this.$t('singbox.off'), value: 'off' },
        { title: this.$t('singbox.recommended'), value: 'tag' },
        { title: this.$t('singbox.custom'), value: 'advanced' },
      ]
    },
    dnsTags(): string[] { return Data().config.dns?.servers?.map((d:any) => d.tag).filter((tag:string) => tag?.length > 0) ?? [] },
    value(): DomainResolveOptions | undefined {
      return this.$props.data?.[this.$props.field]
    },
    mode: {
      get(): string {
        if (this.value == undefined) return 'off'
        return typeof this.value == 'string' ? 'tag' : 'advanced'
      },
      set(v:string) {
        if (v == 'off') {
          delete this.$props.data[this.$props.field]
        } else if (v == 'tag') {
          if (this.dnsTags.length > 0) this.$props.data[this.$props.field] = typeof this.value == 'string' ? this.value : this.dnsTags[0]
          else delete this.$props.data[this.$props.field]
        } else {
          const server = typeof this.value == 'string' ? this.value : this.value?.server
          this.$props.data[this.$props.field] = { server: server || this.dnsTags[0] || '' }
        }
      }
    },
    tagValue: {
      get(): string { return typeof this.value == 'string' ? this.value : this.dnsTags[0] ?? '' },
      set(v:string) { v.length > 0 ? this.$props.data[this.$props.field] = v : delete this.$props.data[this.$props.field] }
    },
    advanced(): ResolverObject {
      if (typeof this.value == 'object' && this.value != null) return this.value
      this.$props.data[this.$props.field] = { server: this.dnsTags[0] || '' }
      return this.$props.data[this.$props.field]
    },
    rewriteTtl: {
      get(): number | undefined { return this.advanced.rewrite_ttl },
      set(v:number | undefined) {
        if (typeof v == 'number' && !isNaN(v) && v > 0) this.advanced.rewrite_ttl = v
        else delete this.advanced.rewrite_ttl
      }
    }
  }
}
</script>
