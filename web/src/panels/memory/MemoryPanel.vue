<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMemoryStore } from '@/stores/memory'
import type { Memory, MemoryType } from '@/types/memory'

const memoryStore = useMemoryStore()

type TypeFilter = 'all' | MemoryType
const typeFilter = ref<TypeFilter>('all')
const error = ref<string | null>(null)
const showAdd = ref(false)
const newKey = ref('')
const newContent = ref('')
const editingId = ref<string | null>(null)
const editingContent = ref('')

const filteredMemories = computed<Memory[]>(() => {
  if (typeFilter.value === 'all') return memoryStore.memories
  return memoryStore.memories.filter(m => m.type === typeFilter.value)
})

async function handleAdd() {
  if (!newKey.value.trim() || !newContent.value.trim()) return
  try {
    await memoryStore.createMemory({
      type: 'user',
      key: newKey.value.trim(),
      content: newContent.value.trim(),
    })
    newKey.value = ''
    newContent.value = ''
    showAdd.value = false
  } catch (e: any) {
    error.value = e.message || '添加失败'
  }
}

function startEdit(mem: Memory) {
  editingId.value = mem.id
  editingContent.value = mem.content
}

async function saveEdit() {
  if (!editingId.value || !editingContent.value.trim()) return
  try {
    await memoryStore.updateMemory(editingId.value, editingContent.value.trim())
    editingId.value = null
    editingContent.value = ''
  } catch (e: any) {
    error.value = e.message || '保存失败'
  }
}

function cancelEdit() {
  editingId.value = null
  editingContent.value = ''
}

async function handleDelete(mem: Memory) {
  try {
    await memoryStore.deleteMemory(mem.id)
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
        <button class="filter-btn" :class="{ active: typeFilter === 'all' }" @click="typeFilter = 'all'">全部</button>
        <button class="filter-btn" :class="{ active: typeFilter === 'user' }" @click="typeFilter = 'user'">用户</button>
        <button class="filter-btn" :class="{ active: typeFilter === 'project' }" @click="typeFilter = 'project'">项目</button>
      </div>
      <button class="add-btn" @click="showAdd = true">+ 新增</button>
    </div>

    <div v-if="showAdd" class="add-form">
      <input v-model="newKey" placeholder="键名" class="add-input" />
      <textarea v-model="newContent" placeholder="内容" class="add-textarea" rows="3"></textarea>
      <div class="add-actions">
        <button class="btn-cancel" @click="showAdd = false">取消</button>
        <button class="btn-save" @click="handleAdd">保存</button>
      </div>
    </div>

    <div v-if="error" class="memory-error">{{ error }}</div>
    <div v-else-if="memoryStore.isLoading" class="memory-loading">加载中...</div>
    <div v-else-if="filteredMemories.length === 0" class="memory-empty">暂无记忆</div>
    <div v-else class="memory-list">
      <div v-for="mem in filteredMemories" :key="mem.id" class="memory-card">
        <div class="memory-header">
          <span class="memory-key">{{ mem.key }}</span>
          <span class="memory-type-tag">{{ mem.type === 'user' ? '用户' : '项目' }}</span>
        </div>
        <div v-if="editingId === mem.id" class="memory-edit">
          <textarea v-model="editingContent" class="edit-textarea" rows="3"></textarea>
          <div class="edit-actions">
            <button class="btn-cancel" @click="cancelEdit">取消</button>
            <button class="btn-save" @click="saveEdit">保存</button>
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
</template>

<style scoped>
.memory-panel { display: flex; flex-direction: column; height: 100%; }
.memory-toolbar { padding: 8px 12px; border-bottom: 1px solid var(--color-border); display: flex; align-items: center; justify-content: space-between; }
.scope-filter { display: flex; gap: 4px; }
.filter-btn { padding: 4px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; transition: all var(--transition-fast); }
.filter-btn:hover { background: var(--color-bg-hover); }
.filter-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.add-btn { padding: 4px 10px; border: 1px solid var(--color-accent); border-radius: 4px; background: transparent; color: var(--color-accent); cursor: pointer; font-size: 11px; }
.add-form { padding: 8px 12px; display: flex; flex-direction: column; gap: 6px; border-bottom: 1px solid var(--color-border); }
.add-input { padding: 6px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-primary); color: var(--color-text-primary); font-size: 12px; }
.add-textarea { padding: 6px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-primary); color: var(--color-text-primary); font-size: 12px; resize: vertical; font-family: var(--font-mono); }
.add-actions { display: flex; gap: 6px; justify-content: flex-end; }
.btn-cancel { padding: 4px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; }
.btn-save { padding: 4px 10px; border: none; border-radius: 4px; background: var(--color-accent); color: white; cursor: pointer; font-size: 11px; }
.memory-error { padding: 16px; color: var(--color-error); font-size: 12px; }
.memory-loading { padding: 16px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.memory-empty { padding: 16px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.memory-list { flex: 1; overflow-y: auto; padding: 8px; }
.memory-card { border: 1px solid var(--color-border); border-radius: 6px; padding: 8px; margin-bottom: 6px; }
.memory-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.memory-key { font-size: 13px; font-weight: 600; color: var(--color-text-primary); font-family: var(--font-mono); }
.memory-type-tag { font-size: 10px; padding: 2px 6px; border-radius: 3px; background: var(--color-bg-tertiary); color: var(--color-text-tertiary); }
.memory-content { font-size: 12px; color: var(--color-text-secondary); white-space: pre-wrap; word-break: break-all; margin-bottom: 6px; }
.memory-edit { display: flex; flex-direction: column; gap: 6px; }
.edit-textarea { padding: 6px 8px; border: 1px solid var(--color-accent); border-radius: 4px; background: var(--color-bg-primary); color: var(--color-text-primary); font-size: 12px; resize: vertical; font-family: var(--font-mono); }
.edit-actions { display: flex; gap: 6px; justify-content: flex-end; }
.memory-footer { display: flex; align-items: center; justify-content: space-between; }
.memory-time { font-size: 10px; color: var(--color-text-tertiary); }
.memory-actions { display: flex; gap: 6px; }
.action-btn { background: none; border: none; cursor: pointer; font-size: 11px; padding: 2px 6px; border-radius: 3px; transition: all var(--transition-fast); }
.edit-btn { color: var(--color-accent); }
.edit-btn:hover { background: var(--color-bg-hover); }
.delete-btn { color: var(--color-error); }
.delete-btn:hover { background: var(--color-bg-hover); }
</style>