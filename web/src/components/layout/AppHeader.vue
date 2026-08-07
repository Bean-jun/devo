<script setup lang="ts">
import { useAppHeader } from './AppHeaderController'

const { sessionTitle } = useAppHeader()
</script>

<template>
  <header class="app-header">
    <div class="header-left">
      <h1 class="header-title">{{ sessionTitle }}</h1>
    </div>
    <div class="header-right">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped src="./AppHeader.css">
</style>