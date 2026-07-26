<template>
  <form-shell
    :dirty="dirty"
    :save-disabled="saveBlocked"
    :loading="loading"
    :title="$t('actions.' + title) + ' ' + $t('objects.rule')"
    @close="closeModal"
    @save="saveChanges"
  >
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
            <v-btn color="primary" @click="ruleData.rules.push(<rule>{})" hide-details>{{ $t('actions.add') + " " + $t('objects.rule') }}</v-btn>
          </v-col>
        </v-row>
        <!-- The modal owns the root child-list anchor (route.rules[0].rules),
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
        <v-card style="background-color: inherit; margin-bottom: 5px;" v-for="(r, index) in ruleData.rules" :key="ruleObjectKey(r)" v-if="ruleData.type == 'logical'">
          <v-card-subtitle class="d-flex align-center justify-space-between">
            <span>{{ $t('objects.rule') + ' ' + (Number(index)+1) }}</span>
            <v-btn v-if="ruleData.rules.length>1" icon="mdi-delete" size="small" variant="text" :aria-label="$t('actions.del')" @click="ruleData.rules.splice(index,1)" />
          </v-card-subtitle>
          <v-card-text style="padding: 0;">
            <RouteRuleNode
              :node="r"
              :path="topChildPath(Number(index))"
              :issues="conditionIssues"
              :clients="clients"
              :in-tags="inTags"
              :out-tags="outTags"
              :rs-tags="rsTags"
              @mutate="clearConditionIssues"
              :field-hints="currentFieldHints" />
          </v-card-text>
        </v-card>
        <RuleOptions
          v-else
          :rule="ruleData.rules[0]"
          :clients="clients"
          :inTags="inTags"
          :outTags="outTags"
          :rsTags="rsTags"
          :field-hints="currentFieldHints" />
        <v-row>
          <v-col cols="12" sm="6" md="4">
            <v-select
              v-model="ruleData.action"
              :items="actions"
              :label="$t('admin.action')"
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
        <v-row v-if="showRouteRuleRecommendedPreset">
          <v-col cols="12">
            <v-btn
              color="primary"
              prepend-icon="mdi-star-plus"
              variant="tonal"
              @click="applyCurrentRouteRuleRecommendations">
              {{ $t('types.rule.recommendedPreset') }}
            </v-btn>
          </v-col>
        </v-row>
        <v-card :subtitle="ruleData.action == 'bypass' ? $t('rule.action.bypass') : $t('rule.action.route')" v-if="['route', 'bypass'].includes(ruleData.action)">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.outbound"
                :items="outTags"
                :label="$t('objects.outbound')"
                :clearable="ruleData.action == 'bypass'"
                @click:clear="delete ruleData.outbound"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('outbound')" :text="fieldHint('outbound')" />
                </template>
              </v-select>
            </v-col>
          </v-row>
        </v-card>
        <v-card :subtitle="$t('rule.action.routeOption')" v-if="['route', 'route-options', 'bypass'].includes(ruleData.action)">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model="ruleData.override_address" :label="$t('types.direct.overrideAddr')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('override_address')" :text="fieldHint('override_address')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field
                v-model.number="ruleData.override_port"
                type="number"
                min="0"
                max="65534"
                :label="$t('types.direct.overridePort')"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('override_port')" :text="fieldHint('override_port')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <div class="d-flex align-center ga-1">
                <v-switch v-model="ruleData.udp_disable_domain_unmapping" :label="$t('rule.udpDisableDomainUnmapping')" hide-details></v-switch>
                <SettingInfo v-if="fieldHint('udp_disable_domain_unmapping')" :text="fieldHint('udp_disable_domain_unmapping')" />
              </div>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <div class="d-flex align-center ga-1">
                <v-switch v-model="ruleData.udp_connect" :label="$t('rule.udpConnect')" hide-details></v-switch>
                <SettingInfo v-if="fieldHint('udp_connect')" :text="fieldHint('udp_connect')" />
              </div>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model="ruleData.udp_timeout" :label="$t('rule.udpTimeout')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('udp_timeout')" :text="fieldHint('udp_timeout')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.network_strategy"
                :items="networkStrategies"
                :label="$t('rule.strategy')"
                clearable
                @click:clear="delete ruleData.network_strategy"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('network_strategy')" :text="fieldHint('network_strategy')" />
                </template>
              </v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field
                v-model.number="ruleData.fallback_delay"
                :label="$t('rule.fallbackDelay')"
                type="number"
                min="0"
                :suffix="$t('date.ms')"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('fallback_delay')" :text="fieldHint('fallback_delay')" />
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <div class="d-flex align-center ga-1">
                <v-switch v-model="tlsRecordFragment" :label="$t('singbox.tlsRecordFragment')" hide-details></v-switch>
                <SettingInfo v-if="fieldHint('tls_record_fragment')" :text="fieldHint('tls_record_fragment')" />
              </div>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <div class="d-flex align-center ga-1">
                <v-switch v-model="tlsFragment" :label="$t('singbox.tlsFragment')" hide-details></v-switch>
                <SettingInfo v-if="fieldHint('tls_fragment')" :text="fieldHint('tls_fragment')" />
              </div>
            </v-col>
            <v-col cols="12" sm="6" md="4" v-if="ruleData.tls_fragment">
              <v-text-field
                v-model="ruleData.tls_fragment_fallback_delay"
                :label="$t('singbox.tlsFragmentFallbackDelay')"
                placeholder="500ms"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('tls_fragment')" :text="fieldHint('tls_fragment')" />
                </template>
              </v-text-field>
            </v-col>
          </v-row>
        </v-card>
        <v-card :subtitle="$t('rule.action.reject')" v-if="ruleData.action == 'reject'">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.method"
                :items="[{ title: 'Default', value: 'default' },{ title: 'Drop', value: 'drop'}, { title: 'Reply', value: 'reply' }]"
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
        <v-card :subtitle="$t('rule.action.sniff')" v-if="ruleData.action == 'sniff'">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.sniffer"
                :items="sniffers"
                :label="$t('rule.sniffer')"
                multiple
                chips
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('sniffer')" :text="fieldHint('sniffer')" />
                </template>
              </v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model="ruleData.timeout" :label="$t('rule.timeout')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('timeout')" :text="fieldHint('timeout')" />
                </template>
              </v-text-field>
            </v-col>
          </v-row>
        </v-card>
        <v-card :subtitle="$t('rule.action.resolve')" v-if="ruleData.action == 'resolve'">
          <v-row>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="ruleData.strategy"
                :items="domainStrategies"
                :label="$t('rule.strategy')"
                clearable
                @click:clear="delete ruleData.strategy"
                hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('strategy')" :text="fieldHint('strategy')" />
                </template>
              </v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-text-field v-model="ruleData.server" :label="$t('basic.dns.server')" hide-details>
                <template #append-inner>
                  <SettingInfo v-if="fieldHint('server')" :text="fieldHint('server')" />
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
  </form-shell>
