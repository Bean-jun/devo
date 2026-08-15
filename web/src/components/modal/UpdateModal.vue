<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useUpdateModal } from './UpdateModalController'

const {
  uiStore,
  isOpen,
  updateInfo,
  releaseBodyHtml,
  publishedDate,
  handleKeydown,
  openReleaseUrl,
} = useUpdateModal()
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click="uiStore.setActiveModal(null)" @keydown="handleKeydown">
    <div class="update-panel" @click.stop>
      <div class="panel-header">
        <h3>版本更新</h3>
        <button class="btn-close" @click="uiStore.setActiveModal(null)"><AppIcon name="x" :size="16" /></button>
      </div>

      <div v-if="updateInfo" class="panel-body">
        <div class="version-compare">
          <div class="version-item current">
            <span class="version-label">当前版本</span>
            <code class="version-value">{{ updateInfo.current_version }}</code>
          </div>
          <AppIcon name="arrow-right" :size="20" class="version-arrow" />
          <div class="version-item latest">
            <span class="version-label">最新版本</span>
            <code class="version-value highlight">{{ updateInfo.latest_version }}</code>
          </div>
        </div>

        <div v-if="updateInfo.release_name" class="release-name">
          {{ updateInfo.release_name }}
        </div>

        <div v-if="publishedDate" class="release-date">
          发布于 {{ publishedDate }}
        </div>

        <div v-if="releaseBodyHtml" class="release-body markdown-body" v-html="releaseBodyHtml" />

        <div class="panel-actions">
          <button class="btn-primary" @click="openReleaseUrl">
            <AppIcon name="arrow-square-out" :size="16" />
            在 GitHub 上查看
          </button>
          <button class="btn-secondary" @click="uiStore.setActiveModal(null)">稍后再说</button>
        </div>
      </div>

      <div v-else class="panel-body panel-empty">
        <p>暂无更新信息</p>
      </div>
    </div>
  </div>
</template>

<style scoped src="./UpdateModal.css">
</style>