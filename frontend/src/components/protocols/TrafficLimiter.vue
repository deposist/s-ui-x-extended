<template>
  <v-card subtitle="Traffic Limiter">
    <v-row>
      <v-col cols="12" sm="6" md="4">
        <v-select
          hide-details
          :label="$t('types.limiter.strategy')"
          :items="strategyOptions"
          v-model="data.strategy">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="data.strategy != 'users' && data.strategy != 'manager'">
        <v-select
          hide-details
          :label="$t('types.limiter.mode')"
          :items="modeOptions"
          v-model="data.mode">
        </v-select>
      </v-col>
      <v-col cols="12" sm="6" md="4" v-if="data.strategy != 'users' && data.strategy != 'manager'">
        <v-combobox
          hide-details
          :label="$t('types.limiter.total')"
          :placeholder="'10GB'"
          :items="sizePresets"
          v-model="data.total">
        </v-combobox>
      </v-col>
    </v-row>

    <!-- Inline users (strategy: users) -->
    <v-card v-if="data.strategy == 'users'" border density="compact" color="background" style="margin-top: 8px;">
      <v-card-subtitle style="padding-top: 8px;">
        {{ $t('types.limiter.users') }}
        <v-chip color="primary" density="compact" variant="elevated" @click="addUser"><v-icon icon="mdi-plus" /></v-chip>
      </v-card-subtitle>
      <v-card-text>
        <v-row v-for="(u, index) in users" :key="index">
          <v-col cols="12">
            <v-card variant="outlined">
              <v-card-text>
                <v-row>
                  <v-col cols="12" sm="6" md="4">
                    <v-text-field hide-details :label="$t('types.limiter.userName')" v-model="u.name"></v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" md="4">
                    <v-select hide-details :label="$t('types.limiter.strategy')" :items="userStrategyOptions" v-model="u.strategy"></v-select>
                  </v-col>
                  <v-col cols="12" sm="6" md="4">
                    <v-select hide-details :label="$t('types.limiter.mode')" :items="modeOptions" v-model="u.mode"></v-select>
                  </v-col>
                  <v-col cols="12" sm="6" md="4">
                    <v-combobox hide-details :label="$t('types.limiter.total')" :placeholder="'100GB'" :items="sizePresets" v-model="u.total"></v-combobox>
                  </v-col>
                  <v-col cols="12" align="end">
                    <v-btn color="error" variant="tonal" @click="delUser(index)">{{ $t('actions.del') }}</v-btn>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <LimiterRoute :route="route" :tags="tags" />
  </v-card>
</template>

<script lang="ts">
import LimiterRoute from '@/components/protocols/LimiterRoute.vue'
import type { TrafficLimiterUser } from '@/types/outbounds'
import { sizePresets } from '@/types/recommended'

export default {
  props: {
    data: { type: Object, required: true },
    tags: { type: Array, default: () => [] },
  },
  components: { LimiterRoute },
  data() {
    return {
      strategyOptions: ['global', 'users', 'manager'],
      userStrategyOptions: ['global'],
      modeOptions: ['bidirectional', 'download', 'upload'],
      sizePresets,
    }
  },
  created() {
    if (!Array.isArray(this.$props.data.users)) this.$props.data.users = []
    if (!this.$props.data.route) this.$props.data.route = {}
  },
  computed: {
    users(): TrafficLimiterUser[] {
      return this.$props.data.users
    },
    route(): any {
      return this.$props.data.route
    },
  },
  methods: {
    addUser() {
      this.users.push({ name: '', strategy: 'global', mode: 'bidirectional', total: '10GB' })
    },
    delUser(index: number) {
      this.users.splice(index, 1)
    },
  },
}
</script>
