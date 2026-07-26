<template>
  <v-card class="rule-node" variant="tonal" :data-rule-path="path" :data-node-shape="isLogical ? 'logical' : 'default'">
    <!-- Issues the core anchored at this exact node. -->
    <v-alert
      v-if="ownIssues.length > 0"
      type="error"
      variant="tonal"
      density="compact"
      class="ma-2"
      :data-rule-issue="path"
    >
      <div v-for="issue in ownIssues" :key="issue.path + issue.message">{{ issue.message }}</div>
    </v-alert>

    <v-row class="px-3 pt-2" no-gutters>
      <v-col cols="12" sm="6" class="d-flex align-center ga-1">
        <v-switch
          color="primary"
          :model-value="isLogical"
          :label="$t('rule.logical')"
          hide-details
          data-testid="dns-rule-node-shape"
          @update:model-value="onToggleLogical"
        />
      </v-col>
      <v-col cols="12" sm="6" class="d-flex align-center ga-1">
        <v-switch
          color="primary"
          :model-value="node.invert === true"
          :label="$t('rule.invert')"
          hide-details
          data-testid="dns-rule-node-invert"
          @update:model-value="setInvert"
        />
      </v-col>
    </v-row>

    <template v-if="isLogical">
      <v-row class="px-3" no-gutters>
        <v-col cols="12" sm="6">
          <v-select
            v-model="nodeMode"
            :items="['and', 'or']"
            :label="$t('rule.mode')"
            hide-details
          />
        </v-col>
      </v-row>

      <v-alert
        v-if="childListIssues.length > 0"
        type="error"
        variant="tonal"
        density="compact"
        class="ma-2"
        :data-rule-issue="childListPathOf(path)"
      >
        <div v-for="issue in childListIssues" :key="issue.path + issue.message">{{ issue.message }}</div>
      </v-alert>

      <div class="nested-children">
        <div v-for="(child, index) in childNodes" :key="nodeKey(child)" class="nested-child">
          <div class="d-flex align-center justify-space-between px-3 pt-2">
            <span class="text-caption text-medium-emphasis">{{ $t('objects.rule') + ' ' + (index + 1) }}</span>
            <v-btn
              icon="mdi-delete"
              size="small"
              variant="text"
              :aria-label="$t('actions.del')"
              data-testid="dns-rule-node-delete-child"
              @click="removeChild(index)"
            />
          </div>
          <DnsRuleNode
            :node="child"
            :path="childNodePathOf(path, index)"
            :issues="issues"
            :clients="clients"
            :in-tags="inTags"
            :rs-tags="rsTags"
            :rule-sets="ruleSets"
            :field-hints="fieldHints"
            @mutate="emit('mutate')"
          />
        </div>
      </div>

      <v-card-actions>
        <v-btn
          color="primary"
          variant="tonal"
          prepend-icon="mdi-plus"
          data-testid="dns-rule-node-add-child"
          @click="addChild"
        >
          {{ $t('actions.add') + ' ' + $t('objects.rule') }}
        </v-btn>
      </v-card-actions>
    </template>

    <DnsRuleOptions
      v-else
      :rule="node"
      :clients="clients"
      :in-tags="inTags"
      :rs-tags="rsTags"
      :rule-sets="ruleSets"
      :field-hints="fieldHints"
    />

    <!-- Classic mode has no mounted ConfirmHost; nexus routes through useConfirm. -->
    <v-dialog v-if="mode !== 'nexus'" v-model="localConfirm.open" max-width="440">
      <v-card class="rounded-xl">
        <v-card-title>{{ localConfirm.title }}</v-card-title>
        <v-card-text>{{ localConfirm.message }}</v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="resolveLocal(false)">{{ $t('actions.close') }}</v-btn>
          <v-btn color="error" variant="flat" @click="resolveLocal(true)">{{ localConfirm.confirmLabel }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'

import DnsRuleOptions from '@/components/DnsRule.vue'
import { dnsDefaultMatchKeys } from '@/types/dns'
import { useUiMode } from '@/uiMode/useUiMode'
import { confirm } from '@/components/nexus/primitives/useConfirm'
import {
  childListPath as childListPathOf,
  childNodePath as childNodePathOf,
  convertDefaultToLogical,
  convertLogicalToDefault,
  discardedDescendantCount,
  discardedMatchKeys,
  isLogicalNode,
  issuesAtPath,
  nodeKey,
  type RuleNode,
} from '@/utils/ruleTree'
import type { RuleConditionIssue } from '@/utils/ruleValidation'

// Edited in place: the parent holds this node by reference and keys it by object
// identity, so a shape conversion must mutate the same object, not replace it.
const props = defineProps<{
  node: RuleNode
  path: string
  issues: RuleConditionIssue[]
  clients?: unknown[]
  inTags?: unknown[]
  rsTags?: unknown[]
  ruleSets?: unknown[]
  fieldHints?: Record<string, string>
}>()
const emit = defineEmits<{
  (e: 'mutate'): void
}>()

const { t } = useI18n()
const { mode } = useUiMode()

const isLogical = computed(() => isLogicalNode(props.node))
const nodeMode = computed<string>({
  get: () => (typeof props.node.mode === 'string' ? props.node.mode : 'and'),
  set: (value) => {
    props.node.mode = value
    emit('mutate')
  },
})
const childNodes = computed<RuleNode[]>(() =>
  Array.isArray(props.node.rules) ? (props.node.rules as RuleNode[]) : [],
)

const ownIssues = computed(() => issuesAtPath(props.issues, props.path))
const childListIssues = computed(() => issuesAtPath(props.issues, childListPathOf(props.path)))

const applyInPlace = (target: RuleNode, source: RuleNode): void => {
  for (const key of Object.keys(target)) {
    if (!Object.prototype.hasOwnProperty.call(source, key)) delete target[key]
  }
  Object.assign(target, source)
}

const setInvert = (value: boolean | null): void => {
  if (value) props.node.invert = true
  else delete props.node.invert
  emit('mutate')
}

const localConfirm = reactive<{
  open: boolean
  title: string
  message: string
  confirmLabel: string
  resolve: ((confirmed: boolean) => void) | null
}>({ open: false, title: '', message: '', confirmLabel: '', resolve: null })

const resolveLocal = (confirmed: boolean): void => {
  localConfirm.open = false
  const resolve = localConfirm.resolve
  localConfirm.resolve = null
  resolve?.(confirmed)
}

const askConfirm = (title: string, message: string, confirmLabel: string): Promise<boolean> => {
  if (mode.value === 'nexus') {
    return confirm({ title, message, confirmLabel, tone: 'error' })
  }
  return new Promise<boolean>((resolve) => {
    localConfirm.title = title
    localConfirm.message = message
    localConfirm.confirmLabel = confirmLabel
    localConfirm.resolve = resolve
    localConfirm.open = true
  })
}

const onToggleLogical = async (next: boolean | null): Promise<void> => {
  if (next) {
    const discarded = discardedMatchKeys(props.node, dnsDefaultMatchKeys)
    if (discarded.length > 0) {
      const ok = await askConfirm(
        t('rule.convert.toLogicalTitle'),
        t('rule.convert.toLogicalMessage', { keys: discarded.join(', ') }),
        t('rule.convert.discardConfirm'),
      )
      if (!ok) return
    }
    applyInPlace(props.node, convertDefaultToLogical(props.node, dnsDefaultMatchKeys))
  } else {
    if (discardedDescendantCount(props.node) > 0) {
      const ok = await askConfirm(
        t('rule.convert.toDefaultTitle'),
        t('rule.convert.toDefaultMessage'),
        t('rule.convert.discardConfirm'),
      )
      if (!ok) return
    }
    applyInPlace(props.node, convertLogicalToDefault(props.node))
  }
  emit('mutate')
}

const addChild = (): void => {
  if (!Array.isArray(props.node.rules)) props.node.rules = []
  ;(props.node.rules as RuleNode[]).push({})
  emit('mutate')
}

const removeChild = (index: number): void => {
  if (Array.isArray(props.node.rules)) (props.node.rules as RuleNode[]).splice(index, 1)
  emit('mutate')
}
</script>

<style scoped>
.nested-children {
  padding-left: 12px;
}
.nested-child {
  border-left: 2px solid rgba(var(--v-border-color), var(--v-border-opacity));
  margin: 8px 0 8px 4px;
}
</style>
