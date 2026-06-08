<template>
  <v-card subtitle="SSH">
    <v-row>
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.ssh.serverVersion')" v-model="data.server_version"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field type="number" hide-details :label="$t('types.ssh.maxAuthTries')" v-model.number="data.max_auth_tries"></v-text-field>
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
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data'],
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
