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
        <v-col cols="12" sm="6" md="4">
          <v-select
            hide-details
            :label="$t('types.amnezia.preset')"
            :items="presetItems"
            v-model="selectedPreset"
            @update:modelValue="applyPreset">
          </v-select>
        </v-col>
        <v-col cols="12" sm="6" md="4" class="d-flex align-center">
          <v-btn
            color="primary"
            variant="tonal"
            size="small"
            prepend-icon="mdi-dice-multiple"
            :loading="randomizing"
            :disabled="randomizing"
            @click="randomize">
            {{ $t('types.amnezia.randomize') }}
          </v-btn>
        </v-col>
      </v-row>
      <v-row v-if="presetDescription">
        <v-col cols="12" class="pt-0">
          <div class="text-caption text-medium-emphasis">{{ presetDescription }}</div>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="6" sm="4" md="2">
          <v-text-field type="number" min="0" :label="$t('types.amnezia.jc')" v-model.number="amnezia.jc"
            :error-messages="errorText('jc')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-text-field type="number" min="0" :label="$t('types.amnezia.jmin')" v-model.number="amnezia.jmin"
            :error-messages="errorText('jmin')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2">
          <v-text-field type="number" min="0" :label="$t('types.amnezia.jmax')" v-model.number="amnezia.jmax"
            :error-messages="errorText('jmax')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" :label="$t('types.amnezia.s1')" v-model.number="amnezia.s1"
            :error-messages="errorText('s1')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" :label="$t('types.amnezia.s2')" v-model.number="amnezia.s2"
            :error-messages="errorText('s2')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" :label="$t('types.amnezia.s3')" v-model.number="amnezia.s3"
            :error-messages="errorText('s3')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="4" md="2" v-if="full">
          <v-text-field type="number" min="0" :label="$t('types.amnezia.s4')" v-model.number="amnezia.s4"
            :error-messages="errorText('s4')"></v-text-field>
        </v-col>
      </v-row>
      <v-row v-if="full">
        <v-col cols="6" sm="3">
          <v-text-field :label="$t('types.amnezia.h1')" v-model="h1"
            :error-messages="errorText('h1')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="3">
          <v-text-field :label="$t('types.amnezia.h2')" v-model="h2"
            :error-messages="errorText('h2')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="3">
          <v-text-field :label="$t('types.amnezia.h3')" v-model="h3"
            :error-messages="errorText('h3')"></v-text-field>
        </v-col>
        <v-col cols="6" sm="3">
          <v-text-field :label="$t('types.amnezia.h4')" v-model="h4"
            :error-messages="errorText('h4')"></v-text-field>
        </v-col>
      </v-row>
      <RecommendedValues
        :model="amnezia"
        :specs="hintSpecs"
        :title="$t('types.amnezia.hints.title')"
        @apply="applyHint" />
      <div class="text-caption text-medium-emphasis mt-2">
        {{ $t('types.amnezia.hints.cpu') }}
      </div>
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
          <v-col cols="12" md="6">
            <v-text-field hide-details :label="$t('types.amnezia.headerProtectionKey')" v-model="headerProtectionKey"
              :error-messages="errorText('header_protection_key')"></v-text-field>
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field hide-details :label="$t('types.amnezia.contentPaddingAddition')" v-model="contentPaddingAddition"
              :error-messages="errorText('content_padding_addition')"></v-text-field>
          </v-col>
        </v-row>
      </template>
      <v-row>
        <v-col cols="12" sm="6" md="4">
          <v-text-field hide-details :label="$t('types.amnezia.rekeyAfterTime')" v-model="rekeyAfterTime"
            :error-messages="errorText('rekey_after_time')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field hide-details :label="$t('types.amnezia.rekeyTimeout')" v-model="rekeyTimeout"
            :error-messages="errorText('rekey_timeout')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field hide-details :label="$t('types.amnezia.rejectAfterTime')" v-model="rejectAfterTime"
            :error-messages="errorText('reject_after_time')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field hide-details :label="$t('types.amnezia.keepaliveTimeout')" v-model="keepaliveTimeout"
            :error-messages="errorText('keepalive_timeout')"></v-text-field>
        </v-col>
        <v-col cols="12" sm="6" md="4">
          <v-text-field hide-details :label="$t('types.amnezia.maxHandshakeAttempts')" v-model="maxHandshakeAttempts"
            :error-messages="errorText('max_handshake_attempts')"></v-text-field>
        </v-col>
      </v-row>
      <div class="text-caption text-medium-emphasis mt-1">
        {{ $t('types.amnezia.hints.timings') }}
      </div>
    </v-card-text>
  </v-card>
