<template>
  <v-expansion-panels class="mb-4">
    <v-expansion-panel :title="$t('nodes.setup.title')" elevation="0" class="border">
      <v-expansion-panel-text>
        <div v-if="panel">
          <div class="mb-4" role="status">{{ $t(`nodes.setup.role.${panel.role}`) }}</div>
          <div v-if="panel.role !== 'slave'">
            <h2 class="text-h6">Master</h2>
            <p class="mb-3">{{ $t('nodes.setup.masterHelp') }}</p>
            <v-btn v-if="panel.role !== 'master'" color="primary" :loading="busy" @click="submit('master')">
              {{ $t('nodes.setup.enableMaster') }}
            </v-btn>
            <v-btn v-else color="primary" :loading="busy" @click="invite">
              {{ $t('nodes.setup.invite') }}
            </v-btn>
            <v-text-field
              v-if="invitation" class="mt-3" :label="$t('nodes.setup.code')"
              :hint="$t('nodes.setup.inviteHelp')" :model-value="invitation" persistent-hint readonly
            />
            <p v-if="inviteExpiresAt">{{ $t('nodes.setup.inviteExpires', { time: new Date(inviteExpiresAt * 1000).toLocaleString() }) }}</p>
            <v-text-field v-if="invitationText" :label="$t('nodes.setup.copy')" :model-value="invitationText" readonly />
            <v-btn v-if="invitationText && canCopy" class="mt-2" variant="text" @click="copyInvitation">
              {{ $t(copied ? 'nodes.setup.copied' : 'nodes.setup.copy') }}
            </v-btn>
          </div>
          <v-divider v-if="panel.role !== 'slave'" class="my-4" />
          <h2 class="text-h6">Slave</h2>
          <p class="mb-3">{{ $t('nodes.setup.slaveHelp') }}</p>
          <form v-if="panel.role !== 'slave'" @submit.prevent="join">
            <v-text-field v-model="pastedInvitation" :label="$t('nodes.setup.pasteInvitation')" :error-messages="pasteError ? [$t('nodes.setup.invalidInvitation')] : []" autocomplete="off" type="password" @paste.prevent="importInvitation($event.clipboardData?.getData('text/plain') ?? '')" @update:model-value="pasteError = false" />
            <v-btn v-if="pastedInvitation" type="button" variant="text" @click="importInvitation(pastedInvitation)">{{ $t('nodes.setup.importInvitation') }}</v-btn>
            <v-text-field v-model="masterUrl" :label="$t('nodes.setup.url')" placeholder="https://master.example/panel/" required />
            <v-text-field v-model="code" :label="$t('nodes.setup.code')" autocomplete="off" maxlength="64" type="password" required />
            <v-btn color="primary" :loading="busy" type="submit">{{ $t('nodes.setup.join') }}</v-btn>
          </form>
          <div v-else>
            <p>{{ $t('nodes.setup.paired', { url: panel.masterUrl }) }}</p>
            <p v-if="panel.lastSync" role="status">{{ $t('nodes.setup.lastSync', { time: new Date(panel.lastSync * 1000).toLocaleString() }) }}</p>
            <v-alert v-if="panel.syncError" type="warning" variant="tonal" class="my-3">{{ panel.syncError }}</v-alert>
            <p class="mt-3">{{ $t('nodes.setup.chooseInbound') }}</p>
            <ul v-if="panel.compatibleInbounds?.length" class="ps-5">
              <li v-for="inbound in panel.compatibleInbounds" :key="inbound.id">{{ inbound.tag }} ({{ inbound.type }})</li>
            </ul>
            <v-btn v-else :to="{ path: '/inbounds', query: { newPeerInbound: 'vless' } }" variant="tonal">{{ $t('nodes.setup.createInbound') }}</v-btn>
            <form class="my-3" @submit.prevent="submit('hostname', { host: publicHost })">
              <p>{{ $t('nodes.setup.confirmAddress') }}</p>
              <v-text-field v-model="publicHost" :label="$t('nodes.linkHost')" :hint="$t('nodes.setup.hostHelp')" maxlength="253" persistent-hint />
              <v-btn class="mt-2" variant="tonal" :loading="busy" type="submit">{{ $t('actions.save') }}</v-btn>
            </form>
            <v-btn v-if="masterNodeURL && panel.publicHost && panel.lastSync && panel.compatibleInbounds?.length" :href="masterNodeURL" target="_blank" rel="noopener noreferrer" variant="tonal">{{ $t('nodes.setup.checkOnMaster') }}</v-btn>
            <p v-else class="mb-2">{{ $t('nodes.setup.waitForCheck') }}</p>
            <v-btn variant="text" :loading="loading" @click="load">{{ $t('actions.update') }}</v-btn>
            <v-btn variant="text" color="error" :loading="busy" @click="unlink">{{ $t('nodes.setup.unlink') }}</v-btn>
            <v-btn v-for="link in localLinks" :key="link.path" :to="link.path" variant="text">{{ $t(link.label) }}</v-btn>
          </div>
        </div>
        <div v-else-if="loading"><v-progress-linear indeterminate /></div>
        <div v-else>
          <v-btn @click="load">{{ $t('actions.update') }}</v-btn>
        </div>
      </v-expansion-panel-text>
    </v-expansion-panel>
  </v-expansion-panels>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import HttpUtils from '@/plugins/httputil'
