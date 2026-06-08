<template>
  <v-card subtitle="VLESS">
    <v-row>
      <v-col cols="12">
        <v-text-field
          v-model="data.decryption"
          :label="$t('types.vless.decryption')"
          :hint="$t('types.vless.decryptionHint')"
          persistent-hint
          clearable
          @click:clear="data.decryption = undefined">
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
