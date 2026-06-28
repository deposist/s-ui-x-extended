<template>
  <v-card subtitle="Redirect">
    <v-alert type="info" variant="tonal" density="compact">
      Redirect inbound has no protocol-specific options beyond Listen.
      It transparently redirects traffic (TPROXY/redirect iptables/nftables)
      to this inbound for processing.
    </v-alert>
  </v-card>
</template>

<script lang="ts">
export default {
  props: ['data'],
}
</script>
