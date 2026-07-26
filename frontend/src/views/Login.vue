<template>
  <!-- The login route renders outside NexusShell, so it needs the same
       palette-aware theme provider; otherwise Submit fell back to Vuetify's
       default blue while the rest of the panel is cyan. -->
  <v-theme-provider class="login-shell" :theme="nexusThemeName" with-background>
    <v-container class="login-page fill-height">
      <v-row justify="center" align="center">
        <v-col cols="12" sm="8" md="5" lg="4">
          <v-card>
            <!-- The form previously opened with a bare "Login" heading and no
                 product identity, while the language/theme controls below drew
                 more attention than the form itself. -->
            <div class="login-brand">
              <span aria-hidden="true" class="login-brand__logo">X</span>
              <span class="login-brand__name">S-UI-X Extended</span>
            </div>
            <v-card-title class="headline" v-text="$t('login.title')"></v-card-title>
            <v-card-text>
              <v-form v-if="!forcePasswordReset" @submit.prevent="login" ref="form">
                <v-text-field v-model="username" :label="$t('login.username')" :rules="usernameRules" required @update:modelValue="errorMsg = ''"></v-text-field>
                <v-text-field
                  v-model="password"
                  :label="$t('login.password')"
                  :rules="passwordRules"
                  :type="showPassword ? 'text' : 'password'"
                  :append-inner-icon="showPassword ? 'lucide:eye-off' : 'lucide:eye'"
                  autocomplete="current-password"
                  required
                  @click:append-inner="showPassword = !showPassword"
                  @update:modelValue="errorMsg = ''"
                ></v-text-field>
                <v-alert v-if="errorMsg" type="error" density="compact" variant="tonal" class="mt-1">{{ errorMsg }}</v-alert>
                <v-btn :loading="loading" type="submit" color="primary" block class="mt-2" v-text="$t('actions.submit')"></v-btn>
              </v-form>
              <v-form v-else @submit.prevent="changeForcedPassword">
                <v-alert type="warning" density="compact" variant="tonal" class="mb-2">{{ $t('login.forcePasswordReset') }}</v-alert>
                <v-text-field v-model="resetUsername" :label="$t('admin.newUname')" :rules="usernameRules" required @update:modelValue="errorMsg = ''"></v-text-field>
                <v-text-field
                  v-model="resetPassword"
                  :label="$t('admin.newPass')"
                  :rules="passwordRules"
                  :type="showPassword ? 'text' : 'password'"
                  :append-inner-icon="showPassword ? 'lucide:eye-off' : 'lucide:eye'"
                  autocomplete="new-password"
                  required
                  @click:append-inner="showPassword = !showPassword"
                  @update:modelValue="errorMsg = ''"
                ></v-text-field>
                <v-alert v-if="errorMsg" type="error" density="compact" variant="tonal" class="mt-1">{{ errorMsg }}</v-alert>
                <v-btn :loading="loading" type="submit" color="primary" block class="mt-2" v-text="$t('actions.save')"></v-btn>
              </v-form>
              <!-- Secondary utilities: keep them visually quiet (plain variant,
                   no filled surface) so the credentials form stays dominant. -->
              <v-select
                density="compact"
                class="mt-4 login-utilities"
                hide-details
                variant="plain"
                :label="$t('menu.language')"
                :items="languages"
                v-model="$i18n.locale"
                @update:modelValue="changeLocale">
                <template v-slot:append>
                  <v-menu>
                    <template v-slot:activator="{ props }">
                      <v-btn
                        :aria-label="$t('menu.theme')"
                        icon
                        size="small"
                        :title="$t('menu.theme')"
                        variant="text"
                        v-bind="props"
                      >
                        <v-icon icon="lucide:sun-moon" />
                      </v-btn>
                    </template>
                    <v-list>
                      <v-list-item
                        v-for="th in themes"
                        :key="th.value"
                        @click="changeTheme(th.value)"
                        :prepend-icon="th.icon"
                        :active="isActiveTheme(th.value)"
                      >
                        <v-list-item-title>{{ $t(`theme.${th.value}`) }}</v-list-item-title>
                      </v-list-item>
                    </v-list>
                  </v-menu>
                </template>
              </v-select>
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </v-theme-provider>
</template>
  
<script lang="ts" setup>
import { ref } from "vue"
import { useLocale,useTheme } from 'vuetify'
import { i18n, languages, setI18nLocale } from '@/locales'
import { useRouter } from 'vue-router'
import HttpUtil, { resetInvalidLoginHandling, markLoginSuccess } from '@/plugins/httputil'
import { useNexusTheme } from '@/uiMode/nexusTheme'


