<template>
  <v-card subtitle="VPN Server">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          v-model="data.address"
          :label="$t('types.vpn.address')"
          :placeholder="'10.0.0.1'"
          hide-details>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="optionTimeout">
        <v-text-field
          v-model="data.connect_timeout"
          :label="$t('types.vpn.connectTimeout')"
          :placeholder="'30s'"
          hide-details>
        </v-text-field>
      </v-col>
    </v-row>

    <v-card :subtitle="$t('types.vpn.users')" class="mt-2">
      <v-row v-for="(user, index) in data.users" :key="index" class="px-2">
        <v-col cols="12" sm="4">
          <v-text-field
            v-model="user.address"
            :label="$t('types.vpn.userAddress')"
            :placeholder="'10.0.0.2'"
            hide-details>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            v-model="user.key"
            :label="$t('types.vpn.key')"
            hide-details
            append-inner-icon="mdi-refresh"
            @click:append-inner="user.key = genKey()">
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="2" class="d-flex align-center">
          <v-btn icon="mdi-delete" variant="text" color="error" @click="delUser(index)"></v-btn>
        </v-col>
      </v-row>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn variant="tonal" prepend-icon="mdi-plus" @click="addUser">{{ $t('actions.add') }}</v-btn>
      </v-card-actions>
    </v-card>

    <v-card :subtitle="$t('types.vpn.inbounds')" class="mt-2">
      <v-card-text>
        <v-textarea
          v-model="inboundsJson"
          :error-messages="inboundsError ? [inboundsError] : []"
          :hint="$t('types.vpn.inboundsHint')"
          persistent-hint
          variant="outlined"
          auto-grow
          rows="8"
          :style="{ 'font-family': 'monospace' }">
        </v-textarea>
      </v-card-text>
    </v-card>

    <v-card-actions>
      <v-spacer></v-spacer>
      <v-menu v-model="menu" :close-on-content-click="false" location="start">
        <template v-slot:activator="{ props }">
          <v-btn v-bind="props" hide-details variant="tonal">{{ $t('types.vpn.options') }}</v-btn>
        </template>
        <v-card>
          <v-list>
            <v-list-item>
              <v-switch v-model="optionTimeout" color="primary" :label="$t('types.vpn.connectTimeout')" hide-details></v-switch>
            </v-list-item>
          </v-list>
        </v-card>
      </v-menu>
    </v-card-actions>
  </v-card>
</template>

<script lang="ts">
import RandomUtil from '@/plugins/randomUtil'

export default {
  props: { data: { type: Object, required: true } },
  data() {
    return {
      menu: false,
      inboundsError: "",
    }
  },
  created() {
    if (!Array.isArray(this.$props.data.users)) this.$props.data.users = []
    if (!Array.isArray(this.$props.data.inbounds)) this.$props.data.inbounds = []
  },
  computed: {
    optionTimeout: {
      get(): boolean { return this.$props.data.connect_timeout != undefined },
      set(v: boolean) { this.$props.data.connect_timeout = v ? "30s" : undefined }
    },
    inboundsJson: {
      get(): string {
        return JSON.stringify(this.$props.data.inbounds ?? [], null, 2)
      },
      set(v: string) {
        if (v.trim().length === 0) {
          this.$props.data.inbounds = []
          this.inboundsError = ""
          return
        }
        try {
          const parsed = JSON.parse(v)
          if (!Array.isArray(parsed)) {
            this.inboundsError = this.$t('types.vpn.inboundsArrayError')
            return
          }
          this.$props.data.inbounds = parsed
          this.inboundsError = ""
        } catch (e: any) {
          this.inboundsError = e.message
        }
      }
    },
  },
  methods: {
    genKey(): string {
      return RandomUtil.randomUUID()
    },
    addUser() {
      const octet = (this.$props.data.users?.length ?? 0) + 2
      if (octet > 254) return
      this.$props.data.users.push({ address: "10.0.0." + octet, key: this.genKey() })
    },
    delUser(index: number | string) {
      this.$props.data.users.splice(Number(index), 1)
    },
  },
}
</script>
