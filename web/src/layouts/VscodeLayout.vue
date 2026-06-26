<script setup lang="ts">
import StatusBar from '@/components/layout/StatusBar.vue'
import ChatView from '@/views/ChatView.vue'
import GlobalModals from '@/components/layout/GlobalModals.vue'
</script>

<template>
  <div class="vscode-layout">
    <StatusBar />
    <ChatView />
    <GlobalModals />
  </div>
</template>

<style scoped>
.vscode-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background: var(--color-bg-primary);
}
</style>