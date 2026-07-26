<template>
  <v-dialog transition="dialog-bottom-transition" width="800">
    <v-card class="rounded-lg">
      <v-card-title>
        {{ $t('actions.' + title) + " " + $t('objects.dnsrule') }}
      </v-card-title>
      <v-divider></v-divider>
      <v-card-text style="padding: 0 16px;">
        <!-- Root-own issue: only reachable when the root is a simple rule that
             the core rejected. Deeper issues are anchored by the nodes below. -->
        <v-alert
          v-if="rootOwnIssues.length > 0"
          type="error"
          variant="tonal"
          density="compact"
          class="mb-3"
          :data-rule-issue="rootPath"
        >
          <div v-for="issue in rootOwnIssues" :key="issue.path + issue.message">{{ issue.message }}</div>
        </v-alert>
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center ga-1">
              <v-switch color="primary" :model-value="logical" :label="$t('rule.logical')" hide-details @update:model-value="onRootLogicalChange"></v-switch>
              <SettingInfo v-if="fieldHint('logical')" :text="fieldHint('logical')" />
            </div>
          </v-col>
          <v-spacer></v-spacer>
          <v-col cols="auto" v-if="logical" justify="center" align="center">
            <v-btn color="primary" @click="ruleData.rules.push(<dnsRule>{})" hide-details>{{ $t('actions.add') + " " + $t('objects.rule') }}</v-btn>
          </v-col>
        </v-row>
        <!-- The modal owns the root child-list anchor (dns.rules[0].rules),
             which the core reports when a logical root has no sub-rules. -->
        <v-alert
          v-if="logical && rootChildListIssues.length > 0"
          type="error"
          variant="tonal"
          density="compact"
          class="mb-3"
          :data-rule-issue="rootChildListPath"
        >
          <div v-for="issue in rootChildListIssues" :key="issue.path + issue.message">{{ issue.message }}</div>
        </v-alert>
        <v-card style="background-color: inherit; margin-bottom: 5px;" v-for="(r, index) in ruleData.rules" :key="nodeKey(r)" v-if="ruleData.type == 'logical'">
          <v-card-subtitle class="d-flex align-center justify-space-between">
            <span>{{ $t('objects.rule') + ' ' + (Number(index)+1) }}</span>
            <v-btn v-if="ruleData.rules.length>1" icon="mdi-delete" size="small" variant="text" :aria-label="$t('actions.del')" @click="ruleData.rules.splice(index,1)" />
          </v-card-subtitle>
          <v-card-text style="padding: 0;">
            <DnsRuleNode
              :node="r"
              :path="topChildPath(Number(index))"
              :issues="conditionIssues"
              :clients="clients"
              :in-tags="inTags"
              :rule-sets="ruleSets"
              @mutate="clearConditionIssues"
              :field-hints="currentFieldHints" />
          </v-card-text>
        </v-card>
        <RuleOptions
          v-else
          :rule="ruleData.rules[0]"
          :clients="clients"
          :inTags="inTags"
          :ruleSets="ruleSets"
          :field-hints="currentFieldHints" />
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-select
              v-model="ruleData.action"
              :items="actions"
              :label="$t('dns.rule.action.title')"
              hide-details>
              <template #append-inner>
                <SettingInfo v-if="fieldHint('action')" :text="fieldHint('action')" />
              </template>
            </v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4" v-if="logical">
            <v-select
              v-model="ruleData.mode"
              :items="['and', 'or']"
              :label="$t('rule.mode')"
              hide-details>
              <template #append-inner>
                <SettingInfo v-if="fieldHint('mode')" :text="fieldHint('mode')" />
              </template>
            </v-select>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center ga-1">
              <v-switch color="primary" v-model="ruleData.invert" :label="$t('rule.invert')" hide-details></v-switch>
              <SettingInfo v-if="fieldHint('invert')" :text="fieldHint('invert')" />
            </div>
          </v-col>
        </v-row>
        <v-row v-if="showDnsRuleRecommendedPreset">
          <v-col cols="12">
            <v-btn
              color="primary"
              prepend-icon="mdi-star-plus"
              variant="tonal"
              @click="applyCurrentDnsRuleRecommendations">
              {{ $t('types.dnsRule.recommendedPreset') }}
            </v-btn>
          </v-col>
        </v-row>
        <v-card :subtitle="$t('dns.rule.action.route')" v-if="['route', 'route-options'].includes(ruleData.action)">
          <v-row v-if="ruleData.action == 'route'">
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.server"
                :items="serverTags"
                :label="$t('dns.server')"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('server')" :text="fieldHint('server')" />
                </template>
              </v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.strategy"
                :items="strategies"
                :label="$t('rule.strategy')"
                clearable
                @click:clear="delete ruleData.strategy"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('strategy')" :text="fieldHint('strategy')" />
                </template>
              </v-select>
            </v-col>
          </v-row>
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <div class="d-flex align-center ga-1">
                <v-switch v-model="ruleData.disable_cache" :label="$t('dns.disableCache')" hide-details></v-switch>
                <SettingInfo v-if="fieldHint('disable_cache')" :text="fieldHint('disable_cache')" />
              </div>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model.number="ruleData.rewrite_ttl" type="number" min="0" :label="$t('dns.rule.action.rewriteTtl')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('rewrite_ttl')" :text="fieldHint('rewrite_ttl')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model="ruleData.client_subnet" :label="$t('dns.rule.action.clientSubnet')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('client_subnet')" :text="fieldHint('client_subnet')" />
                </template>
              </v-text-field>
            </v-col>
          </v-row>
        </v-card>
        <v-card :subtitle="$t('dns.rule.action.reject')" v-if="ruleData.action == 'reject'">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.method"
                :items="[{ title: 'Default', value: 'default' },{ title: 'Drop', value: 'drop'}]"
                :label="$t('rule.method')"
                clearable
                @click:clear="delete ruleData.method"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('method')" :text="fieldHint('method')" />
                </template>
            </v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <div class="d-flex align-center ga-1">
                <v-switch v-model="ruleData.no_drop" :label="$t('rule.noDrop')" hide-details></v-switch>
                <SettingInfo v-if="fieldHint('no_drop')" :text="fieldHint('no_drop')" />
              </div>
            </v-col>
          </v-row>
        </v-card>
        <v-card :subtitle="$t('dns.rule.action.predefined')" v-if="ruleData.action == 'predefined'">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.rcode"
                :items="predefinedRcode"
                :label="$t('dns.rule.action.rcode')"
                clearable
                @click:clear="delete ruleData.rcode"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('rcode')" :text="fieldHint('rcode')" />
                </template>
              </v-select>
            </v-col>
          </v-row>
          <v-row v-if="ruleData.rcode == 'NOERROR'">
            <v-col cols="12" sm="8">
              <v-text-field v-model="answer" :label="$t('dns.rule.action.answer') + ' ' + $t('commaSeparated')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('answer')" :text="fieldHint('answer')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="8">
              <v-text-field v-model="ns" :label="$t('dns.rule.action.ns') + ' ' + $t('commaSeparated')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('answer')" :text="fieldHint('answer')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="8">
              <v-text-field v-model="extra" :label="$t('dns.rule.action.extra') + ' ' + $t('commaSeparated')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('answer')" :text="fieldHint('answer')" />
                </template>
              </v-text-field>
            </v-col>
          </v-row>
        </v-card>
        <v-dialog v-model="rootConfirm.open" max-width="440">
          <v-card class="rounded-xl">
            <v-card-title>{{ $t('rule.convert.toDefaultTitle') }}</v-card-title>
            <v-card-text>{{ $t('rule.convert.toDefaultMessage') }}</v-card-text>
            <v-card-actions>
              <v-spacer />
              <v-btn variant="text" @click="resolveRootConversion(false)">{{ $t('actions.close') }}</v-btn>
              <v-btn color="error" variant="flat" @click="resolveRootConversion(true)">{{ $t('rule.convert.discardConfirm') }}</v-btn>
            </v-card-actions>
          </v-card>
        </v-dialog>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn
          color="primary"
          variant="outlined"
          @click="closeModal"
        >
          {{ $t('actions.close') }}
        </v-btn>
        <v-btn
          color="primary"
          variant="tonal"
          :disabled="saveBlocked"
          :loading="loading"
          @click="saveChanges"
        >
          {{ $t('actions.save') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import { logicalDnsRule, dnsRule, actionDnsRuleKeys } from '@/types/dns'
import RuleOptions from '@/components/DnsRule.vue'
import DnsRuleNode from '@/components/rules/DnsRuleNode.vue'
import { i18n } from '@/locales'
import SettingInfo from '@/components/SettingInfo.vue'
import { applyDnsRuleRecommendedValues, dnsRuleFieldHints, hasDnsRuleRecommendedPreset } from '@/utils/defaultRecommendations'
import { validateRuleConditions, type RuleConditionIssue } from '@/utils/ruleValidation'
import { childListPath, childNodePath, issuesAtPath, nodeKey } from '@/utils/ruleTree'

// The single DNS rule the modal edits is validated as dns.rules[0] (the backend
// wraps it in a one-element array), so every issue path is rooted here.
const ROOT_PATH = 'dns.rules[0]'
export default {
  props: ['visible', 'data', 'index', 'clients', 'inTags', 'serverTags', 'ruleSets'],
  emits: ['close', 'save'],
  data() {
    return {
      title: 'add',
      loading: false,
      conditionIssues: <RuleConditionIssue[]>[],
      validationRequest: 0,
      rootConfirm: <{ open: boolean; resolve: ((confirmed: boolean) => void) | null }>{ open: false, resolve: null },
      ruleData: <any>{
        type: 'logical',
        mode: 'and',
        rules: <dnsRule[]>[{}],
        invert: false,
        action: 'route',
        server: 'local',
      },
      actions: [
        { title: i18n.global.t('dns.rule.action.route'), value: 'route'},
        { title: i18n.global.t('dns.rule.action.routeOptions'), value: 'route-options'},
        { title: i18n.global.t('dns.rule.action.reject'), value: 'reject'},
        { title: i18n.global.t('dns.rule.action.predefined'), value: 'predefined'},
      ],
      strategies: [
        { title: 'Prefer IPv4', value: 'prefer_ipv4' },
        { title: 'Prefer IPv6', value: 'prefer_ipv6' },
        { title: 'IPv4 Only', value: 'ipv4_only' },
        { title: 'IPv6 Only', value: 'ipv6_only' },
      ],
      predefinedRcode: [
        { title: i18n.global.t('dns.rule.action.rcodes.noError'), value: 'NOERROR' },
        { title: i18n.global.t('dns.rule.action.rcodes.formerr'), value: 'FORMERR' },
        { title: i18n.global.t('dns.rule.action.rcodes.servFail'), value: 'SERVFAIL' },
        { title: i18n.global.t('dns.rule.action.rcodes.nxDomain'), value: 'NXDOMAIN' },
        { title: i18n.global.t('dns.rule.action.rcodes.notImp'), value: 'NOTIMP' },
        { title: i18n.global.t('dns.rule.action.rcodes.refused'), value: 'REFUSED' },
      ],
    }
  },
  methods: {
    fieldHint(key: string): string {
      const hintKey = (this.currentFieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    applyCurrentDnsRuleRecommendations() {
      applyDnsRuleRecommendedValues(this.ruleData)
    },
    updateData() {
      this.validationRequest += 1
      this.loading = false
      this.conditionIssues = []
      if (this.$props.index != -1) {
        const newData = JSON.parse(this.$props.data)
        if (newData.type) {
          this.ruleData = newData
        } else {
          this.ruleData = {
            type: 'simple',
            mode: 'and',
            rules: <dnsRule[]>[{}],
          }
          Object.keys(newData).forEach(key => {
            if (actionDnsRuleKeys.includes(key)) {
              this.ruleData[key] = newData[key]
            } else {
              this.ruleData.rules[0][key] = newData[key]
            }
          })
        }
        this.title = 'edit'
      }
      else {
        this.ruleData = <logicalDnsRule>{
            type: 'simple',
            mode: 'and',
            rules: <dnsRule[]>[{}],
            invert: false,
            action: 'route',
            server: this.$props.serverTags[0]?? 'local',
          }
        this.title = 'add'
      }
    },
    closeModal() {
      this.validationRequest += 1
      this.loading = false
      this.$emit('close')
    },
    async saveChanges() {
      this.loading = true
      const validationRequest = ++this.validationRequest
      let newRule = <any>{
        action: this.ruleData.action,
        invert: this.ruleData.invert? this.ruleData.invert : undefined,
      }

      // Filter action data
      switch (newRule.action){
        case 'route':
          newRule.server = this.ruleData.server
          newRule.strategy = this.ruleData.strategy?.length > 0 ? this.ruleData.strategy : undefined
          newRule.disable_cache = this.ruleData.disable_cache? true : undefined
          newRule.rewrite_ttl = this.ruleData.rewrite_ttl > 0 ? this.ruleData.rewrite_ttl : undefined
          newRule.client_subnet = this.ruleData.client_subnet?.length > 0 ? this.ruleData.client_subnet : undefined
          break
        case 'route-options':
          newRule.disable_cache = this.ruleData.disable_cache? true : undefined
          newRule.rewrite_ttl = this.ruleData.rewrite_ttl > 0 ? this.ruleData.rewrite_ttl : undefined
          newRule.client_subnet = this.ruleData.client_subnet?.length > 0 ? this.ruleData.client_subnet : undefined
          break
        case 'reject':
          newRule.method = this.ruleData.method?.length > 0 ? this.ruleData.method : undefined
          newRule.no_drop = this.ruleData.no_drop? true : undefined
          break
        case 'predefined':
          newRule.rcode = this.ruleData.rcode?.length > 0 ? this.ruleData.rcode : undefined
          if (this.ruleData.rcode == 'NOERROR') {
            newRule.answer = this.ruleData.answer
            newRule.ns = this.ruleData.ns
            newRule.extra = this.ruleData.extra
          }
          break
      }

      // Add rules
      if (this.ruleData.type == 'simple'){
        newRule = { ...this.ruleData.rules[0], ...newRule }
      } else {
        newRule.type = 'logical'
        newRule.mode = this.ruleData.mode
        newRule.rules = this.ruleData.rules
      }
      // Validated against the core rather than a local predicate, and against
      // newRule rather than ruleData: newRule is what actually gets persisted,
      // so the pre-normalization editor state is the wrong thing to judge.
      const verdict = await validateRuleConditions('dns', newRule)
      if (validationRequest !== this.validationRequest) return
      // Keep every issue, not just the blocking ones: the recursive nodes anchor
      // each issue to the branch that owns it, so a non-blocking dropped-rule
      // warning still needs to reach its node.
      this.conditionIssues = verdict.issues
      if (!verdict.ok) {
        this.loading = false
        return
      }

      this.$emit('save', newRule)
      this.loading = false
    },
    clearConditionIssues() {
      this.conditionIssues = []
    },
    deleteRule(index:number) {
      this.ruleData.rules.splice(index,1)
      this.clearConditionIssues()
    },
    nodeKey(r: object): string {
      return nodeKey(r)
    },
    // Path of the index-th top-level sub-rule, e.g. dns.rules[0].rules[2].
    // childNodePath already appends `.rules[index]`, so it takes the node path.
    topChildPath(index:number): string {
      return childNodePath(ROOT_PATH, index)
    },
    async onRootLogicalChange(next:boolean | null) {
      if (next) {
        if (!Array.isArray(this.ruleData.rules) || this.ruleData.rules.length === 0) this.ruleData.rules = [{}]
        this.clearConditionIssues()
        this.ruleData.type = 'logical'
        return
      }
      if (Array.isArray(this.ruleData.rules) && this.ruleData.rules.length > 1) {
        const confirmed = await new Promise<boolean>((resolve) => {
          this.rootConfirm.resolve = resolve
          this.rootConfirm.open = true
        })
        if (!confirmed) return
      }
      this.ruleData.rules = [this.ruleData.rules?.[0] ?? {}]
      this.ruleData.type = 'simple'
      this.clearConditionIssues()
    },
    resolveRootConversion(confirmed:boolean) {
      this.rootConfirm.open = false
      const resolve = this.rootConfirm.resolve
      this.rootConfirm.resolve = null
      resolve?.(confirmed)
    }
  },
  computed: {
    // Save is blocked only while the check is in flight. It is never disabled on
    // a local guess: the previous predicate greyed Save out permanently, with no
    // message, for rules the core accepts.
    saveBlocked(): boolean {
      return this.loading
    },
    currentFieldHints(): Record<string, string> {
      return dnsRuleFieldHints()
    },
    rootPath(): string {
      return ROOT_PATH
    },
    rootChildListPath(): string {
      return childListPath(ROOT_PATH)
    },
    // Issues owned by the root rule itself (a rejected simple rule). Excludes the
    // root child-list path, which has its own alert, so the two never double up.
    rootOwnIssues(): RuleConditionIssue[] {
      return issuesAtPath(this.conditionIssues, ROOT_PATH)
    },
    rootChildListIssues(): RuleConditionIssue[] {
      return issuesAtPath(this.conditionIssues, childListPath(ROOT_PATH))
    },
    showDnsRuleRecommendedPreset(): boolean {
      return this.$props.index == -1 && hasDnsRuleRecommendedPreset()
    },
    logical(): boolean {
      return this.ruleData.type == 'logical'
    },
    answer: {
      get() { return this.ruleData.answer?.length > 0 ? this.ruleData.answer.join(',') : "" },
      set(v:string) { this.ruleData.answer = v.length > 0 ? v.split(',') : undefined }
    },
    ns: {
      get() { return this.ruleData.ns?.length > 0 ? this.ruleData.ns.join(',') : "" },
      set(v:string) { this.ruleData.ns = v.length > 0 ? v.split(',') : undefined }
    },
    extra: {
      get() { return this.ruleData.extra?.length > 0 ? this.ruleData.extra.join(',') : "" },
      set(v:string) { this.ruleData.extra = v.length > 0 ? v.split(',') : undefined }
    },
  },
  watch: {
    visible(newValue) {
      if (newValue) {
        this.updateData()
      }
    },
  },
  components: { SettingInfo, RuleOptions, DnsRuleNode }
}

</script>