const theme = useTheme()
const locale = useLocale()
const nexusThemeName = useNexusTheme()

// Match the in-app topbar icon set (lucide); mdi names rendered as blanks here.
const themes = [
  { value: 'light', icon: 'lucide:sun' },
  { value: 'dark', icon: 'lucide:moon' },
  { value: 'system', icon: 'lucide:monitor' },
]

const showPassword = ref(false)

const username = ref('')
const usernameRules = [
  (value: string) => {
    if (value?.length > 0) return true
    return i18n.global.t('login.unRules')
  },
]

const password = ref('')
const resetUsername = ref('')
const resetPassword = ref('')
const forcePasswordReset = ref(false)
const passwordRules = [
  (value: string) => {
    if (value?.length > 0) return true
    return i18n.global.t('login.pwRules')
  },
]

const loading = ref(false)
const errorMsg = ref('')
const router = useRouter()

const login = async () => {
  if (username.value == '' || password.value == '') return
  errorMsg.value = ''
  loading.value=true
  const response = await HttpUtil.post('api/login',{user: username.value, pass: password.value})
  if(response.obj?.forcePasswordReset){
    forcePasswordReset.value = true
    resetUsername.value = response.obj.username || username.value
    resetPassword.value = ''
    loading.value=false
    return
  }
  if(response.success){
    resetInvalidLoginHandling()
    markLoginSuccess()
    loading.value=false
    router.push('/')
  } else {
    loading.value=false
    // Surface the reason inline (e.g. lockout countdown) and fall back to a
    // localized "invalid credentials" message for the common wrong-password case.
    errorMsg.value = response.msg || i18n.global.t('login.invalidCredentials')
  }
}

const changeForcedPassword = async () => {
  if (resetUsername.value == '' || resetPassword.value == '') return
  errorMsg.value = ''
  loading.value = true
  const response = await HttpUtil.post('api/changePass',{oldPass: password.value, newUsername: resetUsername.value, newPass: resetPassword.value})
  loading.value = false
  if(response.success){
    resetInvalidLoginHandling()
    markLoginSuccess()
    router.push('/')
    return
  }
  errorMsg.value = response.msg || i18n.global.t('login.invalidCredentials')
}
const changeLocale = async (l: string | null) => {
  const selectedLocale = await setI18nLocale(l ?? 'en')
  locale.current.value = selectedLocale
}
const changeTheme = (th: string) => {
  theme.change(th)
  localStorage.setItem('theme', th)
}
const isActiveTheme = (th: string) => {
  // Mirror vuetify.ts defaultTheme and the in-app topbar: no stored choice → dark.
  // This screen previously reported 'system' as active while rendering dark.
  const current = localStorage.getItem('theme') ?? 'dark'
  return current == th
}
</script>

<style lang="scss" scoped>
/* The --nexus-* token blocks ship with NexusShell's stylesheet, which never
   mounts on this route, so the brand mark resolved to an empty custom property. */
@use '@/styles/nexus/tokens';

/* Was a fixed margin-top:100px, which pushed the card off-centre on short
   viewports and left it top-heavy on tall ones. */
.login-shell {
  align-items: center;
  display: flex;
  justify-content: center;
  min-height: 100vh;
}

.login-page {
  background: var(--nexus-surface-0);
  /* fill-height only stretches inside a flex parent that has real height; the
     container itself must not force 100vh or the row centring is a no-op. */
  min-height: 0;
}

.login-brand {
  align-items: center;
  display: flex;
  gap: 0.75rem;
  padding: 1.25rem 1.25rem 0;
}

.login-brand__logo {
  align-items: center;
  background: var(--nexus-accent-primary);
  border-radius: var(--nexus-radius-md);
  color: var(--nexus-surface-0);
  display: inline-flex;
  flex: 0 0 auto;
  font-size: 18px;
  font-weight: 700;
  height: 32px;
  justify-content: center;
  line-height: 1;
  width: 32px;
}

.login-brand__name {
  font-size: 1rem;
  font-weight: 600;
}

/* Keep the language picker subdued relative to the submit button. */
.login-utilities :deep(.v-field__input) {
  font-size: 0.85rem;
}

/* Vuetify top-aligns the append slot and reserves room for the floating label,
   which left the theme icon ~8px below the select's dropdown arrow. Centre it
   on the field row so the two icons line up. */
.login-utilities :deep(.v-input__append) {
  align-items: center;
  align-self: center;
  margin-block-start: 0;
  padding-block-start: 0;
}

/* Separate the utility row from the form with a hairline, so it reads as
   secondary chrome rather than another form field. */
.login-utilities {
  border-block-start: 1px solid var(--nexus-border);
  padding-block-start: 0.5rem;
}
</style>
