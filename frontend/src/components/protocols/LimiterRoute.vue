<template>
  <v-card border density="compact" color="background" style="margin-top: 8px;">
    <v-card-subtitle style="padding-top: 8px;">{{ $t('types.limiter.route') }}</v-card-subtitle>
    <v-card-text>
      <v-row>
        <v-col cols="12" sm="6">
          <v-combobox
            clearable
            hide-details
            :label="$t('types.limiter.final')"
            :items="tags"
            v-model="route.final">
          </v-combobox>
        </v-col>
      </v-row>
      <v-row>
        <v-col cols="12">
          <v-textarea
            :label="$t('types.limiter.rules')"
            :model-value="rulesText"
            @update:model-value="updateRules"
            :error-messages="rulesError ? [$t('types.group.invalidJson')] : []"
            rows="4"
            auto-grow
            :style="{ 'font-family': 'monospace' }">
          </v-textarea>
        </v-col>
      </v-row>
    </v-card-text>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['route', 'tags'],
  data() {
    return {
      rulesText: '',
      rulesError: false,
    }
  },
  created() {
    this.syncText()
  },
  methods: {
    syncText() {
      const rules = this.$props.route?.rules
      this.rulesText = Array.isArray(rules) && rules.length > 0 ? JSON.stringify(rules, null, 2) : ''
      this.rulesError = false
    },
    updateRules(value: string) {
      this.rulesText = value
      const trimmed = value.trim()
      if (trimmed === '') {
        delete this.$props.route.rules
        this.rulesError = false
        return
      }
      try {
        const parsed = JSON.parse(value)
        if (!Array.isArray(parsed)) {
          this.rulesError = true
          return
        }
        this.$props.route.rules = parsed
        this.rulesError = false
      } catch {
        this.rulesError = true
      }
    },
  },
}
</script>
