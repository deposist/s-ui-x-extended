<template>
  <v-container fluid class="migrate-xui">
    <v-row>
      <v-col cols="12" md="6">
        <div class="text-h5">{{ $t('migrateXui.title') }}</div>
      </v-col>
    </v-row>

    <v-row class="mb-2">
      <v-col cols="12">
        <v-card class="pa-3" rounded="lg">
          <v-row>
            <v-col
              v-for="item in stepItems"
              :key="item.value"
              cols="12"
              sm="3"
            >
              <v-btn
                block
                :color="step === item.value ? 'primary' : undefined"
                :variant="step === item.value ? 'flat' : 'tonal'"
                :disabled="item.value > maxStep"
                @click="step = item.value"
              >
                <v-icon :icon="item.icon" start></v-icon>
                {{ item.title }}
              </v-btn>
            </v-col>
          </v-row>
        </v-card>
      </v-col>
    </v-row>

    <v-window v-model="step">
      <v-window-item :value="1">
        <v-card class="pa-4" rounded="lg">
          <v-row>
            <v-col cols="12" md="5">
              <v-file-input
                v-model="file"
                accept=".db"
                prepend-icon="mdi-database-import"
                :label="$t('migrateXui.chooseFile')"
                :disabled="loading"
                hide-details
              ></v-file-input>
            </v-col>
            <v-col cols="12" sm="6" md="3">
              <v-select
                v-model="strategy"
                :items="strategyItems"
                :label="$t('migrateXui.strategy')"
                :disabled="loading"
                hide-details
              ></v-select>
            </v-col>
            <v-col cols="12" sm="6" md="4">
              <v-select
                v-model="adminMode"
                :items="adminModeItems"
                :label="$t('migrateXui.adminMode')"
                :disabled="loading"
                hide-details
              ></v-select>
            </v-col>
            <v-col cols="12" md="5">
              <v-checkbox
                v-model="includeSettings"
                :label="$t('migrateXui.includeSettings')"
                :disabled="loading"
                hide-details
              ></v-checkbox>
            </v-col>
            <v-col cols="12" md="4">
              <v-checkbox
                v-model="includeHistory"
                :label="$t('migrateXui.includeHistory')"
                :disabled="loading"
                hide-details
              ></v-checkbox>
            </v-col>
            <v-col cols="12" md="3">
              <v-checkbox
                v-model="includeRouting"
                :label="$t('migrateXui.includeRouting')"
                :disabled="loading"
                hide-details
              ></v-checkbox>
            </v-col>
            <v-spacer></v-spacer>
            <v-col cols="12" md="auto" align-self="center">
              <v-btn
                color="primary"
                prepend-icon="mdi-clipboard-search"
                :loading="loading"
                :disabled="!selectedFile"
                @click="buildPlan"
              >
                {{ $t('migrateXui.buildPlan') }}
              </v-btn>
            </v-col>
          </v-row>
        </v-card>
      </v-window-item>

      <v-window-item :value="2">
        <v-card class="pa-4" rounded="lg">
          <v-row>
            <v-col cols="12" md="4">
              <div class="text-subtitle-1">{{ $t('migrateXui.reviewTitle') }}</div>
              <div class="text-caption text-medium-emphasis">{{ $t('migrateXui.sourceHash') }}: {{ plan?.source?.hash || '-' }}</div>
            </v-col>
            <v-col cols="12" sm="6" md="3">
              <v-select
                v-model="kindFilter"
                :items="kindFilterItems"
                :label="$t('migrateXui.filterKind')"
                hide-details
              ></v-select>
            </v-col>
            <v-col cols="12" sm="6" md="2">
              <v-text-field
                v-model.trim="search"
                prepend-inner-icon="mdi-magnify"
                :label="$t('migrateXui.search')"
                hide-details
              ></v-text-field>
            </v-col>
            <v-col cols="12" md="3" align-self="center">
              <div class="text-caption">
                {{ $t('migrateXui.selectedCount') }}: {{ selectedCount }} / {{ totalItems }}
              </div>
            </v-col>
          </v-row>

          <v-alert
            v-if="applyError"
            class="mt-3"
            type="error"
            variant="tonal"
            data-testid="migrate-xui-apply-error"
            :title="$t('migrateXui.applyFailed')"
          >
            {{ applyError }}
          </v-alert>

          <v-data-table-virtual
            class="mt-3"
            fixed-header
            show-expand
            density="compact"
            height="560"
            item-value="rowKey"
            :headers="headers"
            :items="filteredItems"
          >
            <template #item.import="{ item }">
              <v-checkbox-btn
                :model-value="rowItem(item).action !== 'skip'"
                @update:model-value="setImport(rowItem(item), Boolean($event))"
              ></v-checkbox-btn>
            </template>
            <template #item.kind="{ item }">
              <v-chip size="small" variant="tonal">{{ kindTitle(rowItem(item).kind) }}</v-chip>
            </template>
            <template #item.srcTag="{ item }">
              <span class="text-body-2">{{ rowItem(item).srcTag || rowItem(item).srcId }}</span>
            </template>
            <template #item.dstTag="{ item }">
              <v-text-field
                v-model="rowItem(item).dstTag"
                density="compact"
                hide-details
              ></v-text-field>
            </template>
            <template #item.action="{ item }">
              <v-select
                v-model="rowItem(item).action"
                :items="actionItems"
                density="compact"
                hide-details
              ></v-select>
            </template>
            <template #item.conflict="{ item }">
              <v-chip v-if="rowItem(item).conflict" size="small" color="warning" variant="tonal">
                {{ $t('migrateXui.conflict') }}
              </v-chip>
              <span v-else>-</span>
            </template>
            <template #expanded-row="{ columns, item }">
              <tr>
                <td :colspan="columns.length">
                  <v-expansion-panels class="my-2" variant="accordion">
                    <v-expansion-panel :title="$t('migrateXui.previewJson')">
                      <v-expansion-panel-text>
                        <pre class="preview-json">{{ previewText(rowItem(item)) }}</pre>
                      </v-expansion-panel-text>
                    </v-expansion-panel>
                    <v-expansion-panel v-if="rowItem(item).warnings?.length" :title="$t('migrateXui.warnings')">
                      <v-expansion-panel-text>
                        <ul>
                          <li v-for="warning in rowItem(item).warnings" :key="warning">{{ warning }}</li>
                        </ul>
                      </v-expansion-panel-text>
                    </v-expansion-panel>
                  </v-expansion-panels>
                </td>
              </tr>
            </template>
          </v-data-table-virtual>

          <v-row class="mt-4">
            <v-col cols="auto">
              <v-btn variant="tonal" prepend-icon="mdi-arrow-left" @click="step = 1">{{ $t('migrateXui.back') }}</v-btn>
            </v-col>
            <v-spacer></v-spacer>
            <v-col cols="auto">
              <v-btn color="primary" prepend-icon="mdi-database-check" :disabled="selectedCount === 0" @click="applyPlan">
                {{ $t('migrateXui.apply') }}
              </v-btn>
            </v-col>
          </v-row>
        </v-card>
      </v-window-item>

      <v-window-item :value="3">
        <v-card class="pa-4" rounded="lg">
          <div class="text-subtitle-1 mb-3">{{ $t('migrateXui.progressTitle') }}</div>
          <v-progress-linear
            :model-value="progressPercent"
            color="primary"
            height="12"
            rounded
          ></v-progress-linear>
          <v-row class="mt-3">
            <v-col cols="12" sm="4">{{ $t('migrateXui.current') }}: {{ activeProgress?.step || '-' }}</v-col>
            <v-col cols="12" sm="4">{{ activeProgress?.current || 0 }} / {{ activeProgress?.total || 0 }}</v-col>
            <v-col cols="12" sm="4">{{ activeProgress?.currentTag || activeProgress?.currentName || '-' }}</v-col>
          </v-row>
        </v-card>
      </v-window-item>

      <v-window-item :value="4">
        <v-card class="pa-4" rounded="lg">
          <v-row>
            <v-col cols="12" md="6">
              <div class="text-subtitle-1 mb-2">{{ $t('migrateXui.resultTitle') }}</div>
              <pre class="preview-json">{{ summaryText }}</pre>
            </v-col>
            <v-col cols="12" md="6">
              <div class="text-subtitle-2">{{ $t('migrateXui.backupPath') }}</div>
              <div class="text-body-2 backup-path">{{ report?.backupPath || '-' }}</div>
              <v-row class="mt-3">
                <v-col cols="12" sm="auto">
                  <v-btn prepend-icon="mdi-code-json" variant="tonal" :disabled="!report" @click="downloadJSON">
                    {{ $t('migrateXui.downloadJson') }}
                  </v-btn>
                </v-col>
                <v-col cols="12" sm="auto">
                  <v-btn prepend-icon="mdi-language-markdown" variant="tonal" :disabled="!report" @click="downloadMarkdown">
                    {{ $t('migrateXui.downloadMarkdown') }}
                  </v-btn>
                </v-col>
                <v-col cols="12" sm="auto">
                  <v-btn
                    color="warning"
                    prepend-icon="mdi-database-refresh"
                    :loading="rollbackLoading"
                    :disabled="!report?.backupPath"
                    @click="rollback"
                  >
                    {{ $t('migrateXui.restore') }}
                  </v-btn>
                </v-col>
              </v-row>
            </v-col>
            <v-col v-if="rollbackError" cols="12" md="6" offset-md="6">
              <v-alert
                type="error"
                variant="tonal"
                data-testid="migrate-xui-rollback-error"
                :title="$t('migrateXui.rollbackFailed')"
              >
                {{ rollbackError }}
              </v-alert>
            </v-col>
            <v-col v-if="report?.warnings?.length" cols="12">
              <v-alert type="warning" variant="tonal" :title="$t('migrateXui.warnings')">
                <ul>
                  <li v-for="warning in report.warnings" :key="warning">{{ warning }}</li>
                </ul>
              </v-alert>
            </v-col>
            <v-col v-if="hasGeneratedAdmins" cols="12">
              <v-alert type="info" variant="tonal" :title="$t('migrateXui.generatedAdmins')" data-testid="migrate-xui-generated-admins">
                <div class="mb-2">{{ $t('migrateXui.passwordShownOnce') }}</div>
                <div v-if="!generatedAdminsRevealed" class="text-body-2 mb-2" data-testid="migrate-xui-generated-admins-hidden">
                  {{ $t('migrateXui.passwordsHidden') }}
                </div>
                <v-row class="mb-2" density="compact">
                  <v-col cols="auto">
                    <v-btn
                      variant="tonal"
                      :prepend-icon="generatedAdminsRevealed ? 'mdi-eye-off' : 'mdi-eye'"
                      @click="generatedAdminsRevealed = !generatedAdminsRevealed"
                    >
                      {{ generatedAdminsRevealed ? $t('migrateXui.hideGeneratedAdmins') : $t('migrateXui.revealGeneratedAdmins') }}
                    </v-btn>
                  </v-col>
                  <v-col cols="auto">
                    <v-btn variant="tonal" prepend-icon="mdi-delete-outline" @click="clearGeneratedAdmins">
                      {{ $t('migrateXui.clearGeneratedAdmins') }}
                    </v-btn>
                  </v-col>
                </v-row>
                <pre v-if="generatedAdminsRevealed" class="preview-json" data-testid="migrate-xui-generated-admins-json">{{ generatedAdminsText }}</pre>
              </v-alert>
            </v-col>
          </v-row>
        </v-card>
      </v-window-item>
    </v-window>
  </v-container>
