<script setup lang="ts">
import { useApp } from './AppController'

const { isMobileMode, BrowserLayout, MobileLayout } = useApp()
</script>

<template>
  <MobileLayout v-if="isMobileMode" />
  <BrowserLayout v-else />
</template>