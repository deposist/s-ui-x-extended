<template>
  <v-card subtitle="Bond">
    <v-row>
      <v-col cols="12">
        <span class="text-caption">{{ $t('types.group.ratioHint') }}</span>
      </v-col>
    </v-row>
    <v-row v-for="(item, index) in items" :key="index">
      <v-col cols="12">
        <v-card variant="outlined">
          <v-card-text>
            <v-row>
              <v-col cols="12" sm="4">
                <v-text-field
                  :label="$t('types.group.downloadRatio')"
                  type="number"
                  min="0"
                  max="100"
                  hide-details
                  v-model.number="item.download_ratio">
                </v-text-field>
              </v-col>
              <v-col cols="12" sm="4">
                <v-text-field
                  :label="$t('types.group.uploadRatio')"
                  type="number"
                  min="0"
                  max="100"
                  hide-details
                  v-model.number="item.upload_ratio">
                </v-text-field>
              </v-col>
              <v-col cols="12" sm="4">
                <v-text-field
                  :label="$t('types.group.count')"
                  type="number"
                  min="0"
                  hide-details
                  v-model.number="item.count">
                </v-text-field>
              </v-col>
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
type BondItem = {
  outbound: { [key: string]: any }
  download_ratio: number
  upload_ratio: number
  count?: number
}

export default {
  props: ['data'],
  data() {
    return {
      outboundText: <string[]>[],
      errors: <boolean[]>[],
    }
  },
  created() {
    this.syncText()
  },
  computed: {
    items(): BondItem[] {
      return this.$props.data.outbounds as BondItem[]
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
        JSON.stringify(o.outbound ?? {}, null, 2)
      )
      this.errors = (this.$props.data.outbounds || []).map(() => false)
    },
    addOutbound() {
      const item = {
        outbound: { type: 'vless', server: '', server_port: 443 },
        download_ratio: 50,
        upload_ratio: 50,
      }
      this.$props.data.outbounds.push(item)
      this.outboundText.push(JSON.stringify(item.outbound, null, 2))
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
        this.$props.data.outbounds[index].outbound = parsed
        this.errors[index] = false
      } catch {
        this.errors[index] = true
      }
    },
  },
}
</script>