</template>

<script lang="ts">
import Data from '@/store/modules/data'
import Ws from '@/store/ws'
import HttpUtils from '@/plugins/httputil'
import api from '@/plugins/api'

type PlanItem = {
  rowKey?: string
  kind: string
  srcId: any
  srcTag?: string
  dstTag: string
  action: string
  conflict: boolean
  previewJson: any
  warnings?: string[]
}

type MigrationPlan = {
  items: PlanItem[]
  source: { hash: string }
  defaults: Record<string, any>
}

const generatedAdminsAutoClearMs = 5 * 60 * 1000

export default {
  data() {
    return {
      step: 1,
      maxStep: 1,
      file: null as File | File[] | null,
      strategy: 'merge',
      // Default the optional categories ON so a migration carries over routes,
      // panel settings and traffic history automatically. Admin import stays
      // opt-in (skip) because it creates/overwrites credentials.
      includeSettings: true,
      includeHistory: true,
      includeRouting: true,
      adminMode: 'skip',
      loading: false,
      rollbackLoading: false,
      kindFilter: 'all',
      search: '',
      plan: null as MigrationPlan | null,
      report: null as any,
      progress: null as any,
      applyError: '',
      rollbackError: '',
      generatedAdminsRevealed: false,
      generatedAdminsClearTimer: undefined as ReturnType<typeof setTimeout> | undefined,
    }
  },
  computed: {
    selectedFile(): File | null {
      if (Array.isArray(this.file)) return this.file[0] ?? null
      return this.file
    },
    stepItems(): any[] {
      return [
        { value: 1, title: this.$t('migrateXui.steps.upload'), icon: 'mdi-upload' },
        { value: 2, title: this.$t('migrateXui.steps.review'), icon: 'mdi-format-list-checks' },
        { value: 3, title: this.$t('migrateXui.steps.progress'), icon: 'mdi-progress-clock' },
        { value: 4, title: this.$t('migrateXui.steps.result'), icon: 'mdi-check-circle' },
      ]
    },
    strategyItems(): any[] {
      return ['merge', 'replace', 'skip'].map(value => ({ value, title: this.$t(`migrateXui.actions.${value}`) }))
    },
    actionItems(): any[] {
      return ['create', 'merge', 'replace', 'skip'].map(value => ({ value, title: this.$t(`migrateXui.actions.${value}`) }))
    },
    adminModeItems(): any[] {
      return ['skip', 'new_password', 'reset_required'].map(value => ({ value, title: this.$t(`migrateXui.adminModes.${value}`) }))
    },
    kindFilterItems(): any[] {
      const kinds = ['tls', 'inbound', 'endpoint', 'client', 'setting', 'admin', 'historical', 'routing']
      return [
        { value: 'all', title: this.$t('migrateXui.allKinds') },
        ...kinds.map(value => ({ value, title: this.kindTitle(value) })),
      ]
    },
    headers(): any[] {
      return [
        { title: this.$t('migrateXui.import'), key: 'import', sortable: false, width: 72 },
        { title: this.$t('migrateXui.kind'), key: 'kind', width: 120 },
        { title: this.$t('migrateXui.source'), key: 'srcTag' },
        { title: this.$t('migrateXui.destination'), key: 'dstTag', sortable: false },
        { title: this.$t('migrateXui.action'), key: 'action', sortable: false, width: 180 },
        { title: this.$t('migrateXui.conflict'), key: 'conflict', width: 120 },
      ]
    },
    filteredItems(): PlanItem[] {
      const needle = this.search.toLowerCase()
      return (this.plan?.items ?? []).filter((item) => {
        if (this.kindFilter !== 'all' && item.kind !== this.kindFilter) return false
        if (!needle) return true
        return [item.kind, item.srcTag, item.srcId, item.dstTag, item.action].join(' ').toLowerCase().includes(needle)
      })
    },
    totalItems(): number {
      return this.plan?.items.length ?? 0
    },
    selectedCount(): number {
      return (this.plan?.items ?? []).filter(item => item.action !== 'skip').length
    },
    wsProgress(): any {
      return Ws().xuiImportProgress
    },
    activeProgress(): any {
      return this.progress || this.wsProgress
    },
    progressPercent(): number {
      return Number(this.activeProgress?.percent ?? 0)
    },
    summaryText(): string {
      return JSON.stringify(this.report?.summary ?? {}, null, 2)
    },
    generatedAdmins(): any[] {
      if (Array.isArray(this.report?.generatedAdmins)) return this.report.generatedAdmins
      if (Array.isArray(this.report?.generated_admins)) return this.report.generated_admins
      return []
    },
    hasGeneratedAdmins(): boolean {
      return this.generatedAdmins.length > 0
    },
    generatedAdminsText(): string {
      return JSON.stringify(this.generatedAdmins, null, 2)
    },
  },
  watch: {
    wsProgress(value: any) {
      if (value && this.step === 3) {
        this.progress = value
      }
    },
  },
  mounted() {
    Ws().connect()
  },
  beforeUnmount() {
    this.clearGeneratedAdminsTimer()
  },
  methods: {
    async buildPlan() {
      if (!this.selectedFile) return
      this.loading = true
      this.applyError = ''
      const formData = new FormData()
      formData.append('db', this.selectedFile)
      formData.append('strategy', this.strategy)
      formData.append('includeSettings', this.includeSettings ? '1' : '0')
      formData.append('includeHistory', this.includeHistory ? '1' : '0')
      formData.append('includeRouting', this.includeRouting ? '1' : '0')
      formData.append('adminMode', this.adminMode)
      const msg = await HttpUtils.post('api/import-xui/plan', formData)
      this.loading = false
      if (!msg.success) return
      const plan = msg.obj as MigrationPlan
      plan.items = (plan.items ?? []).map((item, index) => ({
        ...item,
        rowKey: `${item.kind}:${String(item.srcId)}:${index}`,
      }))
      this.plan = plan
      this.clearGeneratedAdminsTimer()
      this.generatedAdminsRevealed = false
      this.report = null
      this.progress = null
      this.maxStep = Math.max(this.maxStep, 2)
      this.step = 2
    },
    async applyPlan() {
      if (!this.selectedFile || !this.plan) return
      this.applyError = ''
      this.progress = { step: 'queued', current: 0, total: Math.max(this.selectedCount, 1), percent: 0 }
      this.maxStep = Math.max(this.maxStep, 3)
      this.step = 3
      const formData = new FormData()
      formData.append('db', this.selectedFile)
      formData.append('plan', JSON.stringify(this.plan))
      const msg = await HttpUtils.post('api/import-xui/apply', formData)
      if (!msg.success) {
        this.step = 2
        this.progress = null
        this.applyError = msg.msg || this.$t('migrateXui.applyFailedFallback')
        return
      }
      this.report = msg.obj
      this.generatedAdminsRevealed = false
      this.scheduleGeneratedAdminsClear()
      this.progress = { step: 'done', current: this.selectedCount, total: Math.max(this.selectedCount, 1), percent: 100 }
      this.maxStep = 4
      this.step = 4
      await Data().loadData()
    },
    async rollback() {
      if (!this.report?.backupPath) return
      this.rollbackError = ''
      this.rollbackLoading = true
      try {
        const body = new URLSearchParams()
        body.set('backup', this.report.backupPath)
        const msg = await HttpUtils.post('api/import-xui/rollback', body)
        if (!msg.success) {
          this.rollbackError = msg.msg || this.$t('migrateXui.rollbackFailedFallback')
          return
        }
        const ready = await this.waitForRollbackReady()
        if (ready) {
          location.reload()
          return
        }
        this.rollbackError = this.$t('migrateXui.rollbackHealthTimeout')
      } finally {
        this.rollbackLoading = false
      }
    },
    async waitForRollbackReady(timeoutMs = 15000, intervalMs = 500): Promise<boolean> {
      const deadline = Date.now() + timeoutMs
      while (Date.now() < deadline) {
        try {
          const response = await api.get('api/status', { params: { r: 'db' } })
          const body = response.data
          if (body?.success && body.obj && Object.prototype.hasOwnProperty.call(body.obj, 'db') && body.obj.db !== null) {
            return true
          }
        } catch {
          // Backend may be restarting after rollback; keep polling until timeout.
        }
        await new Promise(resolve => setTimeout(resolve, intervalMs))
      }
      return false
    },
    clearGeneratedAdminsTimer() {
      if (this.generatedAdminsClearTimer) {
        clearTimeout(this.generatedAdminsClearTimer)
        this.generatedAdminsClearTimer = undefined
      }
    },
    scheduleGeneratedAdminsClear() {
      this.clearGeneratedAdminsTimer()
      if (!this.hasGeneratedAdmins) return
      this.generatedAdminsClearTimer = setTimeout(() => {
        this.clearGeneratedAdmins()
      }, generatedAdminsAutoClearMs)
    },
    clearGeneratedAdmins() {
      this.clearGeneratedAdminsTimer()
      if (this.report) {
        this.report.generatedAdmins = []
      }
      this.generatedAdminsRevealed = false
    },
    setImport(item: PlanItem, enabled: boolean) {
      item.action = enabled ? (item.conflict ? this.strategy : 'create') : 'skip'
    },
    rowItem(item: any): PlanItem {
      return item?.raw ?? item
    },
    kindTitle(kind: string): string {
      return this.$t(`migrateXui.kinds.${kind}`) as string
    },
    previewText(item: PlanItem): string {
      return JSON.stringify(item.previewJson ?? null, null, 2)
    },
    downloadJSON() {
      this.download('xui-import-report.json', JSON.stringify(this.report ?? {}, null, 2), 'application/json')
    },
    downloadMarkdown() {
      this.download('xui-import-report.md', this.markdownReport(), 'text/markdown')
    },
    download(name: string, content: string, type: string) {
      const blob = new Blob([content], { type })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = name
      link.click()
      URL.revokeObjectURL(url)
    },
    markdownReport(): string {
      const summary = this.report?.summary ?? {}
      const lines = ['# 3x-ui import report', '', `Backup: ${this.report?.backupPath || '-'}`, '']
      for (const [key, value] of Object.entries(summary)) {
        lines.push(`## ${key}`, '```json', JSON.stringify(value, null, 2), '```', '')
      }
      if (this.report?.warnings?.length) {
        lines.push('## Warnings', ...this.report.warnings.map((warning: string) => `- ${warning}`), '')
      }
      return lines.join('\n')
    },
  },
}
</script>

<style scoped>
.migrate-xui {
  max-width: 1440px;
}
.preview-json {
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}
.backup-path {
  word-break: break-all;
}
</style>
