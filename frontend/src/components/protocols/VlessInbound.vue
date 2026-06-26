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
    <InboundAdvanced :data="data" />
  </v-card>
</template>

<script lang="ts">
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'
export default {
  props: ['data'],
  components: {InboundAdvanced},
}
</script>
