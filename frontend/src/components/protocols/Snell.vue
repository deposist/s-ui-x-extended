<template>
  <v-card subtitle="Snell">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.snell.version')"
          :items="[{ title: '4', value: 4 }, { title: '6', value: 6 }]"
          v-model.number="data.version">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          hide-details
          :label="$t('types.snell.psk')"
          v-model="data.psk">
          <template #append-inner>
            <v-btn icon="mdi-refresh" size="x-small" variant="text" :aria-label="$t('actions.generate')" @click="genPsk" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          clearable
          :label="$t('types.snell.network')"
          :items="['tcp', 'udp']"
          @click:clear="delete data.network"
          v-model="data.network">
        </v-select>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.snell.userkey')" v-model="data.userkey"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.snell.reuse')" v-model="data.reuse"></v-switch>
      </v-col>
    </v-row>
    <!-- version 4: obfs options -->
    <v-row v-if="data.version === 4">
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          clearable
          :label="$t('types.snell.obfsMode')"
          :items="['none', 'http', 'tls']"
          @click:clear="delete data.obfs_mode"
          v-model="data.obfs_mode">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="data.obfs_mode === 'http'">
        <v-text-field hide-details :label="$t('types.snell.obfsHost')" v-model="data.obfs_host"></v-text-field>
      </v-col>
    </v-row>
    <!-- version 6: mode -->
    <v-row v-if="data.version === 6">
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          clearable
          :label="$t('types.snell.mode')"
          :items="['default', 'unshaped', 'unsafe-raw']"
          @click:clear="delete data.mode"
          v-model="data.mode">
        </v-select>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import RandomUtil from '@/plugins/randomUtil'

export default {
  props: {
    data: { type: Object, required: true },
  },
  methods: {
    genPsk() {
      this.$props.data.psk = RandomUtil.randomShadowsocksPassword(32)
    },
  },
}
</script>
