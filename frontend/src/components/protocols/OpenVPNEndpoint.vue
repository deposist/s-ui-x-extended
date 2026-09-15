<template>
  <v-card subtitle="OpenVPN">
    <!-- Servers (client only): a server endpoint has no remotes list -->
    <template v-if="isClient">
      <v-card-subtitle>
        {{ $t('types.openvpn.servers') }}
        <v-chip color="primary" density="compact" variant="elevated" @click="addServer"><v-icon icon="mdi-plus" /></v-chip>
      </v-card-subtitle>
      <v-row v-for="(s, index) in servers" :key="index">
        <v-col cols="12" sm="6">
          <v-text-field hide-details :label="$t('out.addr')" v-model="s.server"></v-text-field>
        </v-col>
        <v-col cols="8" sm="4">
          <v-text-field type="number" min="1" hide-details :label="$t('out.port')" v-model.number="s.server_port"></v-text-field>
        </v-col>
        <v-col cols="4" sm="2" class="d-flex align-center">
          <v-icon color="error" icon="mdi-delete" @click="delServer(index)" />
        </v-col>
      </v-row>
    </template>

    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.openvpn.name')" v-model="data.name"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select hide-details :label="$t('types.openvpn.proto')" :items="['udp', 'tcp']" v-model="data.network"></v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.openvpn.sysIf')" v-model="data.system"></v-switch>
      </v-col>
    </v-row>

    <!-- Client TLS-mode options -->
    <template v-if="isClient">
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-select hide-details :label="$t('types.openvpn.mode')" :items="['tls', 'static_key']" v-model="data.mode"></v-select>
        </v-col>
        <v-col cols="12" sm="6" md="4" v-if="tlsMode">
          <v-combobox
            clearable
            hide-details
            multiple
            chips
            :label="$t('types.openvpn.dataCiphers')"
            :items="openvpnCiphers"
            v-model="data.data_ciphers">
          </v-combobox>
        </v-col>
        <v-col cols="12" sm="6" md="4" v-else>
          <v-combobox
            clearable
            hide-details
            :label="$t('types.openvpn.cipher')"
            :items="openvpnCiphers"
            v-model="data.cipher">
          </v-combobox>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-combobox
            clearable
            hide-details
            :label="$t('types.openvpn.auth')"
            :items="openvpnAuthDigests"
            v-model="data.auth">
          </v-combobox>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12" sm="6">
          <v-text-field hide-details :label="$t('types.un')" v-model="data.username"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field hide-details :label="$t('types.pw')" v-model="data.password"></v-text-field>
        </v-col>
      </v-row>
    </template>

    <!-- Server listen address -->
    <v-row v-else>
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.openvpn.listenAddress') + ' ' + $t('commaSeparated')" v-model="addressCsv"></v-text-field>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12" sm="6">
        <v-combobox hide-details :label="$t('types.openvpn.pingInterval')" :items="durationPresets" v-model="data.ping_interval"></v-combobox>
      </v-col>
      <v-col cols="12" sm="6" v-if="isClient">
        <v-combobox hide-details :label="$t('types.openvpn.reconnectDelay')" :items="durationPresets" v-model="data.reconnect_delay"></v-combobox>
      </v-col>
    </v-row>

    <!-- Control-channel wrap (tls_auth / tls_crypt) -->
    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.openvpn.controlWrap') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-select clearable hide-details :label="$t('types.openvpn.controlWrapType')"
              :items="['tls_auth', 'tls_crypt', 'tls_crypt_v2']" v-model="controlWrap.type"></v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4" v-if="controlWrap.type == 'tls_auth'">
            <v-select clearable hide-details :label="$t('types.openvpn.controlWrapDirection')"
              :items="['server', 'client']" v-model="controlWrap.direction"></v-select>
          </v-col>
        </v-row>
        <v-row v-if="controlWrap.type">
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('types.openvpn.controlWrapKey')" v-model="controlWrapKey"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.openvpn.controlWrapKeyPath')" v-model="controlWrap.key_path"></v-text-field>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <!-- TLS material -->
    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('objects.tls') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('types.openvpn.ca')" v-model="tlsCertificate"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.openvpn.caPath')" v-model="tls.certificate_path"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('tls.cert')" v-model="tlsClientCertificate"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('tls.certPath')" v-model="tls.client_certificate_path"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('tls.key')" v-model="tlsClientKey"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('tls.keyPath')" v-model="tls.client_key_path"></v-text-field>
          </v-col>
        </v-row>
        <v-row v-if="isClient">
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.openvpn.serverName')" v-model="tls.server_name"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-select clearable hide-details :label="$t('types.openvpn.serverNameType')"
              :items="['subject', 'name', 'name-prefix', 'name-suffix']" v-model="tls.server_name_type"></v-select>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>
  </v-card>
</template>

<script lang="ts">
import { openvpnCiphers, openvpnAuthDigests, durationPresets } from '@/types/recommended'
import type { OpenVPNControlWrap, OpenVPNEndpointTLS } from '@/types/endpoints'

// Lines-join helper: certificate/key material is a Listable[string] in the
// endpoint schema but edited as a single PEM textarea.
function joinLines(v: unknown): string {
  if (Array.isArray(v)) return v.join('\n')
  return typeof v === 'string' ? v : ''
}
function splitLines(s: string): string[] | undefined {
  const t = s.trim()
  return t === '' ? undefined : [t]
}

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {
      openvpnCiphers,
      openvpnAuthDigests,
      durationPresets,
    }
  },
  created() {
    if (!this.$props.data.tls) this.$props.data.tls = {}
    if (this.isClient && !Array.isArray(this.$props.data.servers)) this.$props.data.servers = []
  },
  computed: {
    isClient(): boolean {
      return this.$props.data.type === 'openvpn-client'
    },
    tlsMode(): boolean {
      return (this.$props.data.mode ?? 'tls') !== 'static_key'
    },
    servers(): any[] {
      return this.$props.data.servers ?? (this.$props.data.servers = [])
    },
    tls(): OpenVPNEndpointTLS {
      return this.$props.data.tls as OpenVPNEndpointTLS
    },
    controlWrap(): OpenVPNControlWrap {
      const tls = this.tls
      if (!tls.control_wrap) tls.control_wrap = {}
      return tls.control_wrap
    },
    addressCsv: {
      get(): string {
        const a = this.$props.data.address
        return Array.isArray(a) ? a.join(',') : ''
      },
      set(v: string) {
        this.$props.data.address = v.split(',').map((s: string) => s.trim()).filter((s: string) => s.length > 0)
      },
    },
    controlWrapKey: {
      get(): string { return joinLines(this.controlWrap.key) },
      set(v: string) { this.controlWrap.key = splitLines(v) },
    },
    tlsCertificate: {
      get(): string { return joinLines(this.tls.certificate) },
      set(v: string) { this.tls.certificate = splitLines(v) },
    },
    tlsClientCertificate: {
      get(): string { return joinLines(this.tls.client_certificate) },
      set(v: string) { this.tls.client_certificate = splitLines(v) },
    },
    tlsClientKey: {
      get(): string { return joinLines(this.tls.client_key) },
      set(v: string) { this.tls.client_key = splitLines(v) },
    },
  },
  methods: {
    addServer() {
      this.servers.push({ server: '', server_port: 1194 })
    },
    delServer(index: number) {
      this.servers.splice(index, 1)
    },
  },
}
</script>
