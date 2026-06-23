<template>
  <v-navigation-drawer
    :aria-label="title"
    aria-modal="true"
    class="nexus-drawer"
    location="right"
    :model-value="modelValue"
    role="dialog"
    temporary
    :width="width"
    @keydown.esc="requestClose"
    @update:model-value="onModel"
  >
    <div class="nexus-drawer__header">
      <v-btn
        :aria-label="$t('actions.close')"
        icon="lucide:arrow-left"
        variant="text"
        @click="requestClose"
      />
      <span class="nexus-drawer__title">{{ title }}</span>
      <v-spacer />
      <v-btn
        :aria-label="$t('actions.close')"
        icon="lucide:x"
        variant="text"
        @click="requestClose"
      />
    </div>

    <div class="nexus-drawer__body">
      <!-- Render the form only while open (v-dialog is lazy; a temporary
           navigation drawer is not). Avoids mounting an entity form against its
           incomplete default object before updateData() runs on open. -->
      <template v-if="modelValue">
        <slot v-if="loading" name="loading">
          <v-skeleton-loader type="card, list-item-two-line, list-item-two-line" />
        </slot>
        <slot v-else />
      </template>
    </div>

    <div class="nexus-drawer__footer">
      <v-chip
        v-if="dirty"
        class="nexus-drawer__dirty"
        color="warning"
        prepend-icon="lucide:alert-triangle"
        size="small"
        variant="tonal"
      >
        {{ $t('form.unsavedChanges') }}
      </v-chip>
      <v-spacer />
      <v-btn variant="text" @click="requestClose">{{ $t('actions.close') }}</v-btn>
      <v-btn
        color="primary"
        :disabled="saveDisabled"
        :loading="saving"
        variant="flat"
        @click="emit('save')"
      >
        {{ $t('actions.save') }}
      </v-btn>
    </div>
  </v-navigation-drawer>
</template>

<script lang="ts" setup>
import { useI18n } from 'vue-i18n'

import { useConfirm } from '@/components/nexus/primitives/useConfirm'

const props = withDefaults(defineProps<{
  modelValue: boolean
  title: string
  width?: number
  loading?: boolean
  dirty?: boolean
  saving?: boolean
  saveDisabled?: boolean
}>(), {
  width: 560,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: []
  close: []
}>()

const { t } = useI18n()
const { confirm } = useConfirm()

const requestClose = async () => {
  if (props.dirty) {
    const discard = await confirm({
      title: t('form.leaveTitle'),
      message: t('form.leaveConfirm'),
      confirmLabel: t('form.discard'),
      cancelLabel: t('actions.close'),
      tone: 'error',
    })

    if (!discard) return
  }

  emit('update:modelValue', false)
  emit('close')
}

// The scrim-click path comes through update:model-value(false); route it
// through the same dirty guard so unsaved edits can't be lost by an outside
// click. (Esc is handled separately via @keydown.esc, since a temporary
// navigation drawer has no built-in Esc-to-close.)
const onModel = (value: boolean) => {
  if (value) {
    emit('update:modelValue', true)
    return
  }

  requestClose()
}
</script>

<style scoped>
.nexus-drawer {
  background: var(--nexus-surface-1);
}

.nexus-drawer :deep(.v-navigation-drawer__content) {
  display: flex;
  flex-direction: column;
}

.nexus-drawer__header {
  align-items: center;
  background: var(--nexus-surface-1);
  border-block-end: 1px solid var(--nexus-border);
  display: flex;
  gap: var(--nexus-gap-2);
  min-height: 60px;
  padding-inline: var(--nexus-gap-3);
  position: sticky;
  inset-block-start: 0;
  z-index: 2;
}

.nexus-drawer__title {
  color: var(--nexus-text-primary);
  font-size: 1rem;
  font-weight: 650;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nexus-drawer__body {
  flex: 1 1 auto;
  overflow-y: auto;
  padding: var(--nexus-gap-5);
}

/* Slim drawer scrollbar (reference 8px). */
.nexus-drawer__body::-webkit-scrollbar {
  width: 8px;
}

.nexus-drawer__body::-webkit-scrollbar-thumb {
  background: var(--nexus-border);
  border-radius: 4px;
}

/* Tabbed forms (e.g. Client: Basics/Config/Links) render <v-tabs> flush above
   the first field row; without a gap the first row's floating labels (Group,
   …) clip into the tab strip. Add breathing room below any tab strip. */
.nexus-drawer__body :deep(.v-tabs) {
  margin-block-end: var(--nexus-gap-4);
}

/* Every field in the nexus drawers uses hide-details, which strips the ~22px
   details buffer Vuetify normally reserves below a field. Without it, stacked
   rows collapse together and an outlined field's floating label (which sits on
   the top border) crowds the field, heading, or tab strip directly above it.
   Give every field a little top breathing room so labels always stay clear. */
.nexus-drawer__body :deep(.v-input) {
  margin-block-start: var(--nexus-gap-2);
}

.nexus-drawer__footer {
  align-items: center;
  background: var(--nexus-surface-1);
  border-block-start: 1px solid var(--nexus-border);
  display: flex;
  gap: var(--nexus-gap-2);
  min-height: 64px;
  padding-inline: var(--nexus-gap-4);
  position: sticky;
  inset-block-end: 0;
  z-index: 2;
}
</style>
