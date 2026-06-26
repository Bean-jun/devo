<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMemoryStore } from '@/stores/memory'
import { useUiStore } from '@/stores/ui'
import type { Memory } from '@/types/memory'

const memoryStore = useMemoryStore()
const uiStore = useUiStore()

type ScopeFilter = 'all' | 'global' | 'workspace'
const scopeFilter = ref<ScopeFilter>('all')
const error = ref<string | null>(null)
const showAdd = ref(false)
const newKey = ref('')
const newValue = ref('')
const editingKey = ref<string | null>(null)
const editingValue = ref('')

const filteredMemories = computed<Memory[]>(() => {
  let list = memoryStore.memories
  if (scopeFilter.value === 'global') {
    list = memoryStore.globalMemories
  } else if (scopeFilter.value === 'workspace' && uiStore.activeWorkspace) {
    list = memoryStore.workspaceMemories(uiStore.activeWorkspace)
  }
  return list
})

async function handleAdd() {
  if (!newKey.value.trim() || !newValue.value.trim()) return
  try {
    await memoryStore.createMemory({
      key: newKey.value.trim(),
      value: newValue.value.trim(),
      scope: uiStore.activeWorkspace ? `workspace:${uiStore.activeWorkspace}` : 'global',
    })
    newKey.value = ''
    newValue.value = ''
    showAdd.value = false
  } catch (e: any) {
    error.value = e.message || '添加失败'
  }
}

function startEdit(mem: Memory) {
  editingKey.value = mem.key
  editingValue.value = mem.value
}

async function saveEdit() {
  if (!editingKey.value || !editingValue.value.trim()) return
  try {
    await memoryStore.updateMemory(editingKey.value, editingValue.value.trim())
    editingKey.value = null
    editingValue.value = ''
  } catch (e: any) {
    error.value = e.message || '保存失败'
  }
}

function cancelEdit() {
  editingKey.value = null
  editingValue.value = ''
}

async function handleDelete(mem: Memory) {
  try {
    await memoryStore.deleteMemory(mem.key, mem.scope)
  } catch (e: any) {
    error.value = e.message || '删除失败'
  }
}

onMounted(async () => {
  try {
    await memoryStore.fetchMemories()
  } catch (e: any) {
    error.value = e.message
  }
})
</script>

<template>
  <div class="memory-panel">
    <div class="memory-toolbar">
      <div class="scope-filter">
        <button class="filter-btn" :class="{ active: scopeFilter === 'all' }" @click="scopeFilter = 'all'">全部</button>
        <button class="filter-btn" :class="{ active: scopeFilter === 'global' }" @click="scopeFilter = 'global'">全局</button>
        <button class="filter-btn" :class="{ active: scopeFilter === 'workspace' }" @click="scopeFilter = 'workspace'">项目</button>
      </div>
      <button class="add-btn" @click="showAdd = true">+ 新增</button>
    </div>

    <div v-if="showAdd" class="add-form">
      <input v-model="newKey" placeholder="键名" class="add-input" />
      <textarea v-model="newValue" placeholder="值" class="add-textarea" rows="3"></textarea>
      <div class="add-actions">
        <button class="btn-cancel" @click="showAdd = false">取消</button>
        <button class="btn-save" @click="handleAdd">保存</button>
      </div>
    </div>

    <div v-if="error" class="memory-error">{{ error }}</div>
    <div v-else-if="memoryStore.isLoading" class="memory-loading">加载中...</div>
    <div v-else-if="filteredMemories.length === 0" class="memory-empty">暂无记忆</div>
    <div v-else class="memory-list">
      <div v-for="mem in filteredMemories" :key="mem.key + ':' + mem.scope" class="memory-card">
        <div v-if="editingKey === mem.key" class="memory-edit">
          <textarea v-model="editingValue" class="edit-textarea" rows="3"></textarea>
          <div class="edit-actions">
            <button class="btn-cancel" @click="cancelEdit">取消</button>
            <button class="btn-save" @click="saveEdit">保存</button>
          </div>
        </div>
        <div v-else class="memory-view">
          <div class="memory-header">
            <span class="memory-key">{{ mem.key }}</span>
            <span class="memory-scope">{{ mem.scope === 'global' ? '全局' : '项目' }}</span>
          </div>
          <div class="memory-value">{{ mem.value }}</div>
          <div class="memory-actions">
            <button class="edit-btn" @click="startEdit(mem)">编辑</button>
            <button class="delete-btn" @click="handleDelete(mem)">删除</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.memory-panel { display: flex; flex-direction: column; height: 100%; }
.memory-toolbar { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.scope-filter { display: flex; gap: 4px; }
.filter-btn { padding: 4px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; }
.filter-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.add-btn { padding: 4px 12px; background: var(--color-accent); color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 11px; }
.add-form { padding: 12px; border-bottom: 1px solid var(--color-border); display: flex; flex-direction: column; gap: 8px; }
.add-input, .add-textarea, .edit-textarea { width: 100%; padding: 6px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-primary); color: var(--color-text-primary); font-size: 12px; font-family: inherit; resize: vertical; }
.add-actions, .edit-actions { display: flex; justify-content: flex-end; gap: 8px; }
.btn-cancel { padding: 4px 12px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; }
.btn-save { padding: 4px 12px; background: var(--color-accent); color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 11px; }
.memory-error { padding: 16px; color: var(--color-error); font-size: 12px; }
.memory-loading { padding: 16px; color: var(--color-text-tertiary); text-align: center; }
.memory-empty { padding: 16px; color: var(--color-text-tertiary); text-align: center; font-size: 12px; }
.memory-list { flex: 1; overflow-y: auto; padding: 8px; }
.memory-card { border: 1px solid var(--color-border); border-radius: 6px; margin-bottom: 8px; overflow: hidden; }
.memory-view { padding: 10px; }
.memory-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.memory-key { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.memory-scope { font-size: 10px; padding: 2px 6px; border-radius: 3px; background: var(--color-bg-secondary); color: var(--color-text-tertiary); }
.memory-value { font-size: 12px; color: var(--color-text-secondary); white-space: pre-wrap; word-break: break-word; }
.memory-actions { display: flex; gap: 8px; margin-top: 8px; }
.edit-btn { padding: 2px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; }
.delete-btn { padding: 2px 8px; border: 1px solid var(--color-error); border-radius: 4px; background: transparent; color: var(--color-error); cursor: pointer; font-size: 11px; }
.memory-edit { padding: 10px; display: flex; flex-direction: column; gap: 8px; }
</style>