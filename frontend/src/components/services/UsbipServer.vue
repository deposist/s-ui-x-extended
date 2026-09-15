<template>
  <v-card subtitle="USB/IP Server">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          :label="$t('types.usbip.provider')"
          hide-details
          :items="providers"
          v-model="data.provider">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="provider" />
          </template>
        </v-select>
      </v-col>
    </v-row>
    <v-card-text v-if="data.provider != 'dynamic'" class="px-0">
      <v-alert v-if="(data.devices || []).length === 0" type="info" variant="tonal" density="compact">
        {{ $t('types.usbip.noDevices') }}
      </v-alert>
      <DeviceList :devices="data.devices" :field-hints="fieldHints" />
    </v-card-text>
  </v-card>
</template>

<script lang="ts">
import { USBIPServer } from '@/types/services'
import FieldHint from '@/components/FieldHint.vue'
import DeviceList from '@/components/services/usbip/DeviceList.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      // An empty provider means "default" to the core, so both are offered.
      providers: [
        { title: 'default', value: 'default' },
        { title: 'dynamic', value: 'dynamic' },
      ],
    }
  },
  computed: {
    typed: function(): USBIPServer { return <USBIPServer>this.$props.data },
  },
  components: { FieldHint, DeviceList },
}
</script>