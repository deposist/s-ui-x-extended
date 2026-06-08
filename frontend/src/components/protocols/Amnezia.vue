<template>
  <v-card border density="compact" color="background" style="margin-top: 8px;">
    <v-card-subtitle style="padding-top: 8px;">
      {{ $t('types.amnezia.title') }}
      <v-switch
        class="d-inline-block"
        style="vertical-align: middle; margin-left: 8px;"
        color="primary"
        hide-details
        :label="$t('enable')"
        v-model="enabled">
      </v-switch>
    </v-card-subtitle>
    <v-card-text v-if="enabled">
      <v-row>
        <v-col cols="6" sm="4" md="2">
          <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.jc')" v-model.number="amnezia.jc"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.jmin')" v-model.number="amnezia.jmin"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.jmax')" v-model.number="amnezia.jmax"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.s1')" v-model.number="amnezia.s1"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.s2')" v-model.number="amnezia.s2"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.s3')" v-model.number="amnezia.s3"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.s4')" v-model.number="amnezia.s4"></v-text-field>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="6" sm="3">
          <v-text-field hide-details :label="$t('types.amnezia.h1')" v-model="h1"></v-text-field>
        </v-col>
        <v-col cols="6" sm="3">
          <v-text-field hide-details :label="$t('types.amnezia.h2')" v-model="h2"></v-text-field>
        </v-col>
        <v-col cols="6" sm="3">
          <v-text-field hide-details :label="$t('types.amnezia.h3')" v-model="h3"></v-text-field>
        </v-col>
        <v-col cols="6" sm="3">
          <v-text-field hide-details :label="$t('types.amnezia.h4')" v-model="h4"></v-text-field>
        </v-col>
      </v-row>
      <template v-if="full">
        <v-row>
          <v-col cols="12" md="6">
            <v-text-field hide-details :label="$t('types.amnezia.i1')" v-model="amnezia.i1"></v-text-field>
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field hide-details :label="$t('types.amnezia.i2')" v-model="amnezia.i2"></v-text-field>
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field hide-details :label="$t('types.amnezia.i3')" v-model="amnezia.i3"></v-text-field>
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field hide-details :label="$t('types.amnezia.i4')" v-model="amnezia.i4"></v-text-field>
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field hide-details :label="$t('types.amnezia.i5')" v-model="amnezia.i5"></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" md="4">
            <v-text-field hide-details :label="$t('types.amnezia.j1')" v-model="amnezia.j1"></v-text-field>
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field hide-details :label="$t('types.amnezia.j2')" v-model="amnezia.j2"></v-text-field>
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field hide-details :label="$t('types.amnezia.j3')" v-model="amnezia.j3"></v-text-field>
          </v-col>
          <v-col cols="6" sm="3">
            <v-text-field type="number" min="0" hide-details :label="$t('types.amnezia.itime')" v-model.number="amnezia.itime"></v-text-field>
          </v-col>
        </v-row>
      </template>
    </v-card-text>
  </v-card>
</template>

<script lang="ts">
export default {
  props: {
    data: { type: Object, required: true },
    // full = WireGuard (s-params + junk packets), else WARP (jc/jmin/jmax + h1-h4)
    full: { type: Boolean, default: false },
  },
  computed: {
    enabled: {
      get(): boolean { return this.data.amnezia != undefined },
      set(v: boolean) {
        if (v) {
          this.data.amnezia = { jc: 120, jmin: 23, jmax: 911, h1: 1, h2: 2, h3: 3, h4: 4 }
        } else {
          delete this.data.amnezia
        }
      },
    },
    amnezia(): any {
      return this.data.amnezia
    },
    h1: {
      get(): string { return this.amnezia.h1 != undefined ? String(this.amnezia.h1) : '' },
      set(v: string) { this.setRange('h1', v) },
    },
    h2: {
      get(): string { return this.amnezia.h2 != undefined ? String(this.amnezia.h2) : '' },
      set(v: string) { this.setRange('h2', v) },
    },
    h3: {
      get(): string { return this.amnezia.h3 != undefined ? String(this.amnezia.h3) : '' },
      set(v: string) { this.setRange('h3', v) },
    },
    h4: {
      get(): string { return this.amnezia.h4 != undefined ? String(this.amnezia.h4) : '' },
      set(v: string) { this.setRange('h4', v) },
    },
  },
  methods: {
    // h1-h4 accept either a single integer or a "from-to" range (sing-box Range type)
    setRange(key: string, v: string) {
      const trimmed = (v ?? '').trim()
      if (trimmed.length === 0) {
        delete this.amnezia[key]
        return
      }
      if (/^\d+$/.test(trimmed)) {
        this.amnezia[key] = parseInt(trimmed, 10)
      } else {
        this.amnezia[key] = trimmed
      }
    },
  },
}
</script>
