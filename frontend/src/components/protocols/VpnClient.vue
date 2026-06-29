<template>
  <v-card subtitle="VPN Client">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          v-model="data.address"
          :label="$t('types.vpn.address')"
          :placeholder="'10.0.0.2'"
          hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="address" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field
          v-model="data.key"
          :label="$t('types.vpn.key')"
          hide-details
          append-icon="mdi-refresh"
          @click:append="data.key = genKey()">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="vpn_user" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>

    <v-card :subtitle="$t('types.vpn.outbound')" class="mt-2">
      <v-card-text>
        <v-textarea
          v-model="outboundJson"
          :error-messages="outboundError ? [outboundError] : []"
          :hint="$t('types.vpn.outboundHint')"
          persistent-hint
          variant="outlined"
          auto-grow
          rows="8"
          :style="{ 'font-family': 'monospace' }">
        </v-textarea>
      </v-card-text>
    </v-card>
  </v-card>
</template>

<script lang="ts">
import RandomUtil from '@/plugins/randomUtil'
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      outboundError: "",
    }
  },
  computed: {
    outboundJson: {
      get(): string {
        const outbound = typeof this.$props.data.outbound == 'object' && !Array.isArray(this.$props.data.outbound) && this.$props.data.outbound != null
          ? this.$props.data.outbound
          : {}
        return JSON.stringify(outbound, null, 2)
      },
      set(v: string) {
        if (v.trim().length === 0) {
          this.$props.data.outbound = {}
          this.outboundError = ""
          return
        }
        try {
          const parsed = JSON.parse(v)
          if (typeof parsed != 'object' || Array.isArray(parsed) || parsed == null) {
            this.outboundError = this.$t('types.vpn.outboundObjectError')
            return
          }
          this.$props.data.outbound = parsed
          this.outboundError = ""
        } catch (e: any) {
          this.outboundError = e.message
        }
      }
    },
  },
  methods: {
    genKey(): string {
      return RandomUtil.randomUUID()
    },
  },
  components: { FieldHint },
}
</script>
