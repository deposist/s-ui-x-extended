<template>
  <v-card style="background-color: inherit;">
    <v-row>
      <v-col cols="12" v-if="optionInbound">
        <v-combobox
          v-model="rule.inbound"
          :items="inTags"
          :label="$t('pages.inbounds')"
          multiple
          chips
          hide-details
        ></v-combobox>
      </v-col>
      <v-col cols="12" v-if="optionClient">
        <v-combobox
          v-model="rule.auth_user"
          :items="clients"
          :label="$t('pages.clients')"
          multiple
          chips
          hide-details
        ></v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionIPver">
        <v-select
          hide-details
          :label="$t('rule.ipVer')"
          :items="[4,6]"
          v-model.number="rule.ip_version">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionQueryType">
        <v-combobox
          v-model="rule.query_type"
          :items="queryTypes"
          :label="$t('dns.rule.queryType')"
          multiple
          chips
          hide-details>
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionQueryType">
        <v-textarea v-model="query_client_subnet" :label="$t('dns.rule.queryClientSubnet')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionQueryType">
        <v-switch v-model="rule.query_dnssec" color="primary" :label="$t('dns.rule.queryDnssec')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionNetwork">
        <v-select
          hide-details
          multiple
          chips
          :label="$t('network')"
          :items="['tcp','udp']"
          v-model="rule.network">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" v-if="optionProtocol">
        <v-combobox
          v-model="rule.protocol"
          :items="['http','tls', 'quic', 'stun', 'dns']"
          :label="$t('protocol')"
          multiple
          chips
          hide-details
        ></v-combobox>
      </v-col>
    </v-row>
    <v-row v-if="optionDomain">
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :items="domainKeys"
          @update:model-value="updateDomainOption($event)"
          v-model="domainOption">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.domain != undefined">
        <v-text-field
        :label="$t('rule.domain') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="domain"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.domain_suffix != undefined">
        <v-text-field
        :label="$t('rule.domainSufix') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="domain_suffix"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.domain_keyword != undefined">
        <v-text-field
        :label="$t('rule.domainKw') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="domain_keyword"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.domain_regex != undefined">
        <v-text-field
        :label="$t('rule.domainRgx') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="domain_regex"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.ip_cidr != undefined">
        <v-text-field
        :label="$t('rule.ip') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="ip_cidr"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.ip_is_private != undefined">
        <v-switch v-model="rule.ip_is_private" color="primary" :label="$t('rule.privateIp')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.ip_accept_any != undefined">
        <v-switch v-model="rule.ip_accept_any" color="primary" :label="$t('dns.rule.ipAcceptAny')" hide-details></v-switch>
      </v-col>
    </v-row>
    <v-row v-if="optionPort">
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :items="portKeys"
          @update:model-value="updatePortOption($event)"
          v-model="portOption">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.port != undefined">
        <v-text-field
        :label="$t('rule.port') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="port"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.port_range != undefined">
        <v-text-field
        :label="$t('rule.portRange') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="port_range"></v-text-field>
      </v-col>
    </v-row>
    <v-row v-if="optionSrcIP">
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :items="srcIPKeys"
          @update:model-value="updateSrcIPOption($event)"
          v-model="srcIPOption">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.source_ip_cidr != undefined">
        <v-text-field
        :label="$t('rule.srcCidr') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="source_ip_cidr"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.source_ip_is_private != undefined">
        <v-switch v-model="rule.source_ip_is_private" color="primary" :label="$t('rule.srcPrivateIp')" hide-details></v-switch>
      </v-col>
    </v-row>
    <v-row v-if="optionSrcPort">
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :items="srcPortKeys"
          @update:model-value="updateSrcPortOption($event)"
          v-model="srcPortOption">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.source_port != undefined">
        <v-text-field
        :label="$t('rule.srcPort') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="source_port"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.source_port_range != undefined">
        <v-text-field
        :label="$t('rule.srcPortRange') + ' ' + $t('commaSeparated')"
        hide-details
        v-model="source_port_range"></v-text-field>
      </v-col>
    </v-row>
    <v-row v-if="optionRuleSet">
      <v-col cols="12" sm="6">
        <v-combobox
          v-model="rule.rule_set"
          :items="ruleSets"
          :label="$t('rule.ruleset')"
          multiple
          chips
          hide-details
        ></v-combobox>
      </v-col>
      <v-col cols="12" sm="6">
        <v-switch v-model="rule.rule_set_ip_cidr_match_source" color="primary" :label="$t('rule.rulesetMatchSrc')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6">
        <v-switch v-model="rule.rule_set_ip_cidr_accept_empty" color="primary" :label="$t('dns.rule.rulesetAcceptEmpty')" hide-details></v-switch>
      </v-col>
    </v-row>
    <v-row v-if="optionProcess">
      <v-col cols="12" sm="6" md="4">
        <v-select v-model="processOption" :items="processKeys" hide-details @update:model-value="updateProcessOption($event)" />
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.process_name != undefined">
        <v-textarea v-model="process_name" :label="$t('rule.processName')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.process_path != undefined">
        <v-textarea v-model="process_path" :label="$t('rule.processPath')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.process_path_regex != undefined">
        <v-textarea v-model="process_path_regex" :label="$t('rule.processPathRegex')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.package_name != undefined">
        <v-textarea v-model="package_name" :label="$t('rule.packageName')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.package_name_regex != undefined">
        <v-textarea v-model="package_name_regex" :label="$t('rule.packageNameRegex')" rows="2" no-resize hide-details density="compact" />
      </v-col>
    </v-row>
    <v-row v-if="optionDevice">
      <v-col cols="12" sm="6" v-if="rule.source_mac_address != undefined">
        <v-textarea v-model="source_mac_address" :label="$t('rule.srcMacAddress')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6" v-if="rule.source_hostname != undefined">
        <v-textarea v-model="source_hostname" :label="$t('rule.srcHostname')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6">
        <v-combobox
          v-model="rule.preferred_by"
          :items="[]"
          :label="$t('rule.preferredBy')"
          multiple
          chips
          clearable
          hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="preferred_by" />
          </template>
        </v-combobox>
      </v-col>
    </v-row>
    <v-row v-if="optionResponse">
      <v-col cols="12" sm="6" md="4">
        <div class="d-flex align-center ga-1">
          <v-switch v-model="matchResponseEnabled" color="primary" :label="$t('dns.rule.matchResponse')" hide-details></v-switch>
          <FieldHint :field-hints="fieldHints" field="match_response" />
        </div>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="matchResponseEnabled">
        <v-text-field v-model="matchResponseTag" :label="$t('objects.tag')" hide-details clearable />
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          v-model="rule.response_rcode"
          :items="responseRcodes"
          :label="$t('dns.rule.responseRcode')"
          clearable
          @click:clear="delete rule.response_rcode"
          hide-details />
      </v-col>
      <v-col cols="12" sm="6">
        <v-textarea v-model="response_answer" :label="$t('dns.rule.responseAnswer')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6">
        <v-textarea v-model="response_ns" :label="$t('dns.rule.responseNs')" rows="2" no-resize hide-details density="compact" />
      </v-col>
      <v-col cols="12" sm="6">
        <v-textarea v-model="response_extra" :label="$t('dns.rule.responseExtra')" rows="2" no-resize hide-details density="compact" />
      </v-col>
    </v-row>
    <RuleNetworkState v-if="optionNetworkState" :rule="rule" />
    <RuleInterfaceAddress v-if="optionInterface" :rule="rule" />
    <v-card-actions>
      <v-spacer></v-spacer>
      <v-menu v-model="menu" :close-on-content-click="false" location="start">
        <template v-slot:activator="{ props }">
          <div class="d-flex align-center ga-1">
            <v-btn v-bind="props" hide-details variant="tonal">{{ $t('rule.options') }}</v-btn>
            <FieldHint :field-hints="fieldHints" field="match_fields" />
          </div>
        </template>
        <v-card>
          <v-list>
            <v-list-item>
              <v-switch v-model="optionInbound" color="primary" :label="$t('pages.inbounds')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionClient" color="primary" :label="$t('pages.clients')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionIPver" color="primary" :label="$t('rule.ipVer')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionQueryType" color="primary" :label="$t('dns.rule.queryType')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionNetwork" color="primary" :label="$t('network')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionProtocol" color="primary" :label="$t('protocol')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionDomain" color="primary" :label="$t('rule.domainRules')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionPort" color="primary" :label="$t('in.port')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionSrcIP" color="primary" :label="$t('rule.srcIpRules')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionSrcPort" color="primary" :label="$t('rule.srcPortRules')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionNetworkState" color="primary" :label="$t('rule.networkState')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionInterface" color="primary" :label="$t('rule.interfaceAddr')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionRuleSet" color="primary" :label="$t('rule.ruleset')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionProcess" color="primary" :label="$t('rule.process')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionDevice" color="primary" :label="$t('rule.deviceIdentity')" hide-details></v-switch>
            </v-list-item>
            <v-list-item>
              <v-switch v-model="optionResponse" color="primary" :label="$t('dns.rule.responseMatch')" hide-details></v-switch>
            </v-list-item>
          </v-list>
        </v-card>
      </v-menu>
    </v-card-actions>
  </v-card>