</template>

<script lang="ts">
import { logicalRule, rule, isRouteActionKey, routeDialerActionKeys } from '@/types/rules'
import RuleOptions from '@/components/Rule.vue'
import RouteRuleNode from '@/components/rules/RouteRuleNode.vue'
import FormShell from '@/components/nexus/drawers/FormShell.vue'
import SettingInfo from '@/components/SettingInfo.vue'
import { applyRouteRuleRecommendedValues, hasRouteRuleRecommendedPreset, routeRuleFieldHints } from '@/utils/defaultRecommendations'
import { validateRuleConditions, type RuleConditionIssue } from '@/utils/ruleValidation'
import { childListPath, childNodePath, issuesAtPath } from '@/utils/ruleTree'

// The single rule the modal edits is validated as route.rules[0] (the backend
// wraps it in a one-element array), so every issue path the core returns is
// rooted here. The recursive nodes hang off route.rules[0].rules[i].
const ROOT_PATH = 'route.rules[0]'

// Stable identity key for each sub-rule object so the v-for is not keyed by array
// index. Splicing out a middle rule then re-binds the remaining RuleOptions
// instances by object identity (not position), avoiding stale child widget state.
// A WeakMap keeps it off the rule object, so nothing leaks into the saved config.
const ruleObjectKeys = new WeakMap<object, number>()
let ruleObjectKeySeq = 0

