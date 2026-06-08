<template>
  <v-dialog transition="dialog-bottom-transition" width="460">
    <v-card class="rounded-lg">
      <v-card-title>
        {{ $t('admin.addAdmin') }}
      </v-card-title>
      <v-divider></v-divider>
      <v-card-text>
        <v-alert
          v-if="error"
          type="error"
          density="compact"
          variant="tonal"
          class="mb-4"
        >
          {{ error }}
        </v-alert>
        <v-row>
          <v-col>
            <v-text-field
              v-model="newData.currentPass"
              :label="$t('admin.oldPass')"
              type="password"
              autocomplete="current-password"
              required
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col>
            <v-text-field
              v-model="newData.username"
              :label="$t('admin.newUname')"
              autocomplete="username"
              required
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col>
            <v-text-field
              v-model="newData.password"
              :label="$t('admin.newPass')"
              type="password"
              autocomplete="new-password"
              required
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col>
            <v-text-field
              v-model="newData.confirmPassword"
              :label="$t('admin.confirmPass')"
              type="password"
              autocomplete="new-password"
              required
            ></v-text-field>
          </v-col>
        </v-row>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn
          color="primary"
          variant="outlined"
          @click="closeModal"
        >
          {{ $t('actions.close') }}
        </v-btn>
        <v-btn
          color="primary"
          variant="tonal"
          @click="saveChanges"
        >
          {{ $t('actions.add') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import { i18n } from '@/locales'
import {
  addAdminPasswordsMatch,
  isAddAdminFormComplete,
  normalizeAdminUsername,
} from '@/views/adminForms'

export default {
  props: ['visible'],
  data() {
    return {
      error: '',
      newData: {
        currentPass: '',
        username: '',
        password: '',
        confirmPassword: '',
      },
    }
  },
  methods: {
    resetData() {
      this.error = ''
      this.newData.currentPass = ''
      this.newData.username = ''
      this.newData.password = ''
      this.newData.confirmPassword = ''
    },
    closeModal() {
      this.resetData()
      this.$emit('close')
    },
    saveChanges() {
      this.error = ''
      if (!isAddAdminFormComplete(this.newData)) {
        this.error = i18n.global.t('admin.addValidation') as string
        return
      }
      if (!addAdminPasswordsMatch(this.newData)) {
        this.error = i18n.global.t('admin.passwordMismatch') as string
        return
      }
      this.$emit('save', {
        currentPass: this.newData.currentPass,
        username: normalizeAdminUsername(this.newData.username),
        password: this.newData.password,
      })
    },
  },
  watch: {
    visible(newValue) {
      if (newValue) {
        this.resetData()
      }
    },
  },
}
</script>
