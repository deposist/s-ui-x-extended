<template>
  <confirm-dialog
    :cancel-label="request?.cancelLabel"
    :confirm-label="request?.confirmLabel"
    :message="request?.message"
    :model-value="!!request"
    :title="request?.title ?? ''"
    :tone="request?.tone ?? 'error'"
    @cancel="onCancel"
    @confirm="onConfirm"
  />
</template>

<script lang="ts" setup>
import ConfirmDialog from './ConfirmDialog.vue'
import { activeConfirm, resolveActiveConfirm } from './useConfirm'

// One host is mounted in the shell; useConfirm()'s confirm() drives it.
const request = activeConfirm

const onConfirm = () => resolveActiveConfirm(true)
const onCancel = () => resolveActiveConfirm(false)
</script>
