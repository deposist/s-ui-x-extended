<template>
  <page-header v-if="nexus" :title="$t('pages.settings')" />
  <v-card :loading="loading" :flat="nexus" :class="{ 'settings-nexus-card': nexus }">
    <v-tabs
    v-model="tab"
    color="primary"
    align-tabs="center"
    show-arrows
  >
    <v-tab value="t1">{{ $t('setting.interface') }}</v-tab>
    <v-tab value="t2">{{ $t('setting.sub') }}</v-tab>
    <v-tab value="t3">{{ $t('setting.jsonSub') }}</v-tab>
    <v-tab value="t4">{{ $t('setting.clashSub') }}</v-tab>
    <v-tab value="t6">Basics (Singbox)</v-tab>
    <v-tab value="t5">{{ $t('setting.maintenance') }}</v-tab>
  </v-tabs>
  <v-card-text>
    <v-row
      v-if="tab !== 't5'"
      align="center"
      class="settings-actions"
      :class="{ 'settings-actions--nexus': nexus }"
      justify="center"
    >
      <v-col cols="auto" v-if="tab !== 't6'">
        <v-btn color="primary" @click="save" :loading="loading" :disabled="!stateChange">
          {{ $t('actions.save') }}
        </v-btn>
      </v-col>
      <v-col cols="auto" v-else>
        <v-btn color="primary" @click="saveBasicsConfig" :loading="loading" :disabled="!basicsStateChange">
          {{ $t('actions.save') }}
        </v-btn>
      </v-col>
      <v-col cols="auto">
        <v-btn variant="outlined" color="warning" @click="restartApp" :loading="loading" :disabled="tab !== 't6' ? stateChange : basicsStateChange">
          {{ $t('actions.restartApp') }}
        </v-btn>
      </v-col>
    </v-row>
    <v-window v-model="tab">
      <v-window-item value="t1">
        <v-row v-if="!nexus">
          <v-col cols="12" sm="6" md="4" v-if="showNexusControls">
            <ui-mode-control variant="select" />
          </v-col>
        </v-row>
        <v-row class="settings-grid" v-if="nexus">
          <v-col cols="12" md="6">
            <v-card variant="outlined" class="settings-section-card">
              <v-card-title class="settings-section-title">{{ $t('setting.interface') }}</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" v-if="showNexusControls">
                    <ui-mode-control variant="select" />
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.webListen" :label="$t('setting.addr')" placeholder="0.0.0.0" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webListen')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model.number="webPort" min="1" type="number" :label="$t('setting.port')" placeholder="2095" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webPort')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.webPath" :label="$t('setting.webPath')" placeholder="/app/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webPath')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.webDomain" :label="$t('setting.domain')" placeholder="example.com" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webDomain')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12">
                    <v-text-field v-model="settings.webURI" :label="$t('setting.webUri')" placeholder="https://panel.example.com/app/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webUri')" /></template>
                    </v-text-field>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>

          <v-col cols="12" md="6">
            <v-card variant="outlined" class="settings-section-card">
              <v-card-title class="settings-section-title">Security & Maintenance</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.webKeyFile" :label="$t('setting.sslKey')" placeholder="/etc/s-ui/panel.key" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sslKey')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.webCertFile" :label="$t('setting.sslCert')" placeholder="/etc/s-ui/panel.crt" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sslCert')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field
                      type="number"
                      v-model.number="sessionMaxAge"
                      min="0"
                      :label="$t('setting.sessionAge')"
                      :suffix="$t('date.m')"
                      placeholder="0"
                      persistent-placeholder
                      hide-details
                      >
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sessionAge')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field
                      type="number"
                      v-model.number="trafficAge"
                      min="0"
                      :label="$t('setting.trafficAge')"
                      :suffix="$t('date.d')"
                      placeholder="30"
                      persistent-placeholder
                      hide-details
                      >
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.trafficAge')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12">
                    <v-autocomplete
                      v-model="settings.timeLocation"
                      :items="timezones"
                      :label="$t('setting.timeLoc')"
                      placeholder="Europe/Moscow"
                      persistent-placeholder
                      auto-select-first
                      hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.timeLoc')" /></template>
                    </v-autocomplete>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>

        <!-- Classic Fallback layout: -->
        <v-row v-else>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webListen" :label="$t('setting.addr')" placeholder="0.0.0.0" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webListen')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model.number="webPort" min="1" type="number" :label="$t('setting.port')" placeholder="2095" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webPort')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webPath" :label="$t('setting.webPath')" placeholder="/app/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webPath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webDomain" :label="$t('setting.domain')" placeholder="example.com" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webDomain')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webKeyFile" :label="$t('setting.sslKey')" placeholder="/etc/s-ui/panel.key" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sslKey')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webCertFile" :label="$t('setting.sslCert')" placeholder="/etc/s-ui/panel.crt" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sslCert')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.webURI" :label="$t('setting.webUri')" placeholder="https://panel.example.com/app/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.webUri')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="sessionMaxAge"
              min="0"
              :label="$t('setting.sessionAge')"
              :suffix="$t('date.m')"
              placeholder="0"
              persistent-placeholder
              hide-details
              >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.sessionAge')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="trafficAge"
              min="0"
              :label="$t('setting.trafficAge')"
              :suffix="$t('date.d')"
              placeholder="30"
              persistent-placeholder
              hide-details
              >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.trafficAge')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-autocomplete
              v-model="settings.timeLocation"
              :items="timezones"
              :label="$t('setting.timeLoc')"
              placeholder="Europe/Moscow"
              persistent-placeholder
              auto-select-first
              hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.timeLoc')" /></template>
            </v-autocomplete>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t2">
        <v-row v-if="!nexus">
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subEncode" :label="$t('setting.subEncode')" hide-details />
              <SettingInfo :text="$t('setting.hint.subEncode')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subShowInfo" :label="$t('setting.subInfo')" hide-details />
              <SettingInfo :text="$t('setting.hint.subInfo')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subSecretRequired" :label="$t('setting.subSecretRequired')" hide-details />
              <SettingInfo :text="$t('setting.hint.subSecretRequired')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subLinkEnable" :label="$t('setting.subLinkEnable')" hide-details />
              <SettingInfo :text="$t('setting.hint.subLinkEnable')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subJsonEnable" :label="$t('setting.subJsonEnable')" hide-details />
              <SettingInfo :text="$t('setting.hint.subJsonEnable')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subClashEnable" :label="$t('setting.subClashEnable')" hide-details />
              <SettingInfo :text="$t('setting.hint.subClashEnable')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subListen" :label="$t('setting.addr')" placeholder="0.0.0.0" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subListen')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="subPort"
              min="1"
              :label="$t('setting.port')"
              placeholder="2096"
              persistent-placeholder
              hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subPort')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subKeyFile" :label="$t('setting.sslKey')" placeholder="/etc/s-ui/sub.key" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subKeyFile')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subCertFile" :label="$t('setting.sslCert')" placeholder="/etc/s-ui/sub.crt" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subCertFile')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subDomain" :label="$t('setting.domain')" placeholder="example.com" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subDomain')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subPath" :label="$t('setting.path')" placeholder="/sub/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subPath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              type="number"
              v-model.number="subUpdates"
              min="0"
              :label="$t('setting.update')"
              :suffix="$t('date.h')"
              placeholder="12"
              persistent-placeholder
              hide-details
              >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.update')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subURI" :label="$t('setting.subUri')" placeholder="https://sub.example.com/sub/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subUri')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" class="v-card-subtitle">{{ $t('setting.subAdvanced') }}</v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subTitle" :label="$t('setting.subTitle')" placeholder="My VPN" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subTitle')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subSupportUrl" :label="$t('setting.subSupportUrl')" placeholder="https://t.me/yoursupport" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subSupportUrl')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subProfileUrl" :label="$t('setting.subProfileUrl')" placeholder="https://example.com" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subProfileUrl')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field
              v-model.number="subRateLimitPerIP"
              min="0"
              type="number"
              :label="$t('setting.subRateLimitPerIP')"
              placeholder="60"
              persistent-placeholder
              hide-details
            >
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subRateLimitPerIP')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <div class="d-flex align-center">
              <v-switch color="primary" v-model="subNameInRemark" :label="$t('setting.subNameInRemark')" hide-details />
              <SettingInfo :text="$t('setting.hint.subNameInRemark')" class="ms-1" />
            </div>
          </v-col>
          <v-col cols="12">
            <v-textarea v-model="settings.subAnnounce" :label="$t('setting.subAnnounce')" rows="2" placeholder="Welcome!" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subAnnounce')" /></template>
            </v-textarea>
          </v-col>
        </v-row>

        <v-row class="settings-grid" v-else>
          <v-col cols="12" md="6">
            <v-card variant="outlined" class="settings-section-card">
              <v-card-title class="settings-section-title">Subscription Toggles</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <div class="d-flex align-center">
                      <v-switch color="primary" v-model="subEncode" :label="$t('setting.subEncode')" hide-details />
                      <SettingInfo :text="$t('setting.hint.subEncode')" class="ms-1" />
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="d-flex align-center">
                      <v-switch color="primary" v-model="subShowInfo" :label="$t('setting.subInfo')" hide-details />
                      <SettingInfo :text="$t('setting.hint.subInfo')" class="ms-1" />
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="d-flex align-center">
                      <v-switch color="primary" v-model="subSecretRequired" :label="$t('setting.subSecretRequired')" hide-details />
                      <SettingInfo :text="$t('setting.hint.subSecretRequired')" class="ms-1" />
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="d-flex align-center">
                      <v-switch color="primary" v-model="subLinkEnable" :label="$t('setting.subLinkEnable')" hide-details />
                      <SettingInfo :text="$t('setting.hint.subLinkEnable')" class="ms-1" />
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="d-flex align-center">
                      <v-switch color="primary" v-model="subJsonEnable" :label="$t('setting.subJsonEnable')" hide-details />
                      <SettingInfo :text="$t('setting.hint.subJsonEnable')" class="ms-1" />
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <div class="d-flex align-center">
                      <v-switch color="primary" v-model="subClashEnable" :label="$t('setting.subClashEnable')" hide-details />
                      <SettingInfo :text="$t('setting.hint.subClashEnable')" class="ms-1" />
                    </div>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>

          <v-col cols="12" md="6">
            <v-card variant="outlined" class="settings-section-card">
              <v-card-title class="settings-section-title">Subscription Connection</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subListen" :label="$t('setting.addr')" placeholder="0.0.0.0" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subListen')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field
                      type="number"
                      v-model.number="subPort"
                      min="1"
                      :label="$t('setting.port')"
                      placeholder="2096"
                      persistent-placeholder
                      hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subPort')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subKeyFile" :label="$t('setting.sslKey')" placeholder="/etc/s-ui/sub.key" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subKeyFile')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subCertFile" :label="$t('setting.sslCert')" placeholder="/etc/s-ui/sub.crt" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subCertFile')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subDomain" :label="$t('setting.domain')" placeholder="example.com" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subDomain')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subPath" :label="$t('setting.path')" placeholder="/sub/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subPath')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field
                      type="number"
                      v-model.number="subUpdates"
                      min="0"
                      :label="$t('setting.update')"
                      :suffix="$t('date.h')"
                      placeholder="12"
                      persistent-placeholder
                      hide-details
                      >
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.update')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subURI" :label="$t('setting.subUri')" placeholder="https://sub.example.com/sub/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subUri')" /></template>
                    </v-text-field>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>

          <v-col cols="12">
            <v-card variant="outlined" class="settings-section-card">
              <v-card-title class="settings-section-title">{{ $t('setting.subAdvanced') }}</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6" md="3">
                    <v-text-field v-model="settings.subTitle" :label="$t('setting.subTitle')" placeholder="My VPN" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subTitle')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" md="3">
                    <v-text-field v-model="settings.subSupportUrl" :label="$t('setting.subSupportUrl')" placeholder="https://t.me/yoursupport" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subSupportUrl')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" md="3">
                    <v-text-field v-model="settings.subProfileUrl" :label="$t('setting.subProfileUrl')" placeholder="https://example.com" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subProfileUrl')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" md="3">
                    <v-text-field
                      v-model.number="subRateLimitPerIP"
                      min="0"
                      type="number"
                      :label="$t('setting.subRateLimitPerIP')"
                      placeholder="60"
                      persistent-placeholder
                      hide-details
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subRateLimitPerIP')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12">
                    <div class="d-flex align-center">
                      <v-switch color="primary" v-model="subNameInRemark" :label="$t('setting.subNameInRemark')" hide-details />
                      <SettingInfo :text="$t('setting.hint.subNameInRemark')" class="ms-2" />
                    </div>
                  </v-col>
                  <v-col cols="12">
                    <v-textarea v-model="settings.subAnnounce" :label="$t('setting.subAnnounce')" rows="2" placeholder="Welcome!" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subAnnounce')" /></template>
                    </v-textarea>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t3">
        <v-row v-if="!nexus">
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subJsonPath" :label="$t('setting.jsonPath')" placeholder="/json/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.jsonPath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subJsonURI" :label="$t('setting.jsonSub') + ' ' + $t('setting.subUri')" placeholder="https://sub.example.com/json/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subJsonURI')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <v-row class="settings-grid" v-else>
          <v-col cols="12">
            <v-card variant="outlined" class="settings-section-card">
              <v-card-title class="settings-section-title">JSON Configuration</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subJsonPath" :label="$t('setting.jsonPath')" placeholder="/json/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.jsonPath')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subJsonURI" :label="$t('setting.jsonSub') + ' ' + $t('setting.subUri')" placeholder="https://sub.example.com/json/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subJsonURI')" /></template>
                    </v-text-field>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
        <SubJsonExtVue :settings="settings" />
      </v-window-item>

      <v-window-item value="t4">
        <v-row v-if="!nexus">
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subClashPath" :label="$t('setting.clashPath')" placeholder="/clash/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.clashPath')" /></template>
            </v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="4">
            <v-text-field v-model="settings.subClashURI" :label="$t('setting.clashSub') + ' ' + $t('setting.subUri')" placeholder="https://sub.example.com/clash/" persistent-placeholder hide-details>
              <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subClashURI')" /></template>
            </v-text-field>
          </v-col>
        </v-row>
        <v-row class="settings-grid" v-else>
          <v-col cols="12">
            <v-card variant="outlined" class="settings-section-card">
              <v-card-title class="settings-section-title">Clash Configuration</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subClashPath" :label="$t('setting.clashPath')" placeholder="/clash/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.clashPath')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field v-model="settings.subClashURI" :label="$t('setting.clashSub') + ' ' + $t('setting.subUri')" placeholder="https://sub.example.com/clash/" persistent-placeholder hide-details>
                      <template v-slot:append-inner><SettingInfo :text="$t('setting.hint.subClashURI')" /></template>
                    </v-text-field>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
        <SubClashExtVue :settings="settings" />
      </v-window-item>

      <v-window-item value="t6">
        <!-- Basics layout -->
        <v-row class="settings-grid align-stretch" v-if="nexus">
          <!-- Log & NTP -->
          <v-col cols="12" md="6" class="d-flex flex-column">
            <v-card variant="outlined" class="settings-section-card d-flex flex-column flex-grow-1 mb-4">
              <v-card-title class="settings-section-title">{{ $t('basic.log.title') }}</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <div class="d-flex align-center">
                      <v-switch v-model="appConfig.log.disabled" color="primary" :label="$t('disable')" hide-details></v-switch>
                      <SettingInfo :text="$t('basic.hint.logOutput')" class="ms-1" />
                    </div>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-select
                      hide-details
                      :label="$t('basic.log.level')"
                      :items="levels"
                      clearable
                      @click:clear="delete appConfig.log.level"
                      v-model="appConfig.log.level">
                    </v-select>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-text-field
                      v-model="appConfig.log.output"
                      hide-details
                      placeholder="box.log"
                      persistent-placeholder
                      :label="$t('basic.log.output')"
                    ></v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-switch v-model="appConfig.log.timestamp" color="primary" :label="$t('basic.log.timestamp')" hide-details></v-switch>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>

            <v-card variant="outlined" class="settings-section-card d-flex flex-column flex-grow-1">
              <v-card-title class="settings-section-title">NTP settings</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-switch v-model="enableNtp" color="primary" :label="$t('enable')" hide-details></v-switch>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.ntp?.enabled">
                    <v-text-field
                      v-model="appConfig.ntp.server"
                      hide-details
                      placeholder="time.apple.com"
                      persistent-placeholder
                      :label="$t('out.addr')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.ntpServer')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.ntp?.enabled">
                    <v-text-field
                      v-model="appConfig.ntp.server_port"
                      hide-details
                      type="number"
                      clearable
                      placeholder="123"
                      persistent-placeholder
                      @click:clear="delete appConfig.ntp?.server_port"
                      :label="$t('out.port')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.ntpPort')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.ntp?.enabled">
                    <v-text-field
                      v-model="ntpInterval"
                      hide-details
                      :suffix="$t('date.m')"
                      min="0"
                      type="number"
                      placeholder="30"
                      persistent-placeholder
                      :label="$t('ruleset.interval')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.ntpInterval')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.ntp?.enabled">
                    <v-switch v-model="appConfig.ntp.write_to_system" color="primary" :label="$t('singbox.writeSystemClock')" hide-details></v-switch>
                  </v-col>
                  <v-col cols="12" v-if="appConfig.ntp?.write_to_system">
                    <v-alert density="compact" type="warning" variant="tonal" class="text-caption">
                      {{ $t('singbox.writeSystemClockWarning') }}
                    </v-alert>
                  </v-col>
                </v-row>
                <Dial :dial="appConfig.ntp" v-if="appConfig.ntp?.enabled" />
              </v-card-text>
            </v-card>
          </v-col>

          <!-- Trust -->
          <v-col cols="12" md="6" class="d-flex flex-column">
            <v-card variant="outlined" class="settings-section-card d-flex flex-column flex-grow-1">
              <v-card-title class="settings-section-title">{{ $t('singbox.certificateTrust') }}</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-select v-model="certificateMode" hide-details :label="$t('singbox.preset')" :items="certificateModes"></v-select>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.certificate">
                    <v-select
                      v-model="appConfig.certificate.store"
                      hide-details
                      clearable
                      @click:clear="delete appConfig.certificate?.store"
                      :label="$t('tls.store')"
                      :items="certificateStores">
                    </v-select>
                  </v-col>
                  <v-col cols="12" v-if="appConfig.certificate && (certificateMode == 'file' || certificateMode == 'custom')">
                    <v-textarea v-model="certificatePathText" rows="2" auto-grow hide-details :label="$t('singbox.caFilePaths')"></v-textarea>
                  </v-col>
                  <v-col cols="12" v-if="appConfig.certificate && (certificateMode == 'directory' || certificateMode == 'custom')">
                    <v-textarea v-model="certificateDirectoryText" rows="2" auto-grow hide-details :label="$t('singbox.caDirectoryPaths')"></v-textarea>
                  </v-col>
                  <v-col cols="12" v-if="appConfig.certificate && (certificateMode == 'pem' || certificateMode == 'custom')">
                    <v-textarea v-model="certificateText" rows="4" auto-grow hide-details :label="$t('singbox.pemCertificates')"></v-textarea>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>

          <!-- Clash API -->
          <v-col cols="12" md="6" class="d-flex flex-column">
            <v-card variant="outlined" class="settings-section-card d-flex flex-column flex-grow-1">
              <v-card-title class="settings-section-title">Clash API</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-switch v-model="enableClashApi" color="primary" :label="$t('enable') + ' Clash API'" hide-details></v-switch>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-text-field
                      v-model="appConfig.experimental.clash_api.external_controller"
                      hide-details
                      placeholder="127.0.0.1:9090"
                      persistent-placeholder
                      :label="$t('basic.exp.extController')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashController')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-text-field
                      v-model="appConfig.experimental.clash_api.secret"
                      hide-details
                      :label="$t('basic.exp.secret')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashSecret')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-text-field
                      v-model="appConfig.experimental.clash_api.external_ui"
                      hide-details
                      :label="$t('basic.exp.extUi')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashExtUi')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-text-field
                      v-model="appConfig.experimental.clash_api.external_ui_download_url"
                      hide-details
                      :label="$t('basic.exp.extUiDownloadUrl')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashExtUiUrl')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-select
                      v-model="appConfig.experimental.clash_api.external_ui_download_detour"
                      hide-details
                      :items="outboundTags"
                      clearable
                      @click:clear="delete appConfig.experimental.clash_api.external_ui_download_detour"
                      :label="$t('basic.exp.extUiDownloadDetour')"
                    ></v-select>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-select
                      v-model="appConfig.experimental.clash_api.default_mode"
                      hide-details
                      clearable
                      :items="['rule','global','direct']"
                      placeholder="rule"
                      persistent-placeholder
                      :label="$t('basic.exp.defaultMode')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashDefaultMode')" /></template>
                    </v-select>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-text-field
                      v-model="origin"
                      hide-details
                      :label="$t('basic.exp.allowOrigin')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashAllowOrigin')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.clash_api">
                    <v-switch v-model="appConfig.experimental.clash_api.access_control_allow_private_network" color="primary" :label="$t('basic.exp.allowPrivate')" hide-details></v-switch>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>

          <!-- V2Ray API -->
          <v-col cols="12" md="6" class="d-flex flex-column">
            <v-card variant="outlined" class="settings-section-card d-flex flex-column flex-grow-1">
              <v-card-title class="settings-section-title">V2Ray API</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" sm="6">
                    <v-switch v-model="enableV2rayApi" color="primary" :label="$t('enable') + ' V2Ray API'" hide-details></v-switch>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.v2ray_api">
                    <v-text-field
                      v-model="appConfig.experimental.v2ray_api.listen"
                      hide-details
                      placeholder="127.0.0.1:8080"
                      persistent-placeholder
                      :label="$t('objects.listen')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.v2rayListen')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" v-if="appConfig.experimental.v2ray_api">
                    <v-switch v-model="appConfig.experimental.v2ray_api.stats.enabled"
                      color="primary"
                      :label="$t('stats.enable')"
                      hide-details></v-switch>
                  </v-col>
                </v-row>
                <v-row v-if="appConfig.experimental.v2ray_api?.stats?.enabled">
                  <v-col cols="12" sm="6">
                    <v-select
                      hide-details
                      :label="$t('pages.inbounds')"
                      multiple chips closable-chips
                      :items="inboundTags"
                      v-model="appConfig.experimental.v2ray_api.stats.inbounds">
                    </v-select>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-select
                      hide-details
                      :label="$t('pages.outbounds')"
                      multiple chips closable-chips
                      :items="outboundTags"
                      v-model="appConfig.experimental.v2ray_api.stats.outbounds">
                    </v-select>
                  </v-col>
                  <v-col cols="12" sm="6">
                    <v-select
                      hide-details
                      :label="$t('pages.clients')"
                      multiple chips closable-chips
                      :items="clientNames"
                      v-model="appConfig.experimental.v2ray_api.stats.users">
                    </v-select>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>

          <!-- Experimental settings (full width) -->
          <v-col cols="12" class="d-flex flex-column">
            <v-card variant="outlined" class="settings-section-card d-flex flex-column">
              <v-card-title class="settings-section-title">Experimental settings</v-card-title>
              <v-card-text class="pa-4 pt-2">
                <v-row>
                  <v-col cols="12" class="v-card-subtitle px-0 pb-1">{{ $t('singbox.cacheFile') }}</v-col>
                  <v-col cols="12" sm="6" md="3">
                    <v-switch v-model="enableCacheFile" color="primary" :label="$t('enable')" hide-details></v-switch>
                  </v-col>
                  <v-col cols="12" sm="6" md="3" v-if="appConfig.experimental.cache_file">
                    <v-text-field
                      v-model="appConfig.experimental.cache_file.path"
                      hide-details
                      placeholder="cache.db"
                      persistent-placeholder
                      :label="$t('transport.path')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.cachePath')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" md="3" v-if="appConfig.experimental.cache_file">
                    <v-text-field
                      v-model="appConfig.experimental.cache_file.cache_id"
                      hide-details
                      :label="$t('singbox.cacheId')"
                    >
                      <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.cacheId')" /></template>
                    </v-text-field>
                  </v-col>
                  <v-col cols="12" sm="6" md="3" v-if="appConfig.experimental.cache_file">
                    <v-switch v-model="appConfig.experimental.cache_file.store_fakeip"
                      color="primary"
                      :label="$t('basic.exp.storeFakeIp')"
                      hide-details></v-switch>
                  </v-col>
                  <v-col cols="12" sm="6" md="3" v-if="appConfig.experimental.cache_file">
                    <v-switch v-model="appConfig.experimental.cache_file.store_rdrc"
                      color="primary"
                      :label="$t('singbox.storeRdrc')"
                      hide-details></v-switch>
                  </v-col>
                  <v-col cols="12" sm="6" md="3" v-if="appConfig.experimental.cache_file?.store_rdrc">
                    <v-text-field
                      v-model="appConfig.experimental.cache_file.rdrc_timeout"
                      hide-details
                      placeholder="7d"
                      :label="$t('singbox.rdrcTimeout')">
                    </v-text-field>
                  </v-col>
                </v-row>

                <v-row class="mt-2">
                  <v-col cols="12" class="v-card-subtitle px-0 pb-1">{{ $t('singbox.debug') }}</v-col>
                  <v-col cols="12" sm="6" md="3">
                    <v-switch v-model="enableDebug" color="primary" :label="$t('enable')" hide-details></v-switch>
                  </v-col>
                  <template v-if="appConfig.experimental.debug">
                    <v-col cols="12" sm="6" md="3">
                      <v-text-field v-model="appConfig.experimental.debug.listen" hide-details placeholder="127.0.0.1:8080" persistent-placeholder :label="$t('objects.listen')">
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.debugListen')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3">
                      <v-text-field v-model.number="appConfig.experimental.debug.gc_percent" type="number" hide-details :label="$t('singbox.gcPercent')">
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.gcPercent')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3">
                      <v-text-field v-model="appConfig.experimental.debug.memory_limit" hide-details placeholder="256MiB" persistent-placeholder :label="$t('singbox.memoryLimit')">
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.memoryLimit')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3">
                      <v-text-field v-model.number="appConfig.experimental.debug.max_stack" type="number" hide-details :label="$t('singbox.maxStack')">
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.maxStack')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3">
                      <v-text-field v-model.number="appConfig.experimental.debug.max_threads" type="number" hide-details :label="$t('singbox.maxThreads')">
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.maxThreads')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3">
                      <v-switch v-model="appConfig.experimental.debug.panic_on_fault" color="primary" :label="$t('singbox.panicOnFault')" hide-details></v-switch>
                    </v-col>
                    <v-col cols="12" sm="6" md="3">
                      <v-select
                        v-model="appConfig.experimental.debug.trace_back"
                        hide-details clearable
                        @click:clear="delete appConfig.experimental.debug?.trace_back"
                        :label="$t('singbox.traceback')"
                        :items="['none','single','all','system','crash']">
                      </v-select>
                    </v-col>
                  </template>
                </v-row>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>

        <!-- Classic Fallback layout: -->
        <v-row v-else>
          <v-col cols="12">
            <v-expansion-panels>
              <v-expansion-panel :title="$t('basic.log.title')">
                <v-expansion-panel-text>
                  <v-row>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="appConfig.log.disabled" color="primary" :label="$t('disable')" hide-details></v-switch>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-select
                        hide-details
                        :label="$t('basic.log.level')"
                        :items="levels"
                        clearable
                        @click:clear="delete appConfig.log.level"
                        v-model="appConfig.log.level">
                      </v-select>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-text-field
                        v-model="appConfig.log.output"
                        hide-details
                        placeholder="box.log"
                        persistent-placeholder
                        :label="$t('basic.log.output')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.logOutput')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="appConfig.log.timestamp" color="primary" :label="$t('basic.log.timestamp')" hide-details></v-switch>
                    </v-col>
                  </v-row>
                </v-expansion-panel-text>
              </v-expansion-panel>
              <v-expansion-panel title="NTP">
                <v-expansion-panel-text>
                  <v-row>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="enableNtp" color="primary" :label="$t('enable')" hide-details></v-switch>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
                      <v-text-field
                        v-model="appConfig.ntp.server"
                        hide-details
                        placeholder="time.apple.com"
                        persistent-placeholder
                        :label="$t('out.addr')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.ntpServer')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
                      <v-text-field
                        v-model="appConfig.ntp.server_port"
                        hide-details
                        type="number"
                        clearable
                        placeholder="123"
                        persistent-placeholder
                        @click:clear="delete appConfig.ntp?.server_port"
                        :label="$t('out.port')"
                      >
                        <template v-slot:append><SettingInfo :text="$t('basic.hint.ntpPort')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
                      <v-text-field
                        v-model="ntpInterval"
                        hide-details
                        :suffix="$t('date.m')"
                        min="0"
                        type="number"
                        placeholder="30"
                        persistent-placeholder
                        :label="$t('ruleset.interval')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.ntpInterval')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.ntp?.enabled">
                      <v-switch v-model="appConfig.ntp.write_to_system" color="primary" :label="$t('singbox.writeSystemClock')" hide-details></v-switch>
                    </v-col>
                    <v-col cols="12" v-if="appConfig.ntp?.write_to_system">
                      <v-alert density="compact" type="warning" variant="tonal">
                        {{ $t('singbox.writeSystemClockWarning') }}
                      </v-alert>
                    </v-col>
                  </v-row>
                  <Dial :dial="appConfig.ntp" v-if="appConfig.ntp?.enabled" />
                </v-expansion-panel-text>
              </v-expansion-panel>
              <v-expansion-panel :title="$t('singbox.certificateTrust')">
                <v-expansion-panel-text>
                  <v-row>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-select v-model="certificateMode" hide-details :label="$t('singbox.preset')" :items="certificateModes"></v-select>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.certificate">
                      <v-select
                        v-model="appConfig.certificate.store"
                        hide-details
                        clearable
                        @click:clear="delete appConfig.certificate?.store"
                        :label="$t('tls.store')"
                        :items="certificateStores">
                      </v-select>
                    </v-col>
                  </v-row>
                  <v-row v-if="appConfig.certificate && (certificateMode == 'file' || certificateMode == 'custom')">
                    <v-col cols="12" sm="8">
                      <v-textarea v-model="certificatePathText" rows="2" auto-grow hide-details :label="$t('singbox.caFilePaths')"></v-textarea>
                    </v-col>
                  </v-row>
                  <v-row v-if="appConfig.certificate && (certificateMode == 'directory' || certificateMode == 'custom')">
                    <v-col cols="12" sm="8">
                      <v-textarea v-model="certificateDirectoryText" rows="2" auto-grow hide-details :label="$t('singbox.caDirectoryPaths')"></v-textarea>
                    </v-col>
                  </v-row>
                  <v-row v-if="appConfig.certificate && (certificateMode == 'pem' || certificateMode == 'custom')">
                    <v-col cols="12">
                      <v-textarea v-model="certificateText" rows="5" auto-grow hide-details :label="$t('singbox.pemCertificates')"></v-textarea>
                    </v-col>
                  </v-row>
                </v-expansion-panel-text>
              </v-expansion-panel>
              <v-expansion-panel title="Experimental">
                <v-expansion-panel-text>
                  <v-row>
                    <v-col class="v-card-subtitle">{{ $t('singbox.cacheFile') }}</v-col>
                  </v-row>
                  <v-row>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="enableCacheFile" color="primary" :label="$t('enable')" hide-details></v-switch>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
                      <v-text-field
                        v-model="appConfig.experimental.cache_file.path"
                        hide-details
                        placeholder="cache.db"
                        persistent-placeholder
                        :label="$t('transport.path')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.cachePath')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
                      <v-text-field
                        v-model="appConfig.experimental.cache_file.cache_id"
                        hide-details
                        :label="$t('singbox.cacheId')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.cacheId')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
                      <v-switch v-model="appConfig.experimental.cache_file.store_fakeip"
                        color="primary"
                        :label="$t('basic.exp.storeFakeIp')"
                        hide-details></v-switch>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file">
                      <v-switch v-model="appConfig.experimental.cache_file.store_rdrc"
                        color="primary"
                        :label="$t('singbox.storeRdrc')"
                        hide-details></v-switch>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2" v-if="appConfig.experimental.cache_file?.store_rdrc">
                      <v-text-field
                        v-model="appConfig.experimental.cache_file.rdrc_timeout"
                        hide-details
                        placeholder="7d"
                        :label="$t('singbox.rdrcTimeout')">
                      </v-text-field>
                    </v-col>
                  </v-row>
                  <v-row>
                    <v-col class="v-card-subtitle">{{ $t('singbox.debug') }}</v-col>
                  </v-row>
                  <v-row>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="enableDebug" color="primary" :label="$t('enable')" hide-details></v-switch>
                    </v-col>
                    <template v-if="appConfig.experimental.debug">
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field v-model="appConfig.experimental.debug.listen" hide-details placeholder="127.0.0.1:8080" persistent-placeholder :label="$t('objects.listen')">
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.debugListen')" /></template>
                        </v-text-field>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field v-model.number="appConfig.experimental.debug.gc_percent" type="number" hide-details :label="$t('singbox.gcPercent')">
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.gcPercent')" /></template>
                        </v-text-field>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field v-model="appConfig.experimental.debug.memory_limit" hide-details placeholder="256MiB" persistent-placeholder :label="$t('singbox.memoryLimit')">
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.memoryLimit')" /></template>
                        </v-text-field>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field v-model.number="appConfig.experimental.debug.max_stack" type="number" hide-details :label="$t('singbox.maxStack')">
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.maxStack')" /></template>
                        </v-text-field>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field v-model.number="appConfig.experimental.debug.max_threads" type="number" hide-details :label="$t('singbox.maxThreads')">
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.maxThreads')" /></template>
                        </v-text-field>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-switch v-model="appConfig.experimental.debug.panic_on_fault" color="primary" :label="$t('singbox.panicOnFault')" hide-details></v-switch>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-select
                          v-model="appConfig.experimental.debug.trace_back"
                          hide-details clearable
                          @click:clear="delete appConfig.experimental.debug?.trace_back"
                          :label="$t('singbox.traceback')"
                          :items="['none','single','all','system','crash']">
                        </v-select>
                      </v-col>
                    </template>
                  </v-row>
                  <v-row>
                    <v-col class="v-card-subtitle">Clash API</v-col>
                  </v-row>
                  <v-row>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="enableClashApi" color="primary" :label="$t('enable')" hide-details></v-switch>
                    </v-col>
                    <template v-if="appConfig.experimental.clash_api">
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field
                          v-model="appConfig.experimental.clash_api.external_controller"
                          hide-details
                          placeholder="127.0.0.1:9090"
                          persistent-placeholder
                          :label="$t('basic.exp.extController')"
                        >
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashController')" /></template>
                        </v-text-field>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field
                          v-model="appConfig.experimental.clash_api.secret"
                          hide-details
                          :label="$t('basic.exp.secret')"
                        >
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashSecret')" /></template>
                        </v-text-field>
                      </v-col>
                    </template>
                  </v-row>
                  <v-row v-if="appConfig.experimental.clash_api">
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-text-field
                        v-model="appConfig.experimental.clash_api.external_ui"
                        hide-details
                        :label="$t('basic.exp.extUi')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashExtUi')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="8" md="4">
                      <v-text-field
                        v-model="appConfig.experimental.clash_api.external_ui_download_url"
                        hide-details
                        :label="$t('basic.exp.extUiDownloadUrl')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashExtUiUrl')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-select
                        v-model="appConfig.experimental.clash_api.external_ui_download_detour"
                        hide-details
                        :items="outboundTags"
                        clearable
                        @click:clear="delete appConfig.experimental.clash_api.external_ui_download_detour"
                        :label="$t('basic.exp.extUiDownloadDetour')"
                      ></v-select>
                    </v-col>
                  </v-row>
                  <v-row v-if="appConfig.experimental.clash_api">
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-select
                        v-model="appConfig.experimental.clash_api.default_mode"
                        hide-details
                        clearable
                        :items="['rule','global','direct']"
                        placeholder="rule"
                        persistent-placeholder
                        :label="$t('basic.exp.defaultMode')"
                      >
                        <template v-slot:append><SettingInfo :text="$t('basic.hint.clashDefaultMode')" /></template>
                      </v-select>
                    </v-col>
                    <v-col cols="12" sm="8" md="4">
                      <v-text-field
                        v-model="origin"
                        hide-details
                        :label="$t('basic.exp.allowOrigin') + ' ' + $t('commaSeparated')"
                      >
                        <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.clashAllowOrigin')" /></template>
                      </v-text-field>
                    </v-col>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="appConfig.experimental.clash_api.access_control_allow_private_network" color="primary" :label="$t('basic.exp.allowPrivate')" hide-details></v-switch>
                    </v-col>
                  </v-row>
                  <v-row>
                    <v-col class="v-card-subtitle">V2Ray API</v-col>
                  </v-row>
                  <v-row>
                    <v-col cols="12" sm="6" md="3" lg="2">
                      <v-switch v-model="enableV2rayApi" color="primary" :label="$t('enable')" hide-details></v-switch>
                    </v-col>
                    <template v-if="appConfig.experimental.v2ray_api">
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-text-field
                          v-model="appConfig.experimental.v2ray_api.listen"
                          hide-details
                          placeholder="127.0.0.1:8080"
                          persistent-placeholder
                          :label="$t('objects.listen')"
                        >
                          <template v-slot:append-inner><SettingInfo :text="$t('basic.hint.v2rayListen')" /></template>
                        </v-text-field>
                      </v-col>
                      <v-col cols="12" sm="6" md="3" lg="2">
                        <v-switch v-model="appConfig.experimental.v2ray_api.stats.enabled"
                          color="primary"
                          :label="$t('stats.enable')"
                          hide-details></v-switch>
                      </v-col>
                    </template>
                  </v-row>
                  <v-row v-if="appConfig.experimental.v2ray_api?.stats?.enabled">
                    <v-col cols="12" sm="6">
                      <v-select
                        hide-details
                        :label="$t('pages.inbounds')"
                        multiple chips closable-chips
                        :items="inboundTags"
                        v-model="appConfig.experimental.v2ray_api.stats.inbounds">
                      </v-select>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <v-select
                        hide-details
                        :label="$t('pages.outbounds')"
                        multiple chips closable-chips
                        :items="outboundTags"
                        v-model="appConfig.experimental.v2ray_api.stats.outbounds">
                      </v-select>
                    </v-col>
                    <v-col cols="12" sm="6">
                      <v-select
                        hide-details
                        :label="$t('pages.clients')"
                        multiple chips closable-chips
                        :items="clientNames"
                        v-model="appConfig.experimental.v2ray_api.stats.users">
                      </v-select>
                    </v-col>
                  </v-row>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>
          </v-col>
        </v-row>
      </v-window-item>

      <v-window-item value="t5">
        <MaintenanceTab />
      </v-window-item>
    </v-window>
  </v-card-text>
