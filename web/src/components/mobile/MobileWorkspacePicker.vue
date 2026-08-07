<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useMobileWorkspacePicker } from './MobileWorkspacePickerController'

const emit = defineEmits<{
  close: []
  switched: [path: string]
}>()

const {
  workspaces,
  activeWorkspace,
  selectWorkspace,
  onBackdropClick,
  onSheetClick,
  onTouchStart,
  onTouchMove,
} = useMobileWorkspacePicker(emit as any)
</script>

<template>
  <Teleport to="body">
    <div class="picker-overlay" @click.self="onBackdropClick">
      <div
        class="picker-sheet"
        data-test="workspace-picker"
        @click="onSheetClick"
        @touchstart.passive="onTouchStart"
        @touchmove.passive="onTouchMove"
      >
        <div class="sheet-drag-handle">
          <div class="drag-bar"></div>
        </div>
        <div class="picker-title">切换工作区</div>
        <div class="picker-list">
          <button
            v-for="ws in workspaces"
            :key="ws.id"
            class="picker-item"
            :class="{ active: activeWorkspace === ws.id, removed: !ws.exists }"
            :disabled="!ws.exists"
            @click="ws.exists && selectWorkspace(ws)"
          >
            <AppIcon :name="ws.exists ? 'folder' : 'trash'" :size="18" />
            <div class="picker-item-info">
              <span class="picker-item-name">{{ ws.name }}</span>
              <span class="picker-item-path">{{ ws.exists ? ws.path : '已移除' }}</span>
            </div>
            <AppIcon v-if="activeWorkspace === ws.id" name="circle" :size="10" weight="fill" class="picker-check" />
          </button>
          <div v-if="workspaces.length === 0" class="picker-empty">
            暂无工作区
          </div>
        </div>
        <button class="picker-cancel" @click="emit('close')">取消</button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped src="./MobileWorkspacePicker.css">
</style>