import { ref, computed, onMounted } from 'vue'
import { useMemoryStore } from '@/stores/memory'
import type { Memory, MemoryType } from '@/types/memory'
import AppIcon from '@/components/common/AppIcon.vue'

type TypeFilter = 'all' | MemoryType

export function useMemoryPanel() {
  const memoryStore = useMemoryStore()

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

  return {
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
  }
}