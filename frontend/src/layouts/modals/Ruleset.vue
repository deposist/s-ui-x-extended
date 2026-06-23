<template>
  <form-shell
    :dirty="dirty"
    :loading="loading"
    :title="$t('actions.' + title) + ' ' + $t('objects.ruleset')"
    @close="closeModal"
    @save="saveChanges"
  >
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-select
              hide-details
              :label="$t('type')"
              :items="[{title: $t('ruleset.inline'), value: 'inline'}, {title: $t('ruleset.local'), value: 'local'},{ title: $t('ruleset.remote'), value: 'remote'}]"
              @update:model-value="updateType($event)"
              v-model="rule_set.type">
            </v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="rule_set.tag" :label="$t('objects.tag')" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4" v-if="rule_set.type != 'inline'">
            <v-select
              hide-details
              :label="$t('ruleset.format')"
              :items="['source', 'binary']"
              v-model="rule_set.format">
            </v-select>
          </v-col>
        </v-row>
        <v-row v-if="rule_set.type == 'local'">
          <v-col cols="12">
            <v-text-field v-model="rule_set.path" :label="$t('transport.path')" hide-details></v-text-field>
          </v-col>
        </v-row>
        <v-row v-else>
          <v-col cols="12">
            <v-text-field v-model="rule_set.url" label="URL" hide-details></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-select
              hide-details
              :label="$t('objects.outbound')"
              :items="outTags"
              clearable
              @click:clear="delete rule_set.download_detour"
              v-model="rule_set.download_detour">
            </v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model.number="update_intervals" :suffix="$t('date.d')" type="number" min="0" :label="$t('ruleset.interval')" hide-details></v-text-field>
          </v-col>
        </v-row>
        <template v-if="rule_set.type == 'inline'">
          <v-row>
            <v-col cols="12" align="end">
              <v-btn color="primary" variant="tonal" @click="addRule">{{ $t('actions.add') + ' ' + $t('objects.rule') }}</v-btn>
            </v-col>
          </v-row>
          <HeadlessRule
            v-for="(rule, index) in rule_set.rules"
            :key="index"
            :rule="rule"
            deleteable
            @delete="rule_set.rules?.splice(index, 1)"
          />
        </template>
  </form-shell>
</template>

<script lang="ts">
import RandomUtil from '@/plugins/randomUtil'
import { ruleset } from '@/types/rules'
import HeadlessRule from '@/components/HeadlessRule.vue'
import FormShell from '@/components/nexus/drawers/FormShell.vue'
export default {
  props: ['visible', 'data', 'index', 'outTags'],
  emits: ['close', 'save'],
  data() {
    return {
      title: "add",
      loading: false,
      snapshot: '',
      rule_set: <ruleset>{},
    }
  },
  methods: {
    updateData() {
      if (this.$props.index != -1) {
        this.title = "edit"
        this.rule_set = <ruleset>JSON.parse(this.$props.data)
        if (!this.rule_set.type) this.rule_set.type = 'inline'
        if (this.rule_set.type == 'inline' && !this.rule_set.rules) this.rule_set.rules = []
      }
      else {
        this.title = "add"
        this.rule_set = <ruleset>{type: 'local', tag: "rs-" + RandomUtil.randomSeq(3), format: 'binary'}
      }
      this.snapshot = JSON.stringify(this.rule_set)
    },
    updateType(t:string) {
      if (t == 'inline') {
        delete this.rule_set.format
        delete this.rule_set.path
        delete this.rule_set.url
        delete this.rule_set.download_detour
        delete this.rule_set.update_interval
        if (!this.rule_set.rules) this.rule_set.rules = []
      } else if (t == 'local') {
        if (!this.rule_set.format) this.rule_set.format = 'binary'
        delete this.rule_set.url
        delete this.rule_set.download_detour
        delete this.rule_set.update_interval
        delete this.rule_set.rules
      } else {
        if (!this.rule_set.format) this.rule_set.format = 'binary'
        delete this.rule_set.path
        delete this.rule_set.rules
      }
    },
    addRule() {
      if (!this.rule_set.rules) this.rule_set.rules = []
      this.rule_set.rules.push({})
    },
    closeModal() {
      this.$emit('close')
    },
    saveChanges() {
      this.loading = true
      const savedRuleSet = { ...this.rule_set }
      if (savedRuleSet.type == 'inline') {
        delete savedRuleSet.format
        delete savedRuleSet.path
        delete savedRuleSet.url
        delete savedRuleSet.download_detour
        delete savedRuleSet.update_interval
      }
      this.$emit('save', savedRuleSet)
      this.loading = false
    }
  },
  computed: {
    dirty(): boolean {
      return this.snapshot !== '' && JSON.stringify(this.rule_set) !== this.snapshot
    },
    update_intervals: {
      get() { return this.rule_set.update_interval != undefined ? parseInt(this.rule_set.update_interval.replace('d','')) : 0 },
      set(v:number) { this.rule_set.update_interval = v>0 ?  v + 'd' : undefined }
    },
  },
  watch: {
    visible(newValue) {
      if (newValue) {
        this.updateData()
      }
    },
  },
  components: { FormShell, HeadlessRule },
}
</script>
