<template>
  <v-card subtitle="SSH">
    <v-row>
      <v-col cols="12" sm="6">
        <v-combobox hide-details :label="$t('types.ssh.serverVersion')" :items="sshVersions" v-model="data.server_version">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="server_version" />
          </template>
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field type="number" hide-details :label="$t('types.ssh.maxAuthTries')" v-model.number="data.max_auth_tries">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="max_auth_tries" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12">
        <v-combobox chips multiple clearable hide-details :label="$t('types.ssh.hostKey')"
          :model-value="data.host_key" @update:model-value="setList('host_key', $event)"></v-combobox>
      </v-col>
      <v-col cols="12">
        <v-combobox chips multiple clearable hide-details :label="$t('types.ssh.hostKeyPath')"
          :model-value="data.host_key_path" @update:model-value="setList('host_key_path', $event)"></v-combobox>
      </v-col>
    </v-row>
    <InboundAdvanced :data="data" :field-hints="fieldHints" />
  </v-card>
</template>

<script lang="ts">
import { sshVersions } from '@/types/recommended'
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: ['data', 'fieldHints'],
  components: {InboundAdvanced, FieldHint},
  data() {
    return {
      sshVersions,
    }
  },
  methods: {
    setList(key: string, v: string[]) {
      if (v && v.length > 0) {
        this.$props.data[key] = v
      } else {
        delete this.$props.data[key]
      }
    },
  },
}
</script>
