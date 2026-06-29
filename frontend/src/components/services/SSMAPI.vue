<template>
  <v-card style="padding: 8px;" rounded="xl" class="border">
    <v-card-subtitle>Shadowsocks API
      <v-chip color="primary" density="compact" variant="elevated" @click="add_server"><v-icon icon="mdi-plus" /></v-chip>
    </v-card-subtitle>
    <v-row>
      <v-col cols="12" sm="8">
        <v-text-field
          v-model="cachePath"
          :label="$t('singbox.cachePath')"
          placeholder="/var/lib/sing-box/ssm-api.cache"
          hide-details>
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="cache_path" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
    <v-row v-for="(server, index) in servers">
      <v-col cols="auto" align-self="center" justify-self="center">
        <v-icon @click="del_server(index)" color="error" icon="mdi-delete" />
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field
          :label="$t('transport.path')"
          hide-details
          @input="update_key(index,$event.target.value)"
          v-model="server.name">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="ssm_servers" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          :label="$t('objects.inbound')"
          hide-details
          :items="ssTags"
          @update:model-value="update_value(index,$event)"
          v-model="server.value">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="ssm_servers" />
          </template>
        </v-select>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import FieldHint from '@/components/FieldHint.vue'

type Server = {
  name: string
  value: string
}
export default {
  props: {
    data: { type: Object, required: true },
    ssTags: { type: Array, default: () => [] },
    fieldHints: { type: Object, default: () => ({}) },
  },
  data() {
    return {}
  },
  methods: {
    add_server() {
      const firstTag = typeof this.ssTags[0] === 'string' ? this.ssTags[0] : ''
      this.servers = [...this.servers, {name: "/ss" + this.servers.length, value: firstTag}]
    },
    del_server(i:number) {
      let h = this.servers
      h.splice(i,1)
      this.servers = h
    },
    update_key(i:number,k:string) {
      let h = this.servers
      h[i].name = k
      this.servers = h
    },
    update_value(i:number,v:string) {
      let h = this.servers
      h[i].value = v
      this.servers = h
    },
  },
  computed: {
    cachePath: {
      get(): string {
        return this.$props.data.cache_path ?? ''
      },
      set(v:string) {
        if (v.length > 0) {
          this.$props.data.cache_path = v
        } else {
          delete this.$props.data.cache_path
        }
      }
    },
    servers: {
      get() :Server[] {
        let servers: Server[] = []
        const h = this.$props.data.servers
        if (h) {
          Object.keys(h).forEach(key => {
            if (Array.isArray(h[key])){
              h[key].forEach((v:string) => servers.push({ name: key, value: v }))
            } else {
              servers.push({ name: key, value: h[key] })
            }
          })
        }
        return servers
      },
      set(v:Server[]) {
        if (v.length>0) {
          let servers:any = {}
          v.forEach((h:Server) => {
            if (servers[h.name]) {
              if (Array.isArray(servers[h.name])) {
                servers[h.name].push(h.value)
              } else {
                servers[h.name] = [servers[h.name], h.value]
              }
            } else {
              servers[h.name] = h.value
            }
          })
          this.$props.data.servers = servers
        } else {
          this.$props.data.servers = undefined
        }
      }
    }
  },
  components: { FieldHint },
}
</script>
