<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
const sessionId = route.params.id as string
</script>

<template>
  <div class="placeholder-view">
    <h2>会话存档</h2>
    <p class="placeholder-text">会话 {{ sessionId }} 的存档将在 Phase 5 实现。</p>
  </div>
</template>

<style scoped>
.placeholder-view {
  padding: 24px;
}
.placeholder-view h2 {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 8px;
}
.placeholder-text {
  color: var(--color-text-secondary);
  font-size: 14px;
}
</style>