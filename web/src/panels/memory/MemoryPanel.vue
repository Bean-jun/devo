<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useMemoryPanel } from './MemoryPanelController'

const {
  memoryStore,
  typeFilter,
  error,
  showAdd,
  newType,
  newKey,
  newContent,
  editingId,
  editingContent,
  filteredMemories,
  handleAdd,
  startEdit,
  saveEdit,
  cancelEdit,
  handleDelete,
} = useMemoryPanel()
</script>

<template>
  <div class="memory-panel">
    <div class="memory-toolbar">
      <div class="scope-filter">
        <button class="filter-btn" :class="{ active: typeFilter === 'all' }" @click="typeFilter = 'all'">全部</button>
        <button class="filter-btn" :class="{ active: typeFilter === 'user' }" @click="typeFilter = 'user'">用户</button>
        <button class="filter-btn" :class="{ active: typeFilter === 'project' }" @click="typeFilter = 'project'">项目</button>
      </div>
      <button class="add-btn" @click="showAdd = !showAdd">{{ showAdd ? '收起' : '+ 新增' }}</button>
    </div>

    <div v-if="showAdd" class="add-form">
      <div class="form-row">
        <label class="form-label">类型</label>
        <div class="type-selector">
          <button class="type-btn" :class="{ active: newType === 'user' }" @click="newType = 'user'">
            <span class="type-icon"><AppIcon name="user" :size="14" /></span> 用户
          </button>
          <button class="type-btn" :class="{ active: newType === 'project' }" @click="newType = 'project'">
            <span class="type-icon"><AppIcon name="folder" :size="14" /></span> 项目
          </button>
        </div>
      </div>
      <div class="form-row">
        <label class="form-label">键名</label>
        <input v-model="newKey" placeholder="例如：editor_pref" class="form-input" @keyup.enter="handleAdd" />
      </div>
      <div class="form-row">
        <label class="form-label">内容</label>
        <textarea v-model="newContent" placeholder="记忆内容..." class="form-textarea" rows="2" @keyup.ctrl.enter="handleAdd"></textarea>
      </div>
      <div class="add-actions">
        <button class="btn-cancel" @click="showAdd = false">取消</button>
        <button class="btn-save" @click="handleAdd">保存</button>
      </div>
    </div>

    <div v-if="error" class="memory-error">{{ error }}</div>
    <div v-else-if="memoryStore.isLoading" class="memory-loading">加载中...</div>
    <div v-else-if="filteredMemories.length === 0" class="memory-empty">
      <div class="empty-icon"><AppIcon name="brain" :size="48" /></div>
      <div class="empty-text">暂无记忆</div>
      <div class="empty-hint">点击「+ 新增」添加第一条记忆</div>
    </div>
    <div v-else class="memory-list">
      <div v-for="mem in filteredMemories" :key="mem.id" class="memory-card" :class="'type-' + mem.type">
        <div class="card-accent"></div>
        <div class="card-body">
          <div class="memory-header">
            <span class="memory-key">{{ mem.key }}</span>
            <span class="memory-type-tag" :class="'tag-' + mem.type">{{ mem.type === 'user' ? '用户' : '项目' }}</span>
          </div>
          <div v-if="editingId === mem.id" class="memory-edit">
            <textarea v-model="editingContent" class="edit-textarea" rows="3"></textarea>
            <div class="edit-actions">
              <button class="btn-cancel-sm" @click="cancelEdit">取消</button>
              <button class="btn-save-sm" @click="saveEdit">保存</button>
            </div>
          </div>
          <div v-else class="memory-content">{{ mem.content }}</div>
          <div class="memory-footer">
            <span class="memory-time">{{ new Date(mem.updatedAt).toLocaleString() }}</span>
            <div class="memory-actions">
              <button class="action-btn edit-btn" @click="startEdit(mem)">编辑</button>
              <button class="action-btn delete-btn" @click="handleDelete(mem)">删除</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped src="./MemoryPanel.css">
</style>