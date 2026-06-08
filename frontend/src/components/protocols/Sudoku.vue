<template>
  <v-card subtitle="Sudoku">
    <v-row>
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.sudoku.key')" v-model="data.key"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-select
          clearable
          hide-details
          :label="$t('types.sudoku.aeadMethod')"
          :items="['chacha20-poly1305', 'aes-128-gcm', 'none']"
          v-model="data.aead_method">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="3">
        <v-select
          clearable
          hide-details
          :label="$t('types.sudoku.tableType')"
          :items="tableTypes"
          v-model="data.table_type">
        </v-select>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="6" sm="3">
        <v-text-field type="number" hide-details :label="$t('types.sudoku.paddingMin')" v-model.number="data.padding_min"></v-text-field>
      </v-col>
      <v-col cols="6" sm="3">
        <v-text-field type="number" hide-details :label="$t('types.sudoku.paddingMax')" v-model.number="data.padding_max"></v-text-field>
      </v-col>
      <v-col cols="6" sm="3" v-if="direction == 'in'">
        <v-text-field type="number" hide-details :label="$t('types.sudoku.handshakeTimeout')" v-model.number="data.handshake_timeout"></v-text-field>
      </v-col>
      <v-col cols="6" sm="3">
        <v-switch color="primary" hide-details :label="$t('types.sudoku.pureDownlink')" v-model="data.enable_pure_downlink"></v-switch>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.sudoku.customTable')" v-model="data.custom_table"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-combobox chips multiple clearable hide-details :label="$t('types.sudoku.customTables')"
          :model-value="data.custom_tables" @update:model-value="setCustomTables"></v-combobox>
      </v-col>
    </v-row>

    <!-- inbound http-mask -->
    <v-row v-if="direction == 'in'">
      <v-col cols="12" sm="4">
        <v-switch color="primary" hide-details :label="$t('types.sudoku.disableHttpMask')" v-model="data.disable_http_mask"></v-switch>
      </v-col>
      <v-col cols="12" sm="4">
        <v-select clearable hide-details :label="$t('types.sudoku.httpMaskMode')" :items="maskModes" v-model="data.http_mask_mode"></v-select>
      </v-col>
      <v-col cols="12" sm="4">
        <v-text-field hide-details :label="$t('types.sudoku.pathRoot')" v-model="data.path_root"></v-text-field>
      </v-col>
      <v-col cols="12">
        <v-text-field hide-details :label="$t('types.sudoku.fallback')" v-model="data.fallback"></v-text-field>
      </v-col>
    </v-row>

    <!-- outbound http-mask -->
    <v-card v-if="direction == 'out'" border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.sudoku.httpMask') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="4">
            <v-switch color="primary" hide-details :label="$t('enable')" v-model="httpMask.enabled"></v-switch>
          </v-col>
          <v-col cols="12" sm="4">
            <v-select clearable hide-details :label="$t('types.sudoku.httpMaskMode')" :items="maskModes" v-model="httpMask.mode"></v-select>
          </v-col>
          <v-col cols="12" sm="4">
            <v-select clearable hide-details :label="$t('types.sudoku.multiplex')" :items="['off', 'auto', 'on']" v-model="httpMask.multiplex"></v-select>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.sudoku.host')" v-model="httpMask.host"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.sudoku.pathRoot')" v-model="httpMask.path_root"></v-text-field>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>
  </v-card>
</template>

<script lang="ts">
import { RECOMMENDED } from '@/types/recommended'

export default {
  props: {
    direction: { type: String },
    data: { type: Object, required: true },
  },
  data() {
    return {
      tableTypes: ['prefer_ascii', 'prefer_entropy', 'up_ascii_down_entropy', 'up_entropy_down_ascii'],
      maskModes: ['legacy', 'stream', 'poll', 'auto', 'ws'],
    }
  },
  created() {
    // http_mask is an OUTBOUND-only field; do not inject it into inbound configs.
    if (this.direction == 'out' && !this.$props.data.http_mask) this.$props.data.http_mask = {}
  },
  mounted() {
    this.$props.data.aead_method ??= RECOMMENDED.sudokuAead
    this.$props.data.padding_min ??= RECOMMENDED.sudokuPaddingMin
    this.$props.data.padding_max ??= RECOMMENDED.sudokuPaddingMax
    // handshake_timeout exists only on the inbound Sudoku type.
    if (this.direction == 'in') this.$props.data.handshake_timeout ??= RECOMMENDED.sudokuHandshakeTimeout
  },
  computed: {
    httpMask(): any {
      return this.$props.data.http_mask
    },
  },
  methods: {
    setCustomTables(v: string[]) {
      if (v && v.length > 0) {
        this.$props.data.custom_tables = v
      } else {
        delete this.$props.data.custom_tables
      }
    },
  },
}
</script>
