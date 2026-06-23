<template>
  <v-card :subtitle="$t('types.failover.title')">
    <v-row>
      <v-col cols="12">
        <div class="text-caption mb-1">{{ $t('types.failover.members') }}</div>
        <v-list density="compact" class="pa-0 bg-transparent">
          <v-list-item v-for="(m, i) in members" :key="i" class="px-0">
            <v-row no-gutters align="center">
              <v-col>
                <v-select
                  :model-value="members[i]"
                  @update:modelValue="setMember(i, $event)"
                  :items="memberItems(i)"
                  :label="i === 0 ? $t('types.failover.primary') : $t('types.failover.backup') + ' ' + i"
                  density="compact"
                  hide-details
                ></v-select>
              </v-col>
              <v-col cols="auto" class="d-flex">
                <v-btn icon="mdi-arrow-up" size="small" variant="text" :disabled="i === 0" @click="move(i, -1)"></v-btn>
                <v-btn icon="mdi-arrow-down" size="small" variant="text" :disabled="i === members.length - 1" @click="move(i, 1)"></v-btn>
                <v-btn icon="mdi-delete" size="small" variant="text" color="warning" @click="remove(i)"></v-btn>
              </v-col>
            </v-row>
          </v-list-item>
        </v-list>
        <v-btn variant="tonal" size="small" prepend-icon="mdi-plus" @click="add">{{ $t('types.failover.addMember') }}</v-btn>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <v-text-field
          v-model="probeTarget"
          :label="$t('types.failover.probeTarget')"
          :placeholder="defaultTarget"
          persistent-placeholder
          hide-details
        >
          <template #append-inner>
            <SettingInfo :text="$t('types.failover.probeTargetHint')" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field
          v-model.number="interval"
          :label="$t('types.failover.interval')"
          type="number"
          min="5"
          placeholder="30"
          persistent-placeholder
          :suffix="$t('date.s')"
          hide-details
        >
          <template #append-inner>
            <SettingInfo :text="$t('types.failover.intervalHint')" />
          </template>
        </v-text-field>
      </v-col>
      <v-col cols="12" sm="6">
        <v-text-field
          v-model.number="hysteresis"
          :label="$t('types.failover.hysteresis')"
          type="number"
          min="1"
          placeholder="2"
          persistent-placeholder
          hide-details
        >
          <template #append-inner>
            <SettingInfo :text="$t('types.failover.hysteresisHint')" />
          </template>
        </v-text-field>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12" sm="6">
        <v-switch v-model="enabled" color="primary" :label="$t('types.failover.enabled')" hide-details></v-switch>
      </v-col>
      <v-col cols="12" sm="6">
        <v-switch v-model="data.interrupt_exist_connections" color="primary" :label="$t('types.lb.interruptConn')" hide-details></v-switch>
      </v-col>
    </v-row>
  </v-card>
</template>

<script lang="ts">
import SettingInfo from '@/components/SettingInfo.vue'

export default {
  components: { SettingInfo },
  props: ['data', 'tags'],
  created() {
    if (!Array.isArray(this.$props.data.outbounds)) this.$props.data.outbounds = []
    if (!this.$props.data.failover) {
      this.$props.data.failover = { enabled: true, probe_target: '', interval: '30s', hysteresis: 2 }
    }
  },
  computed: {
    defaultTarget(): string { return 'https://www.gstatic.com/generate_204' },
    members(): string[] { return (this.$props.data.outbounds ?? []) as string[] },
    probeTarget: {
      get(): string { return this.$props.data.failover.probe_target ?? '' },
      set(v: string) { this.$props.data.failover.probe_target = v }
    },
    interval: {
      get(): number {
        const s = this.$props.data.failover.interval || '30s'
        return parseInt(String(s).replace('s', '')) || 30
      },
      set(v: number) { this.$props.data.failover.interval = (v >= 5 ? v : 30) + 's' }
    },
    hysteresis: {
      get(): number { return this.$props.data.failover.hysteresis || 2 },
      set(v: number) { this.$props.data.failover.hysteresis = v >= 1 ? v : 2 }
    },
    enabled: {
      get(): boolean { return this.$props.data.failover.enabled !== false },
      set(v: boolean) { this.$props.data.failover.enabled = v }
    },
  },
  methods: {
    memberItems(i: number): string[] {
      const others = this.$props.data.outbounds.filter((_: string, idx: number) => idx !== i)
      return this.$props.tags.filter((t: string) => t === this.$props.data.outbounds[i] || !others.includes(t))
    },
    setMember(i: number, v: string) { this.$props.data.outbounds.splice(i, 1, v) },
    move(i: number, dir: number) {
      const arr = this.$props.data.outbounds
      const t = i + dir
      if (t < 0 || t >= arr.length) return
      const [m] = arr.splice(i, 1)
      arr.splice(t, 0, m)
    },
    remove(i: number) { this.$props.data.outbounds.splice(i, 1) },
    add() { this.$props.data.outbounds.push('') },
  },
}
</script>