</template>

<script lang="ts">
import HttpUtils from '@/plugins/httputil'
import { validateAmnezia } from '@/utils/amneziaValidation'
import { amneziaPresetCatalog, applyAmneziaPreset, applyAmneziaTimingDefaults, detectAmneziaPreset, type AmneziaPresetId } from '@/components/presets/amneziaPresets'
import RecommendedValues from '@/components/recommendations/RecommendedValues.vue'
import { applyRecommendation, type ResolvedRecommendation } from '@/utils/recommendations'
export default {
  components: { RecommendedValues },
  props: {
    data: { type: Object, required: true },
    // full = WireGuard (s-params + headers + header protection), else WARP
    // (jc/jmin/jmax + 3.0 timings only — WARPAmnezia has no s/h fields)
    full: { type: Boolean, default: false },
  },
  data() {
    return {
      randomizing: false,
      selectedPreset: detectAmneziaPreset(this.data?.amnezia) as AmneziaPresetId,
    }
  },
  computed: {
    enabled: {
      get(): boolean { return this.data.amnezia != undefined },
      set(v: boolean) {
        if (v) {
          // Junk defaults follow the Balanced preset; headers are filled by
          // the server-side crypto/rand generator (the legacy h1:1..h4:4
          // defaults were vanilla WireGuard message types and provided no
          // header masking at all). AWG 3.0 timing ranges follow the official
          // client defaults so the profile is a full 3.0 one out of the box.
          this.data.amnezia = {}
          applyAmneziaPreset(this.data.amnezia, 'balanced')
          applyAmneziaTimingDefaults(this.data.amnezia)
          this.selectedPreset = 'balanced'
          this.fetchRandom(false)
        } else {
          delete this.data.amnezia
        }
      },
    },
    presetItems(): Array<{ title: string, value: string }> {
      return amneziaPresetCatalog.map(preset => ({
        title: this.$t(preset.titleKey),
        value: preset.id,
      }))
    },
    presetDescription(): string {
      const preset = amneziaPresetCatalog.find(item => item.id === this.selectedPreset)
      return preset ? this.$t(preset.descriptionKey) : ''
    },
    hintSpecs(): any[] {
      return [
        {
          id: 'mobile-jmax',
          label: this.$t('types.amnezia.hints.mobileJmax'),
          description: this.$t('types.amnezia.hints.mobileJmaxDescription'),
          path: 'jmax',
          value: 70,
          onlyIfEmpty: false,
          when: () => typeof this.amnezia?.jmax === 'number' && this.amnezia.jmax > 70,
        },
        {
          id: 'mtu-loss',
          label: this.$t('types.amnezia.hints.mtu'),
          description: this.$t('types.amnezia.hints.mtuDescription'),
          path: 'clientMtuHint',
          value: 1280,
          onlyIfEmpty: false,
          // Client-side MTU lives on the endpoint (options.mtu feeds the
          // rendered device config); only offer it where that field exists.
          when: () => this.full && typeof this.data?.mtu !== 'undefined',
        },
      ]
    },
    amnezia(): any {
      return this.data.amnezia
    },
    validationErrors(): Record<string, string> {
      return validateAmnezia(this.data.amnezia, { warp: !this.full })
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
    headerProtectionKey: {
      get(): string { return this.amnezia.header_protection_key != undefined ? String(this.amnezia.header_protection_key) : '' },
      set(v: string) {
        const trimmed = (v ?? '').trim()
        if (trimmed.length === 0) {
          delete this.amnezia.header_protection_key
        } else {
          this.amnezia.header_protection_key = trimmed
        }
      },
    },
    contentPaddingAddition: {
      get(): string { return this.amnezia.content_padding_addition != undefined ? String(this.amnezia.content_padding_addition) : '' },
      set(v: string) { this.setRange('content_padding_addition', v) },
    },
    rekeyAfterTime: {
      get(): string { return this.amnezia.rekey_after_time != undefined ? String(this.amnezia.rekey_after_time) : '' },
      set(v: string) { this.setRange('rekey_after_time', v) },
    },
    rekeyTimeout: {
      get(): string { return this.amnezia.rekey_timeout != undefined ? String(this.amnezia.rekey_timeout) : '' },
      set(v: string) { this.setRange('rekey_timeout', v) },
    },
    rejectAfterTime: {
      get(): string { return this.amnezia.reject_after_time != undefined ? String(this.amnezia.reject_after_time) : '' },
      set(v: string) { this.setRange('reject_after_time', v) },
    },
    keepaliveTimeout: {
      get(): string { return this.amnezia.keepalive_timeout != undefined ? String(this.amnezia.keepalive_timeout) : '' },
      set(v: string) { this.setRange('keepalive_timeout', v) },
    },
    maxHandshakeAttempts: {
      get(): string { return this.amnezia.max_handshake_attempts != undefined ? String(this.amnezia.max_handshake_attempts) : '' },
      set(v: string) { this.setRange('max_handshake_attempts', v) },
    },
  },
  watch: {
    // The endpoint modal reuses this component across opens; re-derive the
    // preset selection whenever another endpoint's options arrive.
    'data.amnezia': {
      handler(value: Record<string, unknown> | undefined) {
        this.selectedPreset = detectAmneziaPreset(value)
      },
      deep: false,
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
    errorText(key: string): string[] {
      const errKey = this.validationErrors[key]
      if (!errKey) return []
      return [this.$t('types.amnezia.errors.' + errKey)]
    },
    applyPreset(id: AmneziaPresetId) {
      this.selectedPreset = id
      if (this.data.amnezia) applyAmneziaPreset(this.data.amnezia, id)
    },
    applyHint(spec: ResolvedRecommendation<Record<string, unknown>>) {
      if (spec.id === 'mtu-loss') {
        // MTU lives on the endpoint object, not inside amnezia options.
        this.data.mtu = 1280
        return
      }
      applyRecommendation(this.amnezia, spec, { model: this.amnezia }, { force: true })
    },
    // Decision 1: Randomize always regenerates H1-H4; junk parameters only
    // when the Balanced preset is active. WARP (full=false) has no H1-H4 in
    // its schema, so only junk is randomized there.
    async randomize() {
      await this.fetchRandom(this.selectedPreset === 'balanced' || !this.full)
    },
    // Server-generated parameters: one source of truth, crypto/rand instead
    // of Math.random. includeJunk mirrors the Balanced-preset decision and is
    // wired up by stage 2 (presets); the manual Randomize button touches
    // headers only (never for WARP, whose schema has no headers).
    async fetchRandom(includeJunk: boolean) {
      if (this.randomizing) return
      this.randomizing = true
      try {
        const query = includeJunk ? { preset: 'balanced' } : undefined
        const msg = await HttpUtils.get('api/awg/obfuscation/random', query)
        if (msg.success && msg.obj && this.data.amnezia) {
          if (this.full) {
            this.amnezia.h1 = msg.obj.h1
            this.amnezia.h2 = msg.obj.h2
            this.amnezia.h3 = msg.obj.h3
            this.amnezia.h4 = msg.obj.h4
          }
          if (includeJunk && msg.obj.jc !== undefined) {
            this.amnezia.jc = msg.obj.jc
            this.amnezia.jmin = msg.obj.jmin
            this.amnezia.jmax = msg.obj.jmax
          }
        }
      } finally {
        this.randomizing = false
      }
    },
  },
}
</script>
