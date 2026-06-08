<template>
  <v-card subtitle="Tun">
    <v-row>
      <v-col cols="12" sm="8">
        <v-text-field v-model="addrs" :label="$t('types.tun.addr') + ' ' + $t('commaSeparated')" placeholder="172.18.0.1/30" hide-details></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field v-model="data.interface_name" :label="$t('types.tun.ifName')" placeholder="tun0" hide-details clearable @click:clear="delete data.interface_name"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" v-model.number="data.mtu" label="MTU" hide-details></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          type="number"
          v-model.number="udpTimeout"
          label="UDP timeout"
          min="1"
          :suffix="$t('date.m')"
          hide-details>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          v-model="data.stack"
          label="Stack"
          :items="['system','gvisor','mixed']"
          hide-details
        ></v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="data.endpoint_independent_nat != undefined">
        <v-switch v-model="optionEndpointIndependentNat" color="primary" :label="$t('singbox.independentNatCompatibility')" hide-details></v-switch>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-switch v-model="autoRoute" color="primary" label="Auto Route" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute">
        <v-switch v-model="autoRedirect" color="primary" label="Auto Redirect" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute">
        <v-switch v-model="strictRoute" color="primary" label="Strict Route" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute && data.auto_redirect">
        <v-switch v-model="excludeMptcp" color="primary" :label="$t('types.tun.excludeMptcp')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute && data.auto_redirect">
        <v-text-field
          type="number"
          v-model.number="fallbackRuleIndex"
          :label="$t('types.tun.fallbackRuleIndex')"
          min="0"
          hide-details>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute && data.auto_redirect">
        <v-text-field
          v-model="autoRedirectResetMark"
          :label="$t('types.tun.resetMark')"
          placeholder="0x2024"
          hide-details>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute && data.auto_redirect">
        <v-text-field v-model="autoRedirectInputMark" :label="$t('singbox.inputMark')" placeholder="0x2023" hide-details></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute && data.auto_redirect">
        <v-text-field v-model="autoRedirectOutputMark" :label="$t('singbox.outputMark')" placeholder="0x2024" hide-details></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="autoRoute && data.auto_redirect">
        <v-text-field
          type="number"
          v-model.number="nfqueue"
          :label="$t('types.tun.nfqueue')"
          min="0"
          hide-details>
        </v-text-field>
      </v-col>
    </v-row>
    <template v-if="autoRoute">
      <v-row>
        <v-col cols="12" class="v-card-subtitle">{{ $t('singbox.splitRouting') }}</v-col>
        <v-col cols="12" sm="6" md="4">
          <v-btn variant="tonal" @click="applyLanDirect">{{ $t('singbox.keepLanDirect') }}</v-btn>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-select
            v-model="routeAddressSet"
            :items="ruleSetTags"
            :label="$t('singbox.ruleSetTunnel')"
            multiple chips closable-chips
            hide-details>
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="4" v-if="data.auto_redirect && data.route_address_set?.length > 0">
          <v-alert density="compact" type="warning" variant="tonal">
            {{ $t('singbox.autoRedirectRuleSetWarning') }}
          </v-alert>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="routeAddressText" rows="2" auto-grow hide-details :label="$t('singbox.routeAddresses')"></v-textarea>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="routeExcludeAddressText" rows="2" auto-grow hide-details :label="$t('singbox.routeExcludeAddresses')"></v-textarea>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" class="v-card-subtitle">{{ $t('singbox.appsUsers') }}</v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="includePackageText" rows="2" auto-grow hide-details :label="$t('singbox.includePackages')"></v-textarea>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="excludePackageText" rows="2" auto-grow hide-details :label="$t('singbox.excludePackages')"></v-textarea>
        </v-col>
        <v-col cols="12" v-if="emptyAppPreset">
          <v-alert density="compact" type="warning" variant="tonal">
            {{ $t('singbox.appPackageWarning') }}
          </v-alert>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="includeUidText" rows="2" auto-grow hide-details :label="$t('singbox.includeUids')"></v-textarea>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="excludeUidText" rows="2" auto-grow hide-details :label="$t('singbox.excludeUids')"></v-textarea>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="includeUidRangeText" rows="2" auto-grow hide-details :label="$t('singbox.includeUidRanges')"></v-textarea>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="excludeUidRangeText" rows="2" auto-grow hide-details :label="$t('singbox.excludeUidRanges')"></v-textarea>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" class="v-card-subtitle">{{ $t('singbox.advanced') }}</v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="includeInterfaceText" rows="2" auto-grow hide-details :label="$t('singbox.includeInterfaces')"></v-textarea>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="excludeInterfaceText" rows="2" auto-grow hide-details :label="$t('singbox.excludeInterfaces')"></v-textarea>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field v-model.number="data.iproute2_table_index" type="number" min="0" hide-details :label="$t('singbox.iproute2TableIndex')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field v-model.number="data.iproute2_rule_index" type="number" min="0" hide-details :label="$t('singbox.iproute2RuleIndex')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6">
          <v-textarea v-model="loopbackAddressText" rows="2" auto-grow hide-details :label="$t('singbox.loopbackAddresses')"></v-textarea>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" class="v-card-subtitle">{{ $t('singbox.httpProxy') }}</v-col>
        <v-col cols="12" sm="6" md="4">
          <v-switch v-model="httpProxyEnabled" color="primary" :label="$t('singbox.enablePlatformHttpProxy')" hide-details></v-switch>
        </v-col>
        <template v-if="data.platform?.http_proxy">
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="data.platform.http_proxy.server" hide-details :label="$t('out.addr')"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model.number="data.platform.http_proxy.server_port" type="number" min="0" max="65535" hide-details :label="$t('out.port')"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-textarea v-model="httpProxyBypassText" rows="2" auto-grow hide-details :label="$t('singbox.bypassDomains')"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-textarea v-model="httpProxyMatchText" rows="2" auto-grow hide-details :label="$t('singbox.matchDomains')"></v-textarea>
          </v-col>
        </template>
      </v-row>
    </template>
  </v-card>
