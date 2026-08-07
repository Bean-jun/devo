<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import ConfirmDeleteDialog from '@/components/modal/ConfirmDeleteDialog.vue'
import { useSkillsPanel } from './SkillsPanelController'

const {
  skillsStore,
  sourceFilter,
  error,
  showDeleteDialog,
  deletingSkill,
  showInstallDialog,
  installPath,
  installing,
  installError,
  filteredSkills,
  handleToggle,
  handleDelete,
  confirmDelete,
  cancelDelete,
  handleReload,
  openInstallDialog,
  closeInstallDialog,
  handleInstall,
  sourceLabel,
} = useSkillsPanel()
</script>

<template>
  <div class="skills-panel">
    <div class="skills-toolbar">
      <div class="scope-filter">
        <button class="filter-btn" :class="{ active: sourceFilter === 'all' }" @click="sourceFilter = 'all'">全部</button>
        <button class="filter-btn" :class="{ active: sourceFilter === 'project' }" @click="sourceFilter = 'project'">项目</button>
        <button class="filter-btn" :class="{ active: sourceFilter === 'global' }" @click="sourceFilter = 'global'">全局</button>
      </div>
      <div class="toolbar-right">
        <button class="reload-btn" :class="{ spinning: skillsStore.isLoading }" :disabled="skillsStore.isLoading" @click="handleReload" title="重新扫描技能目录">
          <AppIcon name="arrow-clockwise" :size="16" />
        </button>
        <button class="add-btn" @click="openInstallDialog" title="安装技能">
          添加
        </button>
      </div>
    </div>
    <div v-if="error" class="skills-error">{{ error }}</div>
    <div v-else-if="skillsStore.isLoading" class="skills-loading">加载中...</div>
    <div v-else-if="filteredSkills.length === 0" class="skills-empty">暂无技能</div>
    <div v-else class="skills-list">
      <div v-for="skill in filteredSkills" :key="skill.name" class="skill-card">
        <div class="skill-info">
          <div class="skill-detail">
            <span class="skill-name">{{ skill.name }}</span>
            <span class="skill-source">{{ sourceLabel(skill.source) }} · {{ skill.location || '内置' }}</span>
          </div>
        </div>
        <div class="skill-actions">
          <span class="skill-source-tag" :class="'source-' + skill.source">{{ sourceLabel(skill.source) }}</span>
          <button class="skill-toggle" :class="{ active: skill.enabled }" @click="handleToggle(skill)">
            {{ skill.enabled ? 'ON' : 'OFF' }}
          </button>
          <button class="skill-delete" @click="handleDelete(skill)"><AppIcon name="x" :size="12" /></button>
        </div>
      </div>
    </div>
  </div>

  <ConfirmDeleteDialog
    :visible="showDeleteDialog"
    :server-name="deletingSkill?.name ?? ''"
    :deleting="false"
    entity-type="技能"
    @confirm="confirmDelete"
    @cancel="cancelDelete"
  />

  <div v-if="showInstallDialog" class="dialog-overlay" @click.self="closeInstallDialog">
    <div class="install-dialog">
      <div class="dialog-header">
        <h3 class="dialog-title">安装技能</h3>
      </div>
      <div class="dialog-body">
        <p class="dialog-hint">输入本地技能目录路径，该目录需包含 SKILL.md 文件。</p>
        <input
          v-model="installPath"
          type="text"
          class="install-input"
          placeholder="例如：/home/user/my-skill"
          @keyup.enter="handleInstall"
        />
        <p v-if="installError" class="install-error">{{ installError }}</p>
      </div>
      <div class="dialog-footer">
        <button class="dialog-btn cancel-btn" @click="closeInstallDialog">取消</button>
        <button class="dialog-btn confirm-btn" :disabled="installing" @click="handleInstall">
          {{ installing ? '安装中...' : '安装' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped src="./SkillsPanel.css">
</style>