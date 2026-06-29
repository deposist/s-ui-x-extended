<template>
  <v-card>
    <v-card-subtitle v-if="direction != 'out_json'">AnyTls</v-card-subtitle>
    <v-row v-if="direction == 'in'">
      <v-col cols="12" sm="8">
        <v-textarea
        label="Padding scheme"
        auto-grow
        hide-details
        v-model="padding_scheme">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="padding_scheme" />
          </template>
        </v-textarea>
      </v-col>
    </v-row>
    <v-row v-else>
      <v-col cols="12" sm="8" v-if="direction == 'out'">
        <v-text-field
        :label="$t('types.pw')"
        hide-details
        v-model="data.password">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="password" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
        :label="$t('types.anytls.idleInterval')"
        type="number"
        min="0"
        hide-details
        :suffix="$t('date.s')"
        v-model.number="idleInterval">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="idle_session_check_interval" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
        :label="$t('types.anytls.idleTimeout')"
        type="number"
        min="0"
        hide-details
        :suffix="$t('date.s')"
        v-model.number="idleTimeout">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="idle_session_timeout" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
        :label="$t('types.anytls.minIdle')"
        type="number"
        min="0"
        hide-details
        v-model.number="minIdle">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="min_idle_session" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
    <InboundAdvanced :data="data" :field-hints="fieldHints" />
  </v-card>
</template>

<script lang="ts">
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: ['data', 'direction', 'fieldHints'],
  components: {InboundAdvanced, FieldHint},
  data() {
    return {}
  },
  computed: {
    padding_scheme: {
      get() { return this.data.padding_scheme?.length > 0 ? this.data.padding_scheme.join("\n") : '' },
      set(v:string) { this.data.padding_scheme = v.length > 0 ? v.split("\n") : undefined }
    },
    idleInterval: {
      get() { return this.data.idle_session_check_interval?.length > 0 ? parseInt(this.data.idle_session_check_interval.replace('s','')) : 30 },
      set(v:number) { this.data.idle_session_check_interval = v && v >= 0 ? `${v}s` : undefined }
    },
    idleTimeout: {
      get() { return this.data.idle_session_timeout?.length > 0 ? parseInt(this.data.idle_session_timeout.replace('s','')) : 30 },
      set(v:number) { this.data.idle_session_timeout = v && v >= 0 ? `${v}s` : undefined }
    },
    minIdle: {
      get() { return this.data.min_idle_session != undefined ? this.data.min_idle_session : 0 },
      set(v:number) { this.data.min_idle_session = v>0 ? v : undefined }
    }
  }
}
</script>