</template>

<script lang="ts">
import Data from '@/store/modules/data'

export default {
  props: ['data'],
  data() {
    return {
      menu: false
    }
  },
  computed: {
    ruleSetTags() { return Data().config.route?.rule_set?.map((rs:any) => rs.tag) ?? [] },
    emptyAppPreset() {
      return (this.$props.data.include_package != undefined && this.$props.data.include_package.length == 0) ||
        (this.$props.data.exclude_package != undefined && this.$props.data.exclude_package.length == 0)
    },
    optionEndpointIndependentNat: {
      get() { return this.$props.data.endpoint_independent_nat ?? false },
      set(v:boolean) { v ? this.$props.data.endpoint_independent_nat = true : delete this.$props.data.endpoint_independent_nat }
    },
    addrs: {
      get() { return this.$props.data.address?.join(',') },
      set(v:string) { this.$props.data.address = v.length > 0 ? v.split(',') : undefined }
    },
    udpTimeout: {
      get() { return this.$props.data.udp_timeout ? parseInt(this.$props.data.udp_timeout.replace('m','')) : 5 },
      set(v:number) { this.$props.data.udp_timeout = v > 0 ? v + 'm' : '5m' }
    },
    autoRoute: {
      get() { return this.$props.data.auto_route ?? false },
      set(v:boolean) {
        if (v) {
          this.$props.data.auto_route = true
        } else {
          delete this.$props.data.auto_route
          delete this.$props.data.auto_redirect
          delete this.$props.data.strict_route
          delete this.$props.data.exclude_mptcp
          delete this.$props.data.auto_redirect_reset_mark
          delete this.$props.data.auto_redirect_input_mark
          delete this.$props.data.auto_redirect_output_mark
          delete this.$props.data.auto_redirect_nfqueue
          delete this.$props.data.auto_redirect_iproute2_fallback_rule_index
          delete this.$props.data.route_address
          delete this.$props.data.route_address_set
          delete this.$props.data.route_exclude_address
          delete this.$props.data.route_exclude_address_set
        }
      }
    },
    autoRedirect: {
      get() { return this.$props.data.auto_redirect === true },
      set(v:boolean) { v ? this.$props.data.auto_redirect = true : delete this.$props.data.auto_redirect }
    },
    strictRoute: {
      get() { return this.$props.data.strict_route === true },
      set(v:boolean) { v ? this.$props.data.strict_route = true : delete this.$props.data.strict_route }
    },
    excludeMptcp: {
      get() { return this.$props.data.exclude_mptcp === true },
      set(v:boolean) { v ? this.$props.data.exclude_mptcp = true : delete this.$props.data.exclude_mptcp }
    },
    routeAddressSet: {
      get() { return this.$props.data.route_address_set ?? [] },
      set(v:string[]) { v.length > 0 ? this.$props.data.route_address_set = v : delete this.$props.data.route_address_set }
    },
    autoRedirectResetMark: markText('auto_redirect_reset_mark'),
    autoRedirectInputMark: markText('auto_redirect_input_mark'),
    autoRedirectOutputMark: markText('auto_redirect_output_mark'),
    fallbackRuleIndex: {
      get() { return this.$props.data.auto_redirect_iproute2_fallback_rule_index ?? 32768 },
      set(v: number) {
        const val = typeof v === 'number' && !isNaN(v) && v >= 0 ? v : undefined
        this.$props.data.auto_redirect_iproute2_fallback_rule_index = val
      }
    },
    nfqueue: {
      get() { return this.$props.data.auto_redirect_nfqueue ?? 0 },
      set(v: number) {
        this.$props.data.auto_redirect_nfqueue = typeof v === 'number' && !isNaN(v) && v > 0 ? v : undefined
      }
    },
    routeAddressText: listText('route_address'),
    routeExcludeAddressText: listText('route_exclude_address'),
    includePackageText: listText('include_package'),
    excludePackageText: listText('exclude_package'),
    includeInterfaceText: listText('include_interface'),
    excludeInterfaceText: listText('exclude_interface'),
    includeUidRangeText: listText('include_uid_range'),
    excludeUidRangeText: listText('exclude_uid_range'),
    loopbackAddressText: listText('loopback_address'),
    includeUidText: numberListText('include_uid'),
    excludeUidText: numberListText('exclude_uid'),
    httpProxyEnabled: {
      get() { return this.$props.data.platform?.http_proxy != undefined },
      set(v:boolean) {
        if (v) {
          if (!this.$props.data.platform) this.$props.data.platform = {}
          this.$props.data.platform.http_proxy = { enabled: true, server: '127.0.0.1', server_port: 8080 }
        } else if (this.$props.data.platform) {
          delete this.$props.data.platform.http_proxy
          if (Object.keys(this.$props.data.platform).length == 0) delete this.$props.data.platform
        }
      }
    },
    httpProxyBypassText: nestedListText(['platform', 'http_proxy'], 'bypass_domain'),
    httpProxyMatchText: nestedListText(['platform', 'http_proxy'], 'match_domain')
  },
  methods: {
    applyLanDirect() {
      this.$props.data.route_exclude_address = ['10.0.0.0/8','172.16.0.0/12','192.168.0.0/16','fc00::/7','fe80::/10']
    }
  }
}

