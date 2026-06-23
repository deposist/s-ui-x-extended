<template>
  <v-overlay
    :model-value="loading"
    persistent
    content-class="text-center"
    class="align-center justify-center"
  >
    <v-progress-circular
      indeterminate
      size="64"
    ></v-progress-circular>
    <br />
    {{ $t('loading') }}
  </v-overlay>
  <Message />
  <router-view />
</template>

<script lang="ts" setup>
import Message from '@/components/message.vue'
import { inject, ref, Ref } from 'vue'

const loading:Ref = inject('loading')?? ref(false)

// Change page title
document.title = "S-UI-X Extended " + document.location.hostname
</script>

<style>
.v-overlay .v-list-item,
.v-field__input {
  direction: ltr;
}

/* (i) tooltips: guarantee readable contrast in every theme. The nexus dark theme
   sets surface-variant but no on-surface-variant, so Vuetify's auto-derived
   tooltip text was too muted to read (reported on the Telegram/Settings hints).
   No tooltip in the app sets a custom color, so a solid dark-bg/light-text scheme
   is safe app-wide and reads well on both light and dark themes. */
.v-tooltip .v-overlay__content {
  background: rgba(17, 24, 45, 0.98) !important;
  color: #eef2f9 !important;
  opacity: 1 !important;
  border: 1px solid rgba(255, 255, 255, 0.14);
  font-size: 0.78rem;
  line-height: 1.4;
}

/* Don't clip floating field labels. The base .v-label has overflow:hidden +
   text-overflow:ellipsis; with persistent-placeholder + an append-inner (i) icon
   the floating label's width got constrained and short labels were truncated to
   "Add…", "P…", "Do…". Let the floating label size to its content and not clip.
   (Vuetify ships its CSS in @layer, so this unlayered rule wins without !important.) */
.v-label.v-field-label--floating {
  max-width: none;
  overflow: visible;
}
</style>