<template>
  <v-card subtitle="Inline">
    <v-card :subtitle="$t('types.provider.outbounds')" class="mt-2">
      <v-card-text>
        <v-textarea
          v-model="outboundsJson"
          :error-messages="outboundsError ? [outboundsError] : []"
          :hint="$t('types.provider.outboundsHint')"
          persistent-hint
          variant="outlined"
          auto-grow
          rows="10"
          :style="{ 'font-family': 'monospace' }">
        </v-textarea>
      </v-card-text>
    </v-card>
    <v-row class="mt-1">
      <v-col cols="12" sm="6" md="4">
        <v-switch
          color="primary"
          hide-details
          :label="$t('types.provider.removeEmojis')"
          v-model="data.remove_emojis">
        </v-switch>
      </v-col>
    </v-row>
    <HealthCheck :data="data" />
  </v-card>
</template>

<script lang="ts">
import HealthCheck from './HealthCheck.vue'

export default {
  props: { data: { type: Object, required: true } },
  components: { HealthCheck },
  data() {
    return {
      outboundsError: "",
    }
  },
  created() {
    if (!Array.isArray(this.$props.data.outbounds)) this.$props.data.outbounds = []
  },
  computed: {
    outboundsJson: {
      get(): string {
        return JSON.stringify(this.$props.data.outbounds ?? [], null, 2)
      },
      set(v: string) {
        if (v.trim().length === 0) {
          this.$props.data.outbounds = []
          this.outboundsError = ""
          return
        }
        try {
          const parsed = JSON.parse(v)
          if (!Array.isArray(parsed)) {
            this.outboundsError = this.$t('types.provider.outboundsArrayError')
            return
          }
          this.$props.data.outbounds = parsed
          this.outboundsError = ""
        } catch (e: any) {
          this.outboundsError = e.message
        }
      }
    },
  },
}
</script>
