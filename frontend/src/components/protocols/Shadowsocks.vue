<template>
  <v-card subtitle="Shadowsocks">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('in.ssMethod')"
          :items="ssMethods"
          @update:model-value="direction == 'in' ? changeMethod($event) : undefined"
          v-model="data.method">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <Network :data="data" />
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction == 'out'">
        <UoT :data="data" />
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="direction == 'in'">
        <v-switch
          v-model="data.managed"
          color="primary"
          :label="$t('in.ssManageable')"
          hide-details>
        </v-switch>
      </v-col>
    </v-row>
    <v-row v-if="data.method != 'none' || direction == 'out'">
      <v-col cols="12" sm="8">
        <v-text-field
          v-model="data.password"
          :label="$t('types.pw')"
          hide-details
          :append-inner-icon="direction == 'in' ? 'mdi-refresh' : undefined"
          @click:append-inner="changeMethod(data.method)">
        </v-text-field>
      </v-col>
    </v-row>
    <v-row v-if="direction == 'out'">
      <v-col cols="12" sm="6" md="4">
        <v-select
          v-model="plugin"
          :label="$t('singbox.plugin')"
          :items="pluginPresets"
          clearable
          hide-details>
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="8">
        <v-text-field
          v-model="pluginOptions"
          :label="$t('singbox.pluginOptions')"
          placeholder="obfs=http;obfs-host=example.com"
          hide-details>
        </v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import Network from '@/components/Network.vue'
import UoT from '@/components/UoT.vue'
import RandomUtil from '@/plugins/randomUtil'

export default {
  props: ['direction','data'],
  data() {
    return {
      pluginPresets: [
        'obfs-local',
        'v2ray-plugin'
      ],
      ssMethods: [
        "none",
        "aes-128-gcm",
        "aes-192-gcm",
        "aes-256-gcm",
        "chacha20-ietf-poly1305",
        "xchacha20-ietf-poly1305",
        "2022-blake3-aes-128-gcm",
        "2022-blake3-aes-256-gcm",
        "2022-blake3-chacha20-poly1305"
      ]
    }
  },
  computed: {
    plugin: {
      get(): string {
        return this.$props.data.plugin ?? ''
      },
      set(v:string | null) {
        if (v && v.length > 0) {
          this.$props.data.plugin = v
        } else {
          delete this.$props.data.plugin
          delete this.$props.data.plugin_opts
        }
      }
    },
    pluginOptions: {
      get(): string {
        return this.$props.data.plugin_opts ?? ''
      },
      set(v:string) {
        if (v.length > 0) {
          this.$props.data.plugin_opts = v
        } else {
          delete this.$props.data.plugin_opts
        }
      }
    }
  },
  methods: {
    changeMethod(ssMethod :string) {
      if (ssMethod.startsWith('2022')) {
        this.$props.data.password = ssMethod == "2022-blake3-aes-128-gcm" ? RandomUtil.randomShadowsocksPassword(16) : RandomUtil.randomShadowsocksPassword(32)
      } else if (ssMethod == 'none') {
        delete this.$props.data.password
      } else {
        this.$props.data.password = RandomUtil.randomSeq(10)
      }
    }
  },
  components: { Network, UoT }
}
</script>