import { getBaseUrl } from '@/plugins/base-url'
import { createNodeInvitation, parseNodeInvitation } from './nodeInvitation'

const emit = defineEmits<{ 'role-change': [role: string] }>()
const panel = ref<{ role: 'standalone' | 'master' | 'slave'; uuid?: string; masterUrl?: string; lastSync?: number; syncError?: string; publicHost?: string; compatibleInbounds?: { id: number; type: string; tag: string }[] }>()
const loading = ref(false)
const busy = ref(false)
const masterUrl = ref('')
const publicHost = ref('')
const code = ref('')
const invitation = ref('')
const inviteExpiresAt = ref(0)
const pastedInvitation = ref('')
const pasteError = ref(false)
const canCopy = typeof navigator !== 'undefined' && Boolean(navigator.clipboard?.writeText)
const invitationText = computed(() => {
  if (!invitation.value || typeof window === 'undefined') return ''
  return createNodeInvitation(new URL(getBaseUrl(), window.location.origin).href, invitation.value)
})
const masterNodeURL = computed(() => {
  const { masterUrl, uuid } = panel.value ?? {}
  if (!masterUrl || !uuid) return ''
  const url = new URL(`${masterUrl.replace(/\/$/, '')}/nodes`)
  if (url.protocol !== 'https:') return ''
  url.searchParams.set('node', uuid)
  return url.href
})
const copied = ref(false)
const { t } = useI18n()
const localLinks = [
  { path: '/inbounds', label: 'pages.inbounds' },
  { path: '/outbounds', label: 'pages.outbounds' },
  { path: '/rules', label: 'pages.rules' },
  { path: '/tls', label: 'pages.tls' },
]

async function load() {
  loading.value = true
  try {
    const response = await HttpUtils.get('api/nodes/panel')
    if (response.success && response.obj) {
      panel.value = response.obj
      publicHost.value = response.obj.publicHost ?? ''
      emit('role-change', response.obj.role)
    }
  } finally {
    loading.value = false
  }
}

async function submit(action: 'master' | 'invite' | 'join' | 'unlink' | 'hostname', payload: object | null = null) {
  if (busy.value) return
  busy.value = true
  try {
    const endpoint = action === 'invite' ? 'api/nodes/panel/invite?details=1' : `api/nodes/panel/${action}`
    const response = await HttpUtils.post(endpoint, payload, {
      headers: { 'Content-Type': 'application/json' },
    })
    if (!response.success) return
    if (action === 'invite') {
      copied.value = false
      invitation.value = typeof response.obj === 'string' ? response.obj : response.obj.code
      inviteExpiresAt.value = typeof response.obj === 'string' ? 0 : response.obj.expiresAt
    } else {
      code.value = ''
      invitation.value = ''
      inviteExpiresAt.value = 0
      await load()
    }
  } finally {
    if (action === 'join') {
      code.value = ''
      pastedInvitation.value = ''
    }
    busy.value = false
  }
}

const invite = () => submit('invite')
const join = () => submit('join', { masterUrl: masterUrl.value.trim(), code: code.value.trim() })
const unlink = () => {
  if (window.confirm(t('nodes.setup.unlinkConfirm'))) void submit('unlink')
}

async function copyInvitation() {
  try {
    await navigator.clipboard.writeText(invitationText.value)
    copied.value = true
  } catch {
    copied.value = false
  }
}

function importInvitation(text: string) {
  const parsed = parseNodeInvitation(text)
  pasteError.value = parsed == null
  pastedInvitation.value = ''
  if (parsed) {
    masterUrl.value = parsed.masterUrl
    code.value = parsed.code
  }
}

onMounted(load)
</script>