export default {
  props: ['visible', 'data', 'index', 'clients', 'inTags', 'outTags', 'rsTags'],
  emits: ['close', 'save'],
  data() {
    return {
      title: 'add',
      loading: false,
      conditionIssues: <RuleConditionIssue[]>[],
      validationRequest: 0,
      rootConfirm: <{ open: boolean; resolve: ((confirmed: boolean) => void) | null }>{ open: false, resolve: null },
      snapshot: '',
      ruleData: <any>{
        type: 'logical',
        mode: 'and',
        rules: <rule[]>[{}],
        invert: false,
        action: 'route',
        outbound: 'direct',
      },
      actions: [
        { title: 'Route', value: 'route'},
        { title: 'Direct', value: 'direct'},
        { title: 'Route Options', value: 'route-options'},
        { title: 'Bypass', value: 'bypass'},
        { title: 'Reject', value: 'reject'},
        { title: 'Hijack DNS', value: 'hijack-dns'},
        { title: 'Sniff', value: 'sniff'},
        { title: 'Resolve', value: 'resolve'}
      ],
      sniffers: [
        { title: 'HTTP', value: 'http' },
        { title: 'TLS', value: 'tls' },
        { title: 'QUIC', value: 'quic' },
        { title: 'STUN', value: 'stun' },
        { title: 'DNS', value: 'dns' },
        { title: 'BitTorrent', value: 'bittorrent' },
        { title: 'DTLS', value: 'dtls' },
        { title: 'SSH', value: 'ssh' },
        { title: 'RDP', value: 'rdp' },
        { title: 'NTP', value: 'ntp' },
      ],
      domainStrategies: [
        { title: 'Prefer IPv4', value: 'prefer_ipv4' },
        { title: 'Prefer IPv6', value: 'prefer_ipv6' },
        { title: 'IPv4 Only', value: 'ipv4_only' },
        { title: 'IPv6 Only', value: 'ipv6_only' },
      ],
      networkStrategies: [
        { title: 'Fallback', value: 'fallback' },
        { title: 'Hybrid', value: 'hybrid' },
      ],
    }
  },
  methods: {
    ruleObjectKey(r: any): number {
      if (r == null || typeof r !== 'object') return -1
      let k = ruleObjectKeys.get(r)
      if (k === undefined) {
        k = ++ruleObjectKeySeq
        ruleObjectKeys.set(r, k)
      }
      return k
    },
    fieldHint(key: string): string {
      const hintKey = (this.currentFieldHints as Record<string, string>)[key]
      if (!hintKey) return ''
      const translated = this.$t(hintKey)
      return translated === hintKey ? '' : translated
    },
    applyCurrentRouteRuleRecommendations() {
      applyRouteRuleRecommendedValues(this.ruleData)
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
            rules: <rule[]>[{}],
          }
          Object.keys(newData).forEach(key => {
            // Action-aware: a `direct` rule carries DialerOptions, and one of
            // those keys (network_type) is also a match field. A flat lookup
            // pushed every unlisted dialer key into rules[0], which rewrote it
            // as a match condition on save.
            if (isRouteActionKey(key, newData.action)) {
              this.ruleData[key] = newData[key]
            } else {
              this.ruleData.rules[0][key] = newData[key]
            }
          })
        }
        this.title = 'edit'
      }
      else {
        this.ruleData = <logicalRule>{
            type: 'simple',
            mode: 'and',
            rules: <rule[]>[{}],
            invert: false,
            action: 'route',
            outbound: this.$props.outTags[0]?? 'direct',
          }
        this.title = 'add'
      }
      this.snapshot = JSON.stringify(this.ruleData)
    },
    closeModal() {
      this.validationRequest += 1
      this.loading = false
      this.updateData() // reset
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
          newRule.outbound = this.ruleData.outbound
          this.applyRouteOptions(newRule)
          break
        case 'bypass':
          newRule.outbound = this.ruleData.outbound?.length > 0 ? this.ruleData.outbound : undefined
          this.applyRouteOptions(newRule)
          break
        case 'route-options':
          this.applyRouteOptions(newRule)
          break
        case 'direct':
          this.applyDirectOptions(newRule)
          break
        case 'reject':
          newRule.method = this.ruleData.method?.length > 0 ? this.ruleData.method : undefined
          newRule.no_drop = this.ruleData.no_drop? true : undefined
          break
        case 'sniff':
          newRule.sniffer = this.ruleData.sniffer?.length > 0 ? this.ruleData.sniffer : undefined
          newRule.timeout = this.ruleData.timeout?.length > 0 ? this.ruleData.timeout : undefined
          break
        case 'resolve':
          newRule.strategy = this.ruleData.strategy?.length > 0 ? this.ruleData.strategy : undefined
          newRule.server = this.ruleData.server?.length > 0 ? this.ruleData.server : undefined
          newRule.disable_cache = this.ruleData.disable_cache ? true : undefined
          newRule.rewrite_ttl = this.ruleData.rewrite_ttl > 0 ? this.ruleData.rewrite_ttl : undefined
          newRule.client_subnet = this.ruleData.client_subnet?.length > 0 ? this.ruleData.client_subnet : undefined
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
      const verdict = await validateRuleConditions('route', newRule)
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
    // Path of the index-th top-level sub-rule, e.g. route.rules[0].rules[2].
    // childNodePath already appends `.rules[index]`, so it takes the node path,
    // not the child-list path. The recursive node extends this for its own kids.
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
    },
    applyDirectOptions(newRule:any) {
      for (const key of [...routeDialerActionKeys, 'network_strategy', 'fallback_delay']) {
        if (Object.prototype.hasOwnProperty.call(this.ruleData, key)) newRule[key] = this.ruleData[key]
      }
    },
    applyRouteOptions(newRule:any) {
      newRule.override_address = this.ruleData.override_address?.length > 0 ? this.ruleData.override_address : undefined
      newRule.override_port = this.ruleData?.override_port > 0 ? this.ruleData.override_port : undefined
      newRule.override_gateway = this.ruleData.override_gateway?.length > 0 ? this.ruleData.override_gateway : undefined
      newRule.network_strategy = this.ruleData.network_strategy?.length > 0 ? this.ruleData.network_strategy : undefined
      newRule.fallback_delay = this.ruleData.fallback_delay > 0 ? this.ruleData.fallback_delay : undefined
      newRule.udp_disable_domain_unmapping = this.ruleData.udp_disable_domain_unmapping? true : undefined
      newRule.udp_connect = this.ruleData.udp_connect? true : undefined
      newRule.udp_timeout = this.ruleData.udp_timeout?.length > 0 ? this.ruleData.udp_timeout : undefined
      newRule.tls_record_fragment = this.ruleData.tls_record_fragment ? true : undefined
      newRule.tls_fragment = this.ruleData.tls_fragment && !this.ruleData.tls_record_fragment ? true : undefined
      newRule.tls_fragment_fallback_delay = newRule.tls_fragment && this.ruleData.tls_fragment_fallback_delay?.length > 0 ? this.ruleData.tls_fragment_fallback_delay : undefined
    }
  },
  computed: {
    dirty(): boolean {
      return this.snapshot !== '' && JSON.stringify(this.ruleData) !== this.snapshot
    },
    // Save is blocked only while the check is in flight. It is never disabled on
    // a local guess: the previous predicate greyed Save out permanently, with no
    // message, for rules the core accepts.
    saveBlocked(): boolean {
      return this.loading
    },
    currentFieldHints(): Record<string, string> {
      return routeRuleFieldHints()
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
    showRouteRuleRecommendedPreset(): boolean {
      return this.$props.index == -1 && hasRouteRuleRecommendedPreset()
    },
    logical(): boolean {
      return this.ruleData.type == 'logical'
    },
    tlsRecordFragment: {
      get() { return this.ruleData.tls_record_fragment ?? false },
      set(v:boolean) {
        this.ruleData.tls_record_fragment = v ? true : undefined
        if (v) {
          delete this.ruleData.tls_fragment
          delete this.ruleData.tls_fragment_fallback_delay
        }
      }
    },
    tlsFragment: {
      get() { return this.ruleData.tls_fragment ?? false },
      set(v:boolean) {
        this.ruleData.tls_fragment = v ? true : undefined
        if (v) delete this.ruleData.tls_record_fragment
        else delete this.ruleData.tls_fragment_fallback_delay
      }
    }
  },
  watch: {
    visible(newValue) {
      if (newValue) {
        this.updateData()
      }
    },
  },
  components: { FormShell, SettingInfo, RuleOptions, RouteRuleNode }
}

</script>
