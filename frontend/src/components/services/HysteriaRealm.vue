<template>
  <v-card subtitle="Hysteria Realm">
    <v-card-title>
      {{ $t('types.hyRealm.users') }}
      <v-chip color="primary" density="compact" variant="elevated" @click="addUser"><v-icon icon="mdi-plus" /></v-chip>
    </v-card-title>
    <v-card v-for="(user, index) in (data.users || [])" :key="index" class="border" style="margin: 4px; padding: 8px;" rounded="xl">
      <v-row>
        <v-col cols="auto" align-self="center">
          <v-icon @click="delUser(index)" color="error" icon="mdi-delete" />
        </v-col>
        <v-col cols="12" sm="4">
          <v-text-field :label="$t('types.hyRealm.userName')" hide-details v-model="user.name">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="users" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="4">
          <v-text-field :label="$t('types.hyRealm.userToken')" hide-details type="password" v-model="user.token">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="users" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="3">
          <v-text-field
            :label="$t('types.hyRealm.maxRealms')"
            hide-details
            type="number"
            min="0"
            :model-value="user.max_realms ?? ''"
            @update:model-value="setMaxRealms(user, $event)">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="max_realms" />
            </template>
          </v-text-field>
        </v-col>
      </v-row>
    </v-card>
    <v-card-title>{{ $t('types.hyRealm.http2') }}</v-card-title>
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-text-field :label="$t('types.hyRealm.idleTimeout')" hide-details clearable v-model="idleTimeout">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="idle_timeout" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field :label="$t('types.hyRealm.keepAlivePeriod')" hide-details clearable v-model="keepAlivePeriod">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="keep_alive_period" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field :label="$t('types.hyRealm.streamReceiveWindow')" hide-details clearable v-model="streamReceiveWindow">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="stream_receive_window" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field :label="$t('types.hyRealm.connectionReceiveWindow')" hide-details clearable v-model="connectionReceiveWindow">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="connection_receive_window" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field :label="$t('types.hyRealm.maxConcurrentStreams')" hide-details type="number" min="0" clearable v-model="maxConcurrentStreams">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="max_concurrent_streams" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import FieldHint from '@/components/FieldHint.vue'

export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  components: { FieldHint },
  methods: {
    addUser() {
      if (!this.$props.data.users) this.$props.data.users = []
      this.$props.data.users.push({ name: '', token: '' })
    },
    delUser(i: number | string) {
      this.$props.data.users?.splice(Number(i), 1)
    },
    // Empty means "not set": the core treats an omitted field as its default,
    // so clearing a field removes the key instead of writing "" / 0.
    setOptional(key: string, v: unknown) {
      if (v !== '' && v != undefined && !(typeof v === 'number' && Number.isNaN(v))) {
        this.$props.data[key] = v
      } else {
        delete this.$props.data[key]
      }
    },
    setMaxRealms(user: Record<string, any>, v: unknown) {
      const n = Number(v)
      if (v === '' || v == undefined || Number.isNaN(n) || n <= 0) delete user.max_realms
      else user.max_realms = n
    },
  },
  computed: {
    idleTimeout: {
      get(): string { return this.$props.data.idle_timeout ?? '' },
      set(v: string) { this.setOptional('idle_timeout', v) },
    },
    keepAlivePeriod: {
      get(): string { return this.$props.data.keep_alive_period ?? '' },
      set(v: string) { this.setOptional('keep_alive_period', v) },
    },
    streamReceiveWindow: {
      get(): string { return this.$props.data.stream_receive_window ?? '' },
      set(v: string) { this.setOptional('stream_receive_window', v) },
    },
    connectionReceiveWindow: {
      get(): string { return this.$props.data.connection_receive_window ?? '' },
      set(v: string) { this.setOptional('connection_receive_window', v) },
    },
    maxConcurrentStreams: {
      get(): string { return this.$props.data.max_concurrent_streams ?? '' },
      set(v: string) { this.setOptional('max_concurrent_streams', v) },
    },
  },
}
</script>