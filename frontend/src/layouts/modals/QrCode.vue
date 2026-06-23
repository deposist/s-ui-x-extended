<template>
  <v-dialog transition="dialog-bottom-transition" width="680">
    <v-card class="rounded-lg" id="qrcode-modal" :loading="loading">
      <v-card-title>
        <v-row>
          <v-col>{{ $t('delivery.title') }}</v-col>
          <v-spacer></v-spacer>
          <v-col cols="auto"><v-icon icon="mdi-close-box" @click="$emit('close')" /></v-col>
        </v-row>
      </v-card-title>
      <v-divider></v-divider>
      <v-skeleton-loader
          class="mx-auto border"
          width="80%"
          type="text, image, divider, text, image"
          v-if="loading"
        ></v-skeleton-loader>
      <v-card-text style="overflow-y: auto;" :hidden="loading">
        <v-tabs
          v-model="tab"
          density="compact"
          show-arrows
          align-tabs="center"
        >
          <v-tab value="singbox">Sing-box</v-tab>
          <v-tab value="clash">Clash/Mihomo</v-tab>
          <v-tab value="hiddify">Hiddify</v-tab>
          <v-tab value="raw">{{ $t('delivery.rawLinks') }}</v-tab>
        </v-tabs>
        <v-window v-model="tab" class="delivery-window">
          <v-window-item
            v-for="platform in deliveryPlatforms"
            :key="platform.value"
            :value="platform.value"
          >
            <div class="delivery-pane">
              <div class="delivery-qr">
                <v-chip>{{ platform.title }}</v-chip>
                <QrcodeVue
                  :value="platform.qr"
                  :size="size"
                  :margin="1"
                  class="delivery-qr__code"
                  @click="copyToClipboard(platform.copy)"
                />
              </div>
              <div class="delivery-details">
                <v-text-field
                  readonly
                  :label="$t('delivery.subscriptionUrl')"
                  :model-value="platform.copy"
                  variant="outlined"
                >
                  <template #append-inner>
                    <v-btn
                      density="comfortable"
                      icon="lucide:copy"
                      size="small"
                      variant="text"
                      @click="copyToClipboard(platform.copy)"
                    />
                  </template>
                </v-text-field>
                <v-text-field
                  v-if="platform.importUrl && platform.importUrl !== platform.copy"
                  readonly
                  :label="$t('delivery.importUrl')"
                  :model-value="platform.importUrl"
                  variant="outlined"
                >
                  <template #append-inner>
                    <v-btn
                      density="comfortable"
                      icon="lucide:copy"
                      size="small"
                      variant="text"
                      @click="copyToClipboard(platform.importUrl)"
                    />
                  </template>
                </v-text-field>
                <v-btn
                  color="primary"
                  prepend-icon="lucide:activity"
                  variant="tonal"
                  @click="testUrl(platform.copy)"
                >
                  {{ $t('delivery.testUrl') }}
                </v-btn>
              </div>
            </div>
          </v-window-item>
          <v-window-item value="raw">
            <div class="delivery-raw">
              <v-alert v-if="clientLinks.length === 0" density="compact" type="warning" variant="tonal">
                {{ $t('delivery.noRawLinks') }}
              </v-alert>
              <div v-for="l in clientLinks" :key="l.uri" class="delivery-raw__item">
                <div class="delivery-raw__qr">
                  <v-chip>{{ l.remark ?? $t('client.' + l.type) }}</v-chip>
                  <QrcodeVue
                    :value="l.uri"
                    :size="rawQrSize"
                    :margin="1"
                    class="delivery-qr__code"
                    @click="copyToClipboard(l.uri)"
                  />
                </div>
                <v-text-field readonly :model-value="l.uri" variant="outlined">
                  <template #append-inner>
                    <v-btn
                      density="comfortable"
                      icon="lucide:copy"
                      size="small"
                      variant="text"
                      @click="copyToClipboard(l.uri)"
                    />
                  </template>
                </v-text-field>
              </div>
            </div>
          </v-window-item>
        </v-window>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import QrcodeVue from 'qrcode.vue'
import Data from '@/store/modules/data'
import Clipboard from 'clipboard'
import { i18n } from '@/locales'
import { push } from 'notivue'