function splitText(v:string): string[] | undefined {
  const values = v.split('\n').map(item => item.trim()).filter(item => item.length > 0)
  return values.length > 0 ? values : undefined
}

function listText(field:string) {
  return {
    get(this:any): string { return this.$props.data[field]?.join('\n') ?? '' },
    set(this:any, v:string) {
      const values = splitText(v)
      values ? this.$props.data[field] = values : delete this.$props.data[field]
    }
  }
}

function numberListText(field:string) {
  return {
    get(this:any): string { return this.$props.data[field]?.join('\n') ?? '' },
    set(this:any, v:string) {
      const values = splitText(v)?.map(item => Number(item)).filter(item => !isNaN(item))
      values && values.length > 0 ? this.$props.data[field] = values : delete this.$props.data[field]
    }
  }
}

function nestedListText(path:string[], field:string) {
  return {
    get(this:any): string {
      let target = this.$props.data
      for (const key of path) target = target?.[key]
      return target?.[field]?.join('\n') ?? ''
    },
    set(this:any, v:string) {
      let target = this.$props.data
      for (const key of path) target = target?.[key]
      if (!target) return
      const values = splitText(v)
      values ? target[field] = values : delete target[field]
    }
  }
}

function markText(field:string) {
  return {
    get(this:any): string { return this.$props.data[field]?.toString() ?? '' },
    set(this:any, v:string) {
      const trimmed = (v ?? '').toString().trim()
      if (trimmed.length == 0 || trimmed == '0' || trimmed == '0x0') {
        delete this.$props.data[field]
      } else if (trimmed.startsWith('0x')) {
        this.$props.data[field] = trimmed
      } else {
        const parsed = Number(trimmed)
        if (Number.isFinite(parsed) && parsed > 0) this.$props.data[field] = parsed
        else delete this.$props.data[field]
      }
    }
  }
}
</script>
