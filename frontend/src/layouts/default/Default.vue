<template>
  <v-app style="overflow: auto;">
    <drawer :isMobile="isMobile" :displayDrawer="displayDrawer" @toggleDrawer="toggleDrawer" />
    <default-bar :isMobile="isMobile" @toggleDrawer="toggleDrawer" />
    <default-view />
  </v-app>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import DefaultBar from './AppBar.vue'
import Drawer from './Drawer.vue'
import DefaultView from './View.vue'
import { useDisplay } from 'vuetify'

const { smAndDown } = useDisplay()
const displayDrawer = ref(false)

const toggleDrawer = () => {
  displayDrawer.value = !displayDrawer.value
}

// isMobile is a pure derivation of the breakpoint. The drawer's default
// open/closed state follows the breakpoint via a watcher (a computed getter must
// not mutate state — doing so caused drawer thrash on every resize/re-render).
const isMobile = computed( ():boolean => smAndDown.value)
watch(smAndDown, (value) => { displayDrawer.value = !value }, { immediate: true })
</script>

<style>
.v-card-subtitle {
  text-align: center;
  border-bottom: 1px solid gray;
  min-height: 20px;
}
.v-switch.v-input {
  padding-inline-start: .6rem;
}
</style>