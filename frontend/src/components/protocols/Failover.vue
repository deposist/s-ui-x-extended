<template>
  <v-card subtitle="Failover">
    <v-row>
      <v-col cols="12" sm="6">
        <v-select
          :label="$t('types.group.strategy')"
          :items="strategyOptions"
          v-model="data.strategy"
          hide-details>
        </v-select>
      </v-col>
      <v-col cols="12" sm="6">
        <v-combobox
          :label="$t('types.group.delay')"
          :items="durationPresets"
          hide-details
          placeholder="2s"
          v-model="data.delay">
        </v-combobox>
      </v-col>
    </v-row>
    <v-row v-for="(item, index) in items" :key="index">
      <v-col cols="12">
        <v-card variant="outlined">
          <v-card-text>
            <v-row>
              <v-col cols="12">
                <v-textarea
                  :label="$t('types.group.outboundConfig')"
                  :model-value="getText(index)"
                  @update:model-value="(v:string) => updateOutbound(index, v)"
                  :error-messages="getError(index) ? [$t('types.group.invalidJson')] : []"
                  rows="6"
                  auto-grow
                  :style="{ 'font-family': 'monospace' }"
                ></v-textarea>
              </v-col>
              <v-col cols="12" align="end">
                <v-btn color="error" variant="tonal" @click="removeOutbound(index)">{{ $t('actions.del') }}</v-btn>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" align="center">
        <v-btn color="primary" variant="tonal" @click="addOutbound">{{ $t('types.group.addOutbound') }}</v-btn>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import { durationPresets } from '@/types/recommended'

export default {
  props: ['data'],
  data() {
    return {
      durationPresets,
      strategyOptions: [
        { title: this.$t('types.group.strategySequential'), value: 'sequential' },
        { title: this.$t('types.group.strategyCycle'), value: 'cycle' },
      ],
      outboundText: <string[]>[],
      errors: <boolean[]>[],
    }
  },
  created() {
    this.syncText()
  },
  computed: {
    items(): { [key: string]: any }[] {
      return this.$props.data.outbounds as { [key: string]: any }[]
    },
  },
  methods: {
    getText(i: number): string {
      return this.outboundText[i] ?? ''
    },
    getError(i: number): boolean {
      return this.errors[i] === true
    },
    syncText() {
      this.outboundText = (this.$props.data.outbounds || []).map((o: any) =>
        JSON.stringify(o ?? {}, null, 2)
      )
      this.errors = (this.$props.data.outbounds || []).map(() => false)
    },
    addOutbound() {
      const item = { type: 'vless', tag: '', server: '', server_port: 443 }
      this.$props.data.outbounds.push(item)
      this.outboundText.push(JSON.stringify(item, null, 2))
      this.errors.push(false)
    },
    removeOutbound(index: number) {
      this.$props.data.outbounds.splice(index, 1)
      this.outboundText.splice(index, 1)
      this.errors.splice(index, 1)
    },
    updateOutbound(index: number, value: string) {
      this.outboundText[index] = value
      try {
        const parsed = JSON.parse(value)
        this.$props.data.outbounds[index] = parsed
        this.errors[index] = false
      } catch {
        this.errors[index] = true
      }
    },
  },
}
</script>
