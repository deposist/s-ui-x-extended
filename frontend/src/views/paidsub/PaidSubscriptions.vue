<template>
  <page-header v-if="nexus" :title="$t('pages.paidSub')" />
  <v-card flat>
    <v-card-title class="d-flex align-center">
      <template v-if="!nexus">
        <v-icon class="mr-2">mdi-cash-multiple</v-icon>
        {{ $t('pages.paidSub') }}
      </template>
      <v-chip :class="nexus ? '' : 'ml-3'" color="warning" size="small" variant="flat">{{ $t('paidSub.experimental') }}</v-chip>
      <v-spacer />
      <v-btn color="primary" :loading="loading" variant="tonal" @click="reloadAll">
        <v-icon start>lucide:rotate-cw</v-icon>{{ $t('actions.refresh') }}
      </v-btn>
    </v-card-title>

    <v-alert
      v-if="!secretboxKeySet"
      type="warning"
      variant="tonal"
      class="mx-4 mb-2"
      density="comfortable"
    >
      {{ $t('paidSub.secretboxWarning') }}
    </v-alert>

    <v-alert class="mb-3" :type="awgStatus?.enabled && awgStatus?.coreReachable && awgStatus?.encryptionKeyAvailable ? 'success' : 'warning'" variant="tonal">
      AWG: {{ awgStatus?.enabled ? 'enabled' : 'disabled' }} · core {{ awgStatus?.coreReachable ? 'online' : 'offline' }} · desired {{ awgStatus?.desired ?? 0 }} · provisioned {{ awgStatus?.provisioned ?? 0 }} · pending {{ awgStatus?.pending ?? 0 }} · errors {{ awgStatus?.errors ?? 0 }}
    </v-alert>

    <v-tabs v-model="tab" color="primary" class="px-2">
      <v-tab value="bindings">{{ $t('paidSub.tabs.bindings') }}</v-tab>
      <v-tab value="autoreg">{{ $t('paidSub.tabs.autoreg') }}</v-tab>
      <v-tab value="tariffs">{{ $t('paidSub.tabs.tariffs') }}</v-tab>
      <v-tab value="payments">{{ $t('paidSub.tabs.payments') }}</v-tab>
      <v-tab value="messages">{{ $t('paidSub.tabs.messages') }}</v-tab>
      <v-tab value="orders">{{ $t('paidSub.tabs.orders') }}</v-tab>
      <v-tab value="bot">{{ $t('paidSub.tabs.bot') }}</v-tab>
    </v-tabs>

    <v-window v-model="tab" class="pa-4">
      <!-- BINDINGS -->
      <v-window-item value="bindings">
        <div class="d-flex align-center mb-2">
          <div class="text-caption text-medium-emphasis">
            {{ $t('paidSub.bindings.hint') }}
          </div>
          <v-spacer />
          <v-btn color="primary" :disabled="bindings.length === 0" @click="openAddBinding()">
            <v-icon start>lucide:plus</v-icon>{{ $t('paidSub.bindings.add') }}
          </v-btn>
        </div>
        <v-alert v-if="!bindingsLoading && bindings.length === 0" type="info" variant="tonal" density="comfortable">
          {{ $t('paidSub.bindings.empty') }}
        </v-alert>
        <nexus-data-table
          v-else-if="nexus"
          :columns="bindingColumns"
          :items="bindings"
          :loading="bindingsLoading"
          :row-key="(item) => item.clientId"
        >
          <template #col.name="{ item }">
            <span class="paidsub-nexus__name">{{ item.name }}</span>
          </template>
          <template #col.enable="{ item }">
            <status-badge :label="item.enable ? $t('paidSub.active') : $t('paidSub.disabled')" :tone="item.enable ? 'success' : 'error'" />
          </template>
          <template #col.tgUserId="{ item }">
            <span v-if="item.tgUserId">{{ item.tgUserId }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #col.desc="{ item }">
            <span v-if="item.desc">{{ item.desc }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #col.expiry="{ item }">
            <template v-if="item.expiry > 0">
              {{ new Date(item.expiry * 1000).toLocaleString() }}
              (<v-chip size="x-small" label :color="item.expiry <= Date.now() / 1000 ? 'error' : ''">{{ HumanReadable.remainedDays(item.expiry) }}</v-chip>)
            </template>
            <v-chip v-else size="small" color="success" label>{{ HumanReadable.remainedDays(item.expiry) }}</v-chip>
          </template>
          <template #actions="{ item }">
            <row-actions :actions="bindingActions(item)" @action="(key) => handleBindingAction(key, item)" />
          </template>
          <template #empty>
            <empty-state icon="lucide:link" :title="$t('paidSub.bindings.none')" />
          </template>
        </nexus-data-table>
        <v-data-table v-else :headers="bindingHeaders" :items="bindings" :loading="bindingsLoading" density="comfortable">
          <template #item.enable="{ item }">
            <v-chip :color="item.enable ? 'success' : 'error'" size="small" variant="flat">
              {{ item.enable ? $t('paidSub.active') : $t('paidSub.disabled') }}
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
            <v-switch v-model="autoRegister" color="primary" :label="$t('paidSub.autoreg.enable')" hide-details />
          </v-col>
          <v-col cols="12" md="6">
            <v-select
              v-model="autoInbounds"
              :items="inboundOptions"
              item-title="title"
              item-value="value"
              :label="$t('paidSub.autoreg.inbounds')"
              multiple
              chips
            />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubTrialDays" type="number" :label="$t('paidSub.autoreg.trialDays')" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubTrialVolumeGB" type="number" :label="$t('paidSub.autoreg.trialVolume')" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubMaxClients" type="number" :label="$t('paidSub.autoreg.maxClients')" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubStartRateLimitPerMin" type="number" :label="$t('paidSub.autoreg.rateLimit')" />
          </v-col>
        </v-row>
        <v-divider class="my-4" />
        <div class="text-subtitle-2 mb-2">AmneziaWG 3.0</div>
        <v-row>
          <v-col cols="12" md="4"><v-switch v-model="awgEnabled" color="primary" label="Enable managed AWG devices" hide-details /></v-col>
          <v-col cols="12" md="4"><v-text-field v-model="settings.awgEndpointTag" label="Managed endpoint tag" /></v-col>
          <v-col cols="12" md="4"><v-text-field v-model="settings.awgPublicEndpoint" label="Public endpoint host:port" /></v-col>
          <v-col cols="12" md="4"><v-text-field v-model="settings.awgSubnet" label="IPv4 subnet" /></v-col>
          <v-col cols="12" md="4"><v-text-field v-model="settings.awgDNS" label="DNS" /></v-col>
          <v-col cols="6" md="2"><v-text-field v-model="settings.awgDefaultDeviceLimit" type="number" label="Default device limit" /></v-col>
          <v-col cols="6" md="2"><v-text-field v-model="settings.awgMTU" type="number" label="MTU (0 = endpoint)" /></v-col>
          <v-col cols="6" md="2"><v-text-field v-model="settings.awgReconcileIntervalSec" type="number" label="Reconcile seconds" /></v-col>
          <v-col cols="6" md="2"><v-text-field v-model="settings.awgStatsIntervalSec" type="number" label="Stats seconds" /></v-col>
        </v-row>
        <v-btn color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') }}</v-btn>
      </v-window-item>

      <!-- TARIFFS -->
      <v-window-item value="tariffs">
        <div class="d-flex mb-2">
          <v-spacer />
          <v-btn color="primary" @click="openTariff()"><v-icon start>lucide:plus</v-icon>{{ $t('paidSub.tariffs.add') }}</v-btn>
        </div>
        <nexus-data-table
          v-if="nexus"
          :columns="tariffColumns"
          :items="tariffs"
          :loading="tariffsLoading"
          :row-key="(item) => item.id"
        >
          <template #col.name="{ item }">
            <span class="paidsub-nexus__name">{{ item.name }}</span>
          </template>
          <template #col.price="{ item }">{{ (item.price / 100).toFixed(2) }} {{ item.currency }}</template>
          <template #col.starsAmount="{ item }">{{ item.starsAmount || '—' }}</template>
          <template #col.addTrafficBytes="{ item }">{{ item.addTrafficBytes ? (item.addTrafficBytes / (1024*1024*1024)).toFixed(2) + ' GB' : '∞' }}</template>
          <template #col.enabled="{ item }">
            <status-badge :label="item.enabled ? $t('nexus.on') : $t('nexus.off')" :tone="item.enabled ? 'success' : 'info'" />
          </template>
          <template #actions="{ item }">
            <row-actions :actions="tariffActions()" @action="(key) => handleTariffAction(key, item)" />
          </template>
          <template #empty>
            <empty-state icon="lucide:tag" :title="$t('paidSub.tariffs.none')" />
          </template>
        </nexus-data-table>
        <v-data-table v-else :headers="tariffHeaders" :items="tariffs" :loading="tariffsLoading" density="comfortable">
          <template #item.price="{ item }">{{ (item.price / 100).toFixed(2) }} {{ item.currency }}</template>
          <template #item.starsAmount="{ item }">{{ item.starsAmount || '—' }}</template>
          <template #item.addTrafficBytes="{ item }">{{ item.addTrafficBytes ? (item.addTrafficBytes / (1024*1024*1024)).toFixed(2) + ' GB' : '∞' }}</template>
          <template #item.enabled="{ item }">
            <v-chip :color="item.enabled ? 'success' : 'grey'" size="small" variant="flat">{{ item.enabled ? $t('nexus.on') : $t('nexus.off') }}</v-chip>
          </template>
          <template #item.actions="{ item }">
            <v-btn size="small" variant="text" icon="mdi-pencil" @click="openTariff(item)" />
            <v-btn size="small" variant="text" icon="mdi-delete" color="error" @click="openDeleteTariffConfirm(item)" />
          </template>
        </v-data-table>
      </v-window-item>

      <!-- PAYMENTS -->
      <v-window-item value="payments">
        <v-row>
          <v-col cols="12" md="4">
            <v-select v-model="settings.paidSubCurrency" :items="currencies" :label="$t('paidSub.payments.currency')" />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field v-model="settings.paidSubOrderTTLMinutes" type="number" :label="$t('paidSub.payments.orderTtl')" />
          </v-col>
        </v-row>
        <v-divider class="my-2" />
        <v-switch v-model="starsEnabled" color="primary" :label="$t('paidSub.payments.stars')" hide-details />
        <v-divider class="my-2" />
        <v-switch v-model="yooEnabled" color="primary" :label="$t('paidSub.payments.yookassa')" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubYooKassaToken"
          :has-secret="settings.paidSubYooKassaTokenHasSecret"
          :label="$t('paidSub.payments.yookassaToken')"
        />
        <v-divider class="my-2" />
        <v-switch v-model="stripeEnabled" color="primary" :label="$t('paidSub.payments.stripe')" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubStripeToken"
          :has-secret="settings.paidSubStripeTokenHasSecret"
          :label="$t('paidSub.payments.stripeToken')"
        />
        <v-divider class="my-2" />
        <v-switch v-model="paymasterEnabled" color="primary" :label="$t('paidSub.payments.paymaster')" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubPayMasterToken"
          :has-secret="settings.paidSubPayMasterTokenHasSecret"
          :label="$t('paidSub.payments.paymasterToken')"
        />
        <v-divider class="my-2" />
        <v-switch v-model="cryptoEnabled" color="primary" :label="$t('paidSub.payments.crypto')" hide-details />
        <SettingsSecretField
          v-model="settings.paidSubCryptoBotToken"
          :has-secret="settings.paidSubCryptoBotTokenHasSecret"
          :label="$t('paidSub.payments.cryptoToken')"
        />
        <v-divider class="my-2" />
        <v-switch v-model="externalEnabled" color="primary" :label="$t('paidSub.payments.external')" hide-details />
        <v-text-field
          v-model="settings.paidSubExternalUrlTemplate"
          :label="$t('paidSub.payments.externalTemplate')"
        />
        <v-btn class="mt-2" color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') }}</v-btn>
      </v-window-item>

      <!-- MESSAGES -->
      <v-window-item value="messages">
        <div class="text-subtitle-2 mb-1">{{ $t('paidSub.messages.greetingTitle') }}</div>
        <div class="text-caption text-medium-emphasis mb-2">
          {{ $t('paidSub.messages.greetingHint') }}
        </div>
        <v-textarea v-model="settings.paidSubGreeting" :label="$t('paidSub.messages.greetingLabel')" rows="3" auto-grow counter="4096" />
        <v-btn color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') }}</v-btn>

        <v-divider class="my-4" />

        <div class="text-subtitle-2 mb-1">{{ $t('paidSub.messages.broadcastTitle') }}</div>
        <div class="text-caption text-medium-emphasis mb-2">
          {{ $t('paidSub.messages.broadcastHint', { count: recipientCount }) }}
        </div>
        <v-textarea v-model="broadcastText" :label="$t('paidSub.messages.broadcastLabel')" rows="4" auto-grow counter="4096" />
        <v-btn color="primary" :loading="broadcastLoading" :disabled="!broadcastText.trim() || recipientCount === 0" @click="broadcastDialog = true">
          <v-icon start>lucide:megaphone</v-icon>{{ $t('paidSub.messages.sendAll') }}
        </v-btn>
        <v-alert v-if="broadcastResult" type="info" variant="tonal" class="mt-3" density="comfortable">
          {{ $t('paidSub.messages.result', { sent: broadcastResult.sent, failed: broadcastResult.failed }) }}
        </v-alert>
      </v-window-item>

      <!-- ORDERS -->
      <v-window-item value="orders">
        <nexus-data-table
          v-if="nexus"
          :columns="orderColumns"
          :items="orders"
          :loading="ordersLoading"
          :row-key="(item) => item.id"
        >
          <template #col.clientName="{ item }">
            <span v-if="item.clientName">{{ item.clientName }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #col.telegramUserId="{ item }">
            <span v-if="item.telegramUserId">{{ item.telegramUserId }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #col.clientDesc="{ item }">
            <span v-if="item.clientDesc">{{ item.clientDesc }}</span>
            <span v-else class="text-disabled">—</span>
          </template>
          <template #col.amount="{ item }">{{ formatMoney(item.amount, item.currency) }}</template>
          <template #col.status="{ item }">
            <status-badge :label="item.status" :tone="orderStatusTone(item.status)" />
          </template>
          <template #col.createdAt="{ item }">{{ item.createdAt ? new Date(item.createdAt * 1000).toLocaleString() : '' }}</template>
          <template #actions="{ item }">
            <row-actions :actions="orderActions(item)" @action="(key) => handleOrderAction(key, item)" />
          </template>
          <template #empty>
            <!-- No action button here: orders are created by paying customers,
                 not by the admin, so there is nothing for them to click. -->
            <empty-state
              :description="$t('emptyState.ordersHint')"
              icon="lucide:receipt"
              :title="$t('emptyState.orders')"
            />
          </template>
        </nexus-data-table>
        <v-data-table v-else :headers="orderHeaders" :items="orders" :loading="ordersLoading" density="comfortable">
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
            <v-btn v-if="item.status === 'paid'" size="small" variant="text" color="warning" @click="openRefund(item)">{{ $t('paidSub.orders.refund') }}</v-btn>
          </template>
        </v-data-table>
      </v-window-item>

      <!-- BOT -->
      <v-window-item value="bot">
        <v-row>
          <v-col cols="12" md="6">
            <v-switch v-model="enabled" color="primary" :label="$t('paidSub.bot.enable')" hide-details />
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field v-model="settings.paidSubBotPollSeconds" type="number" :label="$t('paidSub.bot.pollTimeout')" />
          </v-col>
          <v-col cols="12">
            <SettingsSecretField
              v-model="settings.paidSubBotToken"
              :has-secret="settings.paidSubBotTokenHasSecret"
              :label="$t('paidSub.bot.token')"
            />
          </v-col>
        </v-row>

        <v-divider class="my-3" />
        <div class="text-subtitle-2 mb-1">{{ $t('paidSub.bot.transportTitle') }}</div>
        <div class="text-caption text-medium-emphasis mb-2">
          {{ $t('paidSub.bot.transportHint') }}
        </div>
        <v-row>
          <v-col cols="12" md="4">
            <v-select
              v-model="settings.paidSubTransportMode"
              :items="transportModes"
              item-title="title"
              item-value="value"
              :label="$t('paidSub.bot.transport')"
            />
          </v-col>
          <v-col v-if="settings.paidSubTransportMode === 'outbound'" cols="12" md="8">
            <v-select
              v-model="settings.paidSubOutboundTag"
              :items="outboundOptions"
              item-title="title"
              item-value="value"
              :label="$t('paidSub.bot.outbound')"
              :hint="outboundOptions.length === 0 ? $t('paidSub.bot.noOutbounds') : ''"
              persistent-hint
            />
          </v-col>
          <template v-else>
            <v-col cols="12" md="8">
              <SettingsSecretField
                v-model="settings.paidSubProxyURL"
                :has-secret="settings.paidSubProxyURLHasSecret"
                :label="$t('paidSub.bot.proxyUrl')"
              />
            </v-col>
            <v-col cols="12" md="6">
              <SettingsSecretField
                v-model="settings.paidSubProxyUsername"
                :has-secret="settings.paidSubProxyUsernameHasSecret"
                :label="$t('paidSub.bot.proxyUser')"
              />
            </v-col>
            <v-col cols="12" md="6">
              <SettingsSecretField
                v-model="settings.paidSubProxyPassword"
                :has-secret="settings.paidSubProxyPasswordHasSecret"
                :label="$t('paidSub.bot.proxyPass')"
              />
            </v-col>
          </template>
        </v-row>

        <v-btn color="primary" :loading="loading" @click="saveSettings">{{ $t('actions.set') }}</v-btn>
      </v-window-item>
    </v-window>
  </v-card>

  <!-- Binding dialog -->
  <v-dialog v-model="bindingDialog" max-width="420">
    <v-card>
      <v-card-title>{{ bindingEdit.isNew ? $t('paidSub.bindingDialog.addTitle') : $t('paidSub.bindingDialog.editTitle', { name: bindingEdit.name }) }}</v-card-title>
      <v-card-text>
        <v-select
          v-if="bindingEdit.isNew"
          v-model="bindingEdit.clientId"
          :items="clientOptions"
          item-title="title"
          item-value="value"
          :label="$t('paidSub.bindingDialog.client')"
        />
        <v-text-field v-model="bindingEdit.tgUserId" type="number" :label="$t('paidSub.bindingDialog.tgId')" autofocus />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="bindingDialog = false">{{ $t('actions.cancel') }}</v-btn>
        <v-btn color="primary" @click="saveBinding">{{ $t('actions.set') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Tariff dialog -->
  <v-dialog v-model="tariffDialog" max-width="560">
    <v-card>
      <v-card-title>{{ tariffEdit.id ? $t('paidSub.tariffs.edit') : $t('paidSub.tariffs.new') }}</v-card-title>
      <v-card-text>
        <v-text-field v-model="tariffEdit.name" :label="$t('paidSub.cols.name')" />
        <v-text-field v-model="tariffEdit.description" :label="$t('paidSub.cols.description')" />
        <v-row>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.priceMajor" type="number" :label="$t('paidSub.tariffs.priceMajor')" /></v-col>
          <v-col cols="6"><v-select v-model="tariffEdit.currency" :items="currencies" :label="$t('paidSub.tariffs.currency')" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.starsAmount" type="number" :label="$t('paidSub.tariffs.starsAmount')" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.addDays" type="number" :label="$t('paidSub.tariffs.addDays')" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.addTrafficGB" type="number" :label="$t('paidSub.tariffs.addTrafficGB')" /></v-col>
		  <v-col cols="6"><v-text-field v-model.number="tariffEdit.maxAwgDevices" type="number" min="0" max="100" :label="$t('paidSub.tariffs.maxAwgDevices')" /></v-col>
          <v-col cols="6"><v-text-field v-model.number="tariffEdit.sort" type="number" :label="$t('paidSub.tariffs.sort')" /></v-col>
        </v-row>
        <v-switch v-model="tariffEdit.enabled" color="primary" :label="$t('paidSub.tariffs.enabledField')" hide-details />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="tariffDialog = false">{{ $t('actions.cancel') }}</v-btn>
        <v-btn color="primary" @click="saveTariff">{{ $t('actions.set') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Delete tariff confirm dialog -->
  <v-dialog v-model="deleteTariffDialog" max-width="440">
    <v-card>
      <v-card-title>{{ $t('actions.del') }} {{ deleteTariffEdit.name }}</v-card-title>
      <v-card-text>{{ $t('confirm') }}</v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="deleteTariffDialog = false">{{ $t('actions.cancel') }}</v-btn>
        <v-btn color="error" :loading="deleteTariffBusy" @click="deleteTariff">{{ $t('actions.del') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Broadcast confirm dialog -->
  <v-dialog v-model="broadcastDialog" max-width="460">
    <v-card>
      <v-card-title>{{ $t('paidSub.messages.confirmTitle') }}</v-card-title>
      <v-card-text>{{ $t('paidSub.messages.confirmText', { count: recipientCount }) }}</v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="broadcastDialog = false">{{ $t('actions.cancel') }}</v-btn>
        <v-btn color="primary" @click="sendBroadcast">{{ $t('paidSub.messages.send') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Refund confirm dialog -->
  <v-dialog v-model="refundDialog" max-width="480">
    <v-card>
      <v-card-title>{{ $t('paidSub.refund.title', { id: refundEdit.id }) }}</v-card-title>
      <v-card-text>
        <div class="mb-2">{{ refundEdit.provider }} · {{ formatMoney(refundEdit.amount, refundEdit.currency) }}</div>
        <v-alert :type="refundEdit.provider === 'stars' ? 'info' : 'warning'" variant="tonal" density="comfortable" class="mb-3">
          {{ refundEdit.provider === 'stars' ? $t('paidSub.refund.starsNote') : $t('paidSub.refund.manualNote') }}
        </v-alert>
        <v-switch
          v-model="refundEdit.revoke"
          color="primary"
          hide-details
          :label="$t('paidSub.refund.revoke')"
        />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="refundDialog = false">{{ $t('actions.cancel') }}</v-btn>
        <v-btn color="warning" :loading="refundBusy" @click="doRefund">{{ $t('paidSub.orders.refund') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Unbind confirm dialog -->
  <v-dialog v-model="unbindDialog" max-width="440">
    <v-card>
      <v-card-title>{{ $t('paidSub.unbind.title', { name: unbindEdit.name }) }}</v-card-title>
      <v-card-text>
        {{ $t('paidSub.unbind.text', { clientId: unbindEdit.clientId, tgUserId: unbindEdit.tgUserId }) }}
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="unbindDialog = false">{{ $t('actions.cancel') }}</v-btn>
        <v-btn color="error" :loading="unbindBusy" @click="doUnbind">{{ $t('paidSub.unbind.confirm') }}</v-btn>
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
import type { Column } from '@/components/nexus/data/dataTableColumns'
import NexusDataTable from '@/components/nexus/data/NexusDataTable.vue'
import RowActions from '@/components/nexus/data/RowActions.vue'
import type { RowAction } from '@/components/nexus/data/rowActions'
import EmptyState from '@/components/nexus/primitives/EmptyState.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import StatusBadge from '@/components/nexus/primitives/StatusBadge.vue'
import { useUiMode } from '@/uiMode/useUiMode'
import { inboundAssignable } from '@/types/capabilities'

const { mode } = useUiMode()
const nexus = computed(() => mode.value === 'nexus')

// Nexus column labels are i18n keys (resolved via $t in NexusTableHeader); the
// classic v-data-table headers below render titles directly, so they use
// i18n.global.t. EN + RU translated; other locales fall back to EN.
const bindingColumns: Column<any>[] = [
  { key: 'name', labelKey: 'paidSub.cols.client', sortable: true },
  { key: 'clientId', labelKey: 'paidSub.cols.clientId', sortable: true },
  { key: 'desc', labelKey: 'paidSub.cols.description' },
  { key: 'tgUserId', labelKey: 'paidSub.cols.telegramId' },
  { key: 'expiry', labelKey: 'paidSub.cols.expiry', sortable: true },
  { key: 'enable', labelKey: 'paidSub.cols.status' },
]
const tariffColumns: Column<any>[] = [
  { key: 'name', labelKey: 'paidSub.cols.name', sortable: true },
  { key: 'price', labelKey: 'paidSub.cols.price' },
  { key: 'starsAmount', labelKey: 'paidSub.cols.stars' },
  { key: 'addDays', labelKey: 'paidSub.cols.addDays' },
  { key: 'addTrafficBytes', labelKey: 'paidSub.cols.addTraffic' },
  { key: 'maxAwgDevices', labelKey: 'paidSub.cols.maxAwgDevices' },
  { key: 'enabled', labelKey: 'paidSub.cols.enabled' },
]
const orderColumns: Column<any>[] = [
  { key: 'id', labelKey: 'paidSub.cols.id', sortable: true },
  { key: 'clientName', labelKey: 'paidSub.cols.clientName' },
  { key: 'telegramUserId', labelKey: 'paidSub.cols.telegramId' },
  { key: 'clientDesc', labelKey: 'paidSub.cols.description' },
  { key: 'provider', labelKey: 'paidSub.cols.provider' },
  { key: 'amount', labelKey: 'paidSub.cols.amount' },
  { key: 'status', labelKey: 'paidSub.cols.status', sortable: true },
  { key: 'createdAt', labelKey: 'paidSub.cols.created', sortable: true },
]

const bindingActions = (item: any): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'unbind', labelKey: 'paidSub.unbind.confirm', icon: 'lucide:unlink', tone: 'error', inline: true, hidden: !item.tgUserId },
]
const tariffActions = (): RowAction[] => [
  { key: 'edit', labelKey: 'actions.edit', icon: 'lucide:pencil', inline: true },
  { key: 'del', labelKey: 'actions.del', icon: 'lucide:trash-2', tone: 'error', inline: true },
]
const orderActions = (item: any): RowAction[] => [
  { key: 'refund', labelKey: 'paidSub.orders.refund', icon: 'lucide:rotate-ccw', inline: true, hidden: item.status !== 'paid' },
]

const handleBindingAction = (key: string, item: any) => {
  if (key === 'edit') openBinding(item)
  else if (key === 'unbind') openUnbindConfirm(item)
}
const handleTariffAction = (key: string, item: any) => {
  if (key === 'edit') openTariff(item)
  else if (key === 'del') openDeleteTariffConfirm(item)
}
const handleOrderAction = (key: string, item: any) => {
  if (key === 'refund') openRefund(item)
}
const orderStatusTone = (status: string): 'info' | 'success' | 'warning' | 'error' =>
  status === 'paid' ? 'success' : status === 'pending' ? 'warning' : status === 'failed' ? 'error' : 'info'

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
  awgEnabled: 'false',
  awgEndpointTag: '',
  awgPublicEndpoint: '',
  awgSubnet: '10.77.0.0/16',
  awgDNS: '1.1.1.1,1.0.0.1',
  awgDefaultDeviceLimit: '3',
  awgReconcileIntervalSec: '30',
  awgStatsIntervalSec: '60',
  awgMTU: '0',
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
  // Warn about unknown keys from the backend so schema drift is visible.
  const known = new Set(Object.keys(defaults))
  const unknown = Object.keys(all).filter(k => !known.has(k))
  if (unknown.length > 0) {
    console.warn('[paidsub] unknown settings keys from backend, ignored:', unknown.join(', '))
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
const awgEnabled = boolSetting('awgEnabled')

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
  } else {
    push.error({ title: i18n.global.t('failed'), message: i18n.global.t('pages.paidSub') + ': settings' })
  }
}

const loadStatus = async () => {
  const msg = await HttpUtils.get('api/paidsub/status')
  if (msg.success) secretboxKeySet.value = !!msg.obj?.secretboxKeySet
  else push.error({ title: i18n.global.t('failed'), message: i18n.global.t('pages.paidSub') + ': status' })
}

const awgStatus = ref<any>(null)
const loadAWGStatus = async () => {
  const msg = await HttpUtils.get('api/paidsub/awg/status')
  if (msg.success) awgStatus.value = msg.obj
}

const saveSettings = async () => {
  settings.value.paidSubCurrency = normalizeCurrency(settings.value.paidSubCurrency)
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
    // Only client-assignable inbounds make sense as auto-register targets
    // (direct/tun/bond/tproxy etc. cannot deliver anything to a client).
    inboundOptions.value = list
      .filter((i: any) => inboundAssignable.includes(i.type))
      .map((i: any) => ({ title: `${i.tag} (${i.type})`, value: i.id }))
  }
}

// ---- transport (proxy / outbound) ----
// Common payment currencies accepted by the backend/payment providers. XTR = Telegram Stars.
const currencies = ['RUB', 'USD', 'EUR', 'GBP', 'UAH', 'KZT', 'BYN', 'XTR']
const isAllowedCurrency = (currency: string) => currencies.includes(String(currency || '').toUpperCase())
const normalizeCurrency = (currency: string, fallback = 'RUB') => {
  const normalized = String(currency || '').toUpperCase()
  return isAllowedCurrency(normalized) ? normalized : fallback
}
const transportModes = [
  { title: i18n.global.t('paidSub.transportModes.proxy'), value: 'proxy' },
  { title: i18n.global.t('paidSub.transportModes.outbound'), value: 'outbound' },
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
  { title: i18n.global.t('paidSub.cols.client'), key: 'name' },
  { title: i18n.global.t('paidSub.cols.clientId'), key: 'clientId' },
  { title: i18n.global.t('paidSub.cols.description'), key: 'desc' },
  { title: i18n.global.t('paidSub.cols.telegramId'), key: 'tgUserId' },
  { title: i18n.global.t('paidSub.cols.expiry'), key: 'expiry' },
  { title: i18n.global.t('paidSub.cols.status'), key: 'enable' },
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
  const rawTgUserId = String(bindingEdit.value.tgUserId ?? '').trim()
  const tgUserId = Number(rawTgUserId)
  if (rawTgUserId === '' || !Number.isSafeInteger(tgUserId) || tgUserId <= 0) {
    push.error({ title: i18n.global.t('failed'), message: i18n.global.t('paidSub.bindingDialog.invalidTgId') })
    return
  }
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
const isActiveBroadcastRecipient = (binding: any, now = Date.now() / 1000) => {
  const tgUserId = Number(binding?.tgUserId)
  const expiry = Number(binding?.expiry || 0)
  return binding?.enable === true && Number.isFinite(tgUserId) && tgUserId > 0 && (expiry === 0 || expiry > now)
}
const recipientCount = computed(() => {
  const now = Date.now() / 1000
  return bindings.value.filter((b: any) => isActiveBroadcastRecipient(b, now)).length
})
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
  { title: i18n.global.t('paidSub.cols.name'), key: 'name' },
  { title: i18n.global.t('paidSub.cols.price'), key: 'price' },
  { title: i18n.global.t('paidSub.cols.stars'), key: 'starsAmount' },
  { title: i18n.global.t('paidSub.cols.addDays'), key: 'addDays' },
  { title: i18n.global.t('paidSub.cols.addTraffic'), key: 'addTrafficBytes' },
  { title: i18n.global.t('paidSub.cols.enabled'), key: 'enabled' },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
]
const tariffDialog = ref(false)
const blankTariff = () => ({ id: 0, name: '', description: '', priceMajor: 0, currency: normalizeCurrency(settings.value.paidSubCurrency), starsAmount: 0, addDays: 30, addTrafficGB: 0, maxAwgDevices: 0, sort: 0, enabled: true })
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
      priceMajor: (item.price || 0) / 100, currency: normalizeCurrency(item.currency),
      starsAmount: item.starsAmount || 0, addDays: item.addDays || 0,
      maxAwgDevices: item.maxAwgDevices || 0,
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
  const currency = normalizeCurrency(e.currency)
  tariffEdit.value.currency = currency
  // Clamp every numeric to >= 0 so a typo / negative spinner value never reaches
  // the backend (which now also rejects negatives — defense in depth).
  const data: any = {
    name: e.name, description: e.description,
    price: Math.max(0, Math.round(Number(e.priceMajor) * 100) || 0),
    currency,
    starsAmount: Math.max(0, Math.round(Number(e.starsAmount) || 0)),
    addDays: Math.max(0, Math.round(Number(e.addDays) || 0)),
    addTrafficBytes: Math.max(0, Math.round((Number(e.addTrafficGB) || 0) * 1024 * 1024 * 1024)),
    maxAwgDevices: Math.min(100, Math.max(0, Math.round(Number(e.maxAwgDevices) || 0))),
    sort: Math.max(0, Math.round(Number(e.sort) || 0)),
    enabled: !!e.enabled,
  }
  const action = e.id ? 'edit' : 'new'
  if (e.id) data.id = e.id
  const msg = await HttpUtils.post('api/paidsub/tariffs', { action, data }, jsonPost)
  if (msg.success) { tariffDialog.value = false; await loadTariffs() }
}
const deleteTariffDialog = ref(false)
const deleteTariffBusy = ref(false)
const deleteTariffEdit = ref<{ id: number; name: string }>({ id: 0, name: '' })
const openDeleteTariffConfirm = (item: any) => {
  deleteTariffEdit.value = { id: Number(item.id), name: item.name || String(item.id) }
  deleteTariffDialog.value = true
}
const deleteTariff = async () => {
  if (!deleteTariffEdit.value.id) return
  deleteTariffBusy.value = true
  const msg = await HttpUtils.post('api/paidsub/tariffs', { action: 'del', data: deleteTariffEdit.value.id }, jsonPost)
  deleteTariffBusy.value = false
  if (msg.success) {
    deleteTariffDialog.value = false
    await loadTariffs()
  }
}

// ---- orders ----
const orders = ref<any[]>([])
const ordersLoading = ref(false)
const orderHeaders = [
  { title: i18n.global.t('paidSub.cols.id'), key: 'id' },
  { title: i18n.global.t('paidSub.cols.clientName'), key: 'clientName' },
  { title: i18n.global.t('paidSub.cols.telegramId'), key: 'telegramUserId' },
  { title: i18n.global.t('paidSub.cols.description'), key: 'clientDesc' },
  { title: i18n.global.t('paidSub.cols.provider'), key: 'provider' },
  { title: i18n.global.t('paidSub.cols.amount'), key: 'amount' },
  { title: i18n.global.t('paidSub.cols.status'), key: 'status' },
  { title: i18n.global.t('paidSub.cols.created'), key: 'createdAt' },
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
  await Promise.all([loadSettings(), loadStatus(), loadAWGStatus(), loadInbounds(), loadOutbounds(), loadBindings(), loadTariffs(), loadOrders()])
  loading.value = false
}

onMounted(reloadAll)
</script>

<style scoped>
.paidsub-nexus__name {
  color: var(--nexus-text-primary);
  font-weight: 600;
}
</style>
