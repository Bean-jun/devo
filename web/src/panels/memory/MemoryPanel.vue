<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMemoryStore } from '@/stores/memory'
import type { Memory, MemoryType } from '@/types/memory'
import AppIcon from '@/components/common/AppIcon.vue'

const memoryStore = useMemoryStore()

type TypeFilter = 'all' | MemoryType
const typeFilter = ref<TypeFilter>('all')
const error = ref<string | null>(null)
const showAdd = ref(false)
const newType = ref<MemoryType>('user')
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
      type: newType.value,
      key: newKey.value.trim(),
      content: newContent.value.trim(),
    })
    newType.value = 'user'
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

<style scoped>
.memory-panel { display: flex; flex-direction: column; height: 100%; background: var(--color-bg-primary); }

/* ---- Toolbar ---- */
.memory-toolbar { padding: 8px 12px; border-bottom: 1px solid var(--color-border); display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; }
.scope-filter { display: flex; gap: 4px; }
.filter-btn { padding: 4px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; transition: all var(--transition-fast); }
.filter-btn:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }
.filter-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.add-btn { height: 26px; padding: 0 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; display: flex; align-items: center; transition: all var(--transition-fast); }
.add-btn:hover { background: var(--color-bg-hover); color: var(--color-accent); border-color: var(--color-accent); }

/* ---- Add Form ---- */
.add-form { margin: 8px 12px; padding: 12px; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-bg-secondary); display: flex; flex-direction: column; gap: 10px; }
.form-row { display: flex; flex-direction: column; gap: 4px; }
.form-label { font-size: 11px; font-weight: 500; color: var(--color-text-tertiary); text-transform: uppercase; letter-spacing: 0.5px; }
.type-selector { display: flex; gap: 6px; }
.type-btn { flex: 1; padding: 6px 10px; border: 1px solid var(--color-border); border-radius: 6px; background: var(--color-bg-primary); color: var(--color-text-secondary); cursor: pointer; font-size: 12px; transition: all var(--transition-fast); display: flex; align-items: center; justify-content: center; gap: 4px; }
.type-btn:hover { border-color: var(--color-accent); color: var(--color-text-primary); }
.type-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.type-icon { font-size: 13px; }
.form-input { padding: 6px 10px; border: 1px solid var(--color-border); border-radius: 6px; background: var(--color-bg-primary); color: var(--color-text-primary); font-size: 12px; outline: none; transition: border-color var(--transition-fast); font-family: var(--font-mono); }
.form-input:focus { border-color: var(--color-accent); }
.form-input::placeholder { color: var(--color-text-tertiary); }
.form-textarea { padding: 6px 10px; border: 1px solid var(--color-border); border-radius: 6px; background: var(--color-bg-primary); color: var(--color-text-primary); font-size: 12px; resize: vertical; outline: none; transition: border-color var(--transition-fast); font-family: var(--font-mono); }
.form-textarea:focus { border-color: var(--color-accent); }
.form-textarea::placeholder { color: var(--color-text-tertiary); }
.add-actions { display: flex; gap: 8px; justify-content: flex-end; }
.btn-cancel { padding: 5px 14px; border: 1px solid var(--color-border); border-radius: 6px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 12px; transition: all var(--transition-fast); }
.btn-cancel:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }
.btn-save { padding: 5px 14px; border: none; border-radius: 6px; background: var(--color-accent); color: white; cursor: pointer; font-size: 12px; font-weight: 500; transition: opacity var(--transition-fast); }
.btn-save:hover { opacity: 0.85; }

/* ---- States ---- */
.memory-error { padding: 16px; color: var(--color-error); font-size: 12px; }
.memory-loading { padding: 16px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.memory-empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 6px; padding: 24px; }
.empty-icon { font-size: 32px; opacity: 0.4; }
.empty-text { font-size: 13px; color: var(--color-text-tertiary); }
.empty-hint { font-size: 11px; color: var(--color-text-tertiary); opacity: 0.6; }

/* ---- Memory List ---- */
.memory-list { flex: 1; overflow-y: auto; padding: 8px 12px; display: flex; flex-direction: column; gap: 6px; }

/* ---- Memory Card ---- */
.memory-card { display: flex; border: 1px solid var(--color-border); border-radius: 8px; overflow: hidden; transition: border-color var(--transition-fast), box-shadow var(--transition-fast); background: var(--color-bg-primary); }
.memory-card:hover { border-color: var(--color-accent); box-shadow: 0 1px 6px rgba(0,0,0,0.06); }
.card-accent { width: 3px; flex-shrink: 0; background: var(--color-border); transition: background var(--transition-fast); }
.type-user .card-accent { background: #60a5fa; }
.type-project .card-accent { background: #a78bfa; }
.card-body { flex: 1; padding: 10px 12px; min-width: 0; }
.memory-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; gap: 8px; }
.memory-key { font-size: 13px; font-weight: 600; color: var(--color-text-primary); font-family: var(--font-mono); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.memory-type-tag { font-size: 9px; padding: 1px 6px; border-radius: 3px; font-weight: 500; flex-shrink: 0; line-height: 1.6; }
.tag-user { background: rgba(96, 165, 250, 0.12); color: #60a5fa; }
.tag-project { background: rgba(167, 139, 250, 0.12); color: #a78bfa; }
.memory-content { font-size: 12px; color: var(--color-text-secondary); white-space: pre-wrap; word-break: break-word; margin-bottom: 8px; line-height: 1.5; }

/* ---- Edit ---- */
.memory-edit { display: flex; flex-direction: column; gap: 6px; margin-bottom: 8px; }
.edit-textarea { padding: 6px 10px; border: 1px solid var(--color-accent); border-radius: 6px; background: var(--color-bg-primary); color: var(--color-text-primary); font-size: 12px; resize: vertical; outline: none; font-family: var(--font-mono); width: 100%; box-sizing: border-box; }
.edit-actions { display: flex; gap: 6px; justify-content: flex-end; }
.btn-cancel-sm { padding: 3px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; transition: all var(--transition-fast); }
.btn-cancel-sm:hover { background: var(--color-bg-hover); }
.btn-save-sm { padding: 3px 10px; border: none; border-radius: 4px; background: var(--color-accent); color: white; cursor: pointer; font-size: 11px; font-weight: 500; transition: opacity var(--transition-fast); }
.btn-save-sm:hover { opacity: 0.85; }

/* ---- Footer ---- */
.memory-footer { display: flex; align-items: center; justify-content: space-between; }
.memory-time { font-size: 10px; color: var(--color-text-tertiary); }
.memory-actions { display: flex; gap: 4px; opacity: 0; transition: opacity var(--transition-fast); }
.memory-card:hover .memory-actions { opacity: 1; }
.action-btn { background: none; border: none; cursor: pointer; font-size: 11px; padding: 2px 8px; border-radius: 4px; transition: all var(--transition-fast); }
.edit-btn { color: var(--color-text-tertiary); }
.edit-btn:hover { color: var(--color-accent); background: var(--color-bg-hover); }
.delete-btn { color: var(--color-text-tertiary); }
.delete-btn:hover { color: var(--color-error); background: var(--color-bg-hover); }
</style>