export default {
  props: ['id', 'visible'],
  data() {
    return {
      tab: "singbox",
      client: <any>{},
      loading: false,
    }
  },
  methods: {
    async load() {
      this.loading = true
      const newData = await Data().loadClients(this.$props.id)
      this.client = newData
      this.loading = false
    },
    async testUrl(url: string) {
      try {
        await fetch(url, { method: 'GET', mode: 'no-cors', credentials: 'omit', cache: 'no-store' })
        push.success({
          message: i18n.global.t('delivery.testOk'),
          duration: 5000,
        })
      } catch {
        push.error({
          message: i18n.global.t('delivery.testFailed'),
          duration: 5000,
        })
      }
    },
    copyToClipboard(txt:string) {
      const hiddenButton = document.createElement('button')
      hiddenButton.className = 'clipboard-btn'
      document.body.appendChild(hiddenButton)

      const clipboard = new Clipboard('.clipboard-btn', {
        text: () => txt,
        container: document.getElementById('qrcode-modal')?? undefined
      });

      clipboard.on('success', () => {
        clipboard.destroy()
        push.success({
          message: i18n.global.t('success') + ": " + i18n.global.t('copyToClipboard'),
          duration: 5000,
        })
      })

      clipboard.on('error', () => {
        clipboard.destroy()
        push.error({
          message: i18n.global.t('failed') + ": " + i18n.global.t('copyToClipboard'),
          duration: 5000,
        })
      })

      // Perform click on hidden button to trigger copy
      hiddenButton.click()
      document.body.removeChild(hiddenButton)
    },
    subscriptionUrl(base:string) {
      const trimmed = (base || "").trim()
      if (!trimmed) return this.subscriptionId
      return (trimmed.endsWith("/") ? trimmed : trimmed + "/") + this.subscriptionId
    }
  },
  computed: {
    subscriptionId() {
      return this.client.subSecret || this.client.name
    },
    clientSub() {
      return this.subscriptionUrl(Data().subURI)
    },
    clientJsonSub() {
      const data = Data()
      if (data.subJsonURI) return this.subscriptionUrl(data.subJsonURI)
      return this.clientSub + '?format=json'
    },
    clientClashSub() {
      const data = Data()
      if (data.subClashURI) return this.subscriptionUrl(data.subClashURI)
      return this.clientSub + '?format=clash'
    },
    singbox() {
      return "sing-box://import-remote-profile?url=" +  encodeURIComponent(this.clientJsonSub) + "#" + this.client.name
    },
    deliveryPlatforms() {
      return [
        {
          value: 'singbox',
          title: 'Sing-box',
          copy: this.clientJsonSub,
          importUrl: this.singbox,
          qr: this.singbox,
        },
        {
          value: 'clash',
          title: 'Clash/Mihomo',
          copy: this.clientClashSub,
          importUrl: this.clientClashSub,
          qr: this.clientClashSub,
        },
        {
          value: 'hiddify',
          title: 'Hiddify',
          copy: this.clientSub,
          importUrl: this.clientSub,
          qr: this.clientSub,
        },
      ]
    },
    clientLinks() {
      return this.client.links?? []
    },
    size() {
      if (window.innerWidth > 640) return 260
      if (window.innerWidth > 380) return 240
      return 210
    },
    rawQrSize() {
      if (window.innerWidth > 640) return 180
      return 150
    }
  },
  watch: {
    visible(v) {
      if (v) {
        this.tab = "singbox"
        this.load()
      }
    },
  },
  components: { QrcodeVue }
}
</script>

<style scoped>
.delivery-window {
  margin-top: 12px;
}

.delivery-pane {
  display: grid;
  gap: 18px;
  grid-template-columns: auto minmax(0, 1fr);
  min-width: 0;
}

.delivery-qr,
.delivery-raw__qr {
  align-items: center;
  display: grid;
  gap: 10px;
  justify-items: center;
}

.delivery-qr__code {
  border-radius: 0.75rem;
  cursor: copy;
}

.delivery-details {
  align-content: start;
  display: grid;
  gap: 10px;
  min-width: 0;
}

.delivery-raw {
  display: grid;
  gap: 14px;
}

.delivery-raw__item {
  display: grid;
  gap: 12px;
  grid-template-columns: auto minmax(0, 1fr);
  min-width: 0;
}

@media (max-width: 640px) {
  .delivery-pane,
  .delivery-raw__item {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
