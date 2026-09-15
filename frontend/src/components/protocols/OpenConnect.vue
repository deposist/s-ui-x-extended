<template>
  <v-card subtitle="OpenConnect">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('out.addr')" v-model="data.server" :placeholder="'vpn.example.com'"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select hide-details :label="$t('types.openconnect.flavor')" :items="flavors" v-model="data.flavor"></v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.wg.sysIf')" v-model="data.system"></v-switch>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.openconnect.username')" v-model="data.username"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details type="password" :label="$t('types.openconnect.password')" v-model="data.password"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.openconnect.authGroup')" v-model="data.auth_group"></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.openconnect.cookie')" v-model="data.cookie"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.openconnect.reportedOs')" :items="reportedOsList" v-model="data.reported_os"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.openconnect.userAgent')" v-model="data.user_agent"></v-text-field>
      </v-col>
    </v-row>

    <!-- token: 2FA (TOTP/HOTP/stoken/OIDC) -->
    <v-card border density="compact" color="background" class="mt-2">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.openconnect.token') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-select hide-details clearable :label="$t('types.openconnect.tokenMode')" :items="tokenModes" @click:clear="clearTokenMode" v-model="data.token.mode"></v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4" v-if="data.token.mode">
            <v-text-field hide-details :label="$t('types.openconnect.tokenSecret')" v-model="data.token.secret"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4" v-if="data.token.mode">
            <v-text-field hide-details :label="$t('types.openconnect.tokenPin')" v-model="data.token.pin"></v-text-field>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <!-- TLS material, edited as raw JSON: the block mirrors the core
         OpenConnectTLSOptions one-for-one (CA/client cert/key, MCA, insecure). -->
    <v-card border density="compact" color="background" class="mt-2">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('objects.tls') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('tls.insecure')" v-model="data.tls.insecure"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field hide-details :label="$t('types.openconnect.tlsServerName')" v-model="data.tls.server_name"></v-text-field>
          </v-col>
        </v-row>
        <v-textarea
          v-model="tlsJson"
          :error-messages="tlsError ? [tlsError] : []"
          :hint="$t('types.openconnect.tlsJsonHint')"
          persistent-hint
          variant="outlined"
          auto-grow
          rows="4"
          :style="{ 'font-family': 'monospace' }">
        </v-textarea>
      </v-card-text>
    </v-card>

    <!-- Advanced toggles + advanced JSON blocks -->
    <v-card border density="compact" color="background" class="mt-2">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.openconnect.advanced') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('types.openconnect.noUdp')" v-model="data.no_udp"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('types.openconnect.compressionDisabled')" v-model="data.compression_disabled"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-select hide-details clearable :label="$t('types.openconnect.compressionMode')" :items="['stateless', 'all']" @click:clear="delete data.compression_mode" v-model="data.compression_mode"></v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field type="number" min="0" hide-details label="MTU" v-model.number="data.mtu"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field hide-details :label="$t('types.openconnect.reconnectTimeout')" placeholder="5s" v-model="data.reconnect_timeout"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('types.openconnect.allowInsecureCrypto')" v-model="data.allow_insecure_crypto"></v-switch>
          </v-col>
        </v-row>
        <v-textarea
          v-model="advancedJson"
          :error-messages="advancedError ? [advancedError] : []"
          :hint="$t('types.openconnect.advancedJsonHint')"
          persistent-hint
          variant="outlined"
          auto-grow
          rows="6"
          :style="{ 'font-family': 'monospace' }">
        </v-textarea>
      </v-card-text>
    </v-card>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('types.openconnect.dtlsLocalPort')"
          hide-details
          type="number"
          min="0"
          max="65535"
          clearable
          @click:clear="delete data.dtls_local_port"
          v-model.number="data.dtls_local_port">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="dtls_local_port" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch
          color="primary"
          :label="$t('types.openconnect.ipv6Disabled')"
          hide-details
          v-model="data.ipv6_disabled">
        </v-switch>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      flavors: ['anyconnect', 'gp', 'fortinet', 'f5', 'pulse', 'nc'],
      tokenModes: ['totp', 'hotp', 'stoken', 'oidc'],
      reportedOsList: ['linux', 'linux-64', 'win', 'mac-intel', 'android', 'apple-ios'],
      tlsError: '',
      advancedError: '',
    }
  },
  created() {
    if (!this.$props.data.token) this.$props.data.token = {}
    if (!this.$props.data.tls) this.$props.data.tls = {}
  },
  computed: {
    // The TLS block minus the two structured fields (insecure/server_name):
    // the JSON textarea owns the certificate material. Writing it back merges,
    // never clobbers the structured switches.
    tlsJson: {
      get(): string {
        const tls = this.jsonBlock('tls')
        const { insecure, server_name, ...rest } = tls
        return Object.keys(rest).length ? JSON.stringify(rest, null, 2) : ''
      },
      set(v: string) {
        this.mergeJsonBlock('tls', v, ['insecure', 'server_name'], (e: string) => this.tlsError = e)
      },
    },
    // Everything the structured form does not own: csd/hip/tncc/mobile/
    // fortinet_host_check/form_entries and the remaining scalars. Merging on
    // write guarantees no field loss for anything the form does not render.
    advancedJson: {
      get(): string {
        const block: Record<string, any> = {}
        for (const key of Object.keys(this.$props.data)) {
          if (structuredKeys.includes(key) || ['id', 'type', 'tag'].includes(key)) continue
          block[key] = this.$props.data[key]
        }
        return Object.keys(block).length ? JSON.stringify(block, null, 2) : ''
      },
      set(v: string) {
        this.mergeAdvanced(v)
      },
    },
  },
  methods: {
    jsonBlock(key: string): Record<string, any> {
      const v = this.$props.data[key]
      return v && typeof v === 'object' && !Array.isArray(v) ? v : {}
    },
    clearTokenMode() {
      this.$props.data.token.mode = undefined
      delete this.$props.data.token.secret
      delete this.$props.data.token.pin
      if (Object.keys(this.jsonBlock('token')).length === 0) delete this.$props.data.token
    },
    mergeJsonBlock(key: string, text: string, keep: string[], onError: (e: string) => void) {
      const base: Record<string, any> = {}
      for (const k of keep) if (this.$props.data[key]?.[k] !== undefined) base[k] = this.$props.data[key][k]
      if (text.trim().length === 0) {
        this.$props.data[key] = base
        onError('')
        return
      }
      try {
        const parsed = JSON.parse(text)
        if (typeof parsed !== 'object' || Array.isArray(parsed) || parsed === null) {
          onError(this.$t('types.vpn.outboundObjectError'))
          return
        }
        this.$props.data[key] = { ...parsed, ...base }
        onError('')
      } catch (e: any) {
        onError(e.message)
      }
    },
    mergeAdvanced(text: string) {
      if (text.trim().length === 0) {
        for (const key of Object.keys(this.$props.data)) {
          if (!structuredKeys.includes(key) && !['id', 'type', 'tag'].includes(key)) delete this.$props.data[key]
        }
        this.advancedError = ''
        return
      }
      try {
        const parsed = JSON.parse(text)
        if (typeof parsed !== 'object' || Array.isArray(parsed) || parsed === null) {
          this.advancedError = this.$t('types.vpn.outboundObjectError')
          return
        }
        // Drop non-structured keys, then re-apply the edited block.
        for (const key of Object.keys(this.$props.data)) {
          if (!structuredKeys.includes(key) && !['id', 'type', 'tag'].includes(key)) delete this.$props.data[key]
        }
        Object.assign(this.$props.data, parsed)
        this.advancedError = ''
      } catch (e: any) {
        this.advancedError = e.message
      }
    },
  },
}

// Keys the structured form owns; everything else flows through advancedJson.
const structuredKeys = [
  'system', 'name', 'udp_timeout', 'udp_mapping', 'udp_filtering', 'udp_nat_max',
  'server', 'flavor', 'username', 'password', 'auth_group', 'cookie', 'token',
  'reported_os', 'user_agent', 'version', 'local_hostname',
  'no_udp', 'compression_disabled', 'compression_mode', 'mtu', 'reconnect_timeout',
  'allow_insecure_crypto', 'tls',
]
</script>
