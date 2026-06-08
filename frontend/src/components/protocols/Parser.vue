<template>
  <v-card subtitle="Parser">
    <v-row>
      <v-col cols="12">
        <v-text-field
          v-model="data.link"
          :label="$t('types.parser.link')"
          :hint="$t('types.parser.linkHint')"
          persistent-hint
          placeholder="vless://...">
        </v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data'],
}
</script>
