<template>
  <v-card subtitle="Profiler">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.listen" :label="$t('objects.listen')" placeholder="127.0.0.1:8964" hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="profiler_listen" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-combobox v-model="data.read_timeout" :items="durationPresets" :label="$t('types.profiler.readTimeout')" placeholder="5s" hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="read_timeout" />
          </template>
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-combobox v-model="data.write_timeout" :items="durationPresets" :label="$t('types.profiler.writeTimeout')" placeholder="5s" hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="write_timeout" />
          </template>
        </v-combobox>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import { durationPresets } from '@/types/recommended'
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      durationPresets,
    }
  },
  components: { FieldHint },
}
</script>
