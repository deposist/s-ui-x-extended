<template>
  <page-header v-if="nexus" :title="$t('donations.title')" />
  <v-card :flat="nexus">
    <template v-if="!nexus">
      <v-card-title>{{ $t('donations.title') }}</v-card-title>
      <v-divider></v-divider>
    </template>
    <v-card-text>
      <v-row justify="center" class="mb-2">
        <v-col cols="12" md="8" lg="6">
          <v-img
            src="@/assets/support-s-ui-x.png"
            :alt="$t('donations.imageAlt')"
            class="donations-hero mx-auto"
            max-width="520"
          />
        </v-col>
      </v-row>

      <p class="donations-intro text-medium-emphasis text-center mx-auto mb-6">
        {{ $t('donations.intro') }}
      </p>

      <v-row justify="center" class="mb-8">
        <v-col cols="12" sm="auto">
          <v-btn
            block
            color="primary"
            variant="flat"
            size="large"
            prepend-icon="lucide:external-link"
            href="https://web.tribute.tg/d/LRJ"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ $t('donations.web') }}
          </v-btn>
        </v-col>
        <v-col cols="12" sm="auto">
          <v-btn
            block
            color="primary"
            variant="tonal"
            size="large"
            prepend-icon="lucide:send"
            href="https://t.me/tribute/app?startapp=dLRJ"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ $t('donations.telegram') }}
          </v-btn>
        </v-col>
      </v-row>

      <div class="donations-crypto mx-auto">
        <div class="text-subtitle-1 font-weight-medium mb-2">{{ $t('donations.cryptoTitle') }}</div>
        <v-table density="comfortable" class="donations-crypto-table">
          <thead>
            <tr>
              <th>{{ $t('donations.network') }}</th>
              <th>{{ $t('donations.address') }}</th>
              <th class="text-end"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="wallet in wallets" :key="wallet.network">
              <td class="font-weight-medium">{{ wallet.network }}</td>
              <td class="donations-address" dir="ltr">{{ wallet.address }}</td>
              <td class="text-end">
                <v-btn
                  :aria-label="$t('donations.copy')"
                  icon="lucide:copy"
                  variant="text"
                  density="comfortable"
                  @click="copy(wallet.address)"
                >
                  <v-icon />
                  <v-tooltip activator="parent" location="top" :text="$t('donations.copy')" />
                </v-btn>
              </td>
            </tr>
          </tbody>
        </v-table>
      </div>
    </v-card-text>
  </v-card>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { push } from 'notivue'
import { i18n } from '@/locales'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import { useUiMode } from '@/uiMode/useUiMode'

const { mode } = useUiMode()
const nexus = computed(() => mode.value === 'nexus')

interface Wallet {
  network: string
  address: string
}

// Static project donation wallets — see donations release/announcement copy.
const wallets: Wallet[] = [
  { network: 'TON', address: 'UQB5-DZ3q5vjXGf3_tVUeOHPNuXMLh8lfY0MPW3uGzjdOzke' },
  { network: 'ETH', address: '0x0e67e1b4363a163c36943Ef4F9227c3126bB952B' },
  { network: 'SOL', address: 'BtFm5E1BrUjpoaDNwv3emc2qbyvqkn6ECnDzwhgRn7Df' },
  { network: 'TRX', address: 'TFqEbp1Z82ZQebzDdsW1MbytMvVsHJGpPd' },
  { network: 'BTC', address: 'bc1qn86mfmsnackfwvjd4czjaalv75sh830fws7xc9' },
]

const copy = async (text: string) => {
  if (await writeClipboard(text)) {
    push.success({
      message: i18n.global.t('success') + ': ' + i18n.global.t('copyToClipboard'),
      duration: 4000,
    })
  } else {
    push.error({
      message: i18n.global.t('failed') + ': ' + i18n.global.t('copyToClipboard'),
      duration: 4000,
    })
  }
}

// Prefer the async Clipboard API (available in the panel's HTTPS/localhost
// secure context); fall back to a hidden textarea for plain-HTTP LAN setups
// where navigator.clipboard is undefined.
const writeClipboard = async (text: string): Promise<boolean> => {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // fall through to the legacy path
  }
  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.focus()
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}
</script>

<style scoped>
.donations-hero {
  border-radius: 12px;
}

.donations-intro {
  max-width: 640px;
  line-height: 1.6;
}

.donations-crypto {
  max-width: 760px;
}

.donations-address {
  font-family: 'Roboto Mono', monospace;
  word-break: break-all;
}
</style>
