<template>
  <div>
    <v-card-title>
      {{ $t('types.usbip.devices') }}
      <v-chip color="primary" density="compact" variant="elevated" @click="addDevice"><v-icon icon="mdi-plus" /></v-chip>
    </v-card-title>
    <v-card
      v-for="(device, index) in devices"
      :key="index"
      class="border"
      style="margin: 4px; padding: 8px;"
      rounded="xl"
    >
      <v-row>
        <v-col cols="auto" align-self="center">
          <v-icon @click="delDevice(index)" color="error" icon="mdi-delete" />
        </v-col>
        <v-col cols="12" sm="6" md="3">
          <v-text-field :label="$t('types.usbip.busId')" hide-details v-model="device.bus_id">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="devices" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="3">
          <v-text-field
            :label="$t('types.usbip.vendorId')"
            hide-details
            :model-value="hexId(device.vendor_id)"
            @update:model-value="setVendorId(device, $event)">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="vendor_id" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="3">
          <v-text-field
            :label="$t('types.usbip.productId')"
            hide-details
            :model-value="hexId(device.product_id)"
            @update:model-value="setProductId(device, $event)">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="product_id" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="3">
          <v-text-field :label="$t('types.usbip.serial')" hide-details v-model="device.serial">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="serial" />
            </template>
          </v-text-field>
        </v-col>
      </v-row>
    </v-card>
  </div>
</template>

<script lang="ts">
import { USBIPDeviceMatch } from '@/types/services'
import FieldHint from '@/components/FieldHint.vue'

// USB/IP device selectors are numeric in the core (vendor_id/product_id are
// uint16), but operators paste them as hex from lsusb, so the fields are edited as
// text and converted on the way in/out.
export default {
  props: {
    devices: { type: Array as () => USBIPDeviceMatch[], default: () => [] },
    fieldHints: { type: Object, default: () => ({}) },
  },
  methods: {
    addDevice() {
      this.$props.devices.push({ bus_id: '', vendor_id: 0, product_id: 0, serial: '' })
    },
    delDevice(index: number) {
      this.$props.devices.splice(index, 1)
    },
    hexId(value?: number) {
      return value == undefined || value === 0 ? '' : '0x' + value.toString(16).padStart(4, '0')
    },
    parseId(value: string) {
      const text = (value || '').trim()
      if (text === '') return undefined
      const parsed = text.startsWith('0x') || text.startsWith('0X') ? parseInt(text.slice(2), 16) : parseInt(text, 10)
      if (!Number.isFinite(parsed) || parsed < 0) return undefined
      return parsed > 0xffff ? undefined : parsed
    },
    setVendorId(device: USBIPDeviceMatch, value: string) {
      const parsed = this.parseId(value)
      if (parsed == undefined) delete device.vendor_id
      else device.vendor_id = parsed
    },
    setProductId(device: USBIPDeviceMatch, value: string) {
      const parsed = this.parseId(value)
      if (parsed == undefined) delete device.product_id
      else device.product_id = parsed
    },
  },
  components: { FieldHint },
}
</script>