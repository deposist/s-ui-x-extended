<template>
  <v-card subtitle="Fallback">
    <v-row>
      <v-col cols="12">
        <v-combobox
          v-model="data.outbounds"
          :items="tags"
          :label="$t('pages.outbounds')"
          multiple
          chips
          hide-details
        ></v-combobox>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data', 'tags'],
  data() {
    return {}
  },
}
</script>
