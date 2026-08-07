<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import ToolCallCard from './ToolCallCard.vue'
import type { Message } from '@/types/message'
import { useToolCallGroup } from './ToolCallGroupController'

const props = defineProps<{
  messages: Message[]
  yoloMode?: boolean
}>()

const {
  expanded,
  toolNames,
  summaryText,
  toggle,
} = useToolCallGroup(props)
</script>

<template>
  <div class="tool-call-group" :class="{ expanded }">
    <div class="group-header" @click="toggle">
      <AppIcon :name="expanded ? 'caret-down' : 'caret-right'" :size="12" class="group-chevron" />
      <AppIcon name="wrench" :size="16" class="group-icon" />
      <span class="group-title">{{ messages.length }} 个工具调用</span>
      <span class="group-tools">{{ toolNames.join(', ') }}</span>
      <span v-if="summaryText" class="group-summary">{{ summaryText }}</span>
    </div>

    <div v-if="expanded" class="group-body">
      <ToolCallCard
        v-for="msg in messages"
        :key="msg.id"
        :tool-call="msg.toolCall!"
        :yolo-mode="props.yoloMode"
      />
    </div>
  </div>
</template>

<style scoped src="./ToolCallGroup.css">
</style>