<template>
  <v-card subtitle="API">
    <v-row>
      <v-col cols="12" sm="6">
        <v-text-field
          :label="$t('types.api.secret')"
          hide-details
          type="password"
          v-model="secret">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="secret" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-combobox
          :label="$t('types.api.accessControlAllowOrigin')"
          hide-details
          multiple
          chips
          v-model="allowOrigin">
          <template #append-inner>
            <FieldHint :field-hints="fieldHints" field="access_control_allow_origin" />
          </template>
        </v-combobox>
      </v-col>
      <v-col cols="12" sm="6">
        <div class="d-flex align-center ga-1">
          <v-switch
            color="primary"
            :label="$t('types.api.accessControlAllowPrivateNetwork')"
            hide-details
            v-model="allowPrivateNetwork" />
          <FieldHint :field-hints="fieldHints" field="access_control_allow_private_network" />
        </div>
      </v-col>
    </v-row>
    <v-card-title>
      {{ $t('types.api.dashboard') }}
      <v-chip color="primary" density="compact" variant="elevated" @click="toggleDashboard">
        <v-icon :icon="dashboardEnabled ? 'mdi-delete' : 'mdi-plus'" />
      </v-chip>
    </v-card-title>
    <template v-if="data.dashboard != undefined && data.dashboard !== false">
      <v-row>
        <v-col cols="12" sm="6">
          <v-text-field
            :label="$t('types.api.dashboardPath')"
            hide-details
            :model-value="dashboardField('path')"
            @update:model-value="setDashboardField('path', $event)">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="dashboard_path" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            :label="$t('types.api.dashboardDownloadURL')"
            hide-details
            :model-value="dashboardField('download_url')"
            @update:model-value="setDashboardField('download_url', $event)">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="dashboard_download_url" />
            </template>
          </v-text-field>
        </v-col>
        <v-col cols="12" sm="6">
          <v-select
            :label="$t('types.api.dashboardHTTPClient')"
            hide-details
            :items="httpClientTags"
            clearable
            :model-value="dashboardField('http_client')"
            @update:model-value="setDashboardField('http_client', $event)">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="dashboard_http_client" />
            </template>
          </v-select>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            :label="$t('types.api.dashboardUpdateInterval')"
            hide-details
            :model-value="dashboardField('update_interval')"
            @update:model-value="setDashboardField('update_interval', $event)">
            <template #append-inner>
              <FieldHint :field-hints="fieldHints" field="dashboard_update_interval" />
            </template>
          </v-text-field>
        </v-col>
      </v-row>
    </template>
  </v-card>
</template>

<script lang="ts">
import FieldHint from '@/components/FieldHint.vue'
import Data from '@/store/modules/data'

// `dashboard` is a tri-state in core JSON: bool (enabled), string (enabled +
// path), or object (full form). Reads normalize it, writes always store the
// object form because core accepts it and it keeps partial edits lossless.
export default {
  props: {
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  components: { FieldHint },
  computed: {
    httpClientTags(): string[] {
      return (Data().config?.http_clients ?? [])
        .map((c: any) => c.tag)
        .filter((t: any) => typeof t === 'string' && t.length > 0)
    },
    dashboardEnabled(): boolean {
      const d = this.$props.data.dashboard
      return d != undefined && d !== false
    },
    secret: {
      get(): string {
        return this.$props.data.secret ?? ''
      },
      set(v: string) {
        if (v.length > 0) this.$props.data.secret = v
        else delete this.$props.data.secret
      },
    },
    allowOrigin: {
      get(): string[] {
        return this.$props.data.access_control_allow_origin ?? []
      },
      set(v: string[]) {
        if (v.length > 0) this.$props.data.access_control_allow_origin = v
        else delete this.$props.data.access_control_allow_origin
      },
    },
    allowPrivateNetwork: {
      get(): boolean {
        return this.$props.data.access_control_allow_private_network ?? false
      },
      set(v: boolean) {
        if (v) this.$props.data.access_control_allow_private_network = true
        else delete this.$props.data.access_control_allow_private_network
      },
    },
  },
  methods: {
    toggleDashboard() {
      if (this.dashboardEnabled) delete this.$props.data.dashboard
      else this.$props.data.dashboard = { enabled: true }
    },
    dashboardField(key: string): string {
      const d = this.$props.data.dashboard
      if (d == undefined || d === false || d === true) return ''
      if (typeof d === 'string') return key === 'path' ? d : ''
      return d[key] ?? ''
    },
    setDashboardField(key: string, v: unknown) {
      const d = this.$props.data.dashboard
      let obj: Record<string, any>
      if (d != undefined && d !== false && typeof d === 'object') obj = { ...d }
      else if (typeof d === 'string') obj = { enabled: true, path: d }
      else obj = { enabled: true }
      if (v !== '' && v != undefined) obj[key] = v
      else delete obj[key]
      this.$props.data.dashboard = obj
    },
  },
}
</script>