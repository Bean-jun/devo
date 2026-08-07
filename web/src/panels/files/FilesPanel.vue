<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import HighlightPreview from '@/components/editor/HighlightPreview.vue'
import { useFilesPanel } from './FilesPanelController'

const {
  panelRef,
  isLoading,
  fetchRoot,
  hasWorkspace,
  error,
  flatNodes,
  selectedPath,
  handleNodeClick,
  expandedPaths,
  getFileIcon,
  formatFileSize,
  startPreviewResize,
  previewHeight,
  closePreview,
  isLoadingContent,
  selectedIsImage,
  selectedContent,
  isPreviewError,
  selectedLanguage,
} = useFilesPanel()
</script>

<template>
  <div ref="panelRef" class="files-panel">
    <div class="files-toolbar">
      <span class="files-title">Files</span>
      <button class="reload-btn" :class="{ spinning: isLoading }" :disabled="isLoading" @click="fetchRoot" title="刷新">
        <AppIcon name="arrow-clockwise" :size="16" />
      </button>
    </div>

    <div v-if="!hasWorkspace" class="files-empty">请先选择一个工作区</div>
    <div v-else-if="error" class="files-error">{{ error }}</div>
    <div v-else class="files-body">
      <div class="files-tree">
        <div v-if="isLoading" class="files-loading">加载中...</div>
        <div v-else-if="flatNodes.length === 0" class="files-empty">目录为空</div>
        <div
          v-for="node in flatNodes"
          :key="node.path"
          class="file-row"
          :class="{ selected: selectedPath === node.path }"
          :style="{
            paddingLeft: (8 + node.depth * 22) + 'px',
            '--depth': node.depth,
          }"
          @click="handleNodeClick(node)"
        >
          <span
            v-for="d in node.depth"
            :key="d"
            class="tree-line"
            :style="{ left: (8 + (d - 1) * 22 + 8) + 'px' }"
          ></span>
          <AppIcon :name="expandedPaths.has(node.path) ? 'caret-down' : 'caret-right'" :size="12" class="file-chevron" />
          <AppIcon :name="getFileIcon(node.type, node.name)" :size="16" class="file-icon" />
          <span class="file-name">{{ node.name }}</span>
          <AppIcon v-if="node.type === 'dir' && !node.loaded && expandedPaths.has(node.path)" name="hourglass" :size="12" class="file-loading-icon" />
          <span class="file-size" v-if="node.type === 'file'">{{ formatFileSize(node.size) }}</span>
        </div>
      </div>

      <div
        v-if="selectedPath"
        class="preview-resize-handle"
        @mousedown="startPreviewResize"
      ></div>
      <div v-if="selectedPath" class="file-preview" :style="{ height: previewHeight + 'px' }">
        <div class="preview-header">
          <span class="preview-title">{{ selectedPath }}</span>
          <button class="preview-close" @click="closePreview"><AppIcon name="x" :size="14" /></button>
        </div>
        <div class="preview-body">
          <div v-if="isLoadingContent" class="preview-loading">加载中...</div>
          <img v-else-if="selectedIsImage" :src="selectedContent ?? ''" class="preview-image" />
          <pre v-else-if="isPreviewError" class="preview-content"><code>{{ selectedContent }}</code></pre>
          <HighlightPreview
            v-else
            :content="selectedContent ?? ''"
            :language="selectedLanguage"
            :auto-height="true"
            :min-height="80"
            :max-height="600"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped src="./FilesPanel.css">
</style>