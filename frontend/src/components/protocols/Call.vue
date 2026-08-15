<template>
  <v-card subtitle="Call">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.call.platform')"
          :items="platforms"
          v-model="data.platform">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-select
          clearable
          hide-details
          :label="$t('types.call.mode')"
          :items="['dc', 'video']"
          v-model="data.mode">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" hide-details :label="$t('types.call.readBuffer')" v-model.number="data.read_buffer"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" hide-details :label="$t('types.call.maxBufferedAmount')" v-model.number="data.max_buffered_amount"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6" md="4">
        <v-text-field type="number" hide-details :label="$t('types.call.memoryLimit')" v-model.number="data.memory_limit"></v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <v-text-field
          hide-details
          :label="$t('types.call.joinLink')"
          :hint="$t('types.call.joinLinkHint')"
          persistent-hint
          v-model="data.join_link"></v-text-field>
      </v-col>
    </v-row>

    <!-- inbound-only: dion re-authentication -->
    <v-row v-if="direction == 'in'">
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.call.email')" v-model="data.email"></v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field hide-details :label="$t('types.call.password')" v-model="data.password"></v-text-field>
      </v-col>
    </v-row>

    <!-- cookies -->
    <v-card border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">{{ $t('types.call.cookies') }}</v-card-subtitle>
      <v-card-text>
        <v-row v-for="(cookie, index) in cookies" :key="index" align="center">
          <v-col cols="12" sm="5">
            <v-text-field hide-details :label="$t('types.call.cookieName')" v-model="cookie.name"></v-text-field>
          </v-col>
          <v-col cols="12" sm="5">
            <v-text-field hide-details :label="$t('types.call.cookieValue')" v-model="cookie.value"></v-text-field>
          </v-col>
          <v-col cols="12" sm="2">
            <v-btn icon="mdi-delete" variant="text" color="error" size="small" @click="removeCookie(index)"></v-btn>
          </v-col>
        </v-row>
        <v-btn color="primary" variant="tonal" size="small" prepend-icon="mdi-plus" @click="addCookie">
          {{ $t('types.call.addCookie') }}
        </v-btn>
      </v-card-text>
    </v-card>
    <InboundAdvanced :data="data" :field-hints="fieldHints" />
  </v-card>
</template>

<script lang="ts">
import InboundAdvanced from '@/components/protocols/InboundAdvanced.vue'

export default {
  props: {
    direction: { type: String },
    data: { type: Object, required: true },
    fieldHints: { type: Object, default: () => ({}) },
  },
  components: { InboundAdvanced },
  data() {
    return {
      // Platform backends wired in the kernel transport/call dispatcher
      // (transport/call/config.go). livekit has no dispatch case in 2.6.x.
      platforms: ['dion', 'telemost', 'vk', 'wbstream'],
    }
  },
  computed: {
    cookies(): Array<{ name: string, value?: string }> {
      if (!this.$props.data.cookies) this.$props.data.cookies = []
      return this.$props.data.cookies
    },
  },
  methods: {
    addCookie() {
      this.$props.data.cookies.push({ name: '', value: '' })
    },
    removeCookie(index: number) {
      this.$props.data.cookies.splice(index, 1)
      if (this.$props.data.cookies.length === 0) {
        delete this.$props.data.cookies
      }
    },
  },
}
</script>