</v-card>
</template>

<script lang="ts" setup>
import UiModeControl from '@/components/UiModeControl.vue'
import SettingInfo from '@/components/SettingInfo.vue'
import PageHeader from '@/components/nexus/primitives/PageHeader.vue'
import { isNexusEnabled } from '@/uiMode/featureGate'
import { useUiMode } from '@/uiMode/useUiMode'
import { i18n } from '@/locales'
import { Ref, computed, inject, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import HttpUtils from '@/plugins/httputil'
import { FindDiff } from '@/plugins/utils'
import SubJsonExtVue from '@/components/SubJsonExt.vue'
import SubClashExtVue from '@/components/SubClashExt.vue'
import MaintenanceTab from '@/components/settings/MaintenanceTab.vue'
import Dial from '@/components/Dial.vue'
import { normalizeSecretFields, stripSecretPlaceholders } from '@/components/settingsSecretField'
import { push } from 'notivue'
import { Config, Ntp } from '@/types/config'
import Data from '@/store/modules/data'

const route = useRoute()
const tab = ref(route.query.tab === 'basics' ? 't6' : 't1')

watch(() => route.query.tab, (value) => {
  tab.value = value === 'basics' ? 't6' : 't1'
})

const oldConfig = ref<any>({})
const appConfig = ref<any>({
  log: {
    disabled: false,
    level: "info",
    output: "",
    timestamp: false
  },
  experimental: {
    cache_file: {
      enabled: false
    }
  }
})

const cloneStoreConfig = (): any => JSON.parse(JSON.stringify(Data().config ?? {}))

const resyncBasicsFromStore = () => {
  appConfig.value = cloneStoreConfig()
  oldConfig.value = cloneStoreConfig()
}

const basicsStateChange = computed(() => {
  return !FindDiff.deepCompare(appConfig.value, oldConfig.value)
})

const saveBasicsConfig = async () => {
  loading.value = true
  try {
    const dataStore = Data()
    const success = await dataStore.save("config", "set", appConfig.value)
    if (success) {
      resyncBasicsFromStore()
    }
  } finally {
    loading.value = false
  }
}

const inboundTags = computed((): string[] => {
  const dataStore = Data()
  return [...dataStore.inbounds?.map((i:any) => i.tag), ...dataStore.endpoints?.filter((e:any) => e.listen_port > 0).map((e:any) => e.tag)]
})

const clientNames = computed((): string[] => {
  const dataStore = Data()
  const clients = <any[]>dataStore.clients
  return clients?.map(c => c.name)
})

const outboundTags = computed((): string[] => {
  const dataStore = Data()
  return [...dataStore.outbounds?.map((o:any) => o.tag), ...dataStore.endpoints?.map((e:any) => e.tag)]
})

const levels = ["trace", "debug", "info", "warn", "error", "fatal", "panic"]
const certificateModes = [
  { title: i18n.global.t('singbox.off'), value: 'off' },
  { title: 'System', value: 'system' },
  { title: 'Mozilla', value: 'mozilla' },
  { title: 'Chrome', value: 'chrome' },
  { title: i18n.global.t('singbox.customCaFile'), value: 'file' },
  { title: i18n.global.t('singbox.customCaDirectory'), value: 'directory' },
  { title: i18n.global.t('singbox.pastePem'), value: 'pem' },
  { title: i18n.global.t('singbox.advanced'), value: 'custom' },
]
const certificateStores = ['system', 'mozilla', 'chrome', 'none']

function textToList(value: string): string[] | undefined {
  const items = value.split('\n').map(item => item.trim()).filter(item => item.length > 0)
  return items.length > 0 ? items : undefined
}

const enableNtp = computed({
  get() { return appConfig.value.ntp?.enabled?? false },
  set(v:boolean) {
    if (v){
      appConfig.value.ntp = <Ntp>{ enabled: true, server: 'time.apple.com', server_port: 123, interval: '30m'}
    } else { delete appConfig.value.ntp }
  }
})

const ntpInterval = computed({
  get():any { return appConfig.value.ntp?.interval? parseInt(appConfig.value.ntp?.interval.replace('m','')) : null },
  set(v:number) { if (appConfig.value.ntp) v>0 ? appConfig.value.ntp.interval =  v + 'm' : delete appConfig.value.ntp.interval }
})

const enableCacheFile = computed({
  get() { return appConfig.value.experimental?.cache_file?.enabled?? false },
  set(v:boolean) {
    if (!appConfig.value.experimental) appConfig.value.experimental = {}
    if (v){
      appConfig.value.experimental.cache_file = { enabled: true }
    } else { delete appConfig.value.experimental.cache_file  }
  }
})

const enableDebug = computed({
  get() { return appConfig.value.experimental?.debug != undefined },
  set(v:boolean) {
    if (!appConfig.value.experimental) appConfig.value.experimental = {}
    v ? appConfig.value.experimental.debug = {} : delete appConfig.value.experimental.debug }
})

const certificateMode = computed({
  get(): string {
    const cert = appConfig.value.certificate
    if (!cert) return 'off'
    if (cert.certificate && cert.certificate.length > 0) return 'pem'
    if (cert.certificate_path && cert.certificate_path.length > 0) return 'file'
    if (cert.certificate_directory_path && cert.certificate_directory_path.length > 0) return 'directory'
    return cert.store || 'system'
  },
  set(v:string) {
    if (v == 'off') {
      delete appConfig.value.certificate
      return
    }
    appConfig.value.certificate = {}
    if (['system', 'mozilla', 'chrome'].includes(v)) {
      appConfig.value.certificate.store = v as 'system' | 'mozilla' | 'chrome'
    } else if (v == 'file') {
      appConfig.value.certificate.certificate_path = []
    } else if (v == 'directory') {
      appConfig.value.certificate.certificate_directory_path = []
    } else if (v == 'pem') {
      appConfig.value.certificate.certificate = []
    } else if (v == 'custom') {
      appConfig.value.certificate.store = 'none'
    }
  }
})

const certificateText = computed({
  get(): string { return appConfig.value.certificate?.certificate?.join('\n') ?? '' },
  set(v:string) {
    if (!appConfig.value.certificate) appConfig.value.certificate = {}
    const values = textToList(v)
    values ? appConfig.value.certificate.certificate = values : delete appConfig.value.certificate.certificate
  }
})

const certificatePathText = computed({
  get(): string { return appConfig.value.certificate?.certificate_path?.join('\n') ?? '' },
  set(v:string) {
    if (!appConfig.value.certificate) appConfig.value.certificate = {}
    const values = textToList(v)
    values ? appConfig.value.certificate.certificate_path = values : delete appConfig.value.certificate.certificate_path
  }
})

const certificateDirectoryText = computed({
  get(): string { return appConfig.value.certificate?.certificate_directory_path?.join('\n') ?? '' },
  set(v:string) {
    if (!appConfig.value.certificate) appConfig.value.certificate = {}
    const values = textToList(v)
    values ? appConfig.value.certificate.certificate_directory_path = values : delete appConfig.value.certificate.certificate_directory_path
  }
})

const enableClashApi = computed({
  get() { return appConfig.value.experimental?.clash_api != undefined },
  set(v:boolean) {
    if (!appConfig.value.experimental) appConfig.value.experimental = {}
    appConfig.value.experimental.clash_api = v ? { external_controller: '127.0.0.1:9090' } : undefined
  }
})

const enableV2rayApi = computed({
  get() { return appConfig.value.experimental?.v2ray_api != undefined },
  set(v:boolean) {
    if (!appConfig.value.experimental) appConfig.value.experimental = {}
    appConfig.value.experimental.v2ray_api = v ? { listen: '127.0.0.1:8080', stats: { enabled: false, inbounds: [], outbounds: [], users: [] }} : undefined
  }
})

const origin = computed({
  get() { return appConfig.value.experimental?.clash_api?.access_control_allow_origin &&
    appConfig.value.experimental.clash_api.access_control_allow_origin.length>0 ? appConfig.value.experimental.clash_api.access_control_allow_origin.join(',') : '' },
  set(v:string) {
    if (appConfig.value.experimental?.clash_api) {
      appConfig.value.experimental.clash_api.access_control_allow_origin = v.length> 0 ? v.split(',') : undefined
    }
  }
})
// Full IANA timezone list for the timezone picker (a strictly-defined set);
// fall back to a common subset on engines without Intl.supportedValuesOf.
const timezones: string[] = (() => {
  try {
    const list = (Intl as any).supportedValuesOf?.('timeZone')
    if (Array.isArray(list) && list.length) return list
  } catch { /* older engine: use fallback */ }
  return ['UTC', 'Europe/Moscow', 'Europe/London', 'Europe/Berlin', 'America/New_York',
    'America/Los_Angeles', 'America/Sao_Paulo', 'Asia/Shanghai', 'Asia/Tokyo', 'Asia/Dubai',
    'Asia/Kolkata', 'Asia/Tehran', 'Australia/Sydney']
})()
const { mode } = useUiMode()
const nexus = computed(() => mode.value === 'nexus')
const showNexusControls = isNexusEnabled()
const loading:Ref = inject('loading')?? ref(false)
const oldSettings = ref({})

const settings = ref({
	webListen: "",
	webDomain: "",
	webPort: "2095",
	webCertFile: "",
	webKeyFile: "",
  webPath: "/app/",
  webURI: "",
	sessionMaxAge: "0",
  trafficAge: "30",
	timeLocation: "Asia/Shanghai",
  subListen: "",
	subPort: "2096",
	subPath: "/sub/",
	subDomain: "",
	subCertFile: "",
	subKeyFile: "",
	subUpdates: "12",
  subEncode: "true",
  subShowInfo: "false",
  subSecretRequired: "false",
  subRateLimitPerIP: "60",
  subLinkEnable: "true",
  subJsonEnable: "true",
  subClashEnable: "true",
	subURI: "",
  subJsonPath: "/json/",
  subClashPath: "/clash/",
  subJsonURI: "",
  subClashURI: "",
  subTitle: "",
  subSupportUrl: "",
  subProfileUrl: "",
  subAnnounce: "",
  subNameInRemark: "false",
  subJsonExt: "",
  subClashExt: "",
})

onMounted(async () => {
  loading.value = true
  await loadData()
  // Poll until the Data store has loaded the sing-box config, exactly like
  // the original Basics.vue did: the store's loadData() may still be in-flight
  // when Settings is opened directly (e.g. via a bookmark to /settings).
  while (Data().lastLoad === 0) {
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  resyncBasicsFromStore()
  loading.value = false
})

const loadData = async () => {
  loading.value = true
  const msg = await HttpUtils.get('api/settings')
  loading.value = false
  if (msg.success) {
    setData(msg.obj)
  }
}

const setData = (data: any) => {
  const normalized = normalizeSecretFields(data)
  settings.value = normalized
  oldSettings.value = { ...normalized }
}

const save = async () => {
  loading.value = true
  const payload = stripSecretPlaceholders(settings.value)
  const restartRequired = subscriptionPathChanged()
  const msg = await HttpUtils.post('api/save', { object: 'settings', action: 'set', data: JSON.stringify(payload) })
  if (msg.success) {
    push.success({
      title: i18n.global.t('success'),
      duration: 5000,
      message: i18n.global.t('actions.set') + " " + i18n.global.t('pages.settings')
    })
    if (restartRequired) {
      push.warning({
        title: i18n.global.t('setting.restartRequired'),
        duration: 8000,
        message: i18n.global.t('setting.subPathRestartNotice')
      })
    }
    setData(msg.obj.settings)
  }
  loading.value = false
}

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

const restartApp = async () => {
  loading.value = true
  const msg = await HttpUtils.post('api/restartApp',{})
  if (msg.success) {
    let url = settings.value.webURI
    if (url !== "") {
      const isTLS = settings.value.webCertFile !== "" || settings.value.webKeyFile !== ""
      url = buildURL(settings.value.webDomain,settings.value.webPort.toString(),isTLS, settings.value.webPath)
    }
    await sleep(3000)
    window.location.replace(url)
  }
  loading.value = false
}

const buildURL = (host: string, port: string, isTLS: boolean, path: string) => {
  if (!host || host.length == 0) host = window.location.hostname
  if (!port || port.length == 0) port = window.location.port

  const protocol = isTLS ? "https:" : "http:"

  if (port === "" || (isTLS && port === "443") || (!isTLS && port === "80")) {
      port = ""
  } else {
      port = `:${port}`
  }

  return `${protocol}//${host}${port}${path}settings`
}

const subEncode = computed({
  get: () => { return settings.value.subEncode == "true" },
  set: (v:boolean) => { settings.value.subEncode = v ? "true" : "false" }
})

const subShowInfo = computed({
  get: () => { return settings.value.subShowInfo == "true" },
  set: (v:boolean) => { settings.value.subShowInfo = v ? "true" : "false" }
})

const subSecretRequired = computed({
  get: () => { return settings.value.subSecretRequired == "true" },
  set: (v:boolean) => { settings.value.subSecretRequired = v ? "true" : "false" }
})

const subLinkEnable = computed({
  get: () => { return settings.value.subLinkEnable == "true" },
  set: (v:boolean) => { settings.value.subLinkEnable = v ? "true" : "false" }
})

const subJsonEnable = computed({
  get: () => { return settings.value.subJsonEnable == "true" },
  set: (v:boolean) => { settings.value.subJsonEnable = v ? "true" : "false" }
})

const subClashEnable = computed({
  get: () => { return settings.value.subClashEnable == "true" },
  set: (v:boolean) => { settings.value.subClashEnable = v ? "true" : "false" }
})

const subNameInRemark = computed({
  get: () => { return settings.value.subNameInRemark == "true" },
  set: (v:boolean) => { settings.value.subNameInRemark = v ? "true" : "false" }
})

const webPort = computed({
  get: () => { return settings.value.webPort.length>0 ? parseInt(settings.value.webPort) : 2095 },
  set: (v:number) => { settings.value.webPort = v>0 ? v.toString() : "2095" }
})

const sessionMaxAge = computed({
  get: () => { return settings.value.sessionMaxAge.length>0 ? parseInt(settings.value.sessionMaxAge) : 0 },
  set: (v:number) => { settings.value.sessionMaxAge = v>0 ? v.toString() : "0" }
})

const trafficAge = computed({
  get: () => { return settings.value.trafficAge.length>0 ? parseInt(settings.value.trafficAge) : 0 },
  set: (v:number) => { settings.value.trafficAge = v>0 ? v.toString() : "0" }
})

const subPort = computed({
  get: () => { return settings.value.subPort.length>0 ? parseInt(settings.value.subPort) : 2096 },
  set: (v:number) => { settings.value.subPort = v>0 ? v.toString() : "2096" }
})

const subUpdates = computed({
  get: () => { return settings.value.subUpdates.length>0 ? parseInt(settings.value.subUpdates) : 12 },
  set: (v:number) => { settings.value.subUpdates = v>0 ? v.toString() : "12" }
})

const subRateLimitPerIP = computed({
  get: () => { return settings.value.subRateLimitPerIP.length>0 ? parseInt(settings.value.subRateLimitPerIP) : 60 },
  set: (v:number) => { settings.value.subRateLimitPerIP = v>=0 ? v.toString() : "60" }
})

const subscriptionPathKeys = ['subPath', 'subJsonPath', 'subClashPath'] as const

const subscriptionPathChanged = () => {
  return subscriptionPathKeys.some((key) => settings.value[key] !== (oldSettings.value as any)[key])
}

const stateChange = computed(() => {
  return !FindDiff.deepCompare(settings.value,oldSettings.value)
})
</script>

<style scoped>
.settings-actions {
  margin-block-end: 10px;
}

.settings-nexus-card {
  background: var(--nexus-surface-1);
  border: 1px solid var(--nexus-border);
  border-radius: var(--nexus-radius-lg);
}

.settings-nexus-card :deep(.v-tabs) {
  border-block-end: 1px solid var(--nexus-border);
}

.settings-nexus-card :deep(.v-card-text) {
  padding: var(--nexus-gap-5);
  padding-block-start: var(--nexus-gap-4);
}

.settings-nexus-card :deep(.v-window) {
  padding-block-start: var(--nexus-gap-2);
}

.settings-section-card {
  background: var(--nexus-surface-2) !important;
  border: 1px solid var(--nexus-border) !important;
  border-radius: var(--nexus-radius-md) !important;
  height: 100%;
}

.settings-section-title {
  font-size: 0.875rem !important;
  font-weight: 700 !important;
  color: var(--nexus-text-primary) !important;
  padding: var(--nexus-gap-3) var(--nexus-gap-4) !important;
  border-bottom: 1px solid var(--nexus-border);
}

.settings-nexus-card :deep(.v-row) {
  row-gap: var(--nexus-gap-2);
}

.settings-nexus-card :deep(.v-col) {
  min-width: 0;
}

.settings-nexus-card :deep(.v-field) {
  overflow: visible;
}

.settings-nexus-card :deep(.v-field-label) {
  max-width: none !important;
  overflow: visible !important;
  text-overflow: clip !important;
  white-space: normal !important;
}

.settings-actions--nexus {
  gap: var(--nexus-gap-2);
  margin-block-end: var(--nexus-gap-4);
}

.settings-actions--nexus :deep(.v-col) {
  padding: var(--nexus-gap-1);
}

@media (max-width: 600px) {
  .settings-nexus-card :deep(.v-card-text) {
    padding: var(--nexus-gap-3);
  }

  .settings-actions--nexus {
    justify-content: stretch !important;
  }

  .settings-actions--nexus :deep(.v-col) {
    flex: 1 1 100%;
    max-width: 100%;
  }

  .settings-actions--nexus :deep(.v-btn) {
    width: 100%;
  }
}
</style>
