<template>
  <v-card style="background-color: inherit; margin: 4px; padding: 8px;" class="border">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="logical" color="primary" :label="$t('rule.logical')" hide-details />
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="rule.invert" color="primary" :label="$t('rule.invert')" hide-details />
      </v-col>
      <v-spacer />
      <v-col cols="auto" v-if="deleteable">
        <v-btn icon="mdi-delete" color="warning" variant="text" @click="$emit('delete')" />
      </v-col>
    </v-row>

    <template v-if="logical">
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-select v-model="rule.mode" :items="['and', 'or']" :label="$t('rule.mode')" hide-details />
        </v-col>
        <v-spacer />
        <v-col cols="auto">
          <v-btn color="primary" variant="tonal" @click="addNestedRule">{{ $t('actions.add') + ' ' + $t('objects.rule') }}</v-btn>
        </v-col>
      </v-row>
      <HeadlessRule
        v-for="(item, index) in rule.rules"
        :key="index"
        :rule="item"
        deleteable
        @delete="rule.rules.splice(index, 1)"
      />
    </template>

    <template v-else>
      <v-row>
        <v-col cols="12" sm="6" md="4" v-if="optionQueryType">
          <v-combobox v-model="rule.query_type" :items="queryTypes" :label="$t('dns.rule.queryType')" multiple chips hide-details />
        </v-col>
        <v-col cols="12" sm="6" md="4" v-if="optionNetwork">
          <v-select v-model="rule.network" :items="['tcp', 'udp']" :label="$t('network')" multiple chips hide-details />
        </v-col>
      </v-row>
      <v-row v-if="optionDomain">
        <v-col cols="12" sm="6" md="4">
          <v-select v-model="domainOption" :items="domainKeys" hide-details @update:model-value="updateDomainOption($event)" />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.domain != undefined">
          <v-textarea v-model="domain" :label="$t('rule.domain')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.domain_suffix != undefined">
          <v-textarea v-model="domain_suffix" :label="$t('rule.domainSufix')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.domain_keyword != undefined">
          <v-textarea v-model="domain_keyword" :label="$t('rule.domainKw')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.domain_regex != undefined">
          <v-textarea v-model="domain_regex" :label="$t('rule.domainRgx')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.ip_cidr != undefined">
          <v-textarea v-model="ip_cidr" :label="$t('rule.ip')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.source_ip_cidr != undefined">
          <v-textarea v-model="source_ip_cidr" :label="$t('rule.srcCidr')" rows="3" no-resize hide-details />
        </v-col>
      </v-row>
      <v-row v-if="optionPort">
        <v-col cols="12" sm="6" md="4">
          <v-select v-model="portOption" :items="portKeys" hide-details @update:model-value="updatePortOption($event)" />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.port != undefined">
          <v-textarea v-model="port" :label="$t('rule.port')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.port_range != undefined">
          <v-textarea v-model="port_range" :label="$t('rule.portRange')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.source_port != undefined">
          <v-textarea v-model="source_port" :label="$t('rule.srcPort')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.source_port_range != undefined">
          <v-textarea v-model="source_port_range" :label="$t('rule.srcPortRange')" rows="3" no-resize hide-details />
        </v-col>
      </v-row>
      <v-row v-if="optionProcess">
        <v-col cols="12" sm="6" md="4">
          <v-select v-model="processOption" :items="processKeys" hide-details @update:model-value="updateProcessOption($event)" />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.process_name != undefined">
          <v-textarea v-model="process_name" :label="$t('rule.processName')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.process_path != undefined">
          <v-textarea v-model="process_path" :label="$t('rule.processPath')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.process_path_regex != undefined">
          <v-textarea v-model="process_path_regex" :label="$t('rule.processPathRegex')" rows="3" no-resize hide-details />
        </v-col>
        <v-col cols="12" sm="6" v-if="rule.package_name != undefined">
          <v-textarea v-model="package_name" :label="$t('rule.packageName')" rows="3" no-resize hide-details />
        </v-col>
      </v-row>
      <RuleNetworkState v-if="optionNetworkState" :rule="rule" />
      <RuleInterfaceAddress v-if="optionInterface" :rule="rule" :include-interface-address="false" />
      <v-card-actions>
        <v-spacer />
        <v-menu v-model="menu" :close-on-content-click="false" location="start">
          <template v-slot:activator="{ props }">
            <v-btn v-bind="props" variant="tonal">{{ $t('rule.options') }}</v-btn>
          </template>
          <v-card>
            <v-list>
              <v-list-item><v-switch v-model="optionQueryType" color="primary" :label="$t('dns.rule.queryType')" hide-details /></v-list-item>
              <v-list-item><v-switch v-model="optionNetwork" color="primary" :label="$t('network')" hide-details /></v-list-item>
              <v-list-item><v-switch v-model="optionDomain" color="primary" :label="$t('rule.domainRules')" hide-details /></v-list-item>
              <v-list-item><v-switch v-model="optionPort" color="primary" :label="$t('in.port')" hide-details /></v-list-item>
              <v-list-item><v-switch v-model="optionProcess" color="primary" :label="$t('rule.process')" hide-details /></v-list-item>
              <v-list-item><v-switch v-model="optionNetworkState" color="primary" :label="$t('rule.networkState')" hide-details /></v-list-item>
              <v-list-item><v-switch v-model="optionInterface" color="primary" :label="$t('rule.interfaceAddr')" hide-details /></v-list-item>
            </v-list>
          </v-card>
        </v-menu>
      </v-card-actions>
    </template>
  </v-card>
