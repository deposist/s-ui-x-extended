<template>
  <v-card subtitle="Bridge">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.wg.ifName')" v-model="data.interface"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.bridge.name')" v-model="data.bridge_name"></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" min="0" hide-details :label="$t('types.bridge.tableIndex')" v-model.number="data.iproute2_table_index"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" min="0" hide-details :label="$t('types.bridge.ruleIndex')" v-model.number="data.iproute2_rule_index"></v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: {
    data: { type: Object, required: true },
  },
}
</script>
