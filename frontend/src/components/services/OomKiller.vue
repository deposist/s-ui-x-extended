<template>
  <v-card subtitle="OOM Killer">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.memory_limit" :label="$t('types.oom.memoryLimit')" hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="memory_limit" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.safety_margin" :label="$t('types.oom.safetyMargin')" hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="safety_margin" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.min_interval" :label="$t('types.oom.minInterval')" placeholder="5s" hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="min_interval" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.max_interval" :label="$t('types.oom.maxInterval')" placeholder="1m" hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="max_interval" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          v-model.number="data.checks_before_limit"
          :label="$t('types.oom.checksBeforeLimit')"
          type="number"
          min="1"
          hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="checks_before_limit" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  components: { FieldHint },
}
</script>
