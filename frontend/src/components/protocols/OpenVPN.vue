<template>
  <v-card subtitle="OpenVPN">
    <!-- Servers -->
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

    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field hide-details :label="$t('types.openvpn.name')" v-model="data.name"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select hide-details :label="$t('types.openvpn.proto')" :items="['udp', 'tcp']" v-model="data.proto">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="proto" />
          </template>
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-switch color="primary" hide-details :label="$t('types.openvpn.sysIf')" v-model="data.system"></v-switch>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-combobox
          clearable
          hide-details
          :label="$t('types.openvpn.cipher')"
          :items="openvpnCiphers"
          v-model="data.cipher">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="cipher" />
          </template>
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-combobox
          clearable
          hide-details
          :label="$t('types.openvpn.auth')"
          :items="openvpnAuthDigests"
          v-model="data.auth">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="auth" />
          </template>
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" hide-details :label="$t('types.openvpn.keyDirection')" v-model.number="data.key_direction"></v-text-field>
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
    <v-row>
      <v-col cols="12" sm="6">
        <v-combobox hide-details :label="$t('types.openvpn.pingInterval')" :items="durationPresets" v-model="data.ping_interval"></v-combobox>
      </v-col>
      <v-col cols="12" sm="6">
        <v-combobox hide-details :label="$t('types.openvpn.reconnectDelay')" :items="durationPresets" v-model="data.reconnect_delay"></v-combobox>
      </v-col>
    </v-row>

    <!-- tls-crypt / tls-auth -->
    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.openvpn.staticKey') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('types.openvpn.tlsCrypt')" v-model="data.tls_crypt"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.openvpn.tlsCryptPath')" v-model="data.tls_crypt_path"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('types.openvpn.tlsAuth')" v-model="data.tls_auth"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.openvpn.tlsAuthPath')" v-model="data.tls_auth_path"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-switch color="primary" hide-details :label="$t('types.openvpn.tlsCryptV2')" v-model="data.tls_crypt_v2"></v-switch>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <!-- OpenVPN TLS -->
    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('objects.tls') }}</v-card-subtitle>
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('types.openvpn.ca')" v-model="tls.ca"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.openvpn.caPath')" v-model="tls.ca_path"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('tls.cert')" v-model="tls.certificate"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('tls.certPath')" v-model="tls.certificate_path"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-textarea rows="3" no-resize hide-details :label="$t('tls.key')" v-model="tls.key"></v-textarea>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('tls.keyPath')" v-model="tls.key_path"></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6">
            <v-text-field hide-details :label="$t('types.openvpn.verifyX509Name')" v-model="tls.verify_x509_name"></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-select clearable hide-details :label="$t('types.openvpn.verifyX509NameMode')"
              :items="['name-prefix', 'name-suffix', 'exact']" v-model="tls.verify_x509_name_mode"></v-select>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" md="8">
            <v-select chips multiple clearable hide-details :label="$t('tls.cs')"
              :items="tlsCipherSuites" :model-value="tls.cipher_suites" @update:model-value="setCipherSuites"></v-select>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('tls.kernelTx')" v-model="tls.kernel_tx"></v-switch>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-switch color="primary" hide-details :label="$t('tls.kernelRx')" v-model="tls.kernel_rx"></v-switch>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>
  </v-card>
</template>

<script lang="ts">
import { openvpnCiphers, openvpnAuthDigests, durationPresets, tlsCipherSuites } from '@/types/recommended'
import FieldHint from '@/components/FieldHint.vue'

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
      tlsCipherSuites,
    }
  },
  created() {
    if (!this.$props.data.tls) this.$props.data.tls = {}
  },
  computed: {
    servers(): any[] {
      return Array.isArray(this.$props.data.servers) ? this.$props.data.servers : []
    },
    tls(): any {
      return this.$props.data.tls
    },
  },
  methods: {
    addServer() {
      if (!Array.isArray(this.$props.data.servers)) this.$props.data.servers = []
      this.$props.data.servers.push({ server: '', server_port: 1194 })
    },
    delServer(index: number) {
      this.servers.splice(index, 1)
    },
    setCipherSuites(v: string[]) {
      if (v && v.length > 0) {
        this.tls.cipher_suites = v
      } else {
        delete this.tls.cipher_suites
      }
    },
  },
  components: { FieldHint },
}
</script>
