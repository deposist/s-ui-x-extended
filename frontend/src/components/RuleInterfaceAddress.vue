<template>
  <v-row>
    <v-col cols="12" sm="6" md="4">
      <v-select
        v-model="addressOption"
        :items="availableAddressKeys"
        :label="$t('rule.interfaceAddr')"
        hide-details
        @update:model-value="updateAddressOption"
      />
    </v-col>
    <v-col cols="12" v-if="addressOption == 'interface_address'">
      <v-textarea
        v-model="interfaceAddressText"
        :label="$t('rule.interfaceAddressMap')"
        placeholder="eth0=192.0.2.0/24,2001:db8::/32"
        rows="4"
        no-resize
        hide-details
      />
    </v-col>
    <template v-if="addressOption == 'network_interface_address'">
      <v-col cols="12" sm="6" v-for="item in interfaceTypes" :key="item">
        <v-textarea
          :model-value="networkInterfaceText(item)"
          :label="item"
          rows="2"
          no-resize
          hide-details
          @update:model-value="setNetworkInterfaceText(item, $event)"
        />
      </v-col>
    </template>
    <v-col cols="12" v-if="addressOption == 'default_interface_address'">
      <v-textarea
        v-model="defaultInterfaceText"
        :label="$t('rule.defaultInterfaceAddr')"
        rows="4"
        no-resize
        hide-details
      />
    </v-col>
  </v-row>
</template>

<script lang="ts">
type AddressMap = Record<string, string[]>

const splitList = (value: string): string[] => value
  .split(/[\n,]/)
  .map((item) => item.trim())
  .filter((item) => item.length > 0)

const normalizeMap = (value: unknown): AddressMap => {
  if (!value || Array.isArray(value) || typeof value !== 'object') return {}
  return Object.entries(value as Record<string, unknown>).reduce((result, [key, rawValue]) => {
    if (Array.isArray(rawValue)) {
      const addresses = rawValue.map((item) => String(item).trim()).filter((item) => item.length > 0)
      if (addresses.length > 0) result[key] = addresses
    }
    return result
  }, {} as AddressMap)
}

export default {
  props: {
    rule: { type: Object, required: true },
    includeInterfaceAddress: { type: Boolean, default: true },
  },
  data() {
    return {
      addressOption: 'interface_address',
      interfaceTypes: ['wifi', 'cellular', 'ethernet', 'other'],
    }
  },
  computed: {
    availableAddressKeys(): string[] {
      const keys = ['network_interface_address', 'default_interface_address']
      return this.includeInterfaceAddress ? ['interface_address', ...keys] : keys
    },
    interfaceAddressText: {
      get(): string {
        const value = this.$props.rule.interface_address
        if (Array.isArray(value)) return value.join('\n')
        const addressMap = normalizeMap(value)
        return Object.entries(addressMap).map(([name, addresses]) => `${name}=${addresses.join(',')}`).join('\n')
      },
      set(value: string) {
        const addressMap: AddressMap = {}
        for (const line of value.split('\n')) {
          const trimmedLine = line.trim()
          if (!trimmedLine.includes('=')) continue
          const separatorIndex = trimmedLine.indexOf('=')
          const name = trimmedLine.slice(0, separatorIndex).trim()
          const addresses = splitList(trimmedLine.slice(separatorIndex + 1))
          if (name.length > 0 && addresses.length > 0) addressMap[name] = addresses
        }
        this.$props.rule.interface_address = addressMap
      },
    },
    defaultInterfaceText: {
      get(): string { return this.$props.rule.default_interface_address?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.default_interface_address = splitList(value) },
    },
  },
  methods: {
    updateAddressOption(option: string) {
      this.availableAddressKeys.forEach((key) => delete this.$props.rule[key])
      if (option == 'interface_address') this.$props.rule.interface_address = {}
      else if (option == 'network_interface_address') this.$props.rule.network_interface_address = {}
      else this.$props.rule.default_interface_address = []
    },
    networkInterfaceText(type: string): string {
      const addressMap = normalizeMap(this.$props.rule.network_interface_address)
      return addressMap[type]?.join('\n') ?? ''
    },
    setNetworkInterfaceText(type: string, value: string) {
      const addressMap = normalizeMap(this.$props.rule.network_interface_address)
      const addresses = splitList(value)
      if (addresses.length > 0) addressMap[type] = addresses
      else delete addressMap[type]
      this.$props.rule.network_interface_address = addressMap
    },
    syncAddressOption() {
      if (Array.isArray(this.$props.rule.interface_address)) {
        this.$props.rule.default_interface_address = this.$props.rule.default_interface_address ?? [...this.$props.rule.interface_address]
        delete this.$props.rule.interface_address
      }
      if (Array.isArray(this.$props.rule.network_interface_address)) {
        this.$props.rule.default_interface_address = this.$props.rule.default_interface_address ?? [...this.$props.rule.network_interface_address]
        delete this.$props.rule.network_interface_address
      }
      const selected = this.availableAddressKeys.find((key) => this.$props.rule[key] !== undefined)
      this.addressOption = selected ?? this.availableAddressKeys[0]
    },
  },
  mounted() {
    this.syncAddressOption()
  },
  watch: {
    rule() {
      this.syncAddressOption()
    },
  },
}
</script>