</template>

<script lang="ts">
import FieldHint from '@/components/FieldHint.vue'
import RuleInterfaceAddress from '@/components/RuleInterfaceAddress.vue'
import RuleNetworkState from '@/components/RuleNetworkState.vue'

export default {
  components: { FieldHint, RuleInterfaceAddress, RuleNetworkState },
  props: ['rule', 'clients', 'inTags', 'rsTags', 'deleteable', 'ruleSets', 'fieldHints'],
  data() {
    return {
      menu: false,
      domainKeys: ['domain', 'domain_suffix', 'domain_keyword', 'domain_regex', 'ip_cidr', 'ip_is_private', 'ip_accept_any'],
      portKeys: ['port', 'port_range'],
      srcIPKeys: ['source_ip_cidr', 'source_ip_is_private'],
      srcPortKeys: ['source_port', 'source_port_range'],
      domainOption: 'domain',
      portOption: 'port',
      srcIPOption: 'source_ip_cidr',
      srcPortOption: 'source_port',
      processKeys: ['process_name', 'process_path', 'process_path_regex', 'package_name', 'package_name_regex'],
      processOption: 'process_name',
      responseRcodes: ['NOERROR', 'FORMERR', 'SERVFAIL', 'NXDOMAIN', 'NOTIMP', 'REFUSED'],
      queryTypes: ['A', 'AAAA', 'CNAME', 'MX', 'NS', 'PTR', 'HTTPS', 'SVCB', 'TXT'],
    }
  },
  methods: {
    updateDomainOption(option:string) {
      this.domainKeys.forEach(k => delete this.$props.rule[k])
      this.$props.rule[option] = ['ip_is_private', 'ip_accept_any'].includes(option) ? false : []
    },
    updatePortOption(option:string) {
      this.portKeys.forEach(k => delete this.$props.rule[k])
      this.$props.rule[option] = []
    },
    updateSrcIPOption(option:string) {
      this.srcIPKeys.forEach(k => delete this.$props.rule[k])
      this.$props.rule[option] = option == 'source_ip_is_private' ? false : []
    },
    updateSrcPortOption(option:string) {
      this.srcPortKeys.forEach(k => delete this.$props.rule[k])
      this.$props.rule[option] = []
    },
    updateProcessOption(option:string) {
      this.processKeys.forEach(k => delete this.$props.rule[k])
      this.$props.rule[option] = []
    },
  },
  computed: {
    optionInbound: {
      get() { return this.$props.rule.inbound != undefined },
      set(v:boolean) { this.$props.rule.inbound = v ? [] : undefined }
    },
    optionClient: {
      get() { return this.$props.rule.auth_user != undefined },
      set(v:boolean) { this.$props.rule.auth_user = v ? [] : undefined }
    },
    optionIPver: {
      get() { return this.$props.rule.ip_version != undefined },
      set(v:boolean) { this.$props.rule.ip_version = v ? 4 : undefined }
    },
    optionQueryType: {
      get() {
        return this.$props.rule.query_type != undefined ||
               this.$props.rule.query_client_subnet != undefined ||
               this.$props.rule.query_dnssec != undefined
      },
      set(v:boolean) {
        if (v) {
          this.$props.rule.query_type = []
        } else {
          delete this.$props.rule.query_type
          delete this.$props.rule.query_client_subnet
          delete this.$props.rule.query_dnssec
        }
      }
    },
    optionNetwork: {
      get() { return this.$props.rule.network != undefined },
      set(v:boolean) { this.$props.rule.network = v ? [] : undefined }
    },
    optionProtocol: {
      get() { return this.$props.rule.protocol != undefined },
      set(v:boolean) { this.$props.rule.protocol = v ? ['http'] : undefined }
    },
    optionDomain: {
      get() { return Object.keys(this.$props.rule).some(r => this.domainKeys.includes(r)) },
      set(v:boolean) { 
        if (v) {
          this.$props.rule.domain = []
        } else {
          this.domainKeys.forEach(k => delete this.$props.rule[k])
        }
        this.domainOption = 'domain'
      }
    },
    optionPort: {
      get() { return Object.keys(this.$props.rule).some(r => this.portKeys.includes(r)) },
      set(v:boolean) { 
        if (v) {
          this.$props.rule.port = []
        } else {
          this.portKeys.forEach(k => delete this.$props.rule[k])
        }
        this.portOption = 'port'
      }
    },
    optionSrcIP: {
      get() { return Object.keys(this.$props.rule).some(r => this.srcIPKeys.includes(r)) },
      set(v:boolean) { 
        if (v) {
          this.$props.rule.source_ip_cidr = []
        } else {
          this.srcIPKeys.forEach(k => delete this.$props.rule[k])
        }
        this.srcIPOption = 'source_ip_cidr'
      }
    },
    optionSrcPort: {
      get() { return Object.keys(this.$props.rule).some(r => this.srcPortKeys.includes(r)) },
      set(v:boolean) { 
        if (v) {
          this.$props.rule.source_port = []
        } else {
          this.srcPortKeys.forEach(k => delete this.$props.rule[k])
        }
        this.srcPortOption = 'source_port'
      }
    },
    optionRuleSet: {
      get() { return this.$props.rule.rule_set != undefined },
      set(v:boolean) { 
        if (v) {
          this.$props.rule.rule_set = []
          this.$props.rule.rule_set_ip_cidr_match_source = false
          this.$props.rule.rule_set_ip_cidr_accept_empty = false
        } else {
          delete this.$props.rule.rule_set
          delete this.$props.rule.rule_set_ip_cidr_match_source
          delete this.$props.rule.rule_set_ip_cidr_accept_empty
        }
      }
    },
    optionNetworkState: {
      get() {
        return this.$props.rule.network_type != undefined ||
               this.$props.rule.network_is_expensive != undefined ||
               this.$props.rule.network_is_constrained != undefined ||
               this.$props.rule.wifi_ssid != undefined ||
               this.$props.rule.wifi_bssid != undefined
      },
      set(v:boolean) {
        if (v) {
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
      }
    },
    optionInterface: {
      get() { return ['interface_address', 'network_interface_address', 'default_interface_address'].some(k => this.$props.rule[k] != undefined) },
      set(v:boolean) {
        if (v) {
          this.$props.rule.interface_address = {}
        } else {
          ;['interface_address', 'network_interface_address', 'default_interface_address'].forEach(k => delete this.$props.rule[k])
        }
      }
    },
    optionProcess: {
      get() { return this.processKeys.some((key) => this.$props.rule[key] != undefined) },
      set(v:boolean) {
        if (v) this.$props.rule[this.processOption] = []
        else this.processKeys.forEach((key) => delete this.$props.rule[key])
      }
    },
    optionDevice: {
      get() {
        return this.$props.rule.source_mac_address != undefined ||
               this.$props.rule.source_hostname != undefined ||
               this.$props.rule.preferred_by != undefined
      },
      set(v:boolean) {
        if (v) {
          if (this.$props.rule.source_mac_address == undefined) this.$props.rule.source_mac_address = []
          if (this.$props.rule.source_hostname == undefined) this.$props.rule.source_hostname = []
        } else {
          delete this.$props.rule.source_mac_address
          delete this.$props.rule.source_hostname
          delete this.$props.rule.preferred_by
        }
      }
    },
    optionResponse: {
      get() {
        return this.$props.rule.match_response != undefined ||
               this.$props.rule.response_rcode != undefined ||
               this.$props.rule.response_answer != undefined ||
               this.$props.rule.response_ns != undefined ||
               this.$props.rule.response_extra != undefined
      },
      set(v:boolean) {
        if (v) {
          this.$props.rule.match_response = true
        } else {
          delete this.$props.rule.match_response
          delete this.$props.rule.response_rcode
          delete this.$props.rule.response_answer
          delete this.$props.rule.response_ns
          delete this.$props.rule.response_extra
        }
      }
    },
    // `match_response` is bool (match any response) or a string tag.
    matchResponseEnabled: {
      get() { return this.$props.rule.match_response != undefined && this.$props.rule.match_response !== false },
      set(v:boolean) {
        if (v) this.$props.rule.match_response = typeof this.$props.rule.match_response === 'string' ? this.$props.rule.match_response : true
        else delete this.$props.rule.match_response
      }
    },
    matchResponseTag: {
      get() { return typeof this.$props.rule.match_response === 'string' ? this.$props.rule.match_response : '' },
      set(v:string) { this.$props.rule.match_response = v.length > 0 ? v : true }
    },
    query_client_subnet: {
      get() { return this.$props.rule.query_client_subnet?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.query_client_subnet = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    process_name: {
      get() { return this.$props.rule.process_name?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.process_name = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    process_path: {
      get() { return this.$props.rule.process_path?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.process_path = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    process_path_regex: {
      get() { return this.$props.rule.process_path_regex?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.process_path_regex = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    package_name: {
      get() { return this.$props.rule.package_name?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.package_name = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    package_name_regex: {
      get() { return this.$props.rule.package_name_regex?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.package_name_regex = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    source_mac_address: {
      get() { return this.$props.rule.source_mac_address?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.source_mac_address = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    source_hostname: {
      get() { return this.$props.rule.source_hostname?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.source_hostname = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    response_answer: {
      get() { return this.$props.rule.response_answer?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.response_answer = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    response_ns: {
      get() { return this.$props.rule.response_ns?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.response_ns = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    response_extra: {
      get() { return this.$props.rule.response_extra?.join('\n') ?? '' },
      set(v:string) { this.$props.rule.response_extra = v.length > 0 ? v.split('\n').map((s:string) => s.trim()).filter((s:string) => s.length > 0) : [] }
    },
    domain: {
      get() { return this.$props.rule.domain?.join(',') },
      set(v:string) { this.$props.rule.domain = v.length>0 ? v.split(',') : [] }
    },
    domain_suffix: {
      get() { return this.$props.rule.domain_suffix?.join(',') },
      set(v:string) { this.$props.rule.domain_suffix = v.length>0 ? v.split(',') : [] }
    },
    domain_keyword: {
      get() { return this.$props.rule.domain_keyword?.join(',') },
      set(v:string) { this.$props.rule.domain_keyword = v.length>0 ? v.split(',') : [] }
    },
    domain_regex: {
      get() { return this.$props.rule.domain_regex?.join(',') },
      set(v:string) { this.$props.rule.domain_regex = v.length>0 ? v.split(',') : [] }
    },
    ip_cidr: {
      get() { return this.$props.rule.ip_cidr?.join(',') },
      set(v:string) { this.$props.rule.ip_cidr = v.length>0 ? v.split(',') : [] }
    },
    port: {
      get() { return this.$props.rule.port?.join(',') },
      set(v:string) {
        if(!v.endsWith(',')) {
          this.$props.rule.port = v.length > 0 ? v.split(',').map(str => parseInt(str, 10)) : []
        }
      }
    },
    port_range: {
      get() { return this.$props.rule.port_range?.join(',') },
      set(v:string) { this.$props.rule.port_range = v.length>0 ? v.split(',') : [] }
    },
    source_ip_cidr: {
      get() { return this.$props.rule.source_ip_cidr?.join(',') },
      set(v:string) { this.$props.rule.source_ip_cidr = v.length>0 ? v.split(',') : [] }
    },
    source_port: {
      get() { return this.$props.rule.source_port?.join(',') },
      set(v:string) {
        if(!v.endsWith(',')) {
          this.$props.rule.source_port = v.length > 0 ? v.split(',').map(str => parseInt(str, 10)) : []
        }
      }
    },
    source_port_range: {
      get() { return this.$props.rule.source_port_range?.join(',') },
      set(v:string) { this.$props.rule.source_port_range = v.length>0 ? v.split(',') : [] }
    },
  },
  mounted() {
    const ruleKeys = Object.keys(this.$props.rule)
    if (this.optionDomain) {
      const enabledOption = this.domainKeys.filter(k => ruleKeys.includes(k))
      this.domainOption = enabledOption.length>0 ? enabledOption[0] : 'domain'
    }
    if (this.optionPort) {
      const enabledOption = this.portKeys.filter(k => ruleKeys.includes(k))
      this.portOption = enabledOption.length>0 ? enabledOption[0] : 'port'
    }
    if (this.optionSrcIP) {
      const enabledOption = this.srcIPKeys.filter(k => ruleKeys.includes(k))
      this.srcIPOption = enabledOption.length>0 ? enabledOption[0] : 'source_ip_cidr'
    }
    if (this.optionSrcPort) {
      const enabledOption = this.srcPortKeys.filter(k => ruleKeys.includes(k))
      this.srcPortOption = enabledOption.length>0 ? enabledOption[0] : 'source_port'
    }
    if (this.optionProcess) {
      const enabledOption = this.processKeys.filter(k => ruleKeys.includes(k))
      this.processOption = enabledOption.length>0 ? enabledOption[0] : 'process_name'
    }
  }
}
</script>