</template>

<script lang="ts">
import RuleInterfaceAddress from '@/components/RuleInterfaceAddress.vue'
import RuleNetworkState from '@/components/RuleNetworkState.vue'

const splitStringList = (value: string): string[] => value
  .split(/[\n,]/)
  .map((item) => item.trim())
  .filter((item) => item.length > 0)

const splitNumberList = (value: string): number[] => splitStringList(value)
  .map((item) => parseInt(item, 10))
  .filter((item) => !isNaN(item))

export default {
  name: 'HeadlessRule',
  components: { RuleInterfaceAddress, RuleNetworkState },
  props: ['rule', 'deleteable'],
  emits: ['delete'],
  data() {
    return {
      menu: false,
      domainKeys: ['domain', 'domain_suffix', 'domain_keyword', 'domain_regex', 'ip_cidr', 'source_ip_cidr'],
      portKeys: ['port', 'port_range', 'source_port', 'source_port_range'],
      processKeys: ['process_name', 'process_path', 'process_path_regex', 'package_name'],
      domainOption: 'domain',
      portOption: 'port',
      processOption: 'process_name',
      queryTypes: ['A', 'AAAA', 'CNAME', 'MX', 'NS', 'PTR', 'HTTPS', 'SVCB', 'TXT'],
    }
  },
  computed: {
    logical: {
      get(): boolean { return this.$props.rule.type == 'logical' },
      set(value: boolean) {
        if (value) {
          Object.keys(this.$props.rule).forEach((key) => delete this.$props.rule[key])
          this.$props.rule.type = 'logical'
          this.$props.rule.mode = 'and'
          this.$props.rule.rules = [{}]
        } else {
          Object.keys(this.$props.rule).forEach((key) => delete this.$props.rule[key])
        }
      },
    },
    optionQueryType: {
      get() { return this.$props.rule.query_type != undefined },
      set(value: boolean) { this.$props.rule.query_type = value ? [] : undefined },
    },
    optionNetwork: {
      get() { return this.$props.rule.network != undefined },
      set(value: boolean) { this.$props.rule.network = value ? [] : undefined },
    },
    optionDomain: {
      get() { return this.domainKeys.some((key) => this.$props.rule[key] != undefined) },
      set(value: boolean) {
        if (value) this.$props.rule.domain = []
        else this.domainKeys.forEach((key) => delete this.$props.rule[key])
        this.domainOption = 'domain'
      },
    },
    optionPort: {
      get() { return this.portKeys.some((key) => this.$props.rule[key] != undefined) },
      set(value: boolean) {
        if (value) this.$props.rule.port = []
        else this.portKeys.forEach((key) => delete this.$props.rule[key])
        this.portOption = 'port'
      },
    },
    optionProcess: {
      get() { return this.processKeys.some((key) => this.$props.rule[key] != undefined) },
      set(value: boolean) {
        if (value) this.$props.rule.process_name = []
        else this.processKeys.forEach((key) => delete this.$props.rule[key])
        this.processOption = 'process_name'
      },
    },
    optionNetworkState: {
      get() {
        return this.$props.rule.network_type != undefined ||
          this.$props.rule.network_is_expensive != undefined ||
          this.$props.rule.network_is_constrained != undefined ||
          this.$props.rule.wifi_ssid != undefined ||
          this.$props.rule.wifi_bssid != undefined
      },
      set(value: boolean) {
        if (value) {
          this.$props.rule.network_type = []
          this.$props.rule.network_is_expensive = false
          this.$props.rule.network_is_constrained = false
          this.$props.rule.wifi_ssid = []
          this.$props.rule.wifi_bssid = []
        } else {
          delete this.$props.rule.network_type
          delete this.$props.rule.network_is_expensive
          delete this.$props.rule.network_is_constrained
          delete this.$props.rule.wifi_ssid
          delete this.$props.rule.wifi_bssid
        }
      },
    },
    optionInterface: {
      get() { return ['network_interface_address', 'default_interface_address'].some((key) => this.$props.rule[key] != undefined) },
      set(value: boolean) {
        if (value) this.$props.rule.network_interface_address = {}
        else ['network_interface_address', 'default_interface_address'].forEach((key) => delete this.$props.rule[key])
      },
    },
    domain: {
      get(): string { return this.$props.rule.domain?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.domain = splitStringList(value) },
    },
    domain_suffix: {
      get(): string { return this.$props.rule.domain_suffix?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.domain_suffix = splitStringList(value) },
    },
    domain_keyword: {
      get(): string { return this.$props.rule.domain_keyword?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.domain_keyword = splitStringList(value) },
    },
    domain_regex: {
      get(): string { return this.$props.rule.domain_regex?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.domain_regex = splitStringList(value) },
    },
    ip_cidr: {
      get(): string { return this.$props.rule.ip_cidr?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.ip_cidr = splitStringList(value) },
    },
    source_ip_cidr: {
      get(): string { return this.$props.rule.source_ip_cidr?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.source_ip_cidr = splitStringList(value) },
    },
    port: {
      get(): string { return this.$props.rule.port?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.port = splitNumberList(value) },
    },
    port_range: {
      get(): string { return this.$props.rule.port_range?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.port_range = splitStringList(value) },
    },
    source_port: {
      get(): string { return this.$props.rule.source_port?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.source_port = splitNumberList(value) },
    },
    source_port_range: {
      get(): string { return this.$props.rule.source_port_range?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.source_port_range = splitStringList(value) },
    },
    process_name: {
      get(): string { return this.$props.rule.process_name?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.process_name = splitStringList(value) },
    },
    process_path: {
      get(): string { return this.$props.rule.process_path?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.process_path = splitStringList(value) },
    },
    process_path_regex: {
      get(): string { return this.$props.rule.process_path_regex?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.process_path_regex = splitStringList(value) },
    },
    package_name: {
      get(): string { return this.$props.rule.package_name?.join('\n') ?? '' },
      set(value: string) { this.$props.rule.package_name = splitStringList(value) },
    },
  },
  methods: {
    addNestedRule() {
      if (!this.$props.rule.rules) this.$props.rule.rules = []
      this.$props.rule.rules.push({})
    },
    updateDomainOption(option: string) {
      this.domainKeys.forEach((key) => delete this.$props.rule[key])
      this.$props.rule[option] = []
    },
    updatePortOption(option: string) {
      this.portKeys.forEach((key) => delete this.$props.rule[key])
      this.$props.rule[option] = []
    },
    updateProcessOption(option: string) {
      this.processKeys.forEach((key) => delete this.$props.rule[key])
      this.$props.rule[option] = []
    },
  },
  mounted() {
    const keys = Object.keys(this.$props.rule)
    this.domainOption = this.domainKeys.find((key) => keys.includes(key)) ?? 'domain'
    this.portOption = this.portKeys.find((key) => keys.includes(key)) ?? 'port'
    this.processOption = this.processKeys.find((key) => keys.includes(key)) ?? 'process_name'
  },
}
</script>
