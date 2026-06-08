<template>
  <v-card flat>
    <v-card-title class="d-flex align-center">
      <v-icon class="mr-2">mdi-cash-multiple</v-icon>
      {{ $t('pages.paidSub') }}
      <v-chip class="ml-3" color="warning" size="small" variant="flat">experimental</v-chip>
      <v-spacer />
      <v-btn color="primary" :loading="loading" variant="tonal" @click="reloadAll">
        <v-icon start>mdi-refresh</v-icon>{{ $t('actions.refresh') ?? 'Refresh' }}
      </v-btn>
    </v-card-title>

    <v-alert
      v-if="!secretboxKeySet"
      type="warning"
      variant="tonal"
      class="mx-4 mb-2"
      density="comfortable"
    >
      For production, set the <strong>SUI_SECRETBOX_KEY</strong> environment variable so payment tokens
      are encrypted with a key kept outside the database.
    </v-alert>

    <v-tabs v-model="tab" color="primary" class="px-2">
      <v-tab value="bindings">Bindings</v-tab>
      <v-tab value="autoreg">Auto-registration</v-tab>
      <v-tab value="tariffs">Tariffs</v-tab>
      <v-tab value="payments">Payments</v-tab>
      <v-tab value="messages">Messages</v-tab>
      <v-tab value="orders">Orders</v-tab>
      <v-tab value="bot">Bot</v-tab>
    </v-tabs>

    <v-window v-model="tab" class="pa-4">
      <!-- BINDINGS -->
      <v-window-item value="bindings">
        <div class="d-flex align-center mb-2">
          <div class="text-caption text-medium-emphasis">
            Bind a panel client to a Telegram user ID. The client then gets links/QR/stats and can pay in the bot.
          </div>
          <v-spacer />
          <v-btn color="primary" :disabled="bindings.length === 0" @click="openAddBinding()">
            <v-icon start>mdi-link-plus</v-icon>Add binding
          </v-btn>
        </div>
        <v-alert v-if="!bindingsLoading && bindings.length === 0" type="info" variant="tonal" density="comfortable">
          No clients found. Create a client on the <strong>Clients</strong> page (or enable Auto-registration),
          then bind it to a Telegram ID here.
        </v-alert>
        <v-data-table v-else :headers="bindingHeaders" :items="bindings" :loading="bindingsLoading" density="comfortable">
          <template #item.enable="{ item }">
            <v-chip :color="item.enable ? 'success' : 'error'" size="small" variant="flat">
              {{ item.enable ? 'active' : 'disabled' }}
            </v-chip>
          </template>
          <template #item.tgUserId="{ item }">
            <span v-if="item.tgUserId">{{ item.tgUserId }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #item.desc="{ item }">
            <span v-if="item.desc">{{ item.desc }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #item.expiry="{ item }">
            <template v-if="item.expiry > 0">
              {{ new Date(item.expiry * 1000).toLocaleString() }}
              (<v-chip size="x-small" label :color="item.expiry <= Date.now() / 1000 ? 'error' : ''">{{ HumanReadable.remainedDays(item.expiry) }}</v-chip>)
            </template>
            <v-chip v-else size="small" color="success" label>{{ HumanReadable.remainedDays(item.expiry) }}</v-chip>
          </template>
          <template #item.actions="{ item }">
            <v-btn size="small" variant="text" icon="mdi-pencil" @click="openBinding(item)" />
            <v-btn v-if="item.tgUserId" size="small" variant="text" icon="mdi-link-off" color="error" @click="openUnbindConfirm(item)" />
          </template>
        </v-data-table>
      </v-window-item>

      <!-- AUTO-REGISTRATION -->
      <v-window-item value="autoreg">
        <v-row>
          <v-col cols="12" md="6">
            <v-switch v-model="autoRegister" color="primary" label="Auto-register unknown users" hide-details />
          </v-col>
          <v-col cols="12" md="6">
            <v-select
              v-model="autoInbounds"
              :items="inboundOptions"
              item-title="title"
              item-value="value"
              label="Inbounds for new clients"
              multiple
              chips
            />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubTrialDays" type="number" label="Trial days" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubTrialVolumeGB" type="number" label="Trial traffic (GB, 0 = unlimited)" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubMaxClients" type="number" label="Max auto-registered clients" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubStartRateLimitPerMin" type="number" label="/start rate limit (per min)" />
          </v-col>
        </v-row>
        <v-btn color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') ?? 'Save' }}</v-btn>
      </v-window-item>

      <!-- TARIFFS -->
      <v-window-item value="tariffs">
        <div class="d-flex mb-2">
          <v-spacer />
          <v-btn color="primary" @click="openTariff()"><v-icon start>mdi-plus</v-icon>Add tariff</v-btn>
        </div>
        <v-data-table :headers="tariffHeaders" :items="tariffs" :loading="tariffsLoading" density="comfortable">
          <template #item.price="{ item }">{{ (item.price / 100).toFixed(2) }} {{ item.currency }}</template>
          <template #item.starsAmount="{ item }">{{ item.starsAmount || '—' }}</template>
          <template #item.addTrafficBytes="{ item }">{{ item.addTrafficBytes ? (item.addTrafficBytes / (1024*1024*1024)).toFixed(2) + ' GB' : '∞' }}</template>
          <template #item.enabled="{ item }">
            <v-chip :color="item.enabled ? 'success' : 'grey'" size="small" variant="flat">{{ item.enabled ? 'on' : 'off' }}</v-chip>
          </template>
          <template #item.actions="{ item }">
            <v-btn size="small" variant="text" icon="mdi-pencil" @click="openTariff(item)" />
            <v-btn size="small" variant="text" icon="mdi-delete" color="error" @click="deleteTariff(item)" />
          </template>
        </v-data-table>
      </v-window-item>

      <!-- PAYMENTS -->
      <v-window-item value="payments">
        <v-row>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubCurrency" label="Default currency (e.g. RUB, USD)" maxlength="3" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubOrderTTLMinutes" type="number" label="Pending order TTL (min)" />
          </v-col>
        </v-row>
        <v-divider class="my-2" />
        <v-switch v-model="starsEnabled" color="primary" label="Telegram Stars (XTR)" hide-details />
        <v-divider class="my-2" />
        <v-switch v-model="yooEnabled" color="primary" label="YooKassa" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubYooKassaToken"
          :has-secret="settings.paidSubYooKassaTokenHasSecret"
          label="YooKassa provider_token (BotFather)"
        />
        <v-divider class="my-2" />
        <v-switch v-model="stripeEnabled" color="primary" label="Stripe" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubStripeToken"
          :has-secret="settings.paidSubStripeTokenHasSecret"
          label="Stripe provider_token (BotFather)"
        />
        <v-divider class="my-2" />
        <v-switch v-model="paymasterEnabled" color="primary" label="PayMaster" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubPayMasterToken"
          :has-secret="settings.paidSubPayMasterTokenHasSecret"
          label="PayMaster provider_token (BotFather)"
        />
        <v-divider class="my-2" />
        <v-switch v-model="cryptoEnabled" color="primary" label="CryptoBot" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubCryptoBotToken"
          :has-secret="settings.paidSubCryptoBotTokenHasSecret"
          label="CryptoBot API token"
        />
        <v-divider class="my-2" />
        <v-switch v-model="externalEnabled" color="primary" label="External payment link" hide-details />
        <v-text-field
          v-model="settings.paidSubExternalUrlTemplate"
          label="External URL template (https://… with {orderId} {amount} {currency} {clientId})"
        />
        <v-btn class="mt-2" color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') ?? 'Save' }}</v-btn>
      </v-window-item>

      <!-- MESSAGES -->
      <v-window-item value="messages">
        <div class="text-subtitle-2 mb-1">Greeting on /start</div>
        <div class="text-caption text-medium-emphasis mb-2">
          Shown to a bound client when they open the bot. Leave empty for the default greeting.
        </div>
        <v-textarea v-model="settings.paidSubGreeting" label="Custom greeting" rows="3" auto-grow counter="4096" />
        <v-btn color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') ?? 'Save' }}</v-btn>

        <v-divider class="my-4" />

        <div class="text-subtitle-2 mb-1">Broadcast to all clients</div>
        <div class="text-caption text-medium-emphasis mb-2">
          Sends a one-off announcement to every bound Telegram user ({{ recipientCount }} recipient(s)).
        </div>
        <v-textarea v-model="broadcastText" label="Announcement text" rows="4" auto-grow counter="4096" />
        <v-btn color="primary" :loading="broadcastLoading" :disabled="!broadcastText.trim() || recipientCount === 0" @click="broadcastDialog = true">
          <v-icon start>mdi-bullhorn</v-icon>Send to all
        </v-btn>
        <v-alert v-if="broadcastResult" type="info" variant="tonal" class="mt-3" density="comfortable">
          Sent: {{ broadcastResult.sent }} · failed: {{ broadcastResult.failed }}
        </v-alert>
      </v-window-item>

      <!-- ORDERS -->
      <v-window-item value="orders">
        <v-data-table :headers="orderHeaders" :items="orders" :loading="ordersLoading" density="comfortable">
          <template #item.clientName="{ item }">
            <span v-if="item.clientName">{{ item.clientName }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #item.telegramUserId="{ item }">
            <span v-if="item.telegramUserId">{{ item.telegramUserId }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #item.clientDesc="{ item }">
            <span v-if="item.clientDesc">{{ item.clientDesc }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #item.amount="{ item }">{{ formatMoney(item.amount, item.currency) }}</template>
          <template #item.status="{ item }">
            <v-chip :color="orderStatusColor(item.status)" size="small" variant="flat">{{ item.status }}</v-chip>
          </template>
          <template #item.createdAt="{ item }">{{ item.createdAt ? new Date(item.createdAt * 1000).toLocaleString() : '' }}</template>
          <template #item.actions="{ item }">
            <v-btn v-if="item.status === 'paid'" size="small" variant="text" color="warning" @click="openRefund(item)">Refund</v-btn>
          </template>
        </v-data-table>
      </v-window-item>

      <!-- BOT -->
      <v-window-item value="bot">
        <v-row>
          <v-col cols="12" md="6">
            <v-switch v-model="enabled" color="primary" label="Enable client bot" hide-details />
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field v-model="settings.paidSubBotPollSeconds" type="number" label="Long-poll timeout (s)" />
          </v-col>
          <v-col cols="12">
            <SettingsSecretField
              v-model="settings.paidSubBotToken"
              :has-secret="settings.paidSubBotTokenHasSecret"
              label="Bot token (separate from admin bot)"
            />
          </v-col>
        </v-row>

        <v-divider class="my-3" />
        <div class="text-subtitle-2 mb-1">Connection / transport</div>
        <div class="text-caption text-medium-emphasis mb-2">
          How this bot reaches Telegram. Independent from the admin Telegram module.
        </div>
        <v-row>
          <v-col cols="12" md="4">
            <v-select
              v-model="settings.paidSubTransportMode"
              :items="transportModes"
              item-title="title"
              item-value="value"
              label="Transport"
            />
          </v-col>
          <v-col v-if="settings.paidSubTransportMode === 'outbound'" cols="12" md="8">
            <v-select
              v-model="settings.paidSubOutboundTag"
              :items="outboundOptions"
              item-title="title"
              item-value="value"
              label="Outbound (sing-box) — requires core running"
              :hint="outboundOptions.length === 0 ? 'No outbounds configured' : ''"
              persistent-hint
            />
          </v-col>
          <template v-else>
            <v-col cols="12" md="8">
              <SettingsSecretField
                v-model="settings.paidSubProxyURL"
                :has-secret="settings.paidSubProxyURLHasSecret"
                label="Proxy URL (http/https/socks5, empty = direct)"
              />
            </v-col>
            <v-col cols="12" md="6">
              <SettingsSecretField
                v-model="settings.paidSubProxyUsername"
                :has-secret="settings.paidSubProxyUsernameHasSecret"
                label="Proxy username (optional)"
              />
            </v-col>
            <v-col cols="12" md="6">
              <SettingsSecretField
                v-model="settings.paidSubProxyPassword"
                :has-secret="settings.paidSubProxyPasswordHasSecret"
                label="Proxy password (optional)"
              />
            </v-col>
          </template>
        </v-row>

        <v-btn color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') ?? 'Save' }}</v-btn>
      </v-window-item>
    </v-window>
  </v-card>

  <!-- Binding dialog -->
  <v-dialog v-model="bindingDialog" max-width="420">
    <v-card>
      <v-card-title>{{ bindingEdit.isNew ? 'Add binding' : 'Bind Telegram to ' + bindingEdit.name }}</v-card-title>
      <v-card-text>
        <v-select
          v-if="bindingEdit.isNew"
          v-model="bindingEdit.clientId"
          :items="clientOptions"
          item-title="title"
          item-value="value"
          label="Client"
        />
        <v-text-field v-model="bindingEdit.tgUserId" type="number" label="Telegram user ID (0 = unbind)" autofocus />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="bindingDialog = false">{{ $t('actions.cancel') ?? 'Cancel' }}</v-btn>
        <v-btn color="primary" @click="saveBinding">{{ $t('actions.set') ?? 'Save' }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Tariff dialog -->
  <v-dialog v-model="tariffDialog" max-width="560">
    <v-card>
      <v-card-title>{{ tariffEdit.id ? 'Edit tariff' : 'New tariff' }}</v-card-title>
      <v-card-text>
        <v-text-field v-model="tariffEdit.name" label="Name" />
        <v-text-field v-model="tariffEdit.description" label="Description" />
        <v-row>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.priceMajor" type="number" label="Price (major units)" /></v-col>
          <v-col cols="6"><v-text-field v-model="tariffEdit.currency" label="Currency" maxlength="3" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.starsAmount" type="number" label="Stars amount (XTR)" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.addDays" type="number" label="+Days" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.addTrafficGB" type="number" label="+Traffic (GB, 0 = unlimited)" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.sort" type="number" label="Sort" /></v-col>
        </v-row>
        <v-switch v-model="tariffEdit.enabled" color="primary" label="Enabled" hide-details />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="tariffDialog = false">{{ $t('actions.cancel') ?? 'Cancel' }}</v-btn>
        <v-btn color="primary" @click="saveTariff">{{ $t('actions.set') ?? 'Save' }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Broadcast confirm dialog -->
  <v-dialog v-model="broadcastDialog" max-width="460">
    <v-card>
      <v-card-title>Send announcement?</v-card-title>
      <v-card-text>This will message all {{ recipientCount }} bound client(s).</v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="broadcastDialog = false">{{ $t('actions.cancel') ?? 'Cancel' }}</v-btn>
        <v-btn color="primary" @click="sendBroadcast">Send</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Refund confirm dialog -->
  <v-dialog v-model="refundDialog" max-width="480">
    <v-card>
      <v-card-title>Refund order #{{ refundEdit.id }}?</v-card-title>
      <v-card-text>
        <div class="mb-2">{{ refundEdit.provider }} · {{ formatMoney(refundEdit.amount, refundEdit.currency) }}</div>
        <v-alert :type="refundEdit.provider === 'stars' ? 'info' : 'warning'" variant="tonal" density="comfortable" class="mb-3">
          {{ refundEdit.provider === 'stars'
            ? 'Telegram Stars will be refunded automatically.'
            : 'Marks the order refunded only — refund the money in the provider\'s dashboard.' }}
        </v-alert>
        <v-switch
          v-model="refundEdit.revoke"
          color="primary"
          hide-details
          label="Revoke granted days/traffic from this order"
        />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="refundDialog = false">{{ $t('actions.cancel') ?? 'Cancel' }}</v-btn>
        <v-btn color="warning" :loading="refundBusy" @click="doRefund">Refund</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Unbind confirm dialog -->
  <v-dialog v-model="unbindDialog" max-width="440">
    <v-card>
      <v-card-title>Unbind Telegram from {{ unbindEdit.name }}?</v-card-title>
      <v-card-text>
        This removes the Telegram link for client #{{ unbindEdit.clientId }}
        (Telegram ID {{ unbindEdit.tgUserId }}). The client stays in the panel; only the binding is cleared.
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="unbindDialog = false">{{ $t('actions.cancel') ?? 'Cancel' }}</v-btn>
        <v-btn color="error" :loading="unbindBusy" @click="doUnbind">Unbind</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { HumanReadable } from '@/plugins/utils'
import SettingsSecretField from '@/components/SettingsSecretField.vue'
import { normalizeSecretFields, stripSecretPlaceholders } from '@/components/settingsSecretField'
import { push } from 'notivue'
import { i18n } from '@/locales'

type SMap = Record<string, string>

// The shared axios instance defaults POST bodies to x-www-form-urlencoded; the
// paidsub endpoints parse JSON, so these POSTs must opt into a JSON body.
const jsonPost = { headers: { 'Content-Type': 'application/json' } }

const defaults: SMap = {
  paidSubEnabled: 'false',
  paidSubBotToken: '',
  paidSubBotTokenHasSecret: 'false',
  paidSubBotPollSeconds: '25',
  paidSubTransportMode: 'proxy',
  paidSubProxyURL: '',
  paidSubProxyURLHasSecret: 'false',
  paidSubProxyUsername: '',
  paidSubProxyUsernameHasSecret: 'false',
  paidSubProxyPassword: '',
  paidSubProxyPasswordHasSecret: 'false',
  paidSubOutboundTag: '',
  paidSubAutoRegister: 'false',
  paidSubAutoInbounds: '[]',
  paidSubTrialDays: '3',
  paidSubTrialVolumeGB: '0',
  paidSubMaxClients: '5000',
  paidSubStartRateLimitPerMin: '3',
  paidSubCurrency: 'RUB',
  paidSubStarsEnabled: 'false',
  paidSubYooKassaEnabled: 'false',
  paidSubYooKassaToken: '',
  paidSubYooKassaTokenHasSecret: 'false',
  paidSubStripeEnabled: 'false',
  paidSubStripeToken: '',
  paidSubStripeTokenHasSecret: 'false',
  paidSubPayMasterEnabled: 'false',
  paidSubPayMasterToken: '',
  paidSubPayMasterTokenHasSecret: 'false',
  paidSubCryptoBotEnabled: 'false',
  paidSubCryptoBotToken: '',
  paidSubCryptoBotTokenHasSecret: 'false',
  paidSubExternalEnabled: 'false',
  paidSubExternalUrlTemplate: '',
  paidSubOrderTTLMinutes: '30',
  paidSubGreeting: '',
}

const tab = ref('bindings')
const loading = ref(false)
const settings = ref<SMap>({ ...defaults })
const secretboxKeySet = ref(true)

const pickSettings = (all: SMap): SMap => {
  const out: SMap = {}
  for (const k of Object.keys(defaults)) {
    if (all[k] !== undefined) out[k] = String(all[k])
    else out[k] = defaults[k]
  }
  return out
}

const boolSetting = (key: string) => computed({
  get: () => settings.value[key] === 'true',
  set: (v: boolean) => { settings.value[key] = v ? 'true' : 'false' },
})
const enabled = boolSetting('paidSubEnabled')
const autoRegister = boolSetting('paidSubAutoRegister')
const starsEnabled = boolSetting('paidSubStarsEnabled')
const yooEnabled = boolSetting('paidSubYooKassaEnabled')
const stripeEnabled = boolSetting('paidSubStripeEnabled')
const paymasterEnabled = boolSetting('paidSubPayMasterEnabled')
const cryptoEnabled = boolSetting('paidSubCryptoBotEnabled')
const externalEnabled = boolSetting('paidSubExternalEnabled')

const autoInbounds = computed<number[]>({
  get: () => {
    try { return JSON.parse(settings.value.paidSubAutoInbounds || '[]') } catch { return [] }
  },
  set: (v: number[]) => { settings.value.paidSubAutoInbounds = JSON.stringify(v) },
})

const loadSettings = async () => {
  const msg = await HttpUtils.get('api/settings')
  if (msg.success) {
    const normalized = normalizeSecretFields({ ...defaults, ...(msg.obj ?? {}) }) as SMap
    settings.value = pickSettings(normalized)
  }
}

const loadStatus = async () => {
  const msg = await HttpUtils.get('api/paidsub/status')
  if (msg.success) secretboxKeySet.value = !!msg.obj?.secretboxKeySet
}

const saveSettings = async () => {
  loading.value = true
  const payload = stripSecretPlaceholders(pickSettings(settings.value)) as SMap
  const msg = await HttpUtils.post('api/save', { object: 'settings', action: 'set', data: JSON.stringify(payload) })
  if (msg.success) {
    push.success({ title: i18n.global.t('success'), message: i18n.global.t('pages.paidSub'), duration: 4000 })
    if (msg.obj?.settings) {
      const normalized = normalizeSecretFields({ ...defaults, ...msg.obj.settings }) as SMap
      settings.value = pickSettings(normalized)
    }
  }
  loading.value = false
}

// ---- inbounds for the auto-reg selector ----
const inboundOptions = ref<{ title: string; value: number }[]>([])
const loadInbounds = async () => {
  const msg = await HttpUtils.get('api/inbounds')
  // api/inbounds returns { obj: { inbounds: [...] } } (LoadPartialData envelope).
  const list = msg?.obj?.inbounds
  if (msg.success && Array.isArray(list)) {
    inboundOptions.value = list.map((i: any) => ({ title: `${i.tag} (${i.type})`, value: i.id }))
  }
}

// ---- transport (proxy / outbound) ----
const transportModes = [
  { title: 'Proxy', value: 'proxy' },
  { title: 'Outbound (sing-box)', value: 'outbound' },
]
const outboundOptions = ref<{ title: string; value: string }[]>([])
const loadOutbounds = async () => {
  const msg = await HttpUtils.get('api/outbounds')
  const list = msg?.obj?.outbounds
  if (msg.success && Array.isArray(list)) {
    outboundOptions.value = list.map((o: any) => ({ title: `${o.tag} (${o.type})`, value: o.tag }))
  }
}

// ---- bindings ----
const bindings = ref<any[]>([])
const bindingsLoading = ref(false)
const bindingHeaders = [
  { title: 'Client', key: 'name' },
  { title: 'Client ID', key: 'clientId' },
  { title: 'Description', key: 'desc' },
  { title: 'Telegram ID', key: 'tgUserId' },
  { title: 'Expiry', key: 'expiry' },
  { title: 'Status', key: 'enable' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
]
const bindingDialog = ref(false)
const bindingEdit = ref<{ clientId: number; name: string; tgUserId: number | string; isNew: boolean }>({ clientId: 0, name: '', tgUserId: 0, isNew: false })

// Clients available for the "Add binding" selector (all clients from the
// bindings list, which already enumerates every client in the panel).
const clientOptions = computed(() => bindings.value.map((b: any) => ({
  title: b.tgUserId ? `${b.name} (bound: ${b.tgUserId})` : b.name,
  value: b.clientId,
})))

const loadBindings = async () => {
  bindingsLoading.value = true
  const msg = await HttpUtils.get('api/paidsub/bindings')
  if (msg.success) bindings.value = msg.obj ?? []
  bindingsLoading.value = false
}
const openBinding = (item: any) => {
  bindingEdit.value = { clientId: item.clientId, name: item.name, tgUserId: item.tgUserId || '', isNew: false }
  bindingDialog.value = true
}
const openAddBinding = () => {
  bindingEdit.value = { clientId: bindings.value[0]?.clientId ?? 0, name: '', tgUserId: '', isNew: true }
  bindingDialog.value = true
}
const saveBinding = async () => {
  if (!bindingEdit.value.clientId) return
  const tgUserId = Number(bindingEdit.value.tgUserId) || 0
  const msg = await HttpUtils.post('api/paidsub/bindings', { clientId: bindingEdit.value.clientId, tgUserId }, jsonPost)
  if (msg.success) { bindingDialog.value = false; await loadBindings() }
}
const unbindDialog = ref(false)
const unbindBusy = ref(false)
const unbindEdit = ref<{ clientId: number; name: string; tgUserId: number | string }>({ clientId: 0, name: '', tgUserId: 0 })
const openUnbindConfirm = (item: any) => {
  unbindEdit.value = { clientId: item.clientId, name: item.name, tgUserId: item.tgUserId }
  unbindDialog.value = true
}
const doUnbind = async () => {
  unbindBusy.value = true
  const msg = await HttpUtils.post('api/paidsub/bindings', { clientId: unbindEdit.value.clientId, tgUserId: 0 }, jsonPost)
  unbindBusy.value = false
  if (msg.success) { unbindDialog.value = false; await loadBindings() }
}

// ---- messages: greeting + broadcast ----
const recipientCount = computed(() => bindings.value.filter((b: any) => b.tgUserId).length)
const broadcastText = ref('')
const broadcastLoading = ref(false)
const broadcastDialog = ref(false)
const broadcastResult = ref<{ sent: number; failed: number } | null>(null)
const sendBroadcast = async () => {
  broadcastDialog.value = false
  broadcastLoading.value = true
  broadcastResult.value = null
  const msg = await HttpUtils.post('api/paidsub/broadcast', { text: broadcastText.value }, jsonPost)
  if (msg.success) {
    broadcastResult.value = { sent: Number(msg.obj?.sent ?? 0), failed: Number(msg.obj?.failed ?? 0) }
    broadcastText.value = ''
  }
  broadcastLoading.value = false
}

// ---- tariffs ----
const tariffs = ref<any[]>([])
const tariffsLoading = ref(false)
const tariffHeaders = [
  { title: 'Name', key: 'name' },
  { title: 'Price', key: 'price' },
  { title: 'Stars', key: 'starsAmount' },
  { title: '+Days', key: 'addDays' },
  { title: '+Traffic', key: 'addTrafficBytes' },
  { title: 'Enabled', key: 'enabled' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
]
const tariffDialog = ref(false)
const blankTariff = () => ({ id: 0, name: '', description: '', priceMajor: 0, currency: settings.value.paidSubCurrency || 'RUB', starsAmount: 0, addDays: 30, addTrafficGB: 0, sort: 0, enabled: true })
const tariffEdit = ref<any>(blankTariff())

const loadTariffs = async () => {
  tariffsLoading.value = true
  const msg = await HttpUtils.get('api/paidsub/tariffs')
  if (msg.success) tariffs.value = msg.obj ?? []
  tariffsLoading.value = false
}
const openTariff = (item?: any) => {
  if (item) {
    tariffEdit.value = {
      id: item.id, name: item.name, description: item.description,
      priceMajor: (item.price || 0) / 100, currency: item.currency,
      starsAmount: item.starsAmount || 0, addDays: item.addDays || 0,
      addTrafficGB: (item.addTrafficBytes || 0) / (1024 * 1024 * 1024),
      sort: item.sort || 0, enabled: !!item.enabled,
    }
  } else {
    tariffEdit.value = blankTariff()
  }
  tariffDialog.value = true
}
const saveTariff = async () => {
  const e = tariffEdit.value
  const data: any = {
    name: e.name, description: e.description,
    price: Math.round(Number(e.priceMajor) * 100),
    currency: (e.currency || 'RUB').toUpperCase(),
    starsAmount: Math.round(Number(e.starsAmount) || 0),
    addDays: Math.round(Number(e.addDays) || 0),
    addTrafficBytes: Math.round((Number(e.addTrafficGB) || 0) * 1024 * 1024 * 1024),
    sort: Math.round(Number(e.sort) || 0),
    enabled: !!e.enabled,
  }
  const action = e.id ? 'edit' : 'new'
  if (e.id) data.id = e.id
  const msg = await HttpUtils.post('api/paidsub/tariffs', { action, data }, jsonPost)
  if (msg.success) { tariffDialog.value = false; await loadTariffs() }
}
const deleteTariff = async (item: any) => {
  const msg = await HttpUtils.post('api/paidsub/tariffs', { action: 'del', data: item.id }, jsonPost)
  if (msg.success) await loadTariffs()
}

// ---- orders ----
const orders = ref<any[]>([])
const ordersLoading = ref(false)
const orderHeaders = [
  { title: 'ID', key: 'id' },
  { title: 'Client Name', key: 'clientName' },
  { title: 'Telegram ID', key: 'telegramUserId' },
  { title: 'Description', key: 'clientDesc' },
  { title: 'Provider', key: 'provider' },
  { title: 'Amount', key: 'amount' },
  { title: 'Status', key: 'status' },
  { title: 'Created', key: 'createdAt' },
  { title: '', key: 'actions', sortable: false },
]
const loadOrders = async () => {
  ordersLoading.value = true
  const msg = await HttpUtils.get('api/paidsub/orders')
  if (msg.success) orders.value = msg.obj ?? []
  ordersLoading.value = false
}
const orderStatusColor = (s: string) =>
  ({ paid: 'success', pending: 'warning', failed: 'error', expired: 'grey', canceled: 'grey', refunded: 'info' } as any)[s] || 'grey'

// ---- refund (admin-initiated) ----
const refundDialog = ref(false)
const refundBusy = ref(false)
const refundEdit = ref<{ id: number; provider: string; amount: number; currency: string; revoke: boolean }>({
  id: 0, provider: '', amount: 0, currency: '', revoke: true,
})
const openRefund = (item: any) => {
  refundEdit.value = { id: item.id, provider: item.provider, amount: item.amount, currency: item.currency, revoke: true }
  refundDialog.value = true
}
const doRefund = async () => {
  refundBusy.value = true
  const msg = await HttpUtils.post('api/paidsub/refund', { orderId: refundEdit.value.id, revoke: refundEdit.value.revoke }, jsonPost)
  refundBusy.value = false
  if (msg.success) {
    refundDialog.value = false
    push.success({ title: i18n.global.t('success'), message: 'Refund processed', duration: 4000 })
    await loadOrders()
  }
}
// Telegram Stars (XTR) are whole units; fiat amounts are stored in minor units.
const formatMoney = (amount: number, currency: string) =>
  currency === 'XTR' ? `${Number(amount)} ${currency}` : `${(Number(amount) / 100).toFixed(2)} ${currency}`

const reloadAll = async () => {
  loading.value = true
  await Promise.all([loadSettings(), loadStatus(), loadInbounds(), loadOutbounds(), loadBindings(), loadTariffs(), loadOrders()])
  loading.value = false
}

onMounted(reloadAll)
</script>
