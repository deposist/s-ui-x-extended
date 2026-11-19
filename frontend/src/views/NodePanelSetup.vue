<template>
  <v-card class="mb-4" :title="$t('nodes.setup.title')" variant="outlined">
    <v-card-text v-if="panel">
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
        <v-btn v-if="invitation && canCopy" class="mt-2" variant="text" @click="copyInvitation">
          {{ $t(copied ? 'nodes.setup.copied' : 'nodes.setup.copy') }}
        </v-btn>
      </div>
      <v-divider v-if="panel.role !== 'slave'" class="my-4" />
      <h2 class="text-h6">Slave</h2>
      <p class="mb-3">{{ $t('nodes.setup.slaveHelp') }}</p>
      <form v-if="panel.role !== 'slave'" @submit.prevent="join">
        <v-text-field v-model="masterUrl" :label="$t('nodes.setup.url')" placeholder="https://master.example/panel/" required />
        <v-text-field v-model="code" :label="$t('nodes.setup.code')" autocomplete="off" maxlength="64" required />
        <v-btn color="primary" :loading="busy" type="submit">{{ $t('nodes.setup.join') }}</v-btn>
      </form>
      <div v-else>
        <p>{{ $t('nodes.setup.paired', { url: panel.masterUrl }) }}</p>
        <p v-if="panel.lastSync" role="status">{{ $t('nodes.setup.lastSync', { time: new Date(panel.lastSync * 1000).toLocaleString() }) }}</p>
        <v-alert v-if="panel.syncError" type="warning" variant="tonal" class="my-3">{{ panel.syncError }}</v-alert>
        <form class="my-3" @submit.prevent="submit('hostname', { host: publicHost })">
          <v-text-field v-model="publicHost" :label="$t('nodes.linkHost')" :hint="$t('nodes.setup.hostHelp')" maxlength="253" persistent-hint />
          <v-btn class="mt-2" variant="tonal" :loading="busy" type="submit">{{ $t('actions.save') }}</v-btn>
        </form>
        <v-btn variant="text" :loading="loading" @click="load">{{ $t('actions.update') }}</v-btn>
        <v-btn variant="text" color="error" :loading="busy" @click="unlink">{{ $t('nodes.setup.unlink') }}</v-btn>
        <v-btn v-for="link in localLinks" :key="link.path" :to="link.path" variant="text">{{ $t(link.label) }}</v-btn>
      </div>
    </v-card-text>
    <v-card-text v-else-if="loading"><v-progress-linear indeterminate /></v-card-text>
    <v-card-text v-else>
      <v-btn @click="load">{{ $t('actions.update') }}</v-btn>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import HttpUtils from '@/plugins/httputil'

const emit = defineEmits<{ 'role-change': [role: string] }>()
const panel = ref<{ role: 'standalone' | 'master' | 'slave'; masterUrl?: string; lastSync?: number; syncError?: string; publicHost?: string }>()
const loading = ref(false)
const busy = ref(false)
const masterUrl = ref('')
const publicHost = ref('')
const code = ref('')
const invitation = ref('')
const canCopy = Boolean(navigator.clipboard?.writeText)
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
    const response = await HttpUtils.post(`api/nodes/panel/${action}`, payload, {
      headers: { 'Content-Type': 'application/json' },
    })
    if (!response.success) return
    if (action === 'invite') {
      copied.value = false
      invitation.value = response.obj
    } else {
      code.value = ''
      invitation.value = ''
      await load()
    }
  } finally {
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
    await navigator.clipboard.writeText(invitation.value)
    copied.value = true
  } catch {
    copied.value = false
  }
}

onMounted(load)
</script